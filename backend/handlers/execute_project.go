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
	"sort"
	"strings"

	"etl/jobrunner"

	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql" // MySQL
	_ "github.com/lib/pq"              // Postgres
)

func RunProject(c *gin.Context) {

	status.ClearJobLogs()
	status.ResetCountStatus()
	status.ResetWorkerStatus()

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
			runner.JobOrder = append(runner.JobOrder, job.ID)

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
	for _, id := range runner.JobOrder {
		if !usedTargets[id] {
			startJobs = append(startJobs, id)
		}
	}

	// Fallback: se nenhuma conexão definida, executa todos
	if len(startJobs) == 0 {
		log.Println("Nenhum job raiz detectado — executando todos os jobs.")
		startJobs = append(startJobs, runner.JobOrder...)
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

func ResumeJob(c *gin.Context) {
	status.ClearJobLogs()
	status.ResetCountStatus()
	status.ResetWorkerStatus()

	projectID := c.Param("id")
	jobID := c.Param("jobId")
	projectPath := filepath.Join("data", "projects", projectID, "project.json")
	project, err := loadProjectFile(projectPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao ler project.json"})
		log.Println("Erro ao ler project.json:", err)
		return
	}
	log.Printf("Projeto %s carregado com sucesso", project.ProjectName)

	decryptProjectFields(&project)

	dialect, err := dialects.NewDialect(project.SourceDatabase.Type)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		log.Println("Erro ao criar dialeto:", err)
		return
	}
	log.Printf("Dialeto %s criado com sucesso", project.SourceDatabase.Type)

	sourceDB, err := sql.Open(project.SourceDatabase.Type, buildDSN(project.SourceDatabase))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao conectar no banco de origem"})
		log.Println("Erro ao conectar no banco de origem:", err)
		return
	}
	log.Printf("Conex??o com o banco de origem %s estabelecida", project.SourceDatabase.Database)

	destDB, err := sql.Open(project.DestinationDatabase.Type, buildDSN(project.DestinationDatabase))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao conectar no banco de destino"})
		log.Println("Erro ao conectar no banco de destino:", err)
		return
	}
	log.Printf("Conex??o com o banco de destino %s estabelecida", project.DestinationDatabase.Database)

	runner := jobrunner.NewJobRunner(sourceDB, destDB, buildDSN(project.SourceDatabase), buildDSN(project.DestinationDatabase), dialect, project.Concurrency, project.ProjectName, projectID)
	jobrunner.SetActiveRunner(runner)

	// Carregar os jobs
	jobCount := 0
	for _, jobPath := range project.Jobs {
		fullPath := filepath.Join("data", "projects", projectID, jobPath)
		jobBytes, err := os.ReadFile(fullPath)
		if err != nil {
			log.Printf("Erro ao ler job %s: %v", jobPath, err)
			continue
		}

		var job models.Job
		if err := json.Unmarshal(jobBytes, &job); err == nil {
			runner.JobMap[job.ID] = job
			log.Printf("Job %s carregado do caminho %s", job.ID, fullPath)
			runner.JobOrder = append(runner.JobOrder, job.ID)
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

	if _, ok := runner.JobMap[jobID]; !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Job n??o encontrado no projeto"})
		return
	}

	// Carregar conex??es
	for _, conn := range project.Connections {
		runner.ConnMap[conn.Source] = append(runner.ConnMap[conn.Source], conn.Target)
		log.Printf("Conex??o adicionada: %s -> %s", conn.Source, conn.Target)
	}

	// Marca apenas o job inicial e seus dependentes como pendentes
	reachable := collectDownstreamJobs(jobID, runner.ConnMap)
	requiredMapKeys, err := collectMapKeysForJobs(reachable, runner.JobMap)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	requiredMemoryJobs, err := collectRequiredMemorySelectJobs(requiredMapKeys, runner.JobOrder, runner.JobMap)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	preloadMemoryJobs := make([]string, 0, len(requiredMemoryJobs))
	for _, id := range requiredMemoryJobs {
		if id == jobID {
			continue
		}
		if _, isDownstream := reachable[id]; isDownstream {
			continue
		}
		preloadMemoryJobs = append(preloadMemoryJobs, id)
	}

	pendingSet := make(map[string]struct{}, len(reachable)+len(preloadMemoryJobs))
	for id := range reachable {
		pendingSet[id] = struct{}{}
	}
	for _, id := range preloadMemoryJobs {
		pendingSet[id] = struct{}{}
	}

	for _, id := range orderedJobIDs(pendingSet, runner.JobOrder) {
		status.UpdateJobStatus(id, func(js *status.JobStatus) {
			js.Status = "pending"
			js.StartedAt = nil
			js.EndedAt = nil
			js.Processed = 0
			js.Total = 0
			js.Progress = 0
			js.Error = ""
			status.NotifySubscribers()
		})
	}

	for _, id := range preloadMemoryJobs {
		runner.ConnMap[id] = nil
	}

	startJobs := make([]string, 0, len(preloadMemoryJobs)+1)
	startJobs = append(startJobs, preloadMemoryJobs...)
	startJobs = append(startJobs, jobID)
	go runner.Run(startJobs)
	c.JSON(http.StatusAccepted, gin.H{"message": "Retomada iniciada", "startJobs": startJobs})
	log.Printf("Retomada do job %s iniciada para o projeto %s (preload memory-select: %v)", jobID, project.ProjectName, preloadMemoryJobs)
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

func collectDownstreamJobs(startID string, connMap map[string][]string) map[string]struct{} {
	visited := make(map[string]struct{})
	stack := []string{startID}
	for len(stack) > 0 {
		id := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if _, ok := visited[id]; ok {
			continue
		}
		visited[id] = struct{}{}
		for _, next := range connMap[id] {
			if _, ok := visited[next]; !ok {
				stack = append(stack, next)
			}
		}
	}
	return visited
}

func collectMapKeysForJobs(jobIDs map[string]struct{}, jobMap map[string]models.Job) (map[string]struct{}, error) {
	keys := make(map[string]struct{})
	for id := range jobIDs {
		job, ok := jobMap[id]
		if !ok {
			continue
		}

		mapKeys, err := jobrunner.ExtractMapDirectiveKeys(job.SelectSQL)
		if err != nil {
			return nil, fmt.Errorf("job %s (%s): %w", job.ID, job.JobName, err)
		}
		for _, key := range mapKeys {
			keys[key] = struct{}{}
		}
	}
	return keys, nil
}

func collectRequiredMemorySelectJobs(requiredKeys map[string]struct{}, jobOrder []string, jobMap map[string]models.Job) ([]string, error) {
	if len(requiredKeys) == 0 {
		return nil, nil
	}

	missingKeys := make(map[string]struct{}, len(requiredKeys))
	for key := range requiredKeys {
		missingKeys[key] = struct{}{}
	}

	requiredJobs := make([]string, 0, len(requiredKeys))
	for _, id := range jobOrder {
		job, ok := jobMap[id]
		if !ok || strings.ToLower(job.Type) != "memory-select" {
			continue
		}

		mapKey, err := jobrunner.NormalizeMemoryMapKey(job.JobName)
		if err != nil {
			return nil, fmt.Errorf("job %s (%s): %w", job.ID, job.JobName, err)
		}

		if _, needed := missingKeys[mapKey]; !needed {
			continue
		}

		requiredJobs = append(requiredJobs, id)
		delete(missingKeys, mapKey)
	}

	if len(missingKeys) > 0 {
		keys := make([]string, 0, len(missingKeys))
		for key := range missingKeys {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		return nil, fmt.Errorf("nenhum job memory-select encontrado para map(s): %s", strings.Join(keys, ", "))
	}

	return requiredJobs, nil
}

func orderedJobIDs(ids map[string]struct{}, jobOrder []string) []string {
	ordered := make([]string, 0, len(ids))
	for _, id := range jobOrder {
		if _, ok := ids[id]; ok {
			ordered = append(ordered, id)
		}
	}
	return ordered
}
