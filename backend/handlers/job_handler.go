package handlers

import (
	"database/sql"
	"encoding/json"
	"etl/models"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

func ListJobs(c *gin.Context) {
	projectID := c.Param("id")
	projectPath := filepath.Join("data", "projects", projectID, "project.json")
	projectBytes, err := ioutil.ReadFile(projectPath)
	if err != nil {
		c.JSON(500, gin.H{"error": "Projeto não encontrado"})
		return
	}

	var project models.Project
	if err := json.Unmarshal(projectBytes, &project); err != nil {
		c.JSON(500, gin.H{"error": "Erro ao ler project.json"})
		return
	}

	var jobs []models.Job
	for _, jobPath := range project.Jobs {
		fullJobPath := filepath.Join("data", "projects", projectID, jobPath)
		jobBytes, err := ioutil.ReadFile(fullJobPath)
		if err != nil {
			continue
		}
		var job models.Job
		if err := json.Unmarshal(jobBytes, &job); err == nil {
			jobs = append(jobs, job)
		}
	}
	c.JSON(200, jobs)
}

func AddJob(c *gin.Context) {
	projectID := c.Param("id")
	var job models.Job
	if err := c.BindJSON(&job); err != nil {
		c.JSON(500, gin.H{"error": "JSON inválido"})
		return
	}

	//job.ID = uuid.New().String()
	jobFileName := job.ID + ".json"
	projectDir := filepath.Join("data", "projects", projectID)
	jobsDir := filepath.Join(projectDir, "jobs")
	jobPath := filepath.Join(jobsDir, jobFileName)

	jobBytes, _ := json.MarshalIndent(job, "", "  ")
	if err := ioutil.WriteFile(jobPath, jobBytes, 0644); err != nil {
		c.JSON(500, gin.H{"error": "Erro ao salvar o job"})
		return
	}

	projectPath := filepath.Join(projectDir, "project.json")
	projectBytes, _ := ioutil.ReadFile(projectPath)
	var project models.Project
	_ = json.Unmarshal(projectBytes, &project)
	project.Jobs = append(project.Jobs, filepath.Join("jobs", jobFileName))
	updatedBytes, _ := json.MarshalIndent(project, "", "  ")
	_ = ioutil.WriteFile(projectPath, updatedBytes, 0644)

	c.JSON(201, job)
}

func UpdateJob(c *gin.Context) {
	projectID := c.Param("id")
	jobID := c.Param("jobId")
	var job models.Job
	if err := c.BindJSON(&job); err != nil {
		c.JSON(400, gin.H{"error": "JSON inválido"})
		return
	}

	job.ID = jobID
	jobFileName := jobID + ".json"
	jobPath := filepath.Join("data", "projects", projectID, "jobs", jobFileName)
	jobBytes, _ := json.MarshalIndent(job, "", "  ")
	if err := ioutil.WriteFile(jobPath, jobBytes, 0644); err != nil {
		c.JSON(500, gin.H{"error": "Erro ao atualizar o job"})
		return
	}
	c.JSON(200, job)
}

func DeleteJob(c *gin.Context) {
	projectID := c.Param("id")
	jobID := c.Param("jobId")
	jobFileName := jobID + ".json"
	projectDir := filepath.Join("data", "projects", projectID)
	jobPath := filepath.Join(projectDir, "jobs", jobFileName)

	if err := os.Remove(jobPath); err != nil {
		c.JSON(404, gin.H{"error": "Job não encontrado"})
		return
	}

	projectPath := filepath.Join(projectDir, "project.json")
	projectBytes, _ := ioutil.ReadFile(projectPath)
	var project models.Project
	_ = json.Unmarshal(projectBytes, &project)

	var updatedJobs []string
	for _, job := range project.Jobs {
		if !strings.Contains(job, jobFileName) {
			updatedJobs = append(updatedJobs, job)
		}
	}
	project.Jobs = updatedJobs
	updatedBytes, _ := json.MarshalIndent(project, "", "  ")
	_ = ioutil.WriteFile(projectPath, updatedBytes, 0644)

	c.JSON(200, gin.H{"message": "Job removido com sucesso!"})
}

func DeleteProject(c *gin.Context) {
	projectID := c.Param("id")
	projectDir := filepath.Join("data", "projects", projectID)

	if _, err := os.Stat(projectDir); os.IsNotExist(err) {
		c.JSON(404, gin.H{"error": "Projeto não encontrado"})
		return
	}

	err := os.RemoveAll(projectDir)
	if err != nil {
		c.JSON(500, gin.H{"error": "Erro ao excluir o projeto"})
		return
	}

	c.JSON(200, gin.H{"message": "Projeto excluído com sucesso"})
}

var db *sql.DB // Add this at the package level and initialize it elsewhere in your application

func ValidateJobHandler(c *gin.Context) {
	var req models.ValidateJobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ValidateJobResponse{
			Valid: false, Message: "Erro ao parsear JSON",
		})
		return
	}

	if req.Limit <= 0 {
		req.Limit = 10
	}

	tx, err := db.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ValidateJobResponse{
			Valid: false, Message: "Erro ao iniciar transação",
		})
		return
	}
	defer tx.Rollback()

	// Tenta extrair colunas com SELECT * LIMIT 0
	testSQL := fmt.Sprintf("SELECT * FROM (%s) AS t LIMIT 0", req.SelectSQL)
	rows, err := tx.Query(testSQL)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ValidateJobResponse{
			Valid: false, Message: fmt.Sprintf("Erro no SELECT: %v", err),
		})
		return
	}
	defer rows.Close()

	columns, _ := rows.Columns()
	insertCols, err := extractInsertColumns(req.InsertSQL)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ValidateJobResponse{
			Valid: false, Message: fmt.Sprintf("Erro no INSERT: %v", err),
		})
		return
	}

	if len(columns) != len(insertCols) {
		c.JSON(http.StatusBadRequest, models.ValidateJobResponse{
			Valid:   false,
			Message: fmt.Sprintf("Número de colunas incompatível. SELECT tem %d, INSERT tem %d", len(columns), len(insertCols)),
		})
		return
	}

	// Coleta dados para preview
	previewSQL := fmt.Sprintf("SELECT * FROM (%s) AS t LIMIT %d", req.SelectSQL, req.Limit)
	rowsPreview, err := tx.Query(previewSQL)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ValidateJobResponse{
			Valid: false, Message: fmt.Sprintf("Erro ao buscar preview: %v", err),
		})
		return
	}
	defer rowsPreview.Close()

	var preview []map[string]interface{}
	cols, _ := rowsPreview.Columns()
	for rowsPreview.Next() {
		values := make([]interface{}, len(cols))
		ptrs := make([]interface{}, len(cols))
		for i := range values {
			ptrs[i] = &values[i]
		}
		rowsPreview.Scan(ptrs...)
		row := make(map[string]interface{})
		for i, col := range cols {
			if b, ok := values[i].([]byte); ok {
				row[col] = string(b)
			} else {
				row[col] = values[i]
			}
		}
		preview = append(preview, row)
	}

	c.JSON(http.StatusOK, models.ValidateJobResponse{
		Columns: columns,
		Preview: preview,
		Valid:   true,
		Message: "Validação bem-sucedida",
	})
}

func extractInsertColumns(sql string) ([]string, error) {
	start := strings.Index(sql, "(")
	end := strings.Index(sql, ")")
	if start == -1 || end == -1 || end <= start {
		return nil, fmt.Errorf("não foi possível extrair colunas do INSERT")
	}

	raw := sql[start+1 : end]
	split := strings.Split(raw, ",")
	cols := make([]string, 0, len(split))
	for _, col := range split {
		cols = append(cols, strings.TrimSpace(col))
	}
	return cols, nil
}
