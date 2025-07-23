package jobrunner

import (
	"database/sql"
	"etl/dialects"
	"etl/logger"
	"etl/models"
	"etl/status"
	"fmt"
	"log"
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
	PipelineLog    *logger.PipelineLog // Adicionado para logging
}

func NewJobRunner(sourceDB, destDB *sql.DB, sourceDSN, destDSN string, dialect dialects.SQLDialect, concurrency int, project string) *JobRunner {
	// Inicializa o log do pipeline
	pipelineLog := &logger.PipelineLog{
		PipelineID: logger.GeneratePipelineID(project),
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
		PipelineLog:    pipelineLog,
	}

	// Salva o log inicial do pipeline
	jr.savePipelineLog()

	return jr
}

func (jr *JobRunner) RunJob(jobID string) {
	job, ok := jr.JobMap[jobID]
	if !ok {
		log.Printf("Job %s não encontrado\n", jobID)
		return
	}

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
	log.Printf("Iniciando job de inserção: %s\n", job.JobName)

	jr.WaitGroup.Add(1)
	go func() {
		defer jr.WaitGroup.Done()

		log.Printf("Iniciando job %s\n", job.JobName)
		start := time.Now()

		// Inicializa o log do job
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

		total, err := jr.Dialect.FetchTotalCount(jr.SourceDB, job)
		if err != nil {
			log.Printf("Erro ao contar registros: %v\n", err)

			// Atualiza log do job com erro
			logger.UpdateJob(jr.PipelineLog, jobID, func(jl *logger.JobLog) {
				jl.Status = "error"
				jl.Error = err.Error()
				jl.EndedAt = time.Now()
			})
			jr.savePipelineLog()

			status.UpdateJobStatus(job.ID, func(js *status.JobStatus) {
				js.Status = "error"
				js.Error = err.Error()
				status.NotifySubscribers()
			})
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

		var batchWG sync.WaitGroup
		var mu sync.Mutex
		var processed int
		var jobHadError atomic.Bool
		var lastErr atomic.Value // armazena string

		log.Printf("Total de registros a serem processados: %d\n", total)

		runBatch := func(offset int) {
			defer func() {
				<-jr.Semaphore
				batchWG.Done()
			}()

			log.Printf("Iniciando batch de offset %d\n", offset)
			batchStart := time.Now()

			// Inicializa log do batch
			batchLog := logger.BatchLog{
				Offset:    offset,
				Limit:     job.RecordsPerPage,
				Status:    "running",
				StartedAt: batchStart,
			}

			var affected int64

			if jr.SourceDSN == jr.DestinationDSN {
				batchSQL := jr.Dialect.BuildPaginatedInsertQuery(job, offset, job.RecordsPerPage)
				log.Printf("SQL gerado: %s\n", batchSQL)

				res, err := jr.DestinationDB.Exec(batchSQL)
				if err != nil {
					log.Printf("Erro ao executar batch: %v\n", err)
					jobHadError.Store(true)
					lastErr.Store(err.Error())

					// Atualiza log do batch com erro
					batchLog.Status = "error"
					batchLog.Error = err.Error()
					batchLog.EndedAt = time.Now()
					logger.AddBatch(jr.PipelineLog, jobID, batchLog)
					jr.savePipelineLog()

					status.UpdateJobStatus(job.ID, func(js *status.JobStatus) {
						js.Status = "error"
						js.Error = err.Error()
						status.NotifySubscribers()
					})
					return
				}
				affected, err = res.RowsAffected()
				if err != nil {
					log.Printf("Erro ao obter RowsAffected: %v\n", err)
				}
			} else {
				err := jr.runBatchDataTransfer(job, offset, job.RecordsPerPage)
				if err != nil {
					log.Printf("Erro na transferência de dados entre bancos: %v\n", err)
					jobHadError.Store(true)
					lastErr.Store(err.Error())

					// Atualiza log do batch com erro
					batchLog.Status = "error"
					batchLog.Error = err.Error()
					batchLog.EndedAt = time.Now()
					logger.AddBatch(jr.PipelineLog, jobID, batchLog)
					jr.savePipelineLog()

					status.UpdateJobStatus(job.ID, func(js *status.JobStatus) {
						js.Status = "error"
						js.Error = err.Error()
						status.NotifySubscribers()
					})
					return
				}
				affected = int64(job.RecordsPerPage)
			}

			// Batch executado com sucesso
			batchLog.Status = "done"
			batchLog.Rows = int(affected)
			batchLog.EndedAt = time.Now()
			logger.AddBatch(jr.PipelineLog, jobID, batchLog)
			jr.savePipelineLog()

			mu.Lock()
			processed += int(affected)
			progress := float64(processed) / float64(total) * 100
			mu.Unlock()

			// Atualiza progresso no log do job
			logger.UpdateJob(jr.PipelineLog, jobID, func(jl *logger.JobLog) {
				jl.Processed = processed
			})
			jr.savePipelineLog()

			status.UpdateJobStatus(job.ID, func(js *status.JobStatus) {
				js.Processed = processed
				js.Progress = progress
				status.NotifySubscribers()
			})
		}

		for offset := 0; offset < total; offset += job.RecordsPerPage {
			jr.Semaphore <- struct{}{}
			batchWG.Add(1)
			go runBatch(offset)
		}

		batchWG.Wait()

		end := time.Now()

		// Se houve erro, finalize como erro e pare se necessário
		if jobHadError.Load() {
			errMsg := "erro durante a execução dos batches"
			if last := lastErr.Load(); last != nil {
				errMsg = last.(string)
			}

			log.Printf("Job %s terminou com erro: %s\n", job.JobName, errMsg)

			// Atualiza log do job com erro
			logger.UpdateJob(jr.PipelineLog, jobID, func(jl *logger.JobLog) {
				jl.Status = "error"
				jl.Error = errMsg
				jl.EndedAt = end
			})
			jr.savePipelineLog()

			status.UpdateJobStatus(job.ID, func(js *status.JobStatus) {
				js.Status = "error"
				js.EndedAt = &end
				js.Error = errMsg
				status.NotifySubscribers()
			})

			if job.StopOnError {
				log.Printf("Job %s falhou e StopOnError está ativo. Não executando dependentes.\n", jobID)
				jr.PipelineLog.Status = "error"
				jr.PipelineLog.EndedAt = end
				jr.savePipelineLog()
				status.UpdateProjectStatus("error")
				return
			}
			// Mesmo com erro, continua se StopOnError for false
		} else {
			log.Printf("Job %s finalizado com sucesso\n", job.JobName)

			// Atualiza log do job como concluído
			logger.UpdateJob(jr.PipelineLog, jobID, func(jl *logger.JobLog) {
				jl.Status = "done"
				jl.EndedAt = end
			})
			jr.savePipelineLog()

			status.UpdateJobStatus(job.ID, func(js *status.JobStatus) {
				js.Status = "done"
				js.EndedAt = &end
				status.NotifySubscribers()
			})
		}

		// Executa os próximos jobs, se houver
		nextJobs := jr.ConnMap[jobID]
		for _, nextID := range nextJobs {
			jr.RunJob(nextID)
		}
	}()
}

func (jr *JobRunner) runExecutionJob(jobID string, job models.Job) {
	log.Printf("Executando job de execução: %s\n", job.JobName)
	start := time.Now()

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
		status.NotifySubscribers()
	})

	_, err := jr.DestinationDB.Exec(job.SelectSQL)
	end := time.Now()

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

	var result bool
	err := jr.SourceDB.QueryRow(job.SelectSQL).Scan(&result)
	end := time.Now()

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

func (jr *JobRunner) Run(startIDs []string) {
	log.Printf("Iniciando pipeline %s para projeto %s\n", jr.PipelineLog.PipelineID, jr.PipelineLog.Project)

	for _, id := range startIDs {
		jr.RunJob(id)
	}
	jr.WaitGroup.Wait()

	// Finaliza o pipeline
	jr.PipelineLog.EndedAt = time.Now()
	if jr.PipelineLog.Status == "running" {
		jr.PipelineLog.Status = "done"
	}
	jr.savePipelineLog()

	log.Printf("Pipeline %s finalizado com status: %s\n", jr.PipelineLog.PipelineID, jr.PipelineLog.Status)
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

func (jr *JobRunner) runBatchDataTransfer(job models.Job, offset int, batchSize int) error {
	query := jr.Dialect.BuildPaginatedSelectQuery(job, offset, batchSize)
	rows, err := jr.SourceDB.Query(query)
	if err != nil {
		return err
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return err
	}

	var records []map[string]interface{}

	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range columns {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return err
		}

		record := make(map[string]interface{})
		for i, col := range columns {
			record[col] = values[i]
		}
		records = append(records, record)
	}

	if len(records) == 0 {
		return nil
	}

	insertSQL, args := jr.Dialect.BuildInsertQuery(job, records)
	_, err = jr.DestinationDB.Exec(insertSQL, args...)
	return err
}
