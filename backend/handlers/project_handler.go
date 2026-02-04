package handlers

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"etl/models"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

var chave = []byte("a1b2c3d4e5f6g7h8a1b2c3d4e5f6g7h8") // 32 bytes para AES-256
var projectFileMu sync.Mutex

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
			project, err := loadProjectFile(projectPath)
			if err != nil {
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
	if err := writeProjectFile(projectPath, &project); err != nil {
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
	project, err := loadProjectFile(projectPath)
	if err != nil {
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

	if err := writeProjectFile(projectPath, &updatedProject); err != nil {
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

// -------------------- Duplicate Project --------------------
func DuplicateProject(c *gin.Context) {
	projectID := c.Param("id")
	sourceDir := filepath.Join("data", "projects", projectID)
	projectPath := filepath.Join(sourceDir, "project.json")

	project, err := loadProjectFile(projectPath)
	if err != nil {
		c.JSON(500, gin.H{"error": "Erro ao interpretar project.json"})
		return
	}

	// prepara novo projeto
	decryptProjectFields(&project)

	var req struct {
		ProjectName string `json:"projectName"`
	}
	_ = c.ShouldBindJSON(&req)

	newProject := project
	newProject.ID = uuid.New().String()
	if strings.TrimSpace(req.ProjectName) != "" {
		newProject.ProjectName = req.ProjectName
	} else {
		newProject.ProjectName = project.ProjectName + " (Cópia)"
	}

	// cria diretório do novo projeto
	destDir := filepath.Join("data", "projects", newProject.ID)
	if err := os.MkdirAll(destDir, os.ModePerm); err != nil {
		c.JSON(500, gin.H{"error": "Erro ao criar diretório do projeto duplicado"})
		return
	}

	// copia todos os arquivos e pastas (exceto project.json)
	if err := copyProjectDir(sourceDir, destDir); err != nil {
		c.JSON(500, gin.H{"error": "Erro ao copiar arquivos do projeto"})
		return
	}

	// salva project.json novo
	encryptProjectFields(&newProject)
	if err := writeProjectFile(filepath.Join(destDir, "project.json"), &newProject); err != nil {
		c.JSON(500, gin.H{"error": "Erro ao salvar project.json do projeto duplicado"})
		return
	}

	decryptProjectFields(&newProject)
	c.JSON(201, newProject)
}

func copyProjectDir(srcDir, destDir string) error {
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		name := entry.Name()
		if name == "project.json" {
			continue
		}
		srcPath := filepath.Join(srcDir, name)
		destPath := filepath.Join(destDir, name)

		if entry.IsDir() {
			if err := os.MkdirAll(destPath, os.ModePerm); err != nil {
				return err
			}
			if err := copyDirRecursive(srcPath, destPath); err != nil {
				return err
			}
			continue
		}

		if err := copyFile(srcPath, destPath); err != nil {
			return err
		}
	}

	return nil
}

func copyDirRecursive(srcDir, destDir string) error {
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(srcDir, entry.Name())
		destPath := filepath.Join(destDir, entry.Name())

		if entry.IsDir() {
			if err := os.MkdirAll(destPath, os.ModePerm); err != nil {
				return err
			}
			if err := copyDirRecursive(srcPath, destPath); err != nil {
				return err
			}
			continue
		}

		if err := copyFile(srcPath, destPath); err != nil {
			return err
		}
	}

	return nil
}

func copyFile(srcPath, destPath string) error {
	src, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer src.Close()

	dest, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer dest.Close()

	if _, err := io.Copy(dest, src); err != nil {
		return err
	}

	if info, err := os.Stat(srcPath); err == nil {
		_ = os.Chmod(destPath, info.Mode())
	}

	return nil
}

// -------------------- Safe Load Helper --------------------
func loadProjectFile(path string) (models.Project, error) {
	var project models.Project
	projectBytes, err := os.ReadFile(path)
	if err != nil {
		return project, err
	}

	dec := json.NewDecoder(bytes.NewReader(projectBytes))
	if err := dec.Decode(&project); err != nil {
		return project, err
	}

	// Detect trailing data; if present, rewrite a clean JSON file.
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		if clean, err2 := json.MarshalIndent(project, "", "  "); err2 == nil {
			_ = os.WriteFile(path, clean, 0644)
		}
	}

	return project, nil
}

func writeProjectFile(path string, project *models.Project) error {
	projectFileMu.Lock()
	defer projectFileMu.Unlock()

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	projectBytes, err := json.MarshalIndent(project, "", "  ")
	if err != nil {
		return err
	}

	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, projectBytes, 0644); err != nil {
		return err
	}

	_ = os.Remove(path)
	return os.Rename(tmpPath, path)
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
