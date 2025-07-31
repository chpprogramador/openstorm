package handlers

import (
	"encoding/json"
	"etl/models"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

// ListVariables lista todas as variáveis de um projeto
func ListVariables(c *gin.Context) {
	projectID := c.Param("id")
	project, err := loadProject(projectID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Projeto não encontrado"})
		return
	}

	c.JSON(http.StatusOK, project.Variables)
}

// CreateVariable cria uma nova variável no projeto
func CreateVariable(c *gin.Context) {
	projectID := c.Param("id")
	
	var variable models.Variable
	if err := c.ShouldBindJSON(&variable); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dados inválidos"})
		return
	}

	project, err := loadProject(projectID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Projeto não encontrado"})
		return
	}

	// Verifica se já existe uma variável com o mesmo nome
	for _, existingVar := range project.Variables {
		if existingVar.Name == variable.Name {
			c.JSON(http.StatusConflict, gin.H{"error": "Variável com este nome já existe"})
			return
		}
	}

	// Adiciona a nova variável
	project.Variables = append(project.Variables, variable)

	// Salva o projeto
	if err := saveProject(project); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao salvar projeto"})
		return
	}

	c.JSON(http.StatusCreated, variable)
}

// UpdateVariable atualiza uma variável existente
func UpdateVariable(c *gin.Context) {
	projectID := c.Param("id")
	variableName := c.Param("variableName")
	
	var updatedVariable models.Variable
	if err := c.ShouldBindJSON(&updatedVariable); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dados inválidos"})
		return
	}

	project, err := loadProject(projectID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Projeto não encontrado"})
		return
	}

	// Encontra e atualiza a variável
	found := false
	for i, variable := range project.Variables {
		if variable.Name == variableName {
			project.Variables[i] = updatedVariable
			found = true
			break
		}
	}

	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "Variável não encontrada"})
		return
	}

	// Salva o projeto
	if err := saveProject(project); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao salvar projeto"})
		return
	}

	c.JSON(http.StatusOK, updatedVariable)
}

// DeleteVariable remove uma variável do projeto
func DeleteVariable(c *gin.Context) {
	projectID := c.Param("id")
	variableName := c.Param("variableName")

	project, err := loadProject(projectID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Projeto não encontrado"})
		return
	}

	// Encontra e remove a variável
	found := false
	for i, variable := range project.Variables {
		if variable.Name == variableName {
			project.Variables = append(project.Variables[:i], project.Variables[i+1:]...)
			found = true
			break
		}
	}

	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "Variável não encontrada"})
		return
	}

	// Salva o projeto
	if err := saveProject(project); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao salvar projeto"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Variável removida com sucesso"})
}

// GetVariable obtém uma variável específica
func GetVariable(c *gin.Context) {
	projectID := c.Param("id")
	variableName := c.Param("variableName")

	project, err := loadProject(projectID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Projeto não encontrado"})
		return
	}

	// Encontra a variável
	for _, variable := range project.Variables {
		if variable.Name == variableName {
			c.JSON(http.StatusOK, variable)
			return
		}
	}

	c.JSON(http.StatusNotFound, gin.H{"error": "Variável não encontrada"})
}

// Funções auxiliares para carregar e salvar projeto
func loadProject(projectID string) (*models.Project, error) {
	projectPath := filepath.Join("data", "projects", projectID, "project.json")
	projectBytes, err := os.ReadFile(projectPath)
	if err != nil {
		return nil, err
	}

	var project models.Project
	if err := json.Unmarshal(projectBytes, &project); err != nil {
		return nil, err
	}

	return &project, nil
}

func saveProject(project *models.Project) error {
	projectPath := filepath.Join("data", "projects", project.ID, "project.json")
	
	// Garante que o diretório existe
	dir := filepath.Dir(projectPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	projectBytes, err := json.MarshalIndent(project, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(projectPath, projectBytes, 0644)
}
