package handlers

import (
	"database/sql"
	"encoding/json"
	"etl/dialects"
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
	project, err := loadProjectFile(projectPath)
	if err != nil {
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
	project, _ := loadProjectFile(projectPath)
	project.Jobs = append(project.Jobs, filepath.Join("jobs", jobFileName))
	_ = writeProjectFile(projectPath, &project)

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
	project, _ := loadProjectFile(projectPath)

	var updatedJobs []string
	for _, job := range project.Jobs {
		if !strings.Contains(job, jobFileName) {
			updatedJobs = append(updatedJobs, job)
		}
	}
	project.Jobs = updatedJobs
	_ = writeProjectFile(projectPath, &project)

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

func ValidateJobHandler(c *gin.Context) {

	println("Validando job...\n")
	var req models.ValidateJobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ValidateJobResponse{
			Valid: false, Message: "Erro ao parsear JSON",
		})
		println("Erro ao parsear JSON: %v\n", err)
		return
	}

	projectID := req.ProjectID
	projectPath := filepath.Join("data", "projects", projectID, "project.json")
	project, err := loadProjectFile(projectPath)
	if err != nil {
		c.JSON(500, gin.H{"error": "Erro ao ler project.json"})
		return
	}

	decryptProjectFields(&project)

	// Conecta ao banco de dados de origem
	sourceDB, err := sql.Open(project.SourceDatabase.Type, buildDSN(project.SourceDatabase))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao conectar no banco de origem"})
		return
	}

	// Conecta ao banco de dados de destino
	destDB, err := sql.Open(project.DestinationDatabase.Type, buildDSN(project.DestinationDatabase))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao conectar no banco de destino"})
		return
	}

	if req.Limit <= 0 {
		req.Limit = 10
	}

	tx, err := sourceDB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ValidateJobResponse{
			Valid: false, Message: "Erro ao iniciar transação",
		})
		return
	}
	defer tx.Rollback()

	// Limpa quebras de linha para evitar problemas no PostgreSQL
	cleanSelectSQL := cleanSQLNewlines(req.SelectSQL)

	// Não modificamos a query original para validação
	// Isso evita problemas com funções de janela (window functions) e outras construções SQL complexas
	
	// Usa subconsulta para aplicar LIMIT 1 de forma segura na query original
	validationSQL := fmt.Sprintf("SELECT * FROM (%s) AS t LIMIT 1", cleanSelectSQL)

	// Tenta extrair colunas
	rows, err := tx.Query(validationSQL)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ValidateJobResponse{
			Valid: false, Message: fmt.Sprintf("Erro no SELECT: %v", err),
		})
		return
	}
	defer rows.Close()

	columns, _ := rows.Columns()

	// Limpa quebras de linha para evitar problemas no PostgreSQL
	cleanInsertSQL := cleanSQLNewlines(req.InsertSQL)

	insertCols, err := extractInsertColumns(cleanInsertSQL)
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

	// Teste de INSERT com rollback

	txd, err := destDB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ValidateJobResponse{
			Valid: false, Message: "Erro ao iniciar transação para INSERT",
		})
		return
	}
	defer txd.Rollback() // Garante o rollback

	// Obter um registro do SourceDB para o INSERT
	var values []interface{}
	_, selectTestSQL := dialects.AnalyzeAndModifySQL(req.SelectSQL)
	selectTestSQL = selectTestSQL + " LIMIT 0"
	selectColumns := columns
	rows2, err := sourceDB.Query(selectTestSQL)
	if err == nil {
		defer rows2.Close()
		if rows2.Next() {
			values = make([]interface{}, len(selectColumns))
			scanArgs := make([]interface{}, len(selectColumns))
			for i := range values {
				scanArgs[i] = &values[i]
			}
			rows2.Scan(scanArgs...)
		}
	}

	if len(values) > 0 {

		_, modifiedSQL := dialects.AnalyzeAndModifySQL(req.SelectSQL)
		modifiedSQL = modifiedSQL + " LIMIT 0" // Limita a 0 para otimizar a contagem

		//testSQL := fmt.Sprintf(" SELECT * FROM (%s) AS t LIMIT 0", req.SelectSQL)
		insertStmt := fmt.Sprintf("%s %s", req.InsertSQL, modifiedSQL)

		_, err = txd.Exec(insertStmt)
		if err != nil {
			c.JSON(http.StatusBadRequest, models.ValidateJobResponse{
				Valid: false, Message: fmt.Sprintf("Erro no INSERT: %v", err),
			})
			return
		}
	}

	c.JSON(http.StatusOK, models.ValidateJobResponse{
		Columns: columns,
		Valid:   true,
		Message: "Validação bem-sucedida",
	})
}

func cleanSQLNewlines(sql string) string {
	// Preserva quebras de linha em comentários de linha (--) para não comentar o restante da query.
	// Normaliza quebras de linha fora de strings/comentários.

	var result strings.Builder
	inString := false
	inDouble := false
	inLineComment := false
	inBlockComment := false

	for i := 0; i < len(sql); i++ {
		char := sql[i]

		if inLineComment {
			if char == '\r' || char == '\n' {
				if char == '\r' && i+1 < len(sql) && sql[i+1] == '\n' {
					i++
				}
				inLineComment = false
				result.WriteByte('\n')
				continue
			}
			result.WriteByte(char)
			continue
		}

		if inBlockComment {
			if char == '*' && i+1 < len(sql) && sql[i+1] == '/' {
				inBlockComment = false
				result.WriteByte(char)
				result.WriteByte(sql[i+1])
				i++
				continue
			}
			if char == '\r' || char == '\n' {
				if char == '\r' && i+1 < len(sql) && sql[i+1] == '\n' {
					i++
				}
				result.WriteByte(' ')
				continue
			}
			result.WriteByte(char)
			continue
		}

		// Verifica se estamos dentro ou fora de uma string literal (aspas simples)
		if inString {
			if char == '\'' {
				if i+1 < len(sql) && sql[i+1] == '\'' {
					result.WriteByte(char)
					result.WriteByte(sql[i+1])
					i++
					continue
				}
				inString = false
			}
			result.WriteByte(char)
			continue
		}

		if inDouble {
			if char == '"' {
				if i+1 < len(sql) && sql[i+1] == '"' {
					result.WriteByte(char)
					result.WriteByte(sql[i+1])
					i++
					continue
				}
				inDouble = false
			}
			result.WriteByte(char)
			continue
		}

		if char == '\'' {
			inString = true
			result.WriteByte(char)
			continue
		}
		if char == '"' {
			inDouble = true
			result.WriteByte(char)
			continue
		}
		if char == '-' && i+1 < len(sql) && sql[i+1] == '-' {
			inLineComment = true
			result.WriteByte(char)
			result.WriteByte(sql[i+1])
			i++
			continue
		}
		if char == '/' && i+1 < len(sql) && sql[i+1] == '*' {
			inBlockComment = true
			result.WriteByte(char)
			result.WriteByte(sql[i+1])
			i++
			continue
		}

		if char == '\n' || char == '\r' {
			if char == '\r' && i+1 < len(sql) && sql[i+1] == '\n' {
				i++
			}
			result.WriteByte(' ')
			continue
		}

		result.WriteByte(char)
	}

	return result.String()
}

func extractInsertColumns(sql string) ([]string, error) {
	println("Extraindo colunas do INSERT...\n")
	start := strings.Index(sql, "(")
	end := strings.Index(sql, ")")
	if start == -1 || end == -1 || end <= start {
		println("Erro: não foi possível encontrar parênteses no INSERT\n")
		return nil, fmt.Errorf("não foi possível extrair colunas do INSERT")
	}

	println("Colunas encontradas entre parênteses: %s\n", sql[start+1:end])
	raw := sql[start+1 : end]
	split := strings.Split(raw, ",")
	cols := make([]string, 0, len(split))
	for _, col := range split {
		cols = append(cols, strings.TrimSpace(col))
	}
	println("Colunas extraídas: %v\n", cols)
	return cols, nil
}
