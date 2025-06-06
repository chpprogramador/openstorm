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
	ConnMap       map[string][]string // map[sourceID][]targetIDs
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

	jr.Semaphore <- struct{}{}
	jr.WaitGroup.Add(1)
	go func() {
		defer func() {
			<-jr.Semaphore
			jr.WaitGroup.Done()
		}()

		log.Printf("Executando job: %s\n", job.JobName)
		start := time.Now()
		status.UpdateJobStatus(job.ID, func(js *status.JobStatus) {
			js.Name = job.JobName
			js.Status = "running"
			js.StartedAt = &start
			status.NotifySubscribers()
		})

		batches, err := jr.Dialect.FetchBatches(jr.SourceDB, job)
		if err != nil {
			log.Printf("Erro ao buscar dados: %v\n", err)
			status.UpdateJobStatus(job.ID, func(js *status.JobStatus) {
				js.Status = "error"
				js.Error = err.Error()
				status.NotifySubscribers()
			})
			return
		}

		processed := 0
		total := 0
		for _, batch := range batches {
			total += len(batch)
		}
		status.UpdateJobStatus(job.ID, func(js *status.JobStatus) {
			js.Total = total
			status.NotifySubscribers()
		})

		for _, batch := range batches {
			if err := jr.Dialect.InsertBatch(jr.DestinationDB, job.InsertSQL, batch); err != nil {
				log.Printf("Erro ao inserir batch: %v\n", err)
				status.UpdateJobStatus(job.ID, func(js *status.JobStatus) {
					js.Status = "error"
					js.Error = err.Error()
					status.NotifySubscribers()
				})
				return
			}
			processed += len(batch)
			progress := float64(processed) / float64(total) * 100
			status.UpdateJobStatus(job.ID, func(js *status.JobStatus) {
				js.Processed = processed
				js.Progress = progress
				status.NotifySubscribers()
			})
		}

		log.Printf("Job %s finalizado\n", job.JobName)
		end := time.Now()
		status.UpdateJobStatus(job.ID, func(js *status.JobStatus) {
			js.Status = "done"
			js.EndedAt = &end
			status.NotifySubscribers()
		})

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
