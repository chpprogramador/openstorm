package handlers

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/json"
	"etl/models"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

var chave []byte = []byte("a1b2c3d4e5f6g7h8")

func ListProjects(c *gin.Context) {
	baseDir := "data/projects"
	entries, err := os.ReadDir(baseDir)
	if err != nil {
		c.JSON(500, gin.H{"error": "Erro ao ler o diretório de projetos"})
		return
	}

	var projects []models.Project

	for _, entry := range entries {
		if entry.IsDir() {
			projectPath := filepath.Join(baseDir, entry.Name(), "project.json")
			projectBytes, err := os.ReadFile(projectPath)
			if err != nil {
				continue // ignora se não conseguir ler
			}

			var project models.Project
			if err := json.Unmarshal(projectBytes, &project); err != nil {
				continue
			}

			// Descriptografa usuário e senha dos bancos de dados
			decryptedUserBytes, _ := Descriptografar([]byte(project.SourceDatabase.User))
			project.SourceDatabase.User = string(decryptedUserBytes)

			decryptedPassBytes, _ := Descriptografar([]byte(project.SourceDatabase.Password))
			project.SourceDatabase.Password = string(decryptedPassBytes)

			decryptedDestUserBytes, _ := Descriptografar([]byte(project.DestinationDatabase.User))
			project.DestinationDatabase.User = string(decryptedDestUserBytes)

			decryptedDestPassBytes, _ := Descriptografar([]byte(project.DestinationDatabase.Password))
			project.DestinationDatabase.Password = string(decryptedDestPassBytes)

			projects = append(projects, project)
		}
	}

	c.JSON(200, projects)
}

func CreateProject(c *gin.Context) {
	var project models.Project
	if err := c.BindJSON(&project); err != nil {
		c.JSON(400, gin.H{"error": "JSON inválido"})
		return
	}

	// Criptografa usuário e senha dos bancos de dados
	decryptedUserBytes, _ := Descriptografar([]byte(project.SourceDatabase.User))
	project.SourceDatabase.User = string(decryptedUserBytes)

	decryptedPassBytes, _ := Descriptografar([]byte(project.SourceDatabase.Password))
	project.SourceDatabase.Password = string(decryptedPassBytes)

	decryptedDestUserBytes, _ := Descriptografar([]byte(project.DestinationDatabase.User))
	project.DestinationDatabase.User = string(decryptedDestUserBytes)

	decryptedDestPassBytes, _ := Descriptografar([]byte(project.DestinationDatabase.Password))
	project.DestinationDatabase.Password = string(decryptedDestPassBytes)

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

	// Descriptografa usuário e senha dos bancos de dados
	decryptedUserBytes, _ := Descriptografar([]byte(project.SourceDatabase.User))
	project.SourceDatabase.User = string(decryptedUserBytes)

	decryptedPassBytes, _ := Descriptografar([]byte(project.SourceDatabase.Password))
	project.SourceDatabase.Password = string(decryptedPassBytes)

	decryptedDestUserBytes, _ := Descriptografar([]byte(project.DestinationDatabase.User))
	project.DestinationDatabase.User = string(decryptedDestUserBytes)

	decryptedDestPassBytes, _ := Descriptografar([]byte(project.DestinationDatabase.Password))
	project.DestinationDatabase.Password = string(decryptedDestPassBytes)

	c.JSON(200, project)
}

func UpdateProject(c *gin.Context) {
	projectID := c.Param("id")
	projectPath := filepath.Join("data", "projects", projectID, "project.json")

	// Verifica se o arquivo existe
	if _, err := os.Stat(projectPath); os.IsNotExist(err) {
		c.JSON(404, gin.H{"error": "Projeto não encontrado"})
		return
	}

	// Lê o corpo da requisição com os novos dados
	var updatedProject models.Project
	if err := c.BindJSON(&updatedProject); err != nil {
		c.JSON(400, gin.H{"error": "JSON inválido"})
		return
	}

	// Garante que o ID continua o mesmo
	updatedProject.ID = projectID

	decryptedUserBytes, _ := Descriptografar([]byte(updatedProject.SourceDatabase.User))
	updatedProject.SourceDatabase.User = string(decryptedUserBytes)

	decryptedPassBytes, _ := Descriptografar([]byte(updatedProject.SourceDatabase.Password))
	updatedProject.SourceDatabase.Password = string(decryptedPassBytes)

	decryptedDestUserBytes, _ := Descriptografar([]byte(updatedProject.DestinationDatabase.User))
	updatedProject.DestinationDatabase.User = string(decryptedDestUserBytes)

	decryptedDestPassBytes, _ := Descriptografar([]byte(updatedProject.DestinationDatabase.Password))
	updatedProject.DestinationDatabase.Password = string(decryptedDestPassBytes)

	// Atualiza o arquivo project.json
	projectBytes, err := json.MarshalIndent(updatedProject, "", "  ")
	if err != nil {
		c.JSON(500, gin.H{"error": "Erro ao serializar o projeto"})
		return
	}

	if err := os.WriteFile(projectPath, projectBytes, 0644); err != nil {
		c.JSON(500, gin.H{"error": "Erro ao salvar o projeto"})
		return
	}

	c.JSON(200, updatedProject)
}

func CloseProject(c *gin.Context) {
	projectID := c.Param("id")
	c.JSON(200, gin.H{"message": "Projeto '" + projectID + "' fechado com sucesso."})
}

func Criptografar(texto []byte) ([]byte, error) {
	block, err := aes.NewCipher(chave)
	if err != nil {
		return nil, err
	}
	ciphertext := make([]byte, aes.BlockSize+len(texto))
	iv := ciphertext[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, err
	}
	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(ciphertext[aes.BlockSize:], texto)
	return ciphertext, nil
}

func Descriptografar(ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(chave)
	if err != nil {
		return nil, err
	}
	iv := ciphertext[:aes.BlockSize]
	texto := ciphertext[aes.BlockSize:]
	stream := cipher.NewCFBDecrypter(block, iv)
	stream.XORKeyStream(texto, texto)
	return texto, nil
}
