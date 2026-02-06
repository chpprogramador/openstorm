package logger

import (
	"encoding/json"
	"etl/status"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type BatchLog struct {
	Offset    int       `json:"offset"`
	Limit     int       `json:"limit"`
	Status    string    `json:"status"`
	Error     string    `json:"error,omitempty"`
	ErrorType string    `json:"error_type,omitempty"` // "sql_error", "connection_error", "validation_error", etc.
	ErrorCode string    `json:"error_code,omitempty"` // Código específico do erro
	Rows      int       `json:"rows"`
	StartedAt time.Time `json:"started_at"`
	EndedAt   time.Time `json:"ended_at"`
}

type JobLog struct {
	JobID        string                 `json:"job_id"`
	JobName      string                 `json:"job_name"`
	Status       string                 `json:"status"`
	Error        string                 `json:"error,omitempty"`
	ErrorType    string                 `json:"error_type,omitempty"`    // "sql_error", "connection_error", "validation_error", etc.
	ErrorCode    string                 `json:"error_code,omitempty"`    // Código específico do erro
	ErrorDetails map[string]interface{} `json:"error_details,omitempty"` // Detalhes adicionais do erro
	StopOnError  bool                   `json:"stop_on_error"`
	StartedAt    time.Time              `json:"started_at"`
	EndedAt      time.Time              `json:"ended_at"`
	Processed    int                    `json:"processed"`
	Total        int                    `json:"total"`
	Batches      []BatchLog             `json:"batches"`
}

type PipelineLog struct {
	PipelineID string    `json:"pipeline_id"`
	ProjectID  string    `json:"project_id,omitempty"`
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
	println("Carregando log do pipeline:", pipelineID)
	path := filepath.Join("logs", fmt.Sprintf("%s.json", pipelineID))
	println("Caminho do arquivo de log:", path)
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
			logs = append(logs, strings.TrimPrefix(strings.TrimSuffix(file.Name(), ".json"), "\\"))
		}
	}

	return logs, nil
}

// UpdatePipelineLogsForProject atualiza logs existentes após renomear o projeto.
// Preenche project_id e atualiza o nome do projeto quando aplicável.

// Função para obter estatísticas de um pipeline
func GetPipelineStats(pipelineID string) (map[string]interface{}, error) {
	log, err := LoadPipelineLog(pipelineID)
	println("Obtendo estatísticas para o pipeline:", pipelineID)
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
		if job.Processed > 0 {
			totalProcessed += job.Processed
		} else {
			// Fallback: soma linhas dos batches quando o contador do job não estiver preenchido
			for _, batch := range job.Batches {
				totalProcessed += batch.Rows
			}
		}
	}

	stats["job_stats"] = jobStats
	stats["total_batches"] = totalBatches
	stats["total_processed"] = totalProcessed

	println("Estatísticas obtidas com sucesso para o pipeline:", stats)

	return stats, nil
}

// ErrorAnalyzer analisa e categoriza erros
type ErrorAnalyzer struct{}

// AnalyzeError analisa um erro e retorna informações estruturadas
func (ea *ErrorAnalyzer) AnalyzeError(err error) (errorType, errorCode string, details map[string]interface{}) {
	if err == nil {
		return "", "", nil
	}

	errorMsg := err.Error()
	details = make(map[string]interface{})
	details["original_error"] = errorMsg
	details["timestamp"] = time.Now()

	// Análise de erros SQL
	if strings.Contains(strings.ToLower(errorMsg), "duplicate key") ||
		strings.Contains(strings.ToLower(errorMsg), "unique constraint") {
		errorType = "duplicate_key_error"
		errorCode = "DUPLICATE_KEY"
		details["suggestion"] = "Verifique se há duplicatas na origem ou use UPSERT"
	} else if strings.Contains(strings.ToLower(errorMsg), "foreign key") {
		errorType = "foreign_key_error"
		errorCode = "FOREIGN_KEY_VIOLATION"
		details["suggestion"] = "Verifique se as referências existem na tabela de destino"
	} else if strings.Contains(strings.ToLower(errorMsg), "connection") ||
		strings.Contains(strings.ToLower(errorMsg), "timeout") {
		errorType = "connection_error"
		errorCode = "CONNECTION_FAILED"
		details["suggestion"] = "Verifique a conectividade com o banco de dados"
	} else if strings.Contains(strings.ToLower(errorMsg), "syntax error") ||
		strings.Contains(strings.ToLower(errorMsg), "invalid sql") {
		errorType = "sql_syntax_error"
		errorCode = "SQL_SYNTAX_ERROR"
		details["suggestion"] = "Verifique a sintaxe da query SQL"
	} else if strings.Contains(strings.ToLower(errorMsg), "permission") ||
		strings.Contains(strings.ToLower(errorMsg), "access denied") {
		errorType = "permission_error"
		errorCode = "PERMISSION_DENIED"
		details["suggestion"] = "Verifique as permissões do usuário do banco"
	} else if strings.Contains(strings.ToLower(errorMsg), "table") &&
		strings.Contains(strings.ToLower(errorMsg), "doesn't exist") {
		errorType = "table_not_found"
		errorCode = "TABLE_NOT_FOUND"
		details["suggestion"] = "Verifique se a tabela existe no banco de destino"
	} else {
		errorType = "unknown_error"
		errorCode = "UNKNOWN_ERROR"
		details["suggestion"] = "Consulte a documentação ou entre em contato com o suporte"
	}

	return errorType, errorCode, details
}

// GetErrorSummary retorna um resumo dos erros de um pipeline
func GetErrorSummary(pipelineID string) (map[string]interface{}, error) {
	log, err := LoadPipelineLog(pipelineID)
	if err != nil {
		return nil, err
	}

	summary := map[string]interface{}{
		"pipeline_id":   pipelineID,
		"total_errors":  0,
		"error_types":   make(map[string]int),
		"error_jobs":    []map[string]interface{}{},
		"error_batches": []map[string]interface{}{},
	}

	analyzer := &ErrorAnalyzer{}

	for _, job := range log.Jobs {
		if job.Error != "" {
			errorType, errorCode, details := analyzer.AnalyzeError(fmt.Errorf(job.Error))

			jobError := map[string]interface{}{
				"job_id":        job.JobID,
				"job_name":      job.JobName,
				"error":         job.Error,
				"error_type":    errorType,
				"error_code":    errorCode,
				"error_details": details,
				"started_at":    job.StartedAt,
				"ended_at":      job.EndedAt,
			}

			summary["error_jobs"] = append(summary["error_jobs"].([]map[string]interface{}), jobError)
			summary["total_errors"] = summary["total_errors"].(int) + 1

			errorTypes := summary["error_types"].(map[string]int)
			errorTypes[errorType]++
		}

		// Analisa erros de batches
		for _, batch := range job.Batches {
			if batch.Error != "" {
				errorType, errorCode, details := analyzer.AnalyzeError(fmt.Errorf(batch.Error))

				batchError := map[string]interface{}{
					"job_id":        job.JobID,
					"job_name":      job.JobName,
					"batch_offset":  batch.Offset,
					"batch_limit":   batch.Limit,
					"error":         batch.Error,
					"error_type":    errorType,
					"error_code":    errorCode,
					"error_details": details,
					"started_at":    batch.StartedAt,
					"ended_at":      batch.EndedAt,
				}

				summary["error_batches"] = append(summary["error_batches"].([]map[string]interface{}), batchError)
				summary["total_errors"] = summary["total_errors"].(int) + 1

				errorTypes := summary["error_types"].(map[string]int)
				errorTypes[errorType]++
			}
		}
	}

	return summary, nil
}
