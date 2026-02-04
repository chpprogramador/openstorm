package handlers

import (
	"database/sql"
	"encoding/json"
	"etl/dialects"
	"etl/models"
	"etl/status"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"etl/jobrunner"

	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql" // MySQL
	_ "github.com/lib/pq"              // Postgres
)

func RunProject(c *gin.Context) {

	status.ClearJobLogs()

	//l? o JSON do projeto
	projectID := c.Param("id")
	projectPath := filepath.Join("data", "projects", projectID, "project.json")
	project, err := loadProjectFile(projectPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao ler project.json"})
		log.Println("Erro ao ler project.json:", err)
		return
	}
	log.Printf("Projeto %s carregado com sucesso", project.ProjectName)

	decryptProjectFields(&project)

	// Busca o dialeto apropriado
	dialect, err := dialects.NewDialect(project.SourceDatabase.Type)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		log.Println("Erro ao criar dialeto:", err)
		return
	}
	log.Printf("Dialeto %s criado com sucesso", project.SourceDatabase.Type)

	// Conecta ao banco de dados de origem
	sourceDB, err := sql.Open(project.SourceDatabase.Type, buildDSN(project.SourceDatabase))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao conectar no banco de origem"})
		log.Println("Erro ao conectar no banco de origem:", err)
		return
	}
	log.Printf("Conexão com o banco de origem %s estabelecida", project.SourceDatabase.Database)

	// Conecta ao banco de dados de destino
	destDB, err := sql.Open(project.DestinationDatabase.Type, buildDSN(project.DestinationDatabase))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao conectar no banco de destino"})
		log.Println("Erro ao conectar no banco de destino:", err)
		return
	}
	log.Printf("Conexão com o banco de destino %s estabelecida", project.DestinationDatabase.Database)

	// cria o JobRunner
	runner := jobrunner.NewJobRunner(sourceDB, destDB, buildDSN(project.SourceDatabase), buildDSN(project.DestinationDatabase), dialect, project.Concurrency, project.ProjectName, projectID)
	jobrunner.SetActiveRunner(runner)

	// Carregar os jobs
	jobCount := 0
	for _, jobPath := range project.Jobs {

		// Lê o JSON do job
		fullPath := filepath.Join("data", "projects", projectID, jobPath)
		jobBytes, err := os.ReadFile(fullPath)
		if err != nil {
			log.Printf("Erro ao ler job %s: %v", jobPath, err)
			continue
		}

		// Deserializa o JSON do job
		var job models.Job
		if err := json.Unmarshal(jobBytes, &job); err == nil {
			runner.JobMap[job.ID] = job
			log.Printf("Job %s carregado do caminho %s", job.ID, fullPath)

			// Atualizar status de todos os jobs para pendente
			status.UpdateJobStatus(job.ID, func(js *status.JobStatus) {
				js.Status = "pending"
				js.StartedAt = nil
				js.EndedAt = nil
				js.Processed = 0
				js.Total = 0
				js.Progress = 0
				js.Error = ""
				status.NotifySubscribers()
			})

			jobCount++
		} else {
			log.Printf("Erro ao interpretar job %s: %v", jobPath, err)
		}
	}
	if jobCount == 0 {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Nenhum job foi carregado"})
		log.Println("Nenhum job foi carregado do projeto")
		return
	}

	// Carregar conexões
	for _, conn := range project.Connections {
		runner.ConnMap[conn.Source] = append(runner.ConnMap[conn.Source], conn.Target)
		log.Printf("Conexão adicionada: %s -> %s", conn.Source, conn.Target)
	}

	// Detectar jobs raiz
	usedTargets := make(map[string]bool)
	for _, conn := range project.Connections {
		usedTargets[conn.Target] = true
	}
	log.Printf("Jobs usados como destino: %v", usedTargets)

	var startJobs []string
	for id := range runner.JobMap {
		if !usedTargets[id] {
			startJobs = append(startJobs, id)
		}
	}

	// Fallback: se nenhuma conexão definida, executa todos
	if len(startJobs) == 0 {
		log.Println("Nenhum job raiz detectado — executando todos os jobs.")
		for id := range runner.JobMap {
			startJobs = append(startJobs, id)
		}
	}

	if len(startJobs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Nenhum job para executar"})
		log.Println("Nenhum job para executar")
		return
	}

	go runner.Run(startJobs)
	c.JSON(http.StatusAccepted, gin.H{"message": "Execução iniciada", "startJobs": startJobs})
	log.Printf("Execução do projeto %s iniciada com %d jobs", project.ProjectName, len(startJobs))
}

func StopProject(c *gin.Context) {
	projectID := c.Param("id")
	stopped := jobrunner.StopActiveRunner(projectID, "interrompido via endpoint")
	if !stopped {
		c.JSON(http.StatusNotFound, gin.H{"error": "Nenhuma pipeline ativa para este projeto"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Pipeline interrompida"})
}

func buildDSN(cfg models.DatabaseConfig) string {
	switch cfg.Type {
	case "postgres":
		return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable", cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.Database)
	case "mysql":
		return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s", cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Database)
	default:
		return ""
	}
}
