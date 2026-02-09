package handlers

import (
	"archive/zip"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func ExportProject(c *gin.Context) {
	projectID := c.Param("id")
	projectDir := filepath.Join("data", "projects", projectID)

	if _, err := os.Stat(projectDir); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Projeto nao encontrado"})
		return
	}

	filename := "project_" + projectID + ".zip"
	c.Header("Content-Type", "application/zip")
	c.Header("Content-Disposition", "attachment; filename=\""+filename+"\"")

	zipWriter := zip.NewWriter(c.Writer)
	defer zipWriter.Close()

	_ = filepath.WalkDir(projectDir, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(projectDir, p)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}

		if shouldSkipLogs(rel) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		relSlash := filepath.ToSlash(rel)

		if d.IsDir() {
			hdr := &zip.FileHeader{
				Name:     relSlash + "/",
				Method:   zip.Deflate,
				Modified: time.Now(),
			}
			_, err := zipWriter.CreateHeader(hdr)
			return err
		}

		info, err := d.Info()
		if err != nil {
			return err
		}

		hdr, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		hdr.Name = relSlash
		hdr.Method = zip.Deflate

		w, err := zipWriter.CreateHeader(hdr)
		if err != nil {
			return err
		}

		f, err := os.Open(p)
		if err != nil {
			return err
		}
		defer f.Close()

		_, err = io.Copy(w, f)
		return err
	})
}

func ImportProject(c *gin.Context) {
	formFile, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Arquivo zip nao enviado"})
		return
	}

	projectName := strings.TrimSpace(c.PostForm("projectName"))

	src, err := formFile.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Erro ao abrir o arquivo zip"})
		return
	}
	defer src.Close()

	tmp, err := os.CreateTemp("", "project-import-*.zip")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao preparar arquivo temporario"})
		return
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	if _, err := io.Copy(tmp, src); err != nil {
		_ = tmp.Close()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao salvar arquivo temporario"})
		return
	}
	if err := tmp.Close(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao finalizar arquivo temporario"})
		return
	}

	zipFile, err := os.Open(tmpPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao reabrir arquivo temporario"})
		return
	}
	defer zipFile.Close()

	stat, err := zipFile.Stat()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao ler tamanho do zip"})
		return
	}

	zr, err := zip.NewReader(zipFile, stat.Size())
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Arquivo zip invalido"})
		return
	}

	newID := uuid.New().String()
	destDir := filepath.Join("data", "projects", newID)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao criar diretorio do projeto"})
		return
	}

	destDirAbs, err := filepath.Abs(destDir)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao resolver diretorio do projeto"})
		return
	}

	for _, f := range zr.File {
		clean := path.Clean(f.Name)
		if clean == "." {
			continue
		}
		if strings.HasPrefix(clean, "/") || strings.HasPrefix(clean, "..") || strings.Contains(clean, "../") {
			continue
		}
		if shouldSkipLogs(clean) {
			continue
		}

		destPath := filepath.Join(destDir, filepath.FromSlash(clean))
		destPathAbs, err := filepath.Abs(destPath)
		if err != nil {
			_ = os.RemoveAll(destDir)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao resolver caminho do zip"})
			return
		}
		if destPathAbs != destDirAbs && !strings.HasPrefix(destPathAbs, destDirAbs+string(os.PathSeparator)) {
			_ = os.RemoveAll(destDir)
			c.JSON(http.StatusBadRequest, gin.H{"error": "Arquivo zip contem caminhos invalidos"})
			return
		}

		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(destPath, 0755); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao criar diretorio do projeto"})
				return
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao criar diretorio do projeto"})
			return
		}

		in, err := f.Open()
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Erro ao ler arquivo do zip"})
			return
		}

		out, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			in.Close()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao criar arquivo do projeto"})
			return
		}

		if _, err := io.Copy(out, in); err != nil {
			in.Close()
			out.Close()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao extrair arquivo do zip"})
			return
		}

		in.Close()
		out.Close()
	}

	projectPath := filepath.Join(destDir, "project.json")
	if _, err := os.Stat(projectPath); os.IsNotExist(err) {
		_ = os.RemoveAll(destDir)
		c.JSON(http.StatusBadRequest, gin.H{"error": "project.json nao encontrado no zip"})
		return
	}

	project, err := loadProjectFile(projectPath)
	if err != nil {
		_ = os.RemoveAll(destDir)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao ler project.json"})
		return
	}

	decryptProjectFields(&project)
	project.ID = newID
	if projectName != "" {
		project.ProjectName = projectName
	}
	encryptProjectFields(&project)

	if err := writeProjectFile(projectPath, &project); err != nil {
		_ = os.RemoveAll(destDir)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao salvar project.json"})
		return
	}

	decryptProjectFields(&project)
	c.JSON(http.StatusCreated, project)
}

func shouldSkipLogs(relPath string) bool {
	relPath = filepath.ToSlash(relPath)
	parts := strings.Split(relPath, "/")
	for _, part := range parts {
		if part == "logs" {
			return true
		}
	}
	return false
}
