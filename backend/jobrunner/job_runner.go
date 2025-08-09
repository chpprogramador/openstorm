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
}

func NewJobRunner(sourceDB, destDB *sql.DB, sourceDSN, destDSN string, dialect dialects.SQLDialect, concurrency int, project string, projectID string) *JobRunner {
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
		Variables:      LoadProjectVariables(projectID),
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
	log.Printf("Iniciando job com leitura e escrita paralela via cursores: %s", job.JobName)

	jr.WaitGroup.Add(1)
	go func() {
		defer jr.WaitGroup.Done()

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

		//total de registros
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

		batchChan := make(chan []map[string]interface{}, jr.Concurrency*5)

		// Workers de escrita
		var writersWG sync.WaitGroup
		var processed int64
		var jobHadError atomic.Bool
		var lastErr atomic.Value

		for w := 0; w < jr.Concurrency; w++ {
			writersWG.Add(1)
			go func(workerID int) {
				defer writersWG.Done()
				for batch := range batchChan {
					batchStart := time.Now()
					batchLog := logger.BatchLog{
						Offset:    int(processed),
						Limit:     len(batch),
						Status:    "running",
						StartedAt: batchStart,
					}

					insertSQL, args := jr.Dialect.BuildInsertQuery(job, batch)
					if _, err := jr.DestinationDB.Exec(insertSQL, args...); err != nil {
						jobHadError.Store(true)
						lastErr.Store(err.Error())
						batchLog.Status = "error"
						batchLog.Error = err.Error()
						batchLog.EndedAt = time.Now()
						logger.AddBatch(jr.PipelineLog, jobID, batchLog)
						jr.savePipelineLog()
						continue
					}

					atomic.AddInt64(&processed, int64(len(batch)))
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
			}(w)
		}

		// Leitores paralelos com cursores
		numReaders := jr.Concurrency
		var readersWG sync.WaitGroup
		for r := 0; r < numReaders; r++ {
			readersWG.Add(1)
			go func(readerID int) {
				defer readersWG.Done()

				// Nova conexão exclusiva para o cursor
				srcConn, err := jr.SourceDB.Conn(context.Background())
				if err != nil {
					log.Printf("[Reader %d] erro ao abrir conexão: %v", readerID, err)
					jobHadError.Store(true)
					lastErr.Store(err.Error())
					return
				}
				defer srcConn.Close()

				cursorName := fmt.Sprintf("cur_reader_%d", readerID)
				if _, err := srcConn.ExecContext(context.Background(),
					fmt.Sprintf("BEGIN; DECLARE %s NO SCROLL CURSOR FOR %s", cursorName, job.SelectSQL)); err != nil {
					log.Printf("[Reader %d] erro ao criar cursor: %v", readerID, err)
					jobHadError.Store(true)
					lastErr.Store(err.Error())
					return
				}

				for {
					rows, err := srcConn.QueryContext(context.Background(),
						fmt.Sprintf("FETCH %d FROM %s", job.RecordsPerPage, cursorName))
					if err != nil {
						log.Printf("[Reader %d] erro no FETCH: %v", readerID, err)
						jobHadError.Store(true)
						lastErr.Store(err.Error())
						break
					}

					cols, _ := rows.Columns()
					var buffer []map[string]interface{}
					for rows.Next() {
						values := make([]interface{}, len(cols))
						ptrs := make([]interface{}, len(cols))
						for i := range cols {
							ptrs[i] = &values[i]
						}
						if err := rows.Scan(ptrs...); err != nil {
							jobHadError.Store(true)
							lastErr.Store(err.Error())
							break
						}
						rec := make(map[string]interface{})
						for i, col := range cols {
							rec[col] = values[i]
						}
						buffer = append(buffer, rec)
					}
					rows.Close()

					if len(buffer) == 0 {
						break // Cursor esgotado
					}
					batchChan <- buffer
				}

				// Fecha cursor
				srcConn.ExecContext(context.Background(), fmt.Sprintf("CLOSE %s; COMMIT;", cursorName))
			}(r)
		}

		// Fechamento do canal quando todos leitores terminarem
		go func() {
			readersWG.Wait()
			close(batchChan)
		}()

		writersWG.Wait()

		end := time.Now()
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

	// Substitui variáveis no SQL
	job.SelectSQL = jr.SubstituteVariables(job.SelectSQL)

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

	// Substitui variáveis no SQL
	job.SelectSQL = jr.SubstituteVariables(job.SelectSQL)

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

func (jr *JobRunner) runBatchDataTransfer(job models.Job, offset int, batchSize int) error {
	// Substitui variáveis antes de executar a query
	job.SelectSQL = jr.SubstituteVariables(job.SelectSQL)
	job.InsertSQL = jr.SubstituteVariables(job.InsertSQL)

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
