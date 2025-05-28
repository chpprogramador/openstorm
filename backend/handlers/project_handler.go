package handlers

import (
	"encoding/json"
	"etl/models"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func CreateProject(c *gin.Context) {
	var project models.Project
	if err := c.BindJSON(&project); err != nil {
		c.JSON(400, gin.H{"error": "JSON inválido"})
		return
	}

	project.ID = uuid.New().String()
	projectDir := filepath.Join("data", "projects", project.ID)
	if err := os.MkdirAll(filepath.Join(projectDir, "jobs"), os.ModePerm); err != nil {
		c.JSON(500, gin.H{"error": "Erro ao criar diretório do projeto"})
		return
	}

	projectPath := filepath.Join(projectDir, "project.json")
	projectBytes, _ := json.MarshalIndent(project, "", "  ")
	if err := ioutil.WriteFile(projectPath, projectBytes, 0644); err != nil {
		c.JSON(500, gin.H{"error": "Erro ao salvar arquivo project.json"})
		return
	}

	c.JSON(201, project)
}

func GetProjectByID(c *gin.Context) {
	projectID := c.Param("id")
	projectPath := filepath.Join("data", "projects", projectID, "project.json")
	projectBytes, err := ioutil.ReadFile(projectPath)
	if err != nil {
		c.JSON(404, gin.H{"error": "Projeto não encontrado"})
		return
	}

	var project models.Project
	if err := json.Unmarshal(projectBytes, &project); err != nil {
		c.JSON(500, gin.H{"error": "Erro ao ler o JSON do projeto"})
		return
	}

	c.JSON(200, project)
}

func CloseProject(c *gin.Context) {
	projectID := c.Param("id")
	c.JSON(200, gin.H{"message": "Projeto '" + projectID + "' fechado com sucesso."})
}
