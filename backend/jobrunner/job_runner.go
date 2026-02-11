package jobrunner

import (
	"context"
	"database/sql"
	"encoding/json"
	"etl/dialects"
	"etl/logger"
	"etl/models"
	"etl/status"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

type JobRunner struct {
	SourceDB       *sql.DB
	DestinationDB  *sql.DB
	SourceDSN      string
	DestinationDSN string
	Dialect        dialects.SQLDialect
	Concurrency    int
	Semaphore      chan struct{}
	WaitGroup      *sync.WaitGroup
	JobMap         map[string]models.Job
	ConnMap        map[string][]string
	JobOrder       []string
	Variables      map[string]string
	PipelineLog    *logger.PipelineLog // Adicionado para logging
	ProjectID      string
	ctx            context.Context
	cancel         context.CancelFunc
	stopped        atomic.Bool
	stopReason     atomic.Value
	countMu        sync.Mutex
	countMap       map[string]*countFuture
	countQueue     chan *countRequest
	countInit      sync.Once
	memoryStoreMu  sync.RWMutex
	memoryStore    map[string]memoryDataset
}

func NewJobRunner(sourceDB, destDB *sql.DB, sourceDSN, destDSN string, dialect dialects.SQLDialect, concurrency int, project string, projectID string) *JobRunner {
	// Inicializa o log do pipeline
	pipelineLog := &logger.PipelineLog{
		PipelineID: logger.GeneratePipelineID(project),
		ProjectID:  projectID,
		Project:    project,
		Status:     "running",
		StartedAt:  time.Now(),
		Jobs:       make([]logger.JobLog, 0),
	}

	log.Printf("Criando JobRunner com Pipeline ID: %s para projeto: %s\n", pipelineLog.PipelineID, project)

	jr := &JobRunner{
		SourceDB:       sourceDB,
		DestinationDB:  destDB,
		SourceDSN:      sourceDSN,
		DestinationDSN: destDSN,
		Dialect:        dialect,
		Concurrency:    concurrency,
		Semaphore:      make(chan struct{}, concurrency),
		WaitGroup:      &sync.WaitGroup{},
		JobMap:         make(map[string]models.Job),
		ConnMap:        make(map[string][]string),
		JobOrder:       make([]string, 0),
		Variables:      LoadProjectVariables(projectID),
		PipelineLog:    pipelineLog,
		ProjectID:      projectID,
		memoryStore:    make(map[string]memoryDataset),
	}
	jr.ctx, jr.cancel = context.WithCancel(context.Background())

	// Salva o log inicial do pipeline
	jr.savePipelineLog()

	return jr
}

func (jr *JobRunner) initCountManager() {
	jr.countInit.Do(func() {
		queueSize := len(jr.JobMap)
		if queueSize < 1 {
			queueSize = 1
		}
		jr.countMap = make(map[string]*countFuture)
		jr.countQueue = make(chan *countRequest, queueSize)
		go jr.countWorker()
	})
}

func (jr *JobRunner) countWorker() {
	for req := range jr.countQueue {
		total, err := jr.Dialect.FetchTotalCount(jr.SourceDB, req.job)
		req.future.total = total
		req.future.err = err
		close(req.future.done)
		status.IncCountDone()
	}
}

func (jr *JobRunner) requestCount(jobID string, job models.Job) *countFuture {
	jr.initCountManager()
	jr.countMu.Lock()
	if future, ok := jr.countMap[jobID]; ok {
		jr.countMu.Unlock()
		return future
	}
	future := &countFuture{done: make(chan struct{})}
	jr.countMap[jobID] = future
	jr.countMu.Unlock()

	jr.countQueue <- &countRequest{
		jobID:  jobID,
		job:    job,
		future: future,
	}
	return future
}

func (jr *JobRunner) awaitCount(future *countFuture) (int, error) {
	<-future.done
	return future.total, future.err
}

func (jr *JobRunner) buildExecutionOrder(startIDs []string) []string {
	reachable := make(map[string]struct{})
	queue := make([]string, 0, len(startIDs))
	queue = append(queue, startIDs...)

	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		if _, seen := reachable[id]; seen {
			continue
		}
		reachable[id] = struct{}{}
		for _, nextID := range jr.ConnMap[id] {
			queue = append(queue, nextID)
		}
	}

	indegree := make(map[string]int, len(reachable))
	for id := range reachable {
		indegree[id] = 0
	}
	for src, targets := range jr.ConnMap {
		if _, ok := reachable[src]; !ok {
			continue
		}
		for _, tgt := range targets {
			if _, ok := reachable[tgt]; !ok {
				continue
			}
			indegree[tgt]++
		}
	}

	pending := make([]string, 0, len(reachable))
	for _, id := range startIDs {
		if _, ok := reachable[id]; ok && indegree[id] == 0 {
			pending = append(pending, id)
		}
	}

	if len(pending) == 0 {
		for _, id := range jr.JobOrder {
			if _, ok := reachable[id]; ok && indegree[id] == 0 {
				pending = append(pending, id)
			}
		}
	}

	order := make([]string, 0, len(reachable))
	visited := make(map[string]struct{}, len(reachable))
	for len(pending) > 0 {
		id := pending[0]
		pending = pending[1:]
		if _, seen := visited[id]; seen {
			continue
		}
		visited[id] = struct{}{}
		order = append(order, id)
		for _, nextID := range jr.ConnMap[id] {
			if _, ok := reachable[nextID]; !ok {
				continue
			}
			indegree[nextID]--
			if indegree[nextID] == 0 {
				pending = append(pending, nextID)
			}
		}
	}

	if len(order) < len(reachable) {
		for _, id := range jr.JobOrder {
			if _, ok := reachable[id]; ok {
				if _, seen := visited[id]; !seen {
					order = append(order, id)
				}
			}
		}
	}

	return order
}

func (jr *JobRunner) preloadCounts(startIDs []string) {
	jr.initCountManager()
	order := jr.buildExecutionOrder(startIDs)
	status.ResetCountStatus()
	total := 0
	for _, jobID := range order {
		job, ok := jr.JobMap[jobID]
		if !ok {
			continue
		}
		if strings.ToLower(job.Type) != "insert" {
			continue
		}
		jobCopy := job
		jobCopy.SelectSQL = jr.SubstituteVariables(jobCopy.SelectSQL)
		_, directives, err := extractMapDirectives(jobCopy.SelectSQL)
		if err == nil && len(directives) > 0 {
			// Jobs insert com diretiva Map fazem count na execucao, em sessao propria.
			continue
		}
		total++
	}
	status.SetCountTotal(total)
	for _, jobID := range order {
		job, ok := jr.JobMap[jobID]
		if !ok {
			continue
		}
		if strings.ToLower(job.Type) != "insert" {
			continue
		}
		jobCopy := job
		jobCopy.SelectSQL = jr.SubstituteVariables(jobCopy.SelectSQL)
		jobCopy.InsertSQL = jr.SubstituteVariables(jobCopy.InsertSQL)
		jobCopy.PostInsert = jr.SubstituteVariables(jobCopy.PostInsert)
		_, directives, err := extractMapDirectives(jobCopy.SelectSQL)
		if err == nil && len(directives) > 0 {
			continue
		}
		jr.requestCount(jobID, jobCopy)
	}
}

type countFuture struct {
	done  chan struct{}
	total int
	err   error
}

type countRequest struct {
	jobID  string
	job    models.Job
	future *countFuture
}

func (jr *JobRunner) RunJob(jobID string) {
	if jr.shouldStop() {
		return
	}
	job, ok := jr.JobMap[jobID]
	if !ok {
		log.Printf("Job %s não encontrado\n", jobID)
		return
	}

	println("tipo de job:", job.Type)
	println("nome do job:", job.JobName)

	switch strings.ToLower(job.Type) {
	case "insert":
		jr.runInsertJob(jobID, job)
	case "execution":
		jr.runExecutionJob(jobID, job)
	case "update":
		jr.runExecutionJob(jobID, job)
	case "condition":
		jr.runConditionJob(jobID, job)
	case "memory-select":
		jr.runMemorySelectJob(jobID, job)
	default:
		log.Printf("Tipo de job desconhecido: %s", job.Type)
		// Log de erro para tipo desconhecido
		jobLog := logger.JobLog{
			JobID:       jobID,
			JobName:     job.JobName,
			Status:      "error",
			Error:       fmt.Sprintf("Tipo de job desconhecido: %s", job.Type),
			StopOnError: job.StopOnError,
			StartedAt:   time.Now(),
			EndedAt:     time.Now(),
			Processed:   0,
			Total:       0,
			Batches:     make([]logger.BatchLog, 0),
		}
		logger.AddJob(jr.PipelineLog, jobLog)
		jr.savePipelineLog()
	}
}

func (jr *JobRunner) runInsertJob(jobID string, job models.Job) {
	log.Printf("Iniciando job com leitura e escrita paralela via hash: %s", job.JobName)

	jr.WaitGroup.Add(1)
	go func() {
		defer jr.WaitGroup.Done()
		if jr.shouldStop() {
			jr.markJobFinalStatus(jobID, job, "error", "pipeline interrompida", time.Now())
			return
		}

		start := time.Now()
		jobLog := logger.JobLog{
			JobID:       jobID,
			JobName:     job.JobName,
			Status:      "running",
			StopOnError: job.StopOnError,
			StartedAt:   start,
			Processed:   0,
			Batches:     make([]logger.BatchLog, 0),
		}
		logger.AddJob(jr.PipelineLog, jobLog)
		jr.savePipelineLog()

		status.UpdateJobStatus(job.ID, func(js *status.JobStatus) {
			js.Name = job.JobName
			js.Status = "running"
			js.StartedAt = &start
			status.NotifySubscribers()
		})

		job.SelectSQL = jr.SubstituteVariables(job.SelectSQL)
		job.InsertSQL = jr.SubstituteVariables(job.InsertSQL)
		job.PostInsert = jr.SubstituteVariables(job.PostInsert)

		resolvedSelectSQL, mapDirectives, err := extractMapDirectives(job.SelectSQL)
		if err != nil {
			log.Printf("Erro ao processar diretivas Map no job %s: %v\n", job.ID, err)
			jr.markJobFinalStatus(jobID, job, "error", err.Error(), time.Now())
			return
		}
		job.SelectSQL = resolvedSelectSQL
		if len(mapDirectives) > 0 {
			log.Printf("Job %s (%s): %d diretiva(s) Map detectadas no select", job.ID, job.JobName, len(mapDirectives))
		}

		// total de registros (count calculado em worker sequencial)
		total := 0
		if len(mapDirectives) > 0 {
			countStart := time.Now()
			total, err = jr.countSelectWithMapDirectives(job, mapDirectives)
			if err != nil {
				log.Printf("Erro ao contar registros com Map: %v\n", err)
				jr.markJobFinalStatus(jobID, job, "error", err.Error(), time.Now())
				return
			}
			log.Printf("Job %s (%s): count com Map concluido em %s (total=%d)", job.ID, job.JobName, time.Since(countStart), total)
		} else {
			countFuture := jr.requestCount(jobID, job)
			total, err = jr.awaitCount(countFuture)
			if err != nil {
				log.Printf("Erro ao contar registros: %v\n", err)
				jr.markJobFinalStatus(jobID, job, "error", err.Error(), time.Now())
				return
			}
		}

		// Atualiza total no log
		logger.UpdateJob(jr.PipelineLog, jobID, func(jl *logger.JobLog) {
			jl.Total = total
		})
		jr.savePipelineLog()

		status.UpdateJobStatus(job.ID, func(js *status.JobStatus) {
			js.Total = total
			status.NotifySubscribers()
		})

		batchChan := make(chan []map[string]interface{}, jr.Concurrency*5)

		// Ajusta concurrency: se total < batchSize, usa apenas 1 worker
		concurrency := jr.Concurrency
		if total <= job.RecordsPerPage {
			concurrency = 1
			log.Printf("Total (%d) menor que batchSize (%d), usando apenas 1 worker", total, job.RecordsPerPage)
		}

		writerConcurrency := jr.Concurrency
		if writerConcurrency < 1 {
			writerConcurrency = 1
		}

		if concurrency > 0 || writerConcurrency > 0 {
			status.AddWorkerTotals(concurrency, writerConcurrency)
			defer status.AddWorkerTotals(-concurrency, -writerConcurrency)
		}

		// Controle de execucao do job
		var processed int64
		var jobHadError atomic.Bool
		var lastErr atomic.Value
		reportJobError := func(errMsg string) {
			status.UpdateJobStatus(job.ID, func(js *status.JobStatus) {
				js.Error = errMsg
				if job.StopOnError {
					js.Status = "error"
					now := time.Now()
					js.EndedAt = &now
				}
				status.NotifySubscribers()
			})
			status.AppendLog(fmt.Sprintf("%s - Job: %s falhou: %s", jr.PipelineLog.Project, job.JobName, errMsg))
		}

		jobCtx, jobCancel := context.WithCancel(jr.ctx)
		defer jobCancel()

		closeOnce := &sync.Once{}
		closeBatch := func() {
			closeOnce.Do(func() {
				close(batchChan)
			})
		}

		// Writers paralelos com transacao independente por writer
		var writerWG sync.WaitGroup
		for w := 0; w < writerConcurrency; w++ {
			writerWG.Add(1)
			go func() {
				defer writerWG.Done()
				status.AddWorkerActive(0, 1)
				defer status.AddWorkerActive(0, -1)

				tx, err := jr.DestinationDB.Begin()
				if err != nil {
					jobHadError.Store(true)
					lastErr.Store(err.Error())
					reportJobError(err.Error())
					jobCancel()
					return
				}

				committed := false
				defer func() {
					if !committed {
						_ = tx.Rollback()
					}
				}()

				for batch := range batchChan {
					if jobHadError.Load() || jr.shouldStop() || jobCtx.Err() != nil {
						return
					}
					batchStart := time.Now()
					startOffset := int(atomic.LoadInt64(&processed))
					batchLog := logger.BatchLog{
						Offset:    startOffset,
						Limit:     len(batch),
						Status:    "running",
						StartedAt: batchStart,
					}

					insertSQL, args := jr.Dialect.BuildInsertQuery(job, batch)

					if _, err := tx.Exec(insertSQL, args...); err != nil {
						jobHadError.Store(true)
						lastErr.Store(err.Error())
						reportJobError(err.Error())

						analyzer := &logger.ErrorAnalyzer{}
						errorType, errorCode, _ := analyzer.AnalyzeError(err)

						batchLog.Status = "error"
						batchLog.Error = err.Error()
						batchLog.ErrorType = errorType
						batchLog.ErrorCode = errorCode
						batchLog.EndedAt = time.Now()
						logger.AddBatch(jr.PipelineLog, jobID, batchLog)
						jr.savePipelineLog()
						jobCancel()
						return
					}

					atomic.AddInt64(&processed, int64(len(batch)))
					// Mantem o contador do job sincronizado com o log do pipeline
					logger.UpdateJob(jr.PipelineLog, jobID, func(jl *logger.JobLog) {
						jl.Processed = int(atomic.LoadInt64(&processed))
					})
					batchLog.Status = "done"
					batchLog.Rows = len(batch)
					batchLog.EndedAt = time.Now()
					logger.AddBatch(jr.PipelineLog, jobID, batchLog)
					jr.savePipelineLog()

					status.UpdateJobStatus(job.ID, func(js *status.JobStatus) {
						js.Processed = int(processed)
						js.Progress = float64(processed) / float64(total) * 100
						status.NotifySubscribers()
					})
				}

				if jobHadError.Load() || jr.shouldStop() || jobCtx.Err() != nil {
					return
				}

				if err := tx.Commit(); err != nil {
					jobHadError.Store(true)
					lastErr.Store(err.Error())
					reportJobError(err.Error())
					return
				}
				committed = true
			}()
		}

		// Resolve tabela principal uma única vez (EXPLAIN)
		mainTable := ""
		if len(mapDirectives) > 0 {
			explainStart := time.Now()
			mainTable, err = jr.getMainTableFromExplainWithMapDirectives(job, mapDirectives, jobCtx)
			log.Printf("Job %s (%s): explain com Map concluido em %s (mainTable=%s)", job.ID, job.JobName, time.Since(explainStart), mainTable)
		} else {
			mainTable, err = jr.getMainTableFromExplain(job)
		}
		if err != nil {
			log.Printf("Erro no EXPLAIN do job %s: %v", job.ID, err)
			jobHadError.Store(true)
			lastErr.Store(err.Error())
			reportJobError(err.Error())
			jobCancel()
			return
		}

		// Leitura paralela por bucket (cada worker lê o seu)
		var readersWG sync.WaitGroup
		for w := 0; w < concurrency; w++ {
			readersWG.Add(1)
			go func(workerID int) {
				defer readersWG.Done()
				status.AddWorkerActive(1, 0)
				defer status.AddWorkerActive(-1, 0)
				if jr.shouldStop() || jobCtx.Err() != nil {
					return
				}

				query := jr.Dialect.BuildSelectQueryByHash(job, workerID, concurrency, mainTable)
				var rows *sql.Rows
				if len(mapDirectives) > 0 {
					mapStart := time.Now()
					conn, err := jr.SourceDB.Conn(jobCtx)
					if err != nil {
						log.Printf("Erro ao obter conexao do bucket %d: %v", workerID, err)
						jobHadError.Store(true)
						lastErr.Store(err.Error())
						reportJobError(err.Error())
						jobCancel()
						return
					}
					defer conn.Close()

					tx, err := conn.BeginTx(jobCtx, nil)
					if err != nil {
						log.Printf("Erro ao iniciar transacao do bucket %d: %v", workerID, err)
						jobHadError.Store(true)
						lastErr.Store(err.Error())
						reportJobError(err.Error())
						jobCancel()
						return
					}
					defer tx.Rollback()

					if err := jr.materializeDirectiveMapsWithExecutor(jobCtx, tx, normalizeDBTypeFromDSN(jr.SourceDSN), mapDirectives); err != nil {
						log.Printf("Erro ao materializar maps no bucket %d: %v", workerID, err)
						jobHadError.Store(true)
						lastErr.Store(err.Error())
						reportJobError(err.Error())
						jobCancel()
						return
					}
					log.Printf("Job %s (%s): worker %d materializou maps em %s", job.ID, job.JobName, workerID, time.Since(mapStart))

					rows, err = tx.QueryContext(jobCtx, query)
					if err != nil {
						log.Printf("Erro na query do bucket %d: %v", workerID, err)
						jobHadError.Store(true)
						lastErr.Store(err.Error())
						reportJobError(err.Error())
						jobCancel()
						return
					}
				} else {
					rows, err = jr.SourceDB.Query(query)
					if err != nil {
						log.Printf("Erro na query do bucket %d: %v", workerID, err)
						jobHadError.Store(true)
						lastErr.Store(err.Error())
						reportJobError(err.Error())
						jobCancel()
						return
					}
				}
				defer rows.Close()

				cols, _ := rows.Columns()
				batchSize := job.RecordsPerPage
				buffer := make([]map[string]interface{}, 0, batchSize)

				for rows.Next() {
					if jr.shouldStop() || jobCtx.Err() != nil {
						return
					}
					values := make([]interface{}, len(cols))
					ptrs := make([]interface{}, len(cols))
					for i := range cols {
						ptrs[i] = &values[i]
					}
					if err := rows.Scan(ptrs...); err != nil {
						jobHadError.Store(true)
						lastErr.Store(err.Error())
						reportJobError(err.Error())
						jobCancel()
						return
					}

					rec := make(map[string]interface{})
					for i, col := range cols {
						rec[col] = values[i]
					}
					buffer = append(buffer, rec)

					if len(buffer) == batchSize {
						select {
						case batchChan <- buffer:
						case <-jobCtx.Done():
							return
						}
						buffer = make([]map[string]interface{}, 0, batchSize)
					}
				}

				if len(buffer) > 0 {
					select {
					case batchChan <- buffer:
					case <-jobCtx.Done():
						return
					}
				}

				if err := rows.Err(); err != nil {
					log.Printf("Erro ao iterar rows no bucket %d: %v", workerID, err)
					jobHadError.Store(true)
					lastErr.Store(err.Error())
					reportJobError(err.Error())
					jobCancel()
					return
				}
			}(w)
		}

		// Fecha o canal quando todos leitores terminarem
		go func() {
			readersWG.Wait()
			closeBatch()
		}()

		writerWG.Wait()
		readersWG.Wait()

		end := time.Now()
		finalProcessed := int(atomic.LoadInt64(&processed))
		if jobHadError.Load() {
			finalProcessed = 0
		}
		logger.UpdateJob(jr.PipelineLog, jobID, func(jl *logger.JobLog) {
			jl.Processed = finalProcessed
		})
		jr.savePipelineLog()
		if jobHadError.Load() {
			status.UpdateJobStatus(job.ID, func(js *status.JobStatus) {
				js.Processed = 0
				js.Progress = 0
				status.NotifySubscribers()
			})
		}
		if !jobHadError.Load() && finalProcessed < total {
			jobHadError.Store(true)
			lastErr.Store(fmt.Sprintf("processados %d de %d registros", finalProcessed, total))
		}

		if jobHadError.Load() || jr.shouldStop() || jobCtx.Err() != nil {
			if err := jr.deleteInsertTarget(job); err != nil {
				log.Printf("Erro ao limpar destino do job %s: %v", job.ID, err)
			}
		}

		if jobHadError.Load() {
			errMsg := "erro durante execução"
			if last := lastErr.Load(); last != nil {
				errMsg = last.(string)
			}
			jr.markJobFinalStatus(jobID, job, "error", errMsg, end)
			if job.StopOnError {
				jr.PipelineLog.Status = "error"
				jr.PipelineLog.EndedAt = end
				jr.savePipelineLog()
				status.UpdateProjectStatus("error")
				return
			}
		} else if jr.shouldStop() {
			jr.markJobFinalStatus(jobID, job, "error", "pipeline interrompida", end)
			return
		} else {
			jr.markJobFinalStatus(jobID, job, "done", "", end)
		}

		for _, nextID := range jr.ConnMap[jobID] {
			jr.RunJob(nextID)
		}
	}()
}

func (jr *JobRunner) runExecutionJob(jobID string, job models.Job) {
	log.Printf("Executando job de execucao: %s\n", job.JobName)
	start := time.Now()
	if jr.shouldStop() {
		jr.markJobFinalStatus(jobID, job, "error", "pipeline interrompida", start)
		return
	}

	jobLog := logger.JobLog{
		JobID:       jobID,
		JobName:     job.JobName,
		Status:      "running",
		StopOnError: job.StopOnError,
		StartedAt:   start,
		Processed:   0,
		Total:       1,
		Batches:     make([]logger.BatchLog, 0),
	}
	logger.AddJob(jr.PipelineLog, jobLog)
	jr.savePipelineLog()

	status.UpdateJobStatus(job.ID, func(js *status.JobStatus) {
		js.Name = job.JobName
		js.Status = "running"
		js.StartedAt = &start
		status.NotifySubscribers()
	})

	job.SelectSQL = jr.SubstituteVariables(job.SelectSQL)
	cleanSQL := strings.ReplaceAll(job.SelectSQL, "\r\n", "\n")
	cleanSQL = strings.ReplaceAll(cleanSQL, "\r", "\n")
	log.Printf("EXECUTION SQL (job=%s): %s", job.ID, cleanSQL)

	targetDB, dbType := jr.resolveExecutionDB(job)
	conn, err := targetDB.Conn(jr.ctx)
	if err != nil {
		jr.handleExecutionJobError(jobID, job, err)
		return
	}
	defer conn.Close()

	resolvedSQL, directives, err := extractMapDirectives(cleanSQL)
	if err != nil {
		jr.handleExecutionJobError(jobID, job, err)
		return
	}

	if len(directives) > 0 {
		tx, txErr := conn.BeginTx(jr.ctx, nil)
		if txErr != nil {
			jr.handleExecutionJobError(jobID, job, txErr)
			return
		}
		committed := false
		defer func() {
			if !committed {
				_ = tx.Rollback()
			}
		}()

		if err := jr.materializeDirectiveMapsWithExecutor(jr.ctx, tx, dbType, directives); err != nil {
			jr.handleExecutionJobError(jobID, job, err)
			return
		}

		if strings.TrimSpace(resolvedSQL) != "" {
			_, err = tx.ExecContext(jr.ctx, resolvedSQL)
		}
		if err == nil {
			err = tx.Commit()
			if err == nil {
				committed = true
			}
		}
	} else {
		if strings.TrimSpace(resolvedSQL) != "" {
			_, err = conn.ExecContext(jr.ctx, resolvedSQL)
		}
	}

	end := time.Now()
	if jr.shouldStop() {
		jr.markJobFinalStatus(jobID, job, "error", "pipeline interrompida", end)
		return
	}

	if err != nil {
		log.Printf("Erro no job de execucao: %v\n", err)
		logger.UpdateJob(jr.PipelineLog, jobID, func(jl *logger.JobLog) {
			jl.Status = "error"
			jl.Error = err.Error()
			jl.EndedAt = end
		})
		jr.savePipelineLog()

		status.UpdateJobStatus(job.ID, func(js *status.JobStatus) {
			js.Status = "error"
			js.Error = err.Error()
			status.NotifySubscribers()
		})

		if job.StopOnError {
			log.Printf("Job %s falhou e StopOnError estah ativo. Nao executando dependentes.\n", jobID)
			jr.PipelineLog.Status = "error"
			jr.PipelineLog.EndedAt = end
			jr.savePipelineLog()
			status.UpdateProjectStatus("error")
			return
		}
	} else {
		logger.UpdateJob(jr.PipelineLog, jobID, func(jl *logger.JobLog) {
			jl.Status = "done"
			jl.Processed = 1
			jl.EndedAt = end
		})
		jr.savePipelineLog()

		status.UpdateJobStatus(job.ID, func(js *status.JobStatus) {
			js.Status = "done"
			js.EndedAt = &end
			status.NotifySubscribers()
		})
	}

	for _, nextID := range jr.ConnMap[jobID] {
		jr.RunJob(nextID)
	}
}
func (jr *JobRunner) runConditionJob(jobID string, job models.Job) {
	log.Printf("Executando job de condição: %s\n", job.JobName)
	start := time.Now()
	if jr.shouldStop() {
		jr.markJobFinalStatus(jobID, job, "error", "pipeline interrompida", start)
		return
	}

	// Inicializa log do job
	jobLog := logger.JobLog{
		JobID:       jobID,
		JobName:     job.JobName,
		Status:      "running",
		StopOnError: job.StopOnError,
		StartedAt:   start,
		Processed:   0,
		Total:       1, // Jobs de condição avaliam 1 condição
		Batches:     make([]logger.BatchLog, 0),
	}
	logger.AddJob(jr.PipelineLog, jobLog)
	jr.savePipelineLog()

	status.UpdateJobStatus(job.ID, func(js *status.JobStatus) {
		js.Name = job.JobName
		js.Status = "running"
		status.NotifySubscribers()
	})

	// Substitui variáveis no SQL
	job.SelectSQL = jr.SubstituteVariables(job.SelectSQL)
	resolvedSQL, directives, err := extractMapDirectives(job.SelectSQL)
	if err != nil {
		end := time.Now()
		jr.markJobFinalStatus(jobID, job, "error", err.Error(), end)
		if job.StopOnError {
			jr.PipelineLog.Status = "error"
			jr.PipelineLog.EndedAt = end
			jr.savePipelineLog()
			status.UpdateProjectStatus("error")
		}
		return
	}
	job.SelectSQL = resolvedSQL

	var result bool
	if len(directives) == 0 {
		err = jr.SourceDB.QueryRow(job.SelectSQL).Scan(&result)
	} else {
		conn, connErr := jr.SourceDB.Conn(jr.ctx)
		if connErr != nil {
			err = connErr
		} else {
			defer conn.Close()
			tx, txErr := conn.BeginTx(jr.ctx, nil)
			if txErr != nil {
				err = txErr
			} else if materializeErr := jr.materializeDirectiveMapsWithExecutor(jr.ctx, tx, normalizeDBTypeFromDSN(jr.SourceDSN), directives); materializeErr != nil {
				err = materializeErr
			} else {
				err = tx.QueryRowContext(jr.ctx, job.SelectSQL).Scan(&result)
			}
			if tx != nil {
				_ = tx.Rollback()
			}
		}
	}
	end := time.Now()

	if jr.shouldStop() {
		jr.markJobFinalStatus(jobID, job, "error", "pipeline interrompida", end)
		return
	}
	if err != nil {
		log.Printf("Erro ao executar condição: %v\n", err)

		// Atualiza log do job com erro
		logger.UpdateJob(jr.PipelineLog, jobID, func(jl *logger.JobLog) {
			jl.Status = "error"
			jl.Error = err.Error()
			jl.EndedAt = end
		})
		jr.savePipelineLog()

		status.UpdateJobStatus(job.ID, func(js *status.JobStatus) {
			js.Status = "error"
			js.Error = err.Error()
			status.NotifySubscribers()
		})

		if job.StopOnError {
			log.Printf("Job %s falhou e StopOnError está ativo. Não executando dependentes.\n", jobID)
			jr.PipelineLog.Status = "error"
			jr.PipelineLog.EndedAt = end
			jr.savePipelineLog()
			status.UpdateProjectStatus("error")
		}
		return
	}

	if !result {
		log.Printf("Condição falhou: %s\n", job.JobName)
		errMsg := "Condição retornou falso"

		// Atualiza log do job com erro de condição
		logger.UpdateJob(jr.PipelineLog, jobID, func(jl *logger.JobLog) {
			jl.Status = "error"
			jl.Error = errMsg
			jl.EndedAt = end
		})
		jr.savePipelineLog()

		status.UpdateJobStatus(job.ID, func(js *status.JobStatus) {
			js.Status = "error"
			js.Error = errMsg
			status.NotifySubscribers()
		})

		if job.StopOnError {
			log.Printf("Job %s falhou e StopOnError está ativo. Não executando dependentes.\n", jobID)
			jr.PipelineLog.Status = "error"
			jr.PipelineLog.EndedAt = end
			jr.savePipelineLog()
			status.UpdateProjectStatus("error")
		}
		return
	} else {
		// Atualiza log do job como concluído
		logger.UpdateJob(jr.PipelineLog, jobID, func(jl *logger.JobLog) {
			jl.Status = "done"
			jl.Processed = 1
			jl.EndedAt = end
		})
		jr.savePipelineLog()

		status.UpdateJobStatus(job.ID, func(js *status.JobStatus) {
			js.Status = "done"
			status.NotifySubscribers()
		})
	}

	for _, nextID := range jr.ConnMap[jobID] {
		jr.RunJob(nextID)
	}
}

func (jr *JobRunner) runMemorySelectJob(jobID string, job models.Job) {
	log.Printf("Executando job memory-select: %s\n", job.JobName)
	start := time.Now()
	if jr.shouldStop() {
		jr.markJobFinalStatus(jobID, job, "error", "pipeline interrompida", start)
		return
	}

	jobLog := logger.JobLog{
		JobID:       jobID,
		JobName:     job.JobName,
		Status:      "running",
		StopOnError: job.StopOnError,
		StartedAt:   start,
		Processed:   0,
		Total:       0,
		Batches:     make([]logger.BatchLog, 0),
	}
	logger.AddJob(jr.PipelineLog, jobLog)
	jr.savePipelineLog()

	status.UpdateJobStatus(job.ID, func(js *status.JobStatus) {
		js.Name = job.JobName
		js.Status = "running"
		js.StartedAt = &start
		status.NotifySubscribers()
	})

	key, err := normalizeMemoryMapKey(job.JobName)
	if err != nil {
		jr.failMemorySelectJob(jobID, job, err)
		return
	}

	if err := validateMemorySelectColumns(job.Columns); err != nil {
		jr.failMemorySelectJob(jobID, job, err)
		return
	}

	if exists := jr.hasMemoryMap(key); exists {
		jr.failMemorySelectJob(jobID, job, fmt.Errorf("map em memoria '%s' ja existe", key))
		return
	}

	job.SelectSQL = jr.SubstituteVariables(job.SelectSQL)
	rows, err := jr.DestinationDB.Query(job.SelectSQL)
	if err != nil {
		jr.failMemorySelectJob(jobID, job, err)
		return
	}
	defer rows.Close()

	resultColumns, err := rows.Columns()
	if err != nil {
		jr.failMemorySelectJob(jobID, job, err)
		return
	}
	colTypes, err := rows.ColumnTypes()
	if err != nil {
		jr.failMemorySelectJob(jobID, job, err)
		return
	}
	if len(resultColumns) != len(colTypes) {
		jr.failMemorySelectJob(jobID, job, fmt.Errorf("metadados de colunas inconsistentes para o job %s", job.ID))
		return
	}

	selectedIndices, err := buildSelectedColumnIndexes(job.Columns, resultColumns)
	if err != nil {
		jr.failMemorySelectJob(jobID, job, err)
		return
	}

	columnDBTypes := make(map[string]string, len(job.Columns))
	for _, idx := range selectedIndices {
		columnDBTypes[job.Columns[idx.targetPos]] = strings.ToUpper(strings.TrimSpace(colTypes[idx.resultPos].DatabaseTypeName()))
	}

	records := make([]map[string]interface{}, 0)
	for rows.Next() {
		values := make([]interface{}, len(resultColumns))
		ptrs := make([]interface{}, len(resultColumns))
		for i := range resultColumns {
			ptrs[i] = &values[i]
		}

		if err := rows.Scan(ptrs...); err != nil {
			jr.failMemorySelectJob(jobID, job, err)
			return
		}

		record := make(map[string]interface{}, len(job.Columns))
		for _, idx := range selectedIndices {
			columnName := job.Columns[idx.targetPos]
			record[columnName] = convertScannedValue(values[idx.resultPos], colTypes[idx.resultPos].DatabaseTypeName())
		}
		records = append(records, record)
	}

	if err := rows.Err(); err != nil {
		jr.failMemorySelectJob(jobID, job, err)
		return
	}

	dataset := memoryDataset{
		Columns:       append([]string(nil), job.Columns...),
		ColumnDBTypes: columnDBTypes,
		Rows:          records,
	}
	if err := jr.storeMemoryMap(key, dataset); err != nil {
		jr.failMemorySelectJob(jobID, job, err)
		return
	}

	end := time.Now()
	logger.UpdateJob(jr.PipelineLog, jobID, func(jl *logger.JobLog) {
		jl.Status = "done"
		jl.Processed = len(records)
		jl.Total = len(records)
		jl.EndedAt = end
	})
	jr.savePipelineLog()

	status.UpdateJobStatus(job.ID, func(js *status.JobStatus) {
		js.Status = "done"
		js.Processed = len(records)
		js.Total = len(records)
		js.Progress = 100
		js.EndedAt = &end
		status.NotifySubscribers()
	})

	for _, nextID := range jr.ConnMap[jobID] {
		jr.RunJob(nextID)
	}
}

func (jr *JobRunner) failMemorySelectJob(jobID string, job models.Job, runErr error) {
	end := time.Now()
	jr.markJobFinalStatus(jobID, job, "error", runErr.Error(), end)
	if job.StopOnError {
		jr.PipelineLog.Status = "error"
		jr.PipelineLog.EndedAt = end
		jr.savePipelineLog()
		status.UpdateProjectStatus("error")
		return
	}
	for _, nextID := range jr.ConnMap[jobID] {
		jr.RunJob(nextID)
	}
}

type selectedColumnIndex struct {
	targetPos int
	resultPos int
}

type memoryDataset struct {
	Columns       []string
	ColumnDBTypes map[string]string
	Rows          []map[string]interface{}
}

type mapDirective struct {
	Raw string
	Key string
}

type directiveExecutor interface {
	ExecContext(context.Context, string, ...interface{}) (sql.Result, error)
	PrepareContext(context.Context, string) (*sql.Stmt, error)
}

var (
	mapDirectiveRegex = regexp.MustCompile(`Map\[\s*'([^']+)'\s*\]\s*;?`)
	mapKeyRegex       = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)
)

func buildSelectedColumnIndexes(targetColumns []string, resultColumns []string) ([]selectedColumnIndex, error) {
	resultIdxByExact := make(map[string]int, len(resultColumns))
	resultIdxByNormalized := make(map[string]int, len(resultColumns))
	for i, col := range resultColumns {
		resultIdxByExact[col] = i
		norm := normalizeColumnName(col)
		if _, exists := resultIdxByNormalized[norm]; !exists {
			resultIdxByNormalized[norm] = i
		}
	}

	selected := make([]selectedColumnIndex, 0, len(targetColumns))
	for i, targetCol := range targetColumns {
		if pos, ok := resultIdxByExact[targetCol]; ok {
			selected = append(selected, selectedColumnIndex{targetPos: i, resultPos: pos})
			continue
		}
		pos, ok := resultIdxByNormalized[normalizeColumnName(targetCol)]
		if !ok {
			return nil, fmt.Errorf("coluna '%s' nao encontrada no resultset", targetCol)
		}
		selected = append(selected, selectedColumnIndex{targetPos: i, resultPos: pos})
	}

	return selected, nil
}

func validateMemorySelectColumns(columns []string) error {
	if len(columns) == 0 {
		return fmt.Errorf("columns e obrigatorio para jobs do tipo memory-select")
	}
	for _, col := range columns {
		if strings.TrimSpace(col) == "" {
			return fmt.Errorf("columns contem valor vazio")
		}
	}
	return nil
}

func normalizeMemoryMapKey(jobName string) (string, error) {
	trimmed := strings.TrimSpace(jobName)
	if trimmed == "" {
		return "", fmt.Errorf("jobName e obrigatorio para jobs do tipo memory-select")
	}

	lowered := strings.ToLower(removeDiacritics(trimmed))
	spaceHyphenRegex := regexp.MustCompile(`[\s-]+`)
	specialCharsRegex := regexp.MustCompile(`[^a-z0-9_]+`)
	underscoreRegex := regexp.MustCompile(`_+`)

	normalized := spaceHyphenRegex.ReplaceAllString(lowered, "_")
	normalized = specialCharsRegex.ReplaceAllString(normalized, "")
	normalized = underscoreRegex.ReplaceAllString(normalized, "_")
	normalized = strings.Trim(normalized, "_")
	if normalized == "" {
		return "", fmt.Errorf("jobName '%s' nao gera chave valida para map em memoria", jobName)
	}
	return normalized, nil
}

func removeDiacritics(input string) string {
	decomposed := norm.NFD.String(input)
	var b strings.Builder
	for _, r := range decomposed {
		if unicode.Is(unicode.Mn, r) {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

func normalizeColumnName(name string) string {
	var b strings.Builder
	for _, r := range strings.TrimSpace(strings.ToLower(name)) {
		if unicode.IsSpace(r) {
			continue
		}
		if r == '"' || r == '\'' || r == '`' {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

func (jr *JobRunner) resolveExecutionDB(job models.Job) (*sql.DB, string) {
	conn := strings.ToLower(strings.TrimSpace(job.Connection))
	switch conn {
	case "origem", "source", "src", "source_db":
		return jr.SourceDB, normalizeDBTypeFromDSN(jr.SourceDSN)
	case "destino", "destination", "dest", "target", "destination_db":
		return jr.DestinationDB, normalizeDBTypeFromDSN(jr.DestinationDSN)
	default:
		return jr.DestinationDB, normalizeDBTypeFromDSN(jr.DestinationDSN)
	}
}

func (jr *JobRunner) handleExecutionJobError(jobID string, job models.Job, err error) {
	end := time.Now()
	jr.markJobFinalStatus(jobID, job, "error", err.Error(), end)
	if job.StopOnError {
		jr.PipelineLog.Status = "error"
		jr.PipelineLog.EndedAt = end
		jr.savePipelineLog()
		status.UpdateProjectStatus("error")
		return
	}
	for _, nextID := range jr.ConnMap[jobID] {
		jr.RunJob(nextID)
	}
}

func extractMapDirectives(sqlText string) (string, []mapDirective, error) {
	matches := mapDirectiveRegex.FindAllStringSubmatch(sqlText, -1)
	directives := make([]mapDirective, 0, len(matches))
	seen := make(map[string]struct{}, len(matches))

	for _, m := range matches {
		if len(m) < 2 {
			continue
		}
		raw := m[0]
		key := strings.TrimSpace(m[1])
		if !mapKeyRegex.MatchString(key) {
			return "", nil, fmt.Errorf("chave de map invalida na diretiva: %s", key)
		}
		if _, exists := seen[key]; exists {
			return "", nil, fmt.Errorf("diretiva Map com chave repetida: %s", key)
		}
		seen[key] = struct{}{}
		directives = append(directives, mapDirective{Raw: raw, Key: key})
	}

	cleanSQL := mapDirectiveRegex.ReplaceAllString(sqlText, "")
	return cleanSQL, directives, nil
}

func (jr *JobRunner) materializeDirectiveMaps(ctx context.Context, conn *sql.Conn, targetDBType string, directives []mapDirective) error {
	return jr.materializeDirectiveMapsWithExecutor(ctx, conn, targetDBType, directives)
}

func (jr *JobRunner) materializeDirectiveMapsWithExecutor(ctx context.Context, executor directiveExecutor, targetDBType string, directives []mapDirective) error {
	for _, directive := range directives {
		directiveStart := time.Now()
		dataset, ok := jr.getMemoryMap(directive.Key)
		if !ok {
			return fmt.Errorf("map '%s' nao encontrado no contexto da pipeline", directive.Key)
		}
		log.Printf("Map materialization: key=%s rows=%d cols=%d target=%s", directive.Key, len(dataset.Rows), len(dataset.Columns), targetDBType)

		createStart := time.Now()
		createSQL, err := buildCreateTempTableSQL(targetDBType, directive.Key, dataset)
		if err != nil {
			return err
		}
		if _, err := executor.ExecContext(ctx, createSQL); err != nil {
			return err
		}
		log.Printf("Map materialization: key=%s create temp table em %s", directive.Key, time.Since(createStart))

		if len(dataset.Rows) == 0 {
			log.Printf("Map materialization: key=%s sem linhas, apenas schema criado (%s)", directive.Key, time.Since(directiveStart))
			continue
		}

		insertStart := time.Now()
		insertSQL, err := buildInsertTempTableSQL(targetDBType, directive.Key, dataset.Columns)
		if err != nil {
			return err
		}

		stmt, err := executor.PrepareContext(ctx, insertSQL)
		if err != nil {
			return err
		}

		for _, row := range dataset.Rows {
			args := make([]interface{}, 0, len(dataset.Columns))
			for _, col := range dataset.Columns {
				args = append(args, row[col])
			}
			if _, err := stmt.ExecContext(ctx, args...); err != nil {
				_ = stmt.Close()
				return err
			}
		}
		if err := stmt.Close(); err != nil {
			return err
		}
		log.Printf("Map materialization: key=%s insert %d linha(s) em %s (total %s)", directive.Key, len(dataset.Rows), time.Since(insertStart), time.Since(directiveStart))
	}
	return nil
}

func buildCreateTempTableSQL(targetDBType, tableName string, dataset memoryDataset) (string, error) {
	if len(dataset.Columns) == 0 {
		return "", fmt.Errorf("map '%s' sem colunas para materializacao", tableName)
	}

	columnDefs := make([]string, 0, len(dataset.Columns))
	for _, col := range dataset.Columns {
		sqlType, err := inferColumnSQLType(targetDBType, col, dataset)
		if err != nil {
			return "", err
		}
		columnDefs = append(columnDefs, fmt.Sprintf("%s %s", quoteIdentifier(targetDBType, col), sqlType))
	}

	createPrefix := "CREATE TEMP TABLE"
	if targetDBType == "mysql" {
		createPrefix = "CREATE TEMPORARY TABLE"
	}
	createSuffix := ""
	if targetDBType == "postgres" {
		createSuffix = " ON COMMIT DROP"
	}

	return fmt.Sprintf("%s %s (%s)%s", createPrefix, quoteIdentifier(targetDBType, tableName), strings.Join(columnDefs, ", "), createSuffix), nil
}

func buildInsertTempTableSQL(targetDBType, tableName string, columns []string) (string, error) {
	if len(columns) == 0 {
		return "", fmt.Errorf("nao e possivel gerar insert sem colunas")
	}

	quotedColumns := make([]string, 0, len(columns))
	for _, col := range columns {
		quotedColumns = append(quotedColumns, quoteIdentifier(targetDBType, col))
	}

	placeholders := make([]string, 0, len(columns))
	for i := range columns {
		if targetDBType == "postgres" {
			placeholders = append(placeholders, fmt.Sprintf("$%d", i+1))
			continue
		}
		placeholders = append(placeholders, "?")
	}

	return fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s)",
		quoteIdentifier(targetDBType, tableName),
		strings.Join(quotedColumns, ", "),
		strings.Join(placeholders, ", "),
	), nil
}

func inferColumnSQLType(targetDBType, column string, dataset memoryDataset) (string, error) {
	valueKind := ""
	for _, row := range dataset.Rows {
		value := row[column]
		kind, err := classifyColumnValue(value)
		if err != nil {
			return "", fmt.Errorf("coluna '%s': %w", column, err)
		}
		if kind == "" {
			continue
		}
		if valueKind == "" {
			valueKind = kind
			continue
		}
		if valueKind != kind {
			return "", fmt.Errorf("inconsistencia de tipos na coluna '%s'", column)
		}
	}

	dbTypeName := strings.ToUpper(strings.TrimSpace(dataset.ColumnDBTypes[column]))
	if dbTypeName != "" {
		return mapDBTypeToSQLType(targetDBType, dbTypeName), nil
	}

	switch valueKind {
	case "int":
		if targetDBType == "postgres" {
			return "BIGINT", nil
		}
		return "BIGINT", nil
	case "float":
		if targetDBType == "postgres" {
			return "DOUBLE PRECISION", nil
		}
		return "DOUBLE", nil
	case "bool":
		return "BOOLEAN", nil
	case "time":
		if targetDBType == "postgres" {
			return "TIMESTAMP", nil
		}
		return "DATETIME", nil
	case "string", "":
		return "TEXT", nil
	default:
		return "", fmt.Errorf("tipo nao suportado na coluna '%s'", column)
	}
}

func classifyColumnValue(value interface{}) (string, error) {
	if value == nil {
		return "", nil
	}
	switch value.(type) {
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return "int", nil
	case float32, float64:
		return "float", nil
	case bool:
		return "bool", nil
	case string:
		return "string", nil
	case time.Time:
		return "time", nil
	default:
		return "", fmt.Errorf("valor com tipo nao suportado: %T", value)
	}
}

func mapDBTypeToSQLType(targetDBType, dbTypeName string) string {
	switch {
	case strings.Contains(dbTypeName, "INT"):
		return "BIGINT"
	case strings.Contains(dbTypeName, "DECIMAL"), strings.Contains(dbTypeName, "NUMERIC"):
		if targetDBType == "postgres" {
			return "NUMERIC"
		}
		return "DECIMAL(38,10)"
	case strings.Contains(dbTypeName, "DOUBLE"), strings.Contains(dbTypeName, "FLOAT"), strings.Contains(dbTypeName, "REAL"):
		if targetDBType == "postgres" {
			return "DOUBLE PRECISION"
		}
		return "DOUBLE"
	case strings.Contains(dbTypeName, "BOOL"), strings.Contains(dbTypeName, "BIT"):
		return "BOOLEAN"
	case strings.Contains(dbTypeName, "TIMESTAMP"):
		return "TIMESTAMP"
	case strings.Contains(dbTypeName, "DATE"):
		if targetDBType == "postgres" {
			return "DATE"
		}
		return "DATE"
	case strings.Contains(dbTypeName, "TIME"):
		if targetDBType == "postgres" {
			return "TIME"
		}
		return "TIME"
	default:
		return "TEXT"
	}
}

func quoteIdentifier(targetDBType, ident string) string {
	if targetDBType == "mysql" {
		escaped := strings.ReplaceAll(ident, "`", "``")
		return "`" + escaped + "`"
	}
	escaped := strings.ReplaceAll(ident, "\"", "\"\"")
	return "\"" + escaped + "\""
}

func normalizeDBTypeFromDSN(dsn string) string {
	lower := strings.ToLower(dsn)
	switch {
	case strings.Contains(lower, "host="), strings.Contains(lower, "sslmode="):
		return "postgres"
	case strings.Contains(lower, "@tcp("):
		return "mysql"
	default:
		return "postgres"
	}
}

func (jr *JobRunner) hasMemoryMap(key string) bool {
	jr.memoryStoreMu.RLock()
	_, exists := jr.memoryStore[key]
	jr.memoryStoreMu.RUnlock()
	return exists
}

func (jr *JobRunner) storeMemoryMap(key string, dataset memoryDataset) error {
	jr.memoryStoreMu.Lock()
	defer jr.memoryStoreMu.Unlock()
	if _, exists := jr.memoryStore[key]; exists {
		return fmt.Errorf("map em memoria '%s' ja existe", key)
	}
	jr.memoryStore[key] = dataset
	return nil
}

func (jr *JobRunner) getMemoryMap(key string) (memoryDataset, bool) {
	jr.memoryStoreMu.RLock()
	dataset, exists := jr.memoryStore[key]
	jr.memoryStoreMu.RUnlock()
	return dataset, exists
}

func (jr *JobRunner) clearMemoryStore() {
	jr.memoryStoreMu.Lock()
	jr.memoryStore = make(map[string]memoryDataset)
	jr.memoryStoreMu.Unlock()
}

func convertScannedValue(value interface{}, dbTypeName string) interface{} {
	if value == nil {
		return nil
	}

	rawBytes, isBytes := value.([]byte)
	if !isBytes {
		return value
	}

	raw := string(rawBytes)
	dbType := strings.ToUpper(strings.TrimSpace(dbTypeName))
	switch {
	case isIntDBType(dbType):
		if parsed, err := strconv.ParseInt(raw, 10, 64); err == nil {
			return parsed
		}
	case isFloatDBType(dbType):
		if parsed, err := strconv.ParseFloat(raw, 64); err == nil {
			return parsed
		}
	case isBoolDBType(dbType):
		if parsed, ok := parseDBBool(raw); ok {
			return parsed
		}
	}

	return raw
}

func isIntDBType(dbType string) bool {
	return strings.Contains(dbType, "INT")
}

func isFloatDBType(dbType string) bool {
	return strings.Contains(dbType, "DECIMAL") ||
		strings.Contains(dbType, "NUMERIC") ||
		strings.Contains(dbType, "FLOAT") ||
		strings.Contains(dbType, "DOUBLE") ||
		strings.Contains(dbType, "REAL")
}

func isBoolDBType(dbType string) bool {
	return strings.Contains(dbType, "BOOL") || strings.Contains(dbType, "BIT")
}

func parseDBBool(raw string) (bool, bool) {
	trimmed := strings.TrimSpace(strings.ToLower(raw))
	switch trimmed {
	case "t", "true", "1", "y", "yes":
		return true, true
	case "f", "false", "0", "n", "no":
		return false, true
	default:
		return false, false
	}
}

// Marca erro no job
func (jr *JobRunner) markJobError(jobID string, job models.Job, err error) {
	end := time.Now()
	logger.UpdateJob(jr.PipelineLog, jobID, func(jl *logger.JobLog) {
		jl.Status = "error"
		jl.Error = err.Error()
		jl.EndedAt = end
	})
	jr.savePipelineLog()
	status.UpdateJobStatus(job.ID, func(js *status.JobStatus) {
		js.Status = "error"
		js.Error = err.Error()
		js.EndedAt = &end
		status.NotifySubscribers()
	})
}

// Marca status final do job
func (jr *JobRunner) markJobFinalStatus(jobID string, job models.Job, statusStr, errMsg string, end time.Time) {
	logger.UpdateJob(jr.PipelineLog, jobID, func(jl *logger.JobLog) {
		jl.Status = statusStr
		jl.Error = errMsg
		jl.EndedAt = end
		jl.Processed = int(jl.Processed)
	})
	jr.savePipelineLog()
	status.UpdateJobStatus(job.ID, func(js *status.JobStatus) {
		js.Status = statusStr
		js.Error = errMsg
		js.EndedAt = &end
		status.NotifySubscribers()
	})
}

func (jr *JobRunner) Run(startIDs []string) {
	log.Printf("Iniciando pipeline %s para projeto %s\n", jr.PipelineLog.PipelineID, jr.PipelineLog.Project)
	defer jr.clearMemoryStore()

	jr.preloadCounts(startIDs)

	for _, id := range startIDs {
		if jr.shouldStop() {
			break
		}
		jr.RunJob(id)
	}
	jr.WaitGroup.Wait()

	// Finaliza o pipeline
	jr.PipelineLog.EndedAt = time.Now()
	if jr.PipelineLog.Status == "running" {
		jr.PipelineLog.Status = "done"
	} else if jr.PipelineLog.Status == "stopped" {
		status.UpdateProjectStatus("stop")
	}
	jr.savePipelineLog()
	clearActiveRunner(jr)

	if jr.countQueue != nil {
		close(jr.countQueue)
	}

	log.Printf("Pipeline %s finalizado com status: %s\n", jr.PipelineLog.PipelineID, jr.PipelineLog.Status)
}

func (jr *JobRunner) Stop(reason string) {
	if jr.stopped.Swap(true) {
		return
	}
	if reason == "" {
		reason = "interrompido"
	}
	jr.stopReason.Store(reason)
	jr.PipelineLog.Status = "stopped"
	jr.PipelineLog.EndedAt = time.Now()
	jr.savePipelineLog()
	status.UpdateProjectStatus("stop")
	status.AppendLog(fmt.Sprintf("%s - Pipeline interrompida: %s", jr.PipelineLog.Project, reason))
	jr.cancel()
	if jr.SourceDB != nil {
		_ = jr.SourceDB.Close()
	}
	if jr.DestinationDB != nil {
		_ = jr.DestinationDB.Close()
	}
	for id, job := range jr.JobMap {
		js := status.GetJobStatus(id)
		if js == nil || (js.Status != "done" && js.Status != "error") {
			jr.markJobFinalStatus(id, job, "error", "pipeline interrompida", time.Now())
		}
	}
}

func (jr *JobRunner) shouldStop() bool {
	if jr.stopped.Load() {
		return true
	}
	return jr.ctx.Err() != nil
}

// Método auxiliar para salvar o log do pipeline
func (jr *JobRunner) savePipelineLog() {
	if err := logger.SavePipelineLog(jr.PipelineLog); err != nil {
		log.Printf("Erro ao salvar log do pipeline: %v\n", err)
		// Debug: mostra detalhes do erro
		log.Printf("PipelineID: %s, Status: %s, Jobs count: %d\n",
			jr.PipelineLog.PipelineID, jr.PipelineLog.Status, len(jr.PipelineLog.Jobs))
	} else {
		log.Printf("Log do pipeline salvo com sucesso: %s\n", jr.PipelineLog.PipelineID)
	}
}

// Método para obter o ID do pipeline (útil para recuperação posterior)
func (jr *JobRunner) GetPipelineID() string {
	return jr.PipelineLog.PipelineID
}

// LoadProjectVariables carrega as variáveis de um projeto específico
func LoadProjectVariables(projectID string) map[string]string {
	variables := make(map[string]string)
	projectPath := filepath.Join("data", "projects", projectID, "project.json")

	projectBytes, err := os.ReadFile(projectPath)
	if err != nil {
		log.Printf("Erro ao ler project.json: %v\n", err)
		return variables
	}

	var project models.Project
	if err := json.Unmarshal(projectBytes, &project); err != nil {
		log.Printf("Erro ao interpretar project.json: %v\n", err)
		return variables
	}

	// Converte as variáveis do projeto para um mapa
	for _, variable := range project.Variables {
		// Converte o valor para string, independentemente do tipo
		var valueStr string
		switch v := variable.Value.(type) {
		case string:
			valueStr = v
		case int, int64, float64:
			valueStr = fmt.Sprintf("%v", v)
		case bool:
			valueStr = fmt.Sprintf("%t", v)
		default:
			valueStr = fmt.Sprintf("%v", v)
		}
		variables[variable.Name] = valueStr
	}

	log.Printf("Carregadas %d variáveis do projeto %s\n", len(variables), projectID)
	return variables
}

// SubstituteVariables substitui placeholders nas queries SQL com os valores das variáveis
func (jr *JobRunner) SubstituteVariables(query string) string {
	for key, value := range jr.Variables {
		placeholder := fmt.Sprintf("${%s}", key)
		query = strings.ReplaceAll(query, placeholder, value)
	}
	return query
}

// CleanSQLNewlines limpa as quebras de linha nas queries SQL para evitar problemas no PostgreSQL
func (jr *JobRunner) CleanSQLNewlines(sql string) string {
	// Preserva quebras de linha em strings literais (entre aspas simples)
	// mas normaliza quebras de linha fora de strings literais

	var result strings.Builder
	inString := false

	for i := 0; i < len(sql); i++ {
		char := sql[i]

		// Verifica se estamos dentro ou fora de uma string literal
		if char == '\'' {
			// Verifica se a aspa não está escapada
			if i == 0 || sql[i-1] != '\\' {
				inString = !inString
			}
		}

		// Trata quebras de linha
		if (char == '\n' || char == '\r') && !inString {
			// Substitui quebras de linha por espaço fora de strings literais
			result.WriteByte(' ')

			// Pula o \n em sequências \r\n
			if char == '\r' && i+1 < len(sql) && sql[i+1] == '\n' {
				i++
			}
		} else {
			result.WriteByte(char)
		}
	}

	return result.String()
}

func (jr *JobRunner) deleteInsertTarget(job models.Job) error {
	table, ok := extractInsertTable(job.InsertSQL)
	if !ok {
		return fmt.Errorf("nao foi possivel identificar a tabela do insert")
	}
	deleteSQL := fmt.Sprintf("DELETE FROM %s", table)
	_, err := jr.DestinationDB.Exec(deleteSQL)
	return err
}

func (jr *JobRunner) getMainTableFromExplain(job models.Job) (string, error) {
	queryExplain := jr.Dialect.BuildExplainSelectQueryByHash(job)
	rowsExplain, err := jr.SourceDB.Query(queryExplain)
	if err != nil {
		return "", err
	}
	defer rowsExplain.Close()

	var explainJSON []byte
	for rowsExplain.Next() {
		var col string
		if err := rowsExplain.Scan(&col); err != nil {
			return "", err
		}
		explainJSON = []byte(col)
	}

	if err := rowsExplain.Err(); err != nil {
		return "", err
	}

	mainTable, err := GetMainTableFromExplain(explainJSON)
	if err != nil {
		return "", err
	}
	return mainTable, nil
}

func (jr *JobRunner) getMainTableFromExplainWithMapDirectives(job models.Job, directives []mapDirective, ctx context.Context) (string, error) {
	conn, err := jr.SourceDB.Conn(ctx)
	if err != nil {
		return "", err
	}
	defer conn.Close()

	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return "", err
	}
	defer tx.Rollback()

	if err := jr.materializeDirectiveMapsWithExecutor(ctx, tx, normalizeDBTypeFromDSN(jr.SourceDSN), directives); err != nil {
		return "", err
	}

	queryExplain := jr.Dialect.BuildExplainSelectQueryByHash(job)
	rowsExplain, err := tx.QueryContext(ctx, queryExplain)
	if err != nil {
		return "", err
	}
	defer rowsExplain.Close()

	var explainJSON []byte
	for rowsExplain.Next() {
		var col string
		if err := rowsExplain.Scan(&col); err != nil {
			return "", err
		}
		explainJSON = []byte(col)
	}

	if err := rowsExplain.Err(); err != nil {
		return "", err
	}

	mainTable, err := GetMainTableFromExplain(explainJSON)
	if err != nil {
		return "", err
	}
	return mainTable, nil
}

func (jr *JobRunner) countSelectWithMapDirectives(job models.Job, directives []mapDirective) (int, error) {
	ctx := jr.ctx
	conn, err := jr.SourceDB.Conn(ctx)
	if err != nil {
		return 0, err
	}
	defer conn.Close()

	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	if err := jr.materializeDirectiveMapsWithExecutor(ctx, tx, normalizeDBTypeFromDSN(jr.SourceDSN), directives); err != nil {
		return 0, err
	}

	selectSQL := strings.TrimSpace(job.SelectSQL)
	selectSQL = strings.TrimSuffix(selectSQL, ";")
	countSQL := fmt.Sprintf("SELECT COUNT(*) FROM (%s) AS t", selectSQL)

	var total int
	if err := tx.QueryRowContext(ctx, countSQL).Scan(&total); err != nil {
		return 0, err
	}
	return total, nil
}

func extractInsertTable(insertSQL string) (string, bool) {
	lower := strings.ToLower(insertSQL)
	idx := strings.Index(lower, "insert into")
	if idx == -1 {
		return "", false
	}
	rest := strings.TrimSpace(insertSQL[idx+len("insert into"):])
	if rest == "" {
		return "", false
	}
	inDouble := false
	var b strings.Builder
	for _, r := range rest {
		if r == '"' {
			inDouble = !inDouble
			b.WriteRune(r)
			continue
		}
		if !inDouble {
			if r == '(' || r == ' ' || r == '\n' || r == '\r' || r == '\t' {
				break
			}
		}
		b.WriteRune(r)
	}
	table := strings.TrimSpace(b.String())
	if table == "" {
		return "", false
	}
	return table, true
}

type PlanNode struct {
	NodeType     string     `json:"Node Type"`
	RelationName string     `json:"Relation Name,omitempty"`
	Schema       string     `json:"Schema,omitempty"`
	Alias        string     `json:"Alias,omitempty"`
	Plans        []PlanNode `json:"Plans,omitempty"`
}

type Explain struct {
	Plan PlanNode `json:"Plan"`
}

func getMainTable(node PlanNode) (string, bool) {
	// Retorna alias se existir
	if node.Alias != "" {
		return node.Alias, true
	}

	// Caso contrário, retorna schema.tabela se possível
	if node.RelationName != "" && node.Schema != "" {
		return node.Schema + "." + node.RelationName, true
	}

	// Busca recursivamente no próximo plano
	if len(node.Plans) > 0 {
		return getMainTable(node.Plans[0])
	}

	return "", false
}

func GetMainTableFromExplain(jsonData []byte) (string, error) {
	var explainOutput []Explain
	if err := json.Unmarshal(jsonData, &explainOutput); err != nil {
		return "", err
	}

	if len(explainOutput) == 0 {
		return "", fmt.Errorf("nenhum plano encontrado")
	}

	if table, found := getMainTable(explainOutput[0].Plan); found {
		return table, nil
	}

	return "", fmt.Errorf("tabela principal não encontrada")
}
