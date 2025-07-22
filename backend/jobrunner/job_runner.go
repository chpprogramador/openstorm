package jobrunner

import (
	"database/sql"
	"etl/dialects"
	"etl/models"
	"etl/status"
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
}

func NewJobRunner(sourceDB, destDB *sql.DB, sourceDSN, destDSN string, dialect dialects.SQLDialect, concurrency int) *JobRunner {
	return &JobRunner{
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
	}
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
	}
}

func (jr *JobRunner) runInsertJob(jobID string, job models.Job) {
	log.Printf("Iniciando job de inserção: %s\n", job.JobName)

	jr.WaitGroup.Add(1)
	go func() {
		defer jr.WaitGroup.Done()

		log.Printf("Iniciando job %s\n", job.JobName)
		start := time.Now()

		status.UpdateJobStatus(job.ID, func(js *status.JobStatus) {
			js.Name = job.JobName
			js.Status = "running"
			js.StartedAt = &start
			status.NotifySubscribers()
		})

		total, err := jr.Dialect.FetchTotalCount(jr.SourceDB, job)
		if err != nil {
			log.Printf("Erro ao contar registros: %v\n", err)
			status.UpdateJobStatus(job.ID, func(js *status.JobStatus) {
				js.Status = "error"
				js.Error = err.Error()
				status.NotifySubscribers()
			})
			return
		}

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

			var affected int64

			if jr.SourceDSN == jr.DestinationDSN {
				batchSQL := jr.Dialect.BuildPaginatedInsertQuery(job, offset, job.RecordsPerPage)
				log.Printf("SQL gerado: %s\n", batchSQL)

				res, err := jr.DestinationDB.Exec(batchSQL)
				if err != nil {
					log.Printf("Erro ao executar batch: %v\n", err)
					jobHadError.Store(true)
					lastErr.Store(err.Error())

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

					status.UpdateJobStatus(job.ID, func(js *status.JobStatus) {
						js.Status = "error"
						js.Error = err.Error()
						status.NotifySubscribers()
					})
					return
				}
				affected = int64(job.RecordsPerPage)
			}

			mu.Lock()
			processed += int(affected)
			progress := float64(processed) / float64(total) * 100
			mu.Unlock()

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

			status.UpdateJobStatus(job.ID, func(js *status.JobStatus) {
				js.Status = "error"
				js.EndedAt = &end
				js.Error = errMsg
				status.NotifySubscribers()
			})

			if job.StopOnError {
				log.Printf("Job %s falhou e StopOnError está ativo. Não executando dependentes.\n", jobID)
				status.UpdateProjectStatus("error")
				return
			}
			// Mesmo com erro, continua se StopOnError for false
		} else {
			log.Printf("Job %s finalizado com sucesso\n", job.JobName)
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
	status.UpdateJobStatus(job.ID, func(js *status.JobStatus) {
		js.Name = job.JobName
		js.Status = "running"
		status.NotifySubscribers()
	})

	_, err := jr.DestinationDB.Exec(job.SelectSQL)
	if err != nil {
		log.Printf("Erro no job de execução: %v\n", err)
		status.UpdateJobStatus(job.ID, func(js *status.JobStatus) {
			js.Status = "error"
			js.Error = err.Error()
			status.NotifySubscribers()
		})
		// Verifica se job falhou e se deve parar em erro
		jobStatus := status.GetJobStatus(jobID)
		if job.StopOnError && jobStatus != nil && jobStatus.Status == "error" {
			log.Printf("Job %s falhou e StopOnError está ativo. Não executando dependentes.\n", jobID)
			status.UpdateProjectStatus("error")
			return
		}
	} else {
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
	status.UpdateJobStatus(job.ID, func(js *status.JobStatus) {
		js.Name = job.JobName
		js.Status = "running"
		status.NotifySubscribers()
	})

	var result bool
	err := jr.SourceDB.QueryRow(job.SelectSQL).Scan(&result)
	if err != nil {
		log.Printf("Erro ao executar condição: %v\n", err)
		status.UpdateJobStatus(job.ID, func(js *status.JobStatus) {
			js.Status = "error"
			js.Error = err.Error()
			status.NotifySubscribers()
		})
		return
	}

	if !result {
		log.Printf("Condição falhou: %s\n", job.JobName)
		status.UpdateJobStatus(job.ID, func(js *status.JobStatus) {
			js.Status = "error"
			js.Error = "Condição retornou falso"
			status.NotifySubscribers()
		})
		jobStatus := status.GetJobStatus(jobID)
		if job.StopOnError && jobStatus != nil {
			log.Printf("Job %s falhou e StopOnError está ativo. Não executando dependentes.\n", jobID)
			status.UpdateProjectStatus("error")
			return
		}

	} else {
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
	for _, id := range startIDs {
		jr.RunJob(id)
	}
	jr.WaitGroup.Wait()
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
