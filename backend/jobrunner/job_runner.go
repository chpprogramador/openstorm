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
	"strings"
	"sync"
	"sync/atomic"
	"time"
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
	Variables      map[string]string
	PipelineLog    *logger.PipelineLog // Adicionado para logging
	ProjectID      string
	ctx            context.Context
	cancel         context.CancelFunc
	stopped        atomic.Bool
	stopReason     atomic.Value
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
		Variables:      LoadProjectVariables(projectID),
		PipelineLog:    pipelineLog,
		ProjectID:      projectID,
	}
	jr.ctx, jr.cancel = context.WithCancel(context.Background())

	// Salva o log inicial do pipeline
	jr.savePipelineLog()

	return jr
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
	case "condition":
		jr.runConditionJob(jobID, job)
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

		// total de registros
		total, err := jr.Dialect.FetchTotalCount(jr.SourceDB, job)
		if err != nil {
			log.Printf("Erro ao contar registros: %v\n", err)
			jr.markJobFinalStatus(jobID, job, "error", err.Error(), time.Now())
			return
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

		// Writer unico com transacao unica por job
		var writerWG sync.WaitGroup
		writerWG.Add(1)
		go func() {
			defer writerWG.Done()

			tx, err := jr.DestinationDB.Begin()
			if err != nil {
				jobHadError.Store(true)
				lastErr.Store(err.Error())
				reportJobError(err.Error())
				jobCancel()
				// Drena batches ate o canal fechar
				for range batchChan {
				}
				return
			}

			committed := false
			defer func() {
				if !committed {
					_ = tx.Rollback()
				}
			}()

			discard := false
			for batch := range batchChan {
				if discard {
					continue
				}
				if jr.shouldStop() || jobCtx.Err() != nil {
					discard = true
					continue
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
				log.Printf("INSERT SQL (job=%s): %s", job.ID, insertSQL)

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
					discard = true
					continue
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

		// Leitura paralela por bucket (cada worker lê o seu)
		var readersWG sync.WaitGroup
		for w := 0; w < concurrency; w++ {
			readersWG.Add(1)
			go func(workerID int) {
				defer readersWG.Done()
				if jr.shouldStop() || jobCtx.Err() != nil {
					return
				}

				queryExplain := jr.Dialect.BuildExplainSelectQueryByHash(job)
				rowsExplain, err := jr.SourceDB.Query(queryExplain)
				if err != nil {
					panic(err)
				}
				defer rowsExplain.Close()

				var explainJSON []byte
				for rowsExplain.Next() {
					var col string
					if err := rowsExplain.Scan(&col); err != nil {
						panic(err)
					}
					explainJSON = []byte(col)
				}

				mainTable, err := GetMainTableFromExplain(explainJSON)
				if err != nil {
					panic(err)
				}

				query := jr.Dialect.BuildSelectQueryByHash(job, workerID, concurrency, mainTable)
				rows, err := jr.SourceDB.Query(query)
				if err != nil {
					log.Printf("Erro na query do bucket %d: %v", workerID, err)
					jobHadError.Store(true)
					lastErr.Store(err.Error())
					reportJobError(err.Error())
					jobCancel()
					return
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
	log.Printf("Executando job de execução: %s\n", job.JobName)
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
		Total:       1, // Jobs de execução processam 1 comando
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

	// Substitui variáveis no SQL
	job.SelectSQL = jr.SubstituteVariables(job.SelectSQL)

	// Preserva quebras de linha para não afetar comentários
	cleanSQL := strings.ReplaceAll(job.SelectSQL, "\r\n", "\n")
	cleanSQL = strings.ReplaceAll(cleanSQL, "\r", "\n")
	log.Printf("EXECUTION SQL (job=%s): %s", job.ID, cleanSQL)

	_, err := jr.DestinationDB.Exec(cleanSQL)
	end := time.Now()

	if jr.shouldStop() {
		jr.markJobFinalStatus(jobID, job, "error", "pipeline interrompida", end)
		return
	}

	if err != nil {
		log.Printf("Erro no job de execução: %v\n", err)

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

		// Verifica se job falhou e se deve parar em erro
		if job.StopOnError {
			log.Printf("Job %s falhou e StopOnError está ativo. Não executando dependentes.\n", jobID)
			jr.PipelineLog.Status = "error"
			jr.PipelineLog.EndedAt = end
			jr.savePipelineLog()
			status.UpdateProjectStatus("error")
			return
		}
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
			js.EndedAt = &end
			status.NotifySubscribers()
		})
	}

	// Chama próximos jobs
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

	var result bool
	err := jr.SourceDB.QueryRow(job.SelectSQL).Scan(&result)
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
