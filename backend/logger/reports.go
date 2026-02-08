package logger

import (
	"bytes"
	"fmt"
	"net/http"
	"strings"
	"time"
	"unicode"

	"github.com/gin-gonic/gin"
	"github.com/phpdave11/gofpdf"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

// removeAccents remove acentos de uma string
func removeAccents(s string) string {
	t := transform.Chain(norm.NFD, transform.RemoveFunc(isMn), norm.NFC)
	result, _, _ := transform.String(t, s)
	return result
}

// isMn verifica se um rune e um marcador de acento
func isMn(r rune) bool {
	return unicode.Is(unicode.Mn, r) // Mn: Mark, nonspacing
}

type PDFExporter struct {
	pdf *gofpdf.Fpdf
}

type theme struct {
	primary     color
	primaryDark color
	accent      color
	success     color
	warning     color
	danger      color
	gray900     color
	gray700     color
	gray500     color
	gray200     color
	gray100     color
}

var reportTheme = theme{
	primary:     color{24, 94, 174},
	primaryDark: color{16, 63, 118},
	accent:      color{0, 153, 168},
	success:     color{40, 167, 69},
	warning:     color{255, 193, 7},
	danger:      color{220, 53, 69},
	gray900:     color{33, 37, 41},
	gray700:     color{73, 80, 87},
	gray500:     color{120, 126, 131},
	gray200:     color{233, 236, 239},
	gray100:     color{248, 249, 250},
}

// NewPDFExporter cria uma nova instância do exportador PDF
func NewPDFExporter() *PDFExporter {
	pdf := gofpdf.New("P", "mm", "A4", "")
	//pdf.AddUTF8Font("DejaVu", "", "./fonts/DejaVuSans.ttf")
	//pdf.AddUTF8Font("DejaVu", "B", "./fonts/DejaVuSans-Bold.ttf")
	pdf.SetFont("Arial", "", 14)
	pdf.SetMargins(18, 18, 18)
	pdf.SetAutoPageBreak(true, 18)
	pdf.SetFooterFunc(func() {
		pdf.SetY(-14)
		pdf.SetFont("Arial", "", 8)
		pdf.SetTextColor(reportTheme.gray500.r, reportTheme.gray500.g, reportTheme.gray500.b)
		pdf.CellFormat(0, 6, fmt.Sprintf("Pagina %d", pdf.PageNo()), "", 0, "C", false, 0, "")
	})
	return &PDFExporter{pdf: pdf}
}

// PipelineReportHandler endpoint para gerar e retornar PDF do pipeline
func PipelineReportHandler(c *gin.Context) {
	pipelineID := c.Param("pipelineId")
	if pipelineID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Pipeline ID e obrigatorio",
		})
		return
	}

	// Carrega o log do pipeline
	log, err := LoadPipelineLog(pipelineID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": fmt.Sprintf("Pipeline nao encontrado: %v", err),
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
			"error": "Pipeline ID e obrigatório",
		})
		return
	}

	// Carrega o log do pipeline
	log, err := LoadPipelineLog(pipelineID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": fmt.Sprintf("Pipeline nao encontrado: %v", err),
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
	projectID := strings.TrimSpace(c.Query("projectId"))

	logs, err := ListPipelineLogs()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Erro ao listar pipelines: %v", err),
		})
		return
	}

	if projectID == "" {
		c.JSON(http.StatusOK, logs)
		return
	}

	filtered := make([]string, 0, len(logs))
	for _, pipelineID := range logs {
		log, err := LoadPipelineLog(pipelineID)
		if err != nil {
			continue
		}
		if log.ProjectID == projectID {
			filtered = append(filtered, pipelineID)
		}
	}

	c.JSON(http.StatusOK, filtered)

	// // Formata a lista com informações básicas
	// var pipelines []map[string]interface{}
	// for _, logFile := range logs {
	// 	pipelineID := strings.TrimSuffix(strings.TrimPrefix(logFile, "pipeline_"), ".json")

	// 	// Tenta carregar informações básicas do pipeline
	// 	if log, err := LoadPipelineLog(pipelineID); err == nil {
	// 		pipelines = append(pipelines, map[string]interface{}{
	// 			"pipeline_id":  pipelineID,
	// 			"project":      log.Project,
	// 			"status":       log.Status,
	// 			"started_at":   log.StartedAt,
	// 			"ended_at":     log.EndedAt,
	// 			"duration":     log.EndedAt.Sub(log.StartedAt).String(),
	// 			"total_jobs":   len(log.Jobs),
	// 			"download_url": fmt.Sprintf("/api/pipeline/%s/report", pipelineID),
	// 			"preview_url":  fmt.Sprintf("/api/pipeline/%s/report/preview", pipelineID),
	// 		})
	// 	}
	// }

	// c.JSON(http.StatusOK, gin.H{
	// 	"pipelines": pipelines,
	// 	"total":     len(pipelines),
	// })
}


// Funções auxiliares para geração do PDF

func (e *PDFExporter) addHeader(log *PipelineLog) {
	pageW, _ := e.pdf.GetPageSize()
	left, _, right, _ := e.pdf.GetMargins()
	bandH := 28.0
	e.pdf.SetFillColor(reportTheme.primary.r, reportTheme.primary.g, reportTheme.primary.b)
	e.pdf.Rect(0, 0, pageW, bandH, "F")

	e.pdf.SetTextColor(255, 255, 255)
	e.pdf.SetFont("Arial", "B", 20)
	e.pdf.SetXY(left, 6)
	e.pdf.CellFormat(pageW-left-right, 8, "RELATORIO DE EXECUCAO", "", 1, "L", false, 0, "")

	e.pdf.SetFont("Arial", "", 11)
	e.pdf.SetXY(left, 15)
	e.pdf.CellFormat(pageW-left-right, 6, strings.ToUpper(removeAccents(log.Project)), "", 1, "L", false, 0, "")

	e.pdf.SetFont("Arial", "", 9)
	e.pdf.SetXY(left, 22)
	e.pdf.CellFormat(pageW-left-right, 5, fmt.Sprintf("Pipeline: %s | Gerado em: %s",
		removeAccents(log.PipelineID), time.Now().Format("02/01/2006 15:04:05")), "", 1, "L", false, 0, "")

	e.pdf.SetY(bandH + 6)
	e.pdf.SetDrawColor(reportTheme.gray200.r, reportTheme.gray200.g, reportTheme.gray200.b)
	e.pdf.SetLineWidth(0.4)
	e.pdf.Line(left, e.pdf.GetY(), pageW-right, e.pdf.GetY())
	e.pdf.Ln(6)
}

func (e *PDFExporter) addStatusSummary(log *PipelineLog) {
	currentY := e.pdf.GetY()

	// Background do status
	statusColor := e.getStatusColor(log.Status)
	e.pdf.SetFillColor(statusColor.r, statusColor.g, statusColor.b)
	e.pdf.Rect(18, currentY, 174, 24, "F")

	// Texto do status
	e.pdf.SetTextColor(255, 255, 255) // Branco
	e.pdf.SetFont("Arial", "B", 15)
	statusText := fmt.Sprintf("STATUS GERAL: %s", strings.ToUpper(e.translateStatus(log.Status)))

	e.pdf.SetXY(22, currentY+4)
	e.pdf.CellFormat(166, 8, statusText, "", 1, "C", false, 0, "")

	// Duração
	duration := log.EndedAt.Sub(log.StartedAt)

	hours := int(duration.Hours())
	minutes := int(duration.Minutes()) % 60
	seconds := int(duration.Seconds()) % 60

	formattedDuration := fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds)

	e.pdf.SetFont("Arial", "", 11)
	e.pdf.SetXY(22, currentY+14)
	e.pdf.CellFormat(166, 6, fmt.Sprintf("Pipeline executado em %s", formattedDuration), "", 1, "C", false, 0, "")

	e.pdf.SetY(currentY + 28)
	e.pdf.Ln(5)
}

func (e *PDFExporter) addGeneralInfo(log *PipelineLog) {
	e.addSectionTitle("INFORMACOES GERAIS")

	// Grid de informações (2 colunas)
	leftX := 18
	rightX := 108
	startY := e.pdf.GetY()

	// Coluna esquerda
	e.pdf.SetXY(float64(leftX), startY)
	duration := log.EndedAt.Sub(log.StartedAt)

	hours := int(duration.Hours())
	minutes := int(duration.Minutes()) % 60
	seconds := int(duration.Seconds()) % 60

	formattedDuration := fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds)

	e.addInfoCard("Pipeline ID", removeAccents(log.PipelineID), 86)

	e.pdf.SetXY(float64(leftX), e.pdf.GetY())
	e.addInfoCard("Projeto", removeAccents(log.Project), 86)

	e.pdf.SetXY(float64(leftX), e.pdf.GetY())
	e.addInfoCard("Status Final", e.translateStatus(log.Status), 86)

	// Coluna direita
	e.pdf.SetXY(float64(rightX), startY)
	e.addInfoCard("Duracao Total", formattedDuration, 86)

	e.pdf.SetXY(float64(rightX), e.pdf.GetY())
	e.addInfoCard("Inicio", log.StartedAt.Format("02/01/2006 15:04:05"), 86)

	e.pdf.SetXY(float64(rightX), e.pdf.GetY())
	e.addInfoCard("Termino", log.EndedAt.Format("02/01/2006 15:04:05"), 86)

	e.pdf.Ln(10)
}

func (e *PDFExporter) addStatistics(log *PipelineLog) {
	e.addSectionTitle("ESTATISTICAS DE EXECUCAO")

	// Calcula estatísticas
	stats := e.calculateStats(log)

	leftX := 18
	rightX := 108
	startY := e.pdf.GetY()

	// Coluna esquerda
	e.pdf.SetXY(float64(leftX), startY)
	e.addInfoCard("Total de Jobs", fmt.Sprintf("%d", stats.totalJobs), 86)

	e.pdf.SetXY(float64(leftX), e.pdf.GetY())
	e.addInfoCard("Jobs Concluidos", fmt.Sprintf("%d", stats.jobsDone), 86)

	// Coluna direita
	e.pdf.SetXY(float64(rightX), startY)
	e.addInfoCard("Total de Batches", fmt.Sprintf("%d", stats.totalBatches), 86)

	e.pdf.SetXY(float64(rightX), e.pdf.GetY())
	e.addInfoCard("Registros Processados", fmt.Sprintf("%d", stats.totalProcessed), 86)

	e.pdf.Ln(10)
}

func (e *PDFExporter) addJobDetails(log *PipelineLog) {
	e.addSectionTitle("DETALHES DOS JOBS EXECUTADOS")

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
	e.pdf.SetFillColor(reportTheme.gray100.r, reportTheme.gray100.g, reportTheme.gray100.b)
	e.pdf.Rect(18, startY, 174, 24, "F")

	// Border do card
	e.pdf.SetDrawColor(reportTheme.gray200.r, reportTheme.gray200.g, reportTheme.gray200.b)
	e.pdf.SetLineWidth(0.4)
	e.pdf.Rect(18, startY, 174, 24, "D")

	// Nome do job
	e.pdf.SetFont("Arial", "B", 12)
	e.pdf.SetTextColor(reportTheme.gray900.r, reportTheme.gray900.g, reportTheme.gray900.b)
	e.pdf.SetXY(22, startY+3)
	e.pdf.CellFormat(120, 6, fmt.Sprintf("%d. %s", index, removeAccents(job.JobName)), "", 0, "L", false, 0, "")

	// Status badge
	statusColor := e.getStatusColor(job.Status)
	e.pdf.SetFillColor(statusColor.r, statusColor.g, statusColor.b)
	e.pdf.SetTextColor(255, 255, 255)
	e.pdf.SetFont("Arial", "B", 8)
	badgeX := 156
	e.pdf.SetXY(float64(badgeX), startY+3)
	e.pdf.CellFormat(32, 7, strings.ToUpper(e.translateStatus(job.Status)), "", 0, "C", true, 0, "")

	// Detalhes do job
	e.pdf.SetFont("Arial", "", 9)
	e.pdf.SetTextColor(reportTheme.gray700.r, reportTheme.gray700.g, reportTheme.gray700.b)
	e.pdf.SetXY(22, startY+10)

	duration := job.EndedAt.Sub(job.StartedAt)

	hours := int(duration.Hours())
	minutes := int(duration.Minutes()) % 60
	seconds := int(duration.Seconds()) % 60

	formattedDuration := fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds)

	processed := job.Processed
	if processed == 0 && len(job.Batches) > 0 {
		processed = e.countBatchRows(job.Batches)
	}
	details := fmt.Sprintf("Job ID: %s | Duracao: %v | Processados: %d registros",
		job.JobID[:8]+"...", formattedDuration, processed)
	e.pdf.CellFormat(150, 4, details, "", 1, "L", false, 0, "")

	e.pdf.SetXY(22, startY+15)
	stopOnError := "Nao"
	if job.StopOnError {
		stopOnError = "Sim"
	}
	e.pdf.CellFormat(150, 4, fmt.Sprintf("Stop on Error: %s", stopOnError), "", 1, "L", false, 0, "")

	// Informações de batches se houver
	if len(job.Batches) > 0 {
		e.pdf.SetXY(22, startY+19)
		e.pdf.SetFont("Arial", "", 8)
		e.pdf.SetTextColor(reportTheme.accent.r, reportTheme.accent.g, reportTheme.accent.b)
		batchInfo := fmt.Sprintf("Batches: %d executados, %d linhas processadas",
			len(job.Batches), e.countBatchRows(job.Batches))
		e.pdf.CellFormat(150, 3, batchInfo, "", 1, "L", false, 0, "")
	}

	e.pdf.SetY(startY + 28)
}

func (e *PDFExporter) addConclusions(log *PipelineLog) {
	if e.pdf.GetY() > 220 {
		e.pdf.AddPage()
	}

	e.addSectionTitle("CONCLUSAO E RECOMENDACOES")

	stats := e.calculateStats(log)

	// Background da conclusão
	startY := e.pdf.GetY()
	e.pdf.SetFillColor(232, 245, 235)
	e.pdf.Rect(18, startY, 174, 42, "F")

	// Border verde
	e.pdf.SetDrawColor(reportTheme.success.r, reportTheme.success.g, reportTheme.success.b)
	e.pdf.SetLineWidth(2.2)
	e.pdf.Line(18, startY, 18, startY+42)

	// Título da conclusão
	e.pdf.SetFont("Arial", "B", 14)
	e.pdf.SetTextColor(21, 87, 36)
	e.pdf.SetXY(23, startY+6)
	if log.Status == "done" {
		e.pdf.CellFormat(164, 8, "Pipeline Executado com Sucesso", "", 1, "L", false, 0, "")
	} else {
		e.pdf.CellFormat(164, 8, "Pipeline com Problemas", "", 1, "L", false, 0, "")
	}

	// Pontos principais
	e.pdf.SetFont("Arial", "", 10)
	e.pdf.SetTextColor(51, 51, 51)
	y := startY + 17

	// Calcula a duração formatada
	duration := log.EndedAt.Sub(log.StartedAt)

	hours := int(duration.Hours())
	minutes := int(duration.Minutes()) % 60
	seconds := int(duration.Seconds()) % 60

	formattedDuration := fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds)

	points := []string{
		fmt.Sprintf("Execucao: Concluido em %v", formattedDuration),
		fmt.Sprintf("Taxa de sucesso: %.0f%% dos jobs executados com sucesso", stats.successRate),
		fmt.Sprintf("Jobs configurados: %d com fail-fast, %d com fail-safe", stats.jobsWithStopOnError, stats.totalJobs-stats.jobsWithStopOnError),
	}

	for _, point := range points {
		e.pdf.SetXY(23, y)
		e.pdf.CellFormat(164, 5, point, "", 1, "L", false, 0, "")
		y += 6
	}

	e.pdf.SetY(startY + 48)
}

// Funções auxiliares

func (e *PDFExporter) addSectionTitle(title string) {
	e.pdf.SetFont("Arial", "B", 13)
	e.pdf.SetTextColor(reportTheme.primaryDark.r, reportTheme.primaryDark.g, reportTheme.primaryDark.b)
	e.pdf.CellFormat(0, 8, title, "", 1, "L", false, 0, "")
	e.pdf.SetDrawColor(reportTheme.gray200.r, reportTheme.gray200.g, reportTheme.gray200.b)
	e.pdf.SetLineWidth(0.4)
	e.pdf.Line(18, e.pdf.GetY(), 192, e.pdf.GetY())
	e.pdf.Ln(4)
}

func (e *PDFExporter) addInfoCard(label, value string, width float64) {
	currentX := e.pdf.GetX()
	currentY := e.pdf.GetY()

	// Background do card
	e.pdf.SetFillColor(reportTheme.gray100.r, reportTheme.gray100.g, reportTheme.gray100.b)
	e.pdf.Rect(currentX, currentY, width, 16, "F")

	// Border esquerda colorida
	e.pdf.SetDrawColor(reportTheme.accent.r, reportTheme.accent.g, reportTheme.accent.b)
	e.pdf.SetLineWidth(2)
	e.pdf.Line(currentX, currentY, currentX, currentY+16)

	// Label
	e.pdf.SetFont("Arial", "B", 8)
	e.pdf.SetTextColor(reportTheme.gray500.r, reportTheme.gray500.g, reportTheme.gray500.b)
	e.pdf.SetXY(currentX+3, currentY+3)
	e.pdf.CellFormat(width-6, 4, strings.ToUpper(label), "", 1, "L", false, 0, "")

	// Value
	e.pdf.SetFont("Arial", "B", 11)
	e.pdf.SetTextColor(reportTheme.gray900.r, reportTheme.gray900.g, reportTheme.gray900.b)
	e.pdf.SetXY(currentX+3, currentY+8)

	if len(value) > 25 {
		value = value[:22] + "..."
	}

	e.pdf.CellFormat(width-6, 6, value, "", 1, "L", false, 0, "")

	// Atualiza a coordenada Y manualmente, se necessario
	e.pdf.SetY(currentY + 19)
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
		return "CONCLUIDO"
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
		if job.Processed > 0 {
			stats.totalProcessed += job.Processed
		} else {
			stats.totalProcessed += e.countBatchRows(job.Batches)
		}
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
