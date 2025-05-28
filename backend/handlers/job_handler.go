package handlers

import (
	"encoding/json"
	"etl/models"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func ListJobs(c *gin.Context) {
	projectName := c.Param("name")
	projectPath := filepath.Join("data", "projects", projectName, "project.json")
	projectBytes, err := ioutil.ReadFile(projectPath)
	if err != nil {
		c.JSON(404, gin.H{"error": "Projeto não encontrado"})
		return
	}

	var project models.Project
	if err := json.Unmarshal(projectBytes, &project); err != nil {
		c.JSON(500, gin.H{"error": "Erro ao ler project.json"})
		return
	}

	var jobs []models.Job
	for _, jobPath := range project.Jobs {
		fullJobPath := filepath.Join("data", "projects", projectName, jobPath)
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
	projectName := c.Param("name")
	var job models.Job
	if err := c.BindJSON(&job); err != nil {
		c.JSON(400, gin.H{"error": "JSON inválido"})
		return
	}

	job.ID = uuid.New().String()
	jobFileName := "job-" + strings.ToLower(strings.ReplaceAll(job.JobName, " ", "-")) + ".json"
	projectDir := filepath.Join("data", "projects", projectName)
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

	c.JSON(201, gin.H{"message": "Job adicionado com sucesso!"})
}

func UpdateJob(c *gin.Context) {
	projectName := c.Param("name")
	jobName := c.Param("jobName")
	var job models.Job
	if err := c.BindJSON(&job); err != nil {
		c.JSON(400, gin.H{"error": "JSON inválido"})
		return
	}

	jobFileName := "job-" + strings.ToLower(strings.ReplaceAll(jobName, " ", "-")) + ".json"
	jobPath := filepath.Join("data", "projects", projectName, "jobs", jobFileName)
	jobBytes, _ := json.MarshalIndent(job, "", "  ")
	if err := ioutil.WriteFile(jobPath, jobBytes, 0644); err != nil {
		c.JSON(500, gin.H{"error": "Erro ao atualizar o job"})
		return
	}
	c.JSON(200, gin.H{"message": "Job atualizado com sucesso!"})
}

func DeleteJob(c *gin.Context) {
	projectName := c.Param("name")
	jobName := c.Param("jobName")
	jobFileName := "job-" + strings.ToLower(strings.ReplaceAll(jobName, " ", "-")) + ".json"
	projectDir := filepath.Join("data", "projects", projectName)
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
