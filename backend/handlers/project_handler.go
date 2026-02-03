package handlers

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"etl/models"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

var chave = []byte("a1b2c3d4e5f6g7h8a1b2c3d4e5f6g7h8") // 32 bytes para AES-256

// -------------------- List Projects --------------------
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
				continue
			}

			var project models.Project
			if err := json.Unmarshal(projectBytes, &project); err != nil {
				continue
			}

			// Descriptografa os campos
			decryptProjectFields(&project)

			projects = append(projects, project)
		}
	}

	c.JSON(200, projects)
}

// -------------------- Create Project --------------------
func CreateProject(c *gin.Context) {
	var project models.Project
	if err := c.BindJSON(&project); err != nil {
		c.JSON(400, gin.H{"error": "JSON inválido"})
		return
	}

	// Criptografa os campos
	encryptProjectFields(&project)

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

	// Retorna descriptografado para o cliente
	decryptProjectFields(&project)

	c.JSON(201, project)
}

// -------------------- Get Project by ID --------------------
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

	decryptProjectFields(&project)

	c.JSON(200, project)
}

// -------------------- Update Project --------------------
func UpdateProject(c *gin.Context) {
	projectID := c.Param("id")
	projectPath := filepath.Join("data", "projects", projectID, "project.json")

	if _, err := os.Stat(projectPath); os.IsNotExist(err) {
		c.JSON(404, gin.H{"error": "Projeto não encontrado"})
		return
	}

	var updatedProject models.Project
	if err := c.BindJSON(&updatedProject); err != nil {
		c.JSON(400, gin.H{"error": "JSON inválido"})
		return
	}

	updatedProject.ID = projectID
	encryptProjectFields(&updatedProject)

	projectBytes, err := json.MarshalIndent(updatedProject, "", "  ")
	if err != nil {
		c.JSON(500, gin.H{"error": "Erro ao serializar o projeto"})
		return
	}

	if err := os.WriteFile(projectPath, projectBytes, 0644); err != nil {
		c.JSON(500, gin.H{"error": "Erro ao salvar o projeto"})
		return
	}

	decryptProjectFields(&updatedProject)
	c.JSON(200, updatedProject)
}

// -------------------- Close Project --------------------
func CloseProject(c *gin.Context) {
	projectID := c.Param("id")
	c.JSON(200, gin.H{"message": "Projeto '" + projectID + "' fechado com sucesso."})
}

// -------------------- Encrypt / Decrypt Helpers --------------------
func encryptProjectFields(p *models.Project) {
	p.SourceDatabase.User, _ = EncryptToBase64([]byte(p.SourceDatabase.User))
	p.SourceDatabase.Password, _ = EncryptToBase64([]byte(p.SourceDatabase.Password))
	p.DestinationDatabase.User, _ = EncryptToBase64([]byte(p.DestinationDatabase.User))
	p.DestinationDatabase.Password, _ = EncryptToBase64([]byte(p.DestinationDatabase.Password))
}

func decryptProjectFields(p *models.Project) {
	p.SourceDatabase.User = string(DecryptFromBase64(p.SourceDatabase.User))
	p.SourceDatabase.Password = string(DecryptFromBase64(p.SourceDatabase.Password))
	p.DestinationDatabase.User = string(DecryptFromBase64(p.DestinationDatabase.User))
	p.DestinationDatabase.Password = string(DecryptFromBase64(p.DestinationDatabase.Password))
}

// -------------------- Criptografia AES-CFB + Base64 --------------------
func EncryptToBase64(plain []byte) (string, error) {
	if len(plain) == 0 {
		return "", nil
	}

	block, err := aes.NewCipher(chave)
	if err != nil {
		return "", err
	}

	ciphertext := make([]byte, aes.BlockSize+len(plain))
	iv := ciphertext[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return "", err
	}

	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(ciphertext[aes.BlockSize:], plain)

	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func DecryptFromBase64(b64 string) []byte {
	if b64 == "" {
		return []byte{}
	}

	if !looksLikeBase64(b64) {
		return []byte(b64)
	}

	ciphertext, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return []byte(b64)
	}

	if len(ciphertext) < aes.BlockSize {
		return []byte(b64)
	}

	block, err := aes.NewCipher(chave)
	if err != nil {
		return []byte(b64)
	}

	iv := ciphertext[:aes.BlockSize]
	texto := make([]byte, len(ciphertext[aes.BlockSize:]))
	copy(texto, ciphertext[aes.BlockSize:])

	stream := cipher.NewCFBDecrypter(block, iv)
	stream.XORKeyStream(texto, texto)

	if !isProbablyText(texto) {
		return []byte(b64)
	}

	return texto
}

func looksLikeBase64(value string) bool {
	if len(value)%4 != 0 {
		return false
	}
	for i := 0; i < len(value); i++ {
		c := value[i]
		if (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '+' || c == '/' || c == '=' {
			continue
		}
		return false
	}
	return true
}

func isProbablyText(value []byte) bool {
	if len(value) == 0 {
		return true
	}
	if !utf8.Valid(value) {
		return false
	}
	printable := 0
	for _, r := range string(value) {
		if r == '\n' || r == '\r' || r == '\t' {
			printable++
			continue
		}
		if r >= 32 && r != 127 {
			printable++
		}
	}
	return printable > 0
}
