package handlers

import (
	"net/http"

	"etl/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ListVisualElements lista todos os elementos visuais de um projeto
func ListVisualElements(c *gin.Context) {
	projectID := c.Param("id")
	project, err := loadProject(projectID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Projeto não encontrado"})
		return
	}

	c.JSON(http.StatusOK, project.VisualElements)
}

// CreateVisualElement cria um novo elemento visual no projeto
func CreateVisualElement(c *gin.Context) {
	projectID := c.Param("id")

	var element models.VisualElement
	if err := c.ShouldBindJSON(&element); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dados inválidos"})
		return
	}

	project, err := loadProject(projectID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Projeto não encontrado"})
		return
	}

	if element.ID == "" {
		element.ID = uuid.New().String()
	}

	project.VisualElements = append(project.VisualElements, element)

	if err := saveProject(project); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao salvar projeto"})
		return
	}

	c.JSON(http.StatusCreated, element)
}

// UpdateVisualElement atualiza um elemento visual existente
func UpdateVisualElement(c *gin.Context) {
	projectID := c.Param("id")
	elementID := c.Param("elementId")

	var updated models.VisualElement
	if err := c.ShouldBindJSON(&updated); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dados inválidos"})
		return
	}

	project, err := loadProject(projectID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Projeto não encontrado"})
		return
	}

	found := false
	for i, element := range project.VisualElements {
		if element.ID == elementID {
			updated.ID = elementID
			project.VisualElements[i] = updated
			found = true
			break
		}
	}

	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "Elemento não encontrado"})
		return
	}

	if err := saveProject(project); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao salvar projeto"})
		return
	}

	c.JSON(http.StatusOK, updated)
}

// DeleteVisualElement remove um elemento visual do projeto
func DeleteVisualElement(c *gin.Context) {
	projectID := c.Param("id")
	elementID := c.Param("elementId")

	project, err := loadProject(projectID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Projeto não encontrado"})
		return
	}

	found := false
	for i, element := range project.VisualElements {
		if element.ID == elementID {
			project.VisualElements = append(project.VisualElements[:i], project.VisualElements[i+1:]...)
			found = true
			break
		}
	}

	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "Elemento não encontrado"})
		return
	}

	if err := saveProject(project); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao salvar projeto"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Elemento removido com sucesso"})
}

// GetVisualElement obtém um elemento visual específico
func GetVisualElement(c *gin.Context) {
	projectID := c.Param("id")
	elementID := c.Param("elementId")

	project, err := loadProject(projectID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Projeto não encontrado"})
		return
	}

	for _, element := range project.VisualElements {
		if element.ID == elementID {
			c.JSON(http.StatusOK, element)
			return
		}
	}

	c.JSON(http.StatusNotFound, gin.H{"error": "Elemento não encontrado"})
}
