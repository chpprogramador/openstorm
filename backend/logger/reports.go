package logger

import (
	"bytes"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/phpdave11/gofpdf"
)

type PDFExporter struct {
	pdf *gofpdf.Fpdf
}

// NewPDFExporter cria uma nova instância do exportador PDF
func NewPDFExporter() *PDFExporter {
	pdf := gofpdf.New("P", "mm", "A4", "")
	return &PDFExporter{pdf: pdf}
}

// PipelineReportHandler endpoint para gerar e retornar PDF do pipeline
func PipelineReportHandler(c *gin.Context) {
	pipelineID := c.Param("pipelineId")
	if pipelineID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Pipeline ID é obrigatório",
		})
		return
	}

	// Carrega o log do pipeline
	log, err := LoadPipelineLog(pipelineID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": fmt.Sprintf("Pipeline não encontrado: %v", err),
		})
		return
	}

	// Gera o PDF em memória
	pdfBytes, err := GeneratePipelineReportPDF(log)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Erro ao gerar PDF: %v", err),
		})
		return
	}

	// Define headers para download do PDF
	filename := fmt.Sprintf("relatorio_%s.pdf", sanitizeFilename(pipelineID))
	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	c.Header("Content-Length", fmt.Sprintf("%d", len(pdfBytes)))

	// Retorna o PDF
	c.Data(http.StatusOK, "application/pdf", pdfBytes)
}

// PipelineReportInlineHandler endpoint para visualizar PDF inline no browser
func PipelineReportInlineHandler(c *gin.Context) {
	pipelineID := c.Param("pipelineId")
	if pipelineID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Pipeline ID é obrigatório",
		})
		return
	}

	// Carrega o log do pipeline
	log, err := LoadPipelineLog(pipelineID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": fmt.Sprintf("Pipeline não encontrado: %v", err),
		})
		return
	}

	// Gera o PDF em memória
	pdfBytes, err := GeneratePipelineReportPDF(log)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Erro ao gerar PDF: %v", err),
		})
		return
	}

	// Define headers para visualização inline
	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", "inline")
	c.Header("Content-Length", fmt.Sprintf("%d", len(pdfBytes)))

	// Retorna o PDF para visualização
	c.Data(http.StatusOK, "application/pdf", pdfBytes)
}

// GeneratePipelineReportPDF gera o PDF em memória e retorna os bytes
func GeneratePipelineReportPDF(log *PipelineLog) ([]byte, error) {
	exporter := NewPDFExporter()

	// Gera o relatório
	if err := exporter.generateReport(log); err != nil {
		return nil, err
	}

	// Retorna os bytes do PDF
	var buf bytes.Buffer
	err := exporter.pdf.Output(&buf)
	if err != nil {
		return nil, fmt.Errorf("erro ao gerar bytes do PDF: %v", err)
	}

	return buf.Bytes(), nil
}

// generateReport gera o relatório PDF completo
func (e *PDFExporter) generateReport(log *PipelineLog) error {
	e.pdf.AddPage()

	// Header
	e.addHeader(log)

	// Status geral
	e.addStatusSummary(log)

	// Informações gerais
	e.addGeneralInfo(log)

	// Estatísticas
	e.addStatistics(log)

	// Nova página para jobs se necessário
	if len(log.Jobs) > 3 {
		e.pdf.AddPage()
	}

	// Detalhes dos jobs
	e.addJobDetails(log)

	// Conclusões
	e.addConclusions(log)

	return nil
}

// ListPipelineReportsHandler endpoint para listar todos os pipelines disponíveis
func ListPipelineReportsHandler(c *gin.Context) {
	logs, err := ListPipelineLogs()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Erro ao listar pipelines: %v", err),
		})
		return
	}

	// Formata a lista com informações básicas
	var pipelines []map[string]interface{}
	for _, logFile := range logs {
		pipelineID := strings.TrimSuffix(strings.TrimPrefix(logFile, "pipeline_"), ".json")

		// Tenta carregar informações básicas do pipeline
		if log, err := LoadPipelineLog(pipelineID); err == nil {
			pipelines = append(pipelines, map[string]interface{}{
				"pipeline_id":  pipelineID,
				"project":      log.Project,
				"status":       log.Status,
				"started_at":   log.StartedAt,
				"ended_at":     log.EndedAt,
				"duration":     log.EndedAt.Sub(log.StartedAt).String(),
				"total_jobs":   len(log.Jobs),
				"download_url": fmt.Sprintf("/api/pipeline/%s/report", pipelineID),
				"preview_url":  fmt.Sprintf("/api/pipeline/%s/report/preview", pipelineID),
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"pipelines": pipelines,
		"total":     len(pipelines),
	})
}

// Funções auxiliares para geração do PDF

func (e *PDFExporter) addHeader(log *PipelineLog) {
	// Logo/Título principal
	e.pdf.SetFont("Arial", "B", 24)
	e.pdf.SetTextColor(0, 123, 255) // Azul
	e.pdf.CellFormat(0, 15, "RELATÓRIO DE MIGRAÇÃO", "", 1, "C", false, 0, "")

	// Subtítulo
	e.pdf.SetFont("Arial", "", 14)
	e.pdf.SetTextColor(102, 102, 102) // Cinza
	e.pdf.CellFormat(0, 8, strings.ToUpper(log.Project), "", 1, "C", false, 0, "")

	// Data de geração
	e.pdf.SetFont("Arial", "", 10)
	e.pdf.CellFormat(0, 6, fmt.Sprintf("Gerado em: %s", time.Now().Format("02/01/2006 15:04:05")), "", 1, "C", false, 0, "")

	// Linha separadora
	e.pdf.SetDrawColor(0, 123, 255)
	e.pdf.SetLineWidth(1)
	e.pdf.Line(20, e.pdf.GetY()+5, 190, e.pdf.GetY()+5)
	e.pdf.Ln(10)
}

func (e *PDFExporter) addStatusSummary(log *PipelineLog) {
	currentY := e.pdf.GetY()

	// Background colorido para o status
	statusColor := e.getStatusColor(log.Status)
	e.pdf.SetFillColor(statusColor.r, statusColor.g, statusColor.b)
	e.pdf.Rect(20, currentY, 170, 25, "F")

	// Texto do status
	e.pdf.SetTextColor(255, 255, 255) // Branco
	e.pdf.SetFont("Arial", "B", 16)
	statusText := fmt.Sprintf("STATUS GERAL: %s", strings.ToUpper(e.translateStatus(log.Status)))
	if log.Status == "done" {
		statusText += " ✓"
	}
	e.pdf.SetXY(25, currentY+5)
	e.pdf.CellFormat(160, 8, statusText, "", 1, "C", false, 0, "")

	// Duração
	duration := log.EndedAt.Sub(log.StartedAt)
	e.pdf.SetFont("Arial", "", 12)
	e.pdf.SetXY(25, currentY+15)
	e.pdf.CellFormat(160, 6, fmt.Sprintf("Pipeline executado em %v", duration), "", 1, "C", false, 0, "")

	e.pdf.SetY(currentY + 30)
	e.pdf.Ln(5)
}

func (e *PDFExporter) addGeneralInfo(log *PipelineLog) {
	e.addSectionTitle("📊 INFORMAÇÕES GERAIS")

	// Grid de informações (2 colunas)
	leftX := 20
	rightX := 110
	startY := e.pdf.GetY()

	// Coluna esquerda
	e.pdf.SetXY(float64(leftX), startY)
	e.addInfoCard("Pipeline ID", log.PipelineID, 85)
	e.addInfoCard("Projeto", log.Project, 85)
	e.addInfoCard("Status Final", e.translateStatus(log.Status), 85)

	// Coluna direita
	e.pdf.SetXY(float64(rightX), startY)
	duration := log.EndedAt.Sub(log.StartedAt)
	e.addInfoCard("Duração Total", duration.String(), 85)
	e.addInfoCard("Início", log.StartedAt.Format("02/01/2006 15:04:05"), 85)
	e.addInfoCard("Término", log.EndedAt.Format("02/01/2006 15:04:05"), 85)

	e.pdf.Ln(10)
}

func (e *PDFExporter) addStatistics(log *PipelineLog) {
	e.addSectionTitle("📈 ESTATÍSTICAS DE EXECUÇÃO")

	// Calcula estatísticas
	stats := e.calculateStats(log)

	leftX := 20
	rightX := 110
	startY := e.pdf.GetY()

	// Coluna esquerda
	e.pdf.SetXY(float64(leftX), startY)
	e.addInfoCard("Total de Jobs", fmt.Sprintf("%d", stats.totalJobs), 85)
	e.addInfoCard("Jobs Concluídos", fmt.Sprintf("%d (%.0f%%)", stats.jobsDone, stats.successRate), 85)

	// Coluna direita
	e.pdf.SetXY(float64(rightX), startY)
	e.addInfoCard("Total de Batches", fmt.Sprintf("%d", stats.totalBatches), 85)
	e.addInfoCard("Registros Processados", fmt.Sprintf("%d", stats.totalProcessed), 85)

	e.pdf.Ln(10)
}

func (e *PDFExporter) addJobDetails(log *PipelineLog) {
	e.addSectionTitle("🔧 DETALHES DOS JOBS EXECUTADOS")

	for i, job := range log.Jobs {
		if e.pdf.GetY() > 250 { // Quebra de página se necessário
			e.pdf.AddPage()
		}

		e.addJobCard(i+1, &job)
		e.pdf.Ln(3)
	}
}

func (e *PDFExporter) addJobCard(index int, job *JobLog) {
	startY := e.pdf.GetY()

	// Background do card
	e.pdf.SetFillColor(248, 249, 250) // Cinza claro
	e.pdf.Rect(20, startY, 170, 20, "F")

	// Border do card
	e.pdf.SetDrawColor(233, 236, 239)
	e.pdf.SetLineWidth(0.5)
	e.pdf.Rect(20, startY, 170, 20, "D")

	// Nome do job
	e.pdf.SetFont("Arial", "B", 12)
	e.pdf.SetTextColor(51, 51, 51)
	e.pdf.SetXY(25, startY+3)
	e.pdf.CellFormat(100, 6, fmt.Sprintf("%d. %s", index, job.JobName), "", 0, "L", false, 0, "")

	// Status badge
	statusColor := e.getStatusColor(job.Status)
	e.pdf.SetFillColor(statusColor.r, statusColor.g, statusColor.b)
	e.pdf.SetTextColor(255, 255, 255)
	e.pdf.SetFont("Arial", "B", 8)
	badgeX := 160
	e.pdf.SetXY(float64(badgeX), startY+2)
	e.pdf.CellFormat(25, 8, strings.ToUpper(e.translateStatus(job.Status)), "", 0, "C", true, 0, "")

	// Detalhes do job
	e.pdf.SetFont("Arial", "", 9)
	e.pdf.SetTextColor(102, 102, 102)
	e.pdf.SetXY(25, startY+9)

	duration := job.EndedAt.Sub(job.StartedAt)
	details := fmt.Sprintf("Job ID: %s | Duração: %v | Processados: %d/%d registros",
		job.JobID[:8]+"...", duration, job.Processed, job.Total)
	e.pdf.CellFormat(130, 4, details, "", 1, "L", false, 0, "")

	e.pdf.SetXY(25, startY+13)
	stopOnError := "Não"
	if job.StopOnError {
		stopOnError = "Sim"
	}
	e.pdf.CellFormat(130, 4, fmt.Sprintf("Stop on Error: %s", stopOnError), "", 1, "L", false, 0, "")

	// Informações de batches se houver
	if len(job.Batches) > 0 {
		e.pdf.SetXY(25, startY+17)
		e.pdf.SetFont("Arial", "", 8)
		e.pdf.SetTextColor(25, 118, 210) // Azul
		batchInfo := fmt.Sprintf("Batches: %d executados, %d linhas processadas",
			len(job.Batches), e.countBatchRows(job.Batches))
		e.pdf.CellFormat(130, 3, batchInfo, "", 1, "L", false, 0, "")
	}

	e.pdf.SetY(startY + 25)
}

func (e *PDFExporter) addConclusions(log *PipelineLog) {
	if e.pdf.GetY() > 220 {
		e.pdf.AddPage()
	}

	e.addSectionTitle("✅ CONCLUSÕES E RECOMENDAÇÕES")

	stats := e.calculateStats(log)

	// Background da conclusão
	startY := e.pdf.GetY()
	e.pdf.SetFillColor(212, 237, 218) // Verde claro
	e.pdf.Rect(20, startY, 170, 40, "F")

	// Border verde
	e.pdf.SetDrawColor(40, 167, 69)
	e.pdf.SetLineWidth(2)
	e.pdf.Line(20, startY, 20, startY+40)

	// Título da conclusão
	e.pdf.SetFont("Arial", "B", 14)
	e.pdf.SetTextColor(21, 87, 36) // Verde escuro
	e.pdf.SetXY(25, startY+5)
	if log.Status == "done" {
		e.pdf.CellFormat(160, 8, "Pipeline Executado com Sucesso", "", 1, "L", false, 0, "")
	} else {
		e.pdf.CellFormat(160, 8, "Pipeline com Problemas", "", 1, "L", false, 0, "")
	}

	// Pontos principais
	e.pdf.SetFont("Arial", "", 10)
	e.pdf.SetTextColor(51, 51, 51)
	y := startY + 15

	points := []string{
		fmt.Sprintf("• Execução rápida: Concluído em %v", log.EndedAt.Sub(log.StartedAt)),
		fmt.Sprintf("• Taxa de sucesso: %.0f%% dos jobs executados com sucesso", stats.successRate),
		fmt.Sprintf("• Processamento: %d registros processados em %d batches", stats.totalProcessed, stats.totalBatches),
		fmt.Sprintf("• Jobs configurados: %d com fail-fast, %d com fail-safe", stats.jobsWithStopOnError, stats.totalJobs-stats.jobsWithStopOnError),
	}

	for _, point := range points {
		e.pdf.SetXY(25, y)
		e.pdf.CellFormat(160, 5, point, "", 1, "L", false, 0, "")
		y += 6
	}

	e.pdf.SetY(startY + 45)
}

// Funções auxiliares

func (e *PDFExporter) addSectionTitle(title string) {
	e.pdf.SetFont("Arial", "B", 14)
	e.pdf.SetTextColor(0, 123, 255)
	e.pdf.CellFormat(0, 10, title, "", 1, "L", false, 0, "")
	e.pdf.Ln(2)
}

func (e *PDFExporter) addInfoCard(label, value string, width float64) {
	currentY := e.pdf.GetY()

	// Background do card
	e.pdf.SetFillColor(248, 249, 250)
	e.pdf.Rect(e.pdf.GetX(), currentY, width, 15, "F")

	// Border esquerda colorida
	e.pdf.SetDrawColor(40, 167, 69) // Verde
	e.pdf.SetLineWidth(2)
	e.pdf.Line(e.pdf.GetX(), currentY, e.pdf.GetX(), currentY+15)

	// Label
	e.pdf.SetFont("Arial", "B", 8)
	e.pdf.SetTextColor(102, 102, 102)
	e.pdf.SetXY(e.pdf.GetX()+3, currentY+2)
	e.pdf.CellFormat(width-6, 4, strings.ToUpper(label), "", 1, "L", false, 0, "")

	// Value
	e.pdf.SetFont("Arial", "B", 11)
	e.pdf.SetTextColor(40, 167, 69)
	e.pdf.SetXY(e.pdf.GetX()+3, currentY+7)

	// Trunca o valor se for muito longo
	if len(value) > 25 {
		value = value[:22] + "..."
	}

	e.pdf.CellFormat(width-6, 6, value, "", 1, "L", false, 0, "")

	e.pdf.SetY(currentY + 18)
}

// Estruturas e funções auxiliares

type color struct {
	r, g, b int
}

func (e *PDFExporter) getStatusColor(status string) color {
	switch status {
	case "done", "completed", "success":
		return color{40, 167, 69} // Verde
	case "error", "failed":
		return color{220, 53, 69} // Vermelho
	case "running", "in_progress":
		return color{255, 193, 7} // Amarelo
	default:
		return color{108, 117, 125} // Cinza
	}
}

func (e *PDFExporter) translateStatus(status string) string {
	switch status {
	case "done":
		return "CONCLUÍDO"
	case "error":
		return "ERRO"
	case "running":
		return "EXECUTANDO"
	case "pending":
		return "PENDENTE"
	default:
		return strings.ToUpper(status)
	}
}

type pipelineStats struct {
	totalJobs           int
	jobsDone            int
	jobsWithError       int
	jobsWithStopOnError int
	totalBatches        int
	totalProcessed      int
	successRate         float64
}

func (e *PDFExporter) calculateStats(log *PipelineLog) pipelineStats {
	stats := pipelineStats{
		totalJobs: len(log.Jobs),
	}

	for _, job := range log.Jobs {
		if job.Status == "done" {
			stats.jobsDone++
		}
		if job.Error != "" {
			stats.jobsWithError++
		}
		if job.StopOnError {
			stats.jobsWithStopOnError++
		}
		stats.totalBatches += len(job.Batches)
		stats.totalProcessed += job.Processed
	}

	if stats.totalJobs > 0 {
		stats.successRate = float64(stats.jobsDone) / float64(stats.totalJobs) * 100
	}

	return stats
}

func (e *PDFExporter) countBatchRows(batches []BatchLog) int {
	total := 0
	for _, batch := range batches {
		total += batch.Rows
	}
	return total
}

// sanitizeFilename remove caracteres inválidos do nome do arquivo
func sanitizeFilename(filename string) string {
	// Remove caracteres especiais que podem causar problemas
	replacer := strings.NewReplacer(
		" ", "_",
		":", "-",
		"/", "-",
		"\\", "-",
		"*", "_",
		"?", "_",
		"\"", "_",
		"<", "_",
		">", "_",
		"|", "_",
	)
	return replacer.Replace(filename)
}
