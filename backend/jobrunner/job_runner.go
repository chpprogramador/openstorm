package jobrunner

import (
	"database/sql"
	"etl/dialects"
	"etl/models"
	"etl/status"
	"log"
	"sync"
	"time"
)

type JobRunner struct {
	SourceDB      *sql.DB
	DestinationDB *sql.DB
	Dialect       dialects.SQLDialect
	Concurrency   int
	Semaphore     chan struct{}
	WaitGroup     *sync.WaitGroup
	JobMap        map[string]models.Job
	ConnMap       map[string][]string
}

func NewJobRunner(sourceDB, destDB *sql.DB, dialect dialects.SQLDialect, concurrency int) *JobRunner {
	return &JobRunner{
		SourceDB:      sourceDB,
		DestinationDB: destDB,
		Dialect:       dialect,
		Concurrency:   concurrency,
		Semaphore:     make(chan struct{}, concurrency),
		WaitGroup:     &sync.WaitGroup{},
		JobMap:        make(map[string]models.Job),
		ConnMap:       make(map[string][]string),
	}
}

func (jr *JobRunner) RunJob(jobID string) {
	job, ok := jr.JobMap[jobID]
	if !ok {
		log.Printf("Job %s não encontrado\n", jobID)
		return
	}

	jr.WaitGroup.Add(1)
	go func() {
		defer jr.WaitGroup.Done()

		log.Printf("Executando job: %s\n", job.JobName)
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
		processed := 0

		// Função para executar um batch paralelo
		runBatch := func(offset int) {
			defer func() {
				<-jr.Semaphore
				batchWG.Done()
			}()

			batchSQL := jr.Dialect.BuildPaginatedInsertQuery(job, offset, job.RecordsPerPage)

			res, err := jr.DestinationDB.Exec(batchSQL)
			if err != nil {
				log.Printf("Erro ao executar batch: %v\nSQL: %s\n", err, batchSQL)
				status.UpdateJobStatus(job.ID, func(js *status.JobStatus) {
					js.Status = "error"
					js.Error = err.Error()
					status.NotifySubscribers()
				})
				return
			}

			affected, err := res.RowsAffected()
			if err != nil {
				log.Printf("Erro ao obter RowsAffected: %v\n", err)
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

		// Inicia os batches em paralelo limitados por Semaphore
		for offset := 0; offset < total; offset += job.RecordsPerPage {
			jr.Semaphore <- struct{}{}
			batchWG.Add(1)
			go runBatch(offset)
		}

		// Espera todos batches terminarem
		batchWG.Wait()

		log.Printf("Job %s finalizado\n", job.JobName)
		end := time.Now()
		status.UpdateJobStatus(job.ID, func(js *status.JobStatus) {
			js.Status = "done"
			js.EndedAt = &end
			status.NotifySubscribers()
		})

		// Executa os próximos jobs dependentes
		nextJobs := jr.ConnMap[jobID]
		for _, nextID := range nextJobs {
			jr.RunJob(nextID)
		}
	}()
}

func (jr *JobRunner) Run(startIDs []string) {
	for _, id := range startIDs {
		jr.RunJob(id)
	}
	jr.WaitGroup.Wait()
}
