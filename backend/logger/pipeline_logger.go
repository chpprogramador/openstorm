package logger

import (
	"encoding/json"
	"etl/status"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type BatchLog struct {
	Offset    int       `json:"offset"`
	Limit     int       `json:"limit"`
	Status    string    `json:"status"`
	Error     string    `json:"error,omitempty"`
	Rows      int       `json:"rows"`
	StartedAt time.Time `json:"started_at"`
	EndedAt   time.Time `json:"ended_at"`
}

type JobLog struct {
	JobID       string     `json:"job_id"`
	JobName     string     `json:"job_name"`
	Status      string     `json:"status"`
	Error       string     `json:"error,omitempty"`
	StopOnError bool       `json:"stop_on_error"`
	StartedAt   time.Time  `json:"started_at"`
	EndedAt     time.Time  `json:"ended_at"`
	Processed   int        `json:"processed"`
	Total       int        `json:"total"`
	Batches     []BatchLog `json:"batches"`
}

type PipelineLog struct {
	PipelineID string    `json:"pipeline_id"`
	Project    string    `json:"project"`
	Status     string    `json:"status"`
	StartedAt  time.Time `json:"started_at"`
	EndedAt    time.Time `json:"ended_at"`
	Jobs       []JobLog  `json:"jobs"`
}

var mu sync.Mutex

func GeneratePipelineID(project string) string {
	return fmt.Sprintf("%s_%s", project, time.Now().Format("2006-01-02_15-04-05"))
}

func SavePipelineLog(log *PipelineLog) error {
	mu.Lock()
	defer mu.Unlock()

	// Cria o diretório logs se não existir
	logsDir := "logs"
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		return fmt.Errorf("erro ao criar diretório de logs: %v", err)
	}

	path := filepath.Join(logsDir, fmt.Sprintf("pipeline_%s.json", log.PipelineID))
	data, err := json.MarshalIndent(log, "", "  ")
	if err != nil {
		return fmt.Errorf("erro ao serializar log: %v", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("erro ao escrever arquivo de log: %v", err)
	}

	fmt.Printf("Log salvo em: %s\n", path)
	return nil
}

func LoadPipelineLog(pipelineID string) (*PipelineLog, error) {
	path := filepath.Join("logs", fmt.Sprintf("pipeline_%s.json", pipelineID))
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var log PipelineLog
	err = json.Unmarshal(data, &log)
	if err != nil {
		return nil, err
	}
	return &log, nil
}

func AddJob(log *PipelineLog, job JobLog) {
	mu.Lock()
	defer mu.Unlock()
	log.Jobs = append(log.Jobs, job)
	fmt.Printf("Job adicionado: %s (Status: %s)\n", job.JobName, job.Status)

	status.AppendLog(log.Project + " - Job: " + job.JobName + " iniciado")
}

func UpdateJob(log *PipelineLog, jobID string, updater func(*JobLog)) {
	mu.Lock()
	defer mu.Unlock()
	for i := range log.Jobs {
		if log.Jobs[i].JobID == jobID {
			oldStatus := log.Jobs[i].Status
			updater(&log.Jobs[i])
			fmt.Printf("Job atualizado: %s (%s -> %s)\n", log.Jobs[i].JobName, oldStatus, log.Jobs[i].Status)

			//status.AppendLog(log.Project + " - " + log.Jobs[i].JobName + " atualizado de " + oldStatus + " para " + log.Jobs[i].Status)
			if log.Jobs[i].Status == "done" {
				status.AppendLog(log.Project + " - Job: " + log.Jobs[i].JobName + " finalizado")
			} else if log.Jobs[i].Status == "error" {
				status.AppendLog(log.Project + " - Job: " + log.Jobs[i].JobName + " falhou")
			}
			break
		}
	}
}

func AddBatch(log *PipelineLog, jobID string, batch BatchLog) {
	mu.Lock()
	defer mu.Unlock()
	for i := range log.Jobs {
		if log.Jobs[i].JobID == jobID {
			log.Jobs[i].Batches = append(log.Jobs[i].Batches, batch)
			fmt.Printf("Batch adicionado ao job %s: offset %d, status %s\n",
				log.Jobs[i].JobName, batch.Offset, batch.Status)
			status.AppendLog(log.Project + " - Job: " + log.Jobs[i].JobName + " - Batch adicionado: offset " + fmt.Sprintf("%d", batch.Offset) + ", status " + batch.Status)
			break
		}
	}
}

// Função auxiliar para listar todos os logs de pipeline
func ListPipelineLogs() ([]string, error) {
	logsDir := "logs"
	files, err := os.ReadDir(logsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil // Diretório não existe ainda
		}
		return nil, err
	}

	var logs []string
	for _, file := range files {
		if filepath.Ext(file.Name()) == ".json" &&
			len(file.Name()) > 9 && file.Name()[:9] == "pipeline_" {
			logs = append(logs, file.Name())
		}
	}
	return logs, nil
}

// Função para obter estatísticas de um pipeline
func GetPipelineStats(pipelineID string) (map[string]interface{}, error) {
	log, err := LoadPipelineLog(pipelineID)
	if err != nil {
		return nil, err
	}

	stats := map[string]interface{}{
		"pipeline_id": log.PipelineID,
		"project":     log.Project,
		"status":      log.Status,
		"started_at":  log.StartedAt,
		"ended_at":    log.EndedAt,
		"duration":    log.EndedAt.Sub(log.StartedAt).String(),
		"total_jobs":  len(log.Jobs),
	}

	// Conta jobs por status
	jobStats := make(map[string]int)
	totalBatches := 0
	totalProcessed := 0

	for _, job := range log.Jobs {
		jobStats[job.Status]++
		totalBatches += len(job.Batches)
		totalProcessed += job.Processed
	}

	stats["job_stats"] = jobStats
	stats["total_batches"] = totalBatches
	stats["total_processed"] = totalProcessed

	return stats, nil
}
