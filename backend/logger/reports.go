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
	primary:     color{31, 41, 55},
	primaryDark: color{17, 24, 39},
	accent:      color{59, 130, 246},
	success:     color{40, 167, 69},
	warning:     color{255, 193, 7},
	danger:      color{220, 53, 69},
	gray900:     color{33, 37, 41},
	gray700:     color{75, 85, 99},
	gray500:     color{120, 126, 131},
	gray200:     color{229, 231, 235},
	gray100:     color{249, 250, 251},
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
	e.addHeader(log)
	e.addStatusSummary(log)
	e.addJobDetails(log)
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
	cardY := 18.0
	cardH := 20.0
	e.pdf.SetFillColor(reportTheme.gray100.r, reportTheme.gray100.g, reportTheme.gray100.b)
	e.pdf.SetDrawColor(reportTheme.gray200.r, reportTheme.gray200.g, reportTheme.gray200.b)
	e.pdf.RoundedRect(left, cardY, pageW-left-right, cardH, 2, "1234", "DF")

	e.pdf.SetTextColor(reportTheme.primaryDark.r, reportTheme.primaryDark.g, reportTheme.primaryDark.b)
	e.pdf.SetFont("Arial", "B", 16)
	e.pdf.SetXY(left+4, cardY+3)
	e.pdf.CellFormat(pageW-left-right-8, 6, "Estatisticas do Pipeline", "", 1, "L", false, 0, "")

	e.pdf.SetFont("Arial", "", 9)
	e.pdf.SetTextColor(reportTheme.gray700.r, reportTheme.gray700.g, reportTheme.gray700.b)
	e.pdf.SetXY(left+4, cardY+10)
	e.pdf.CellFormat(pageW-left-right-8, 5, fmt.Sprintf("Pipeline: %s | Projeto: %s | Gerado em: %s", removeAccents(log.PipelineID), removeAccents(log.Project), time.Now().Format("02/01/2006 15:04:05")), "", 1, "L", false, 0, "")

	e.pdf.SetY(cardY + cardH + 4)
	e.pdf.SetDrawColor(reportTheme.gray200.r, reportTheme.gray200.g, reportTheme.gray200.b)
	e.pdf.SetLineWidth(0.4)
	e.pdf.Line(left, e.pdf.GetY(), pageW-right, e.pdf.GetY())
	e.pdf.Ln(6)
}

func (e *PDFExporter) addStatusSummary(log *PipelineLog) {
	e.addSectionTitle("Resumo da Execucao")
	stats := e.calculateStats(log)
	duration := e.formatDuration(log.EndedAt.Sub(log.StartedAt))

	pageW, _ := e.pdf.GetPageSize()
	left, _, right, _ := e.pdf.GetMargins()
	gap := 4.0
	cardW := (pageW - left - right - gap*3) / 4
	cardH := 18.0
	startY := e.pdf.GetY()

	top := []struct {
		label string
		value string
	}{
		{"Pipeline", removeAccents(log.Project)},
		{"Inicio", log.StartedAt.Format("02/01/2006 15:04:05")},
		{"Fim", log.EndedAt.Format("02/01/2006 15:04:05")},
		{"Duracao", duration},
	}
	for i, c := range top {
		x := left + float64(i)*(cardW+gap)
		e.drawMetricCard(x, startY, cardW, cardH, c.label, c.value, reportTheme.gray100)
	}

	y2 := startY + cardH + 4
	statusColor := e.getStatusColor(log.Status)
	bottom := []struct {
		label string
		value string
		col   color
	}{
		{"Total de Jobs", fmt.Sprintf("%d", stats.totalJobs), reportTheme.gray100},
		{"Registros Processados", fmt.Sprintf("%d", stats.totalProcessed), reportTheme.gray100},
		{"Duracao", duration, reportTheme.gray100},
		{"Status", e.translateStatus(log.Status), statusColor},
	}
	for i, c := range bottom {
		x := left + float64(i)*(cardW+gap)
		e.drawMetricCard(x, y2, cardW, cardH, c.label, c.value, c.col)
	}

	e.pdf.SetY(y2 + cardH + 6)
	e.pdf.SetFont("Arial", "B", 11)
	e.pdf.SetTextColor(reportTheme.primaryDark.r, reportTheme.primaryDark.g, reportTheme.primaryDark.b)
	e.pdf.CellFormat(0, 6, "Status dos Jobs:", "", 1, "L", false, 0, "")
	e.pdf.Ln(1)
	e.drawStatusChips(stats)
	e.pdf.Ln(6)
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
	e.addSectionTitle("Timeline de Execucoes")
	e.pdf.SetFont("Arial", "", 10)
	e.pdf.SetTextColor(reportTheme.gray700.r, reportTheme.gray700.g, reportTheme.gray700.b)
	e.pdf.CellFormat(0, 5, fmt.Sprintf("%d jobs nesta execucao", len(log.Jobs)), "", 1, "L", false, 0, "")
	e.pdf.Ln(1)

	for i, job := range log.Jobs {
		if e.pdf.GetY() > 245 {
			e.pdf.AddPage()
			e.addSectionTitle("Timeline de Execucoes")
		}
		e.addJobCard(i+1, &job)
		e.pdf.Ln(3)
	}
}

func (e *PDFExporter) addJobCard(index int, job *JobLog) {
	startY := e.pdf.GetY()
	cardH := 28.0
	if strings.TrimSpace(job.Error) != "" {
		cardH = 44.0
	}
	pageW, _ := e.pdf.GetPageSize()
	left, _, right, _ := e.pdf.GetMargins()
	cardW := pageW - left - right

	e.pdf.SetFillColor(255, 255, 255)
	e.pdf.SetDrawColor(reportTheme.gray200.r, reportTheme.gray200.g, reportTheme.gray200.b)
	e.pdf.RoundedRect(left, startY, cardW, cardH, 2, "1234", "D")

	e.pdf.SetFont("Arial", "B", 12)
	e.pdf.SetTextColor(reportTheme.primaryDark.r, reportTheme.primaryDark.g, reportTheme.primaryDark.b)
	e.pdf.SetXY(left+4, startY+3)
	e.pdf.CellFormat(110, 6, fmt.Sprintf("%d. %s", index, removeAccents(job.JobName)), "", 0, "L", false, 0, "")

	statusColor := e.getStatusColor(job.Status)
	e.pdf.SetFillColor(statusColor.r, statusColor.g, statusColor.b)
	e.pdf.SetTextColor(255, 255, 255)
	e.pdf.SetFont("Arial", "B", 9)
	e.pdf.SetXY(left+cardW-46, startY+3)
	e.pdf.CellFormat(22, 6, e.translateStatus(job.Status), "", 0, "C", true, 0, "")

	processed := job.Processed
	if processed == 0 && len(job.Batches) > 0 {
		processed = e.countBatchRows(job.Batches)
	}
	totalLabel := "0"
	if job.Total > 0 {
		totalLabel = fmt.Sprintf("%d", job.Total)
	}
	e.pdf.SetTextColor(reportTheme.gray700.r, reportTheme.gray700.g, reportTheme.gray700.b)
	e.pdf.SetFont("Arial", "B", 9)
	e.pdf.SetXY(left+cardW-22, startY+3)
	e.pdf.CellFormat(18, 6, fmt.Sprintf("%d/%s", processed, totalLabel), "", 0, "R", false, 0, "")
	e.pdf.SetXY(left+cardW-22, startY+10)
	e.pdf.CellFormat(18, 5, e.formatDuration(job.EndedAt.Sub(job.StartedAt)), "", 0, "R", false, 0, "")

	e.pdf.SetFont("Arial", "", 9)
	e.pdf.SetTextColor(reportTheme.gray700.r, reportTheme.gray700.g, reportTheme.gray700.b)
	e.pdf.SetXY(left+4, startY+10)
	e.pdf.CellFormat(cardW-56, 5, fmt.Sprintf("Inicio: %s    Fim: %s", job.StartedAt.Format("02/01/2006 15:04:05"), job.EndedAt.Format("02/01/2006 15:04:05")), "", 1, "L", false, 0, "")
	e.pdf.SetXY(left+4, startY+16)
	e.pdf.CellFormat(cardW-56, 4, fmt.Sprintf("ID: %s | Batches: %d | StopOnError: %t", shortJobID(job.JobID), len(job.Batches), job.StopOnError), "", 1, "L", false, 0, "")

	if strings.TrimSpace(job.Error) != "" {
		errorY := startY + 22
		e.pdf.SetFillColor(255, 244, 244)
		e.pdf.SetDrawColor(reportTheme.danger.r, reportTheme.danger.g, reportTheme.danger.b)
		e.pdf.RoundedRect(left+3, errorY, cardW-6, 18, 1.5, "1234", "DF")
		e.pdf.SetFont("Arial", "B", 10)
		e.pdf.SetTextColor(reportTheme.danger.r, reportTheme.danger.g, reportTheme.danger.b)
		e.pdf.SetXY(left+6, errorY+2)
		e.pdf.CellFormat(cardW-12, 4, "Erro", "", 1, "L", false, 0, "")
		e.pdf.SetFont("Arial", "", 9)
		e.pdf.SetTextColor(reportTheme.gray900.r, reportTheme.gray900.g, reportTheme.gray900.b)
		e.pdf.SetXY(left+6, errorY+7)
		e.pdf.MultiCell(cardW-12, 4, removeAccents(job.Error), "", "L", false)
	}

	e.pdf.SetY(startY + cardH + 1)
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

func (e *PDFExporter) drawMetricCard(x, y, w, h float64, label, value string, bg color) {
	e.pdf.SetFillColor(bg.r, bg.g, bg.b)
	e.pdf.SetDrawColor(reportTheme.gray200.r, reportTheme.gray200.g, reportTheme.gray200.b)
	e.pdf.RoundedRect(x, y, w, h, 1.5, "1234", "DF")
	e.pdf.SetFont("Arial", "B", 8)
	e.pdf.SetTextColor(reportTheme.gray500.r, reportTheme.gray500.g, reportTheme.gray500.b)
	e.pdf.SetXY(x+3, y+2)
	e.pdf.CellFormat(w-6, 4, strings.ToUpper(removeAccents(label)), "", 1, "L", false, 0, "")
	e.pdf.SetFont("Arial", "B", 12)
	e.pdf.SetTextColor(reportTheme.gray900.r, reportTheme.gray900.g, reportTheme.gray900.b)
	e.pdf.SetXY(x+3, y+8)
	e.pdf.CellFormat(w-6, 7, removeAccents(value), "", 1, "L", false, 0, "")
}

func (e *PDFExporter) drawStatusChips(stats pipelineStats) {
	type chip struct {
		label string
		value int
		col   color
	}
	chips := []chip{{"Concluidos", stats.jobsDone, reportTheme.success}, {"Com Erro", stats.jobsWithError, reportTheme.danger}}
	x := 18.0
	y := e.pdf.GetY()
	for _, c := range chips {
		if c.value <= 0 {
			continue
		}
		txt := fmt.Sprintf("%s: %d", c.label, c.value)
		w := e.pdf.GetStringWidth(txt) + 10
		e.pdf.SetFillColor(c.col.r, c.col.g, c.col.b)
		e.pdf.SetTextColor(255, 255, 255)
		e.pdf.SetFont("Arial", "B", 9)
		e.pdf.RoundedRect(x, y, w, 7, 3, "1234", "F")
		e.pdf.SetXY(x+5, y+1.3)
		e.pdf.CellFormat(w-8, 4.5, txt, "", 1, "L", false, 0, "")
		x += w + 3
	}
}

func (e *PDFExporter) formatDuration(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60
	return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
}

func shortJobID(id string) string {
	id = strings.TrimSpace(id)
	if len(id) <= 8 {
		return id
	}
	return id[:8] + "..."
}

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
