package logger

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"etl/models"

	"github.com/gin-gonic/gin"
)

var benchmarkTheme = struct {
	bg         color
	card       color
	cardAlt    color
	cardBorder color
	textMain   color
	textMuted  color
	green      color
	orange     color
	red        color
}{
	bg:         color{11, 18, 32},
	card:       color{16, 27, 48},
	cardAlt:    color{18, 32, 54},
	cardBorder: color{32, 49, 78},
	textMain:   color{231, 238, 247},
	textMuted:  color{151, 168, 192},
	green:      color{33, 160, 98},
	orange:     color{196, 140, 64},
	red:        color{180, 72, 72},
}

// BenchmarkReportHandler endpoint para gerar e retornar PDF do benchmark
func BenchmarkReportHandler(c *gin.Context) {
	projectID := c.Param("id")
	runID := c.Param("runId")
	if projectID == "" || runID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Project ID e Run ID são obrigatórios",
		})
		return
	}

	run, err := loadBenchmarkRun(projectID, runID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": fmt.Sprintf("Benchmark não encontrado: %v", err),
		})
		return
	}

	projectName := loadProjectName(projectID)
	pdfBytes, err := GenerateBenchmarkReportPDF(projectName, run)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Erro ao gerar PDF: %v", err),
		})
		return
	}

	filename := fmt.Sprintf("benchmark_%s.pdf", sanitizeFilename(runID))
	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	c.Header("Content-Length", fmt.Sprintf("%d", len(pdfBytes)))
	c.Data(http.StatusOK, "application/pdf", pdfBytes)
}

// BenchmarkHistoryReportHandler endpoint para gerar PDF com histórico de benchmarks
func BenchmarkHistoryReportHandler(c *gin.Context) {
	projectID := c.Param("id")
	if projectID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Project ID é obrigatório",
		})
		return
	}

	limit := parseBenchmarkLimit(c.Query("limit"))
	summaries, err := listBenchmarkSummaries(projectID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Erro ao listar benchmarks: %v", err),
		})
		return
	}
	if len(summaries) == 0 {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Nenhum benchmark encontrado",
		})
		return
	}

	projectName := loadProjectName(projectID)
	pdfBytes, err := GenerateBenchmarkHistoryPDF(projectName, summaries)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Erro ao gerar PDF: %v", err),
		})
		return
	}

	filename := fmt.Sprintf("benchmarks_%s.pdf", sanitizeFilename(projectID))
	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	c.Header("Content-Length", fmt.Sprintf("%d", len(pdfBytes)))
	c.Data(http.StatusOK, "application/pdf", pdfBytes)
}

// GenerateBenchmarkReportPDF gera o PDF de um benchmark específico
func GenerateBenchmarkReportPDF(projectName string, run *models.BenchmarkRun) ([]byte, error) {
	exporter := NewPDFExporter()
	exporter.pdf.AddPage()

	exporter.renderBenchmarkReportUI(projectName, run)

	var buf bytes.Buffer
	if err := exporter.pdf.Output(&buf); err != nil {
		return nil, fmt.Errorf("erro ao gerar bytes do PDF: %v", err)
	}
	return buf.Bytes(), nil
}

// GenerateBenchmarkHistoryPDF gera o PDF do histórico de benchmarks
func GenerateBenchmarkHistoryPDF(projectName string, summaries []models.BenchmarkSummary) ([]byte, error) {
	exporter := NewPDFExporter()
	exporter.pdf.AddPage()

	exporter.addBenchmarkHistoryHeader(projectName, len(summaries))
	exporter.addBenchmarkHistoryTable(summaries)

	var buf bytes.Buffer
	if err := exporter.pdf.Output(&buf); err != nil {
		return nil, fmt.Errorf("erro ao gerar bytes do PDF: %v", err)
	}
	return buf.Bytes(), nil
}

func (e *PDFExporter) addBenchmarkHeader(projectName, runID string) {
	pageW, _ := e.pdf.GetPageSize()
	left, _, right, _ := e.pdf.GetMargins()
	bandH := 26.0
	e.pdf.SetFillColor(reportTheme.primary.r, reportTheme.primary.g, reportTheme.primary.b)
	e.pdf.Rect(0, 0, pageW, bandH, "F")

	e.pdf.SetTextColor(255, 255, 255)
	e.pdf.SetFont("Arial", "B", 18)
	e.pdf.SetXY(left, 6)
	e.pdf.CellFormat(pageW-left-right, 7, "RELATORIO DE BENCHMARK", "", 1, "L", false, 0, "")

	if strings.TrimSpace(projectName) == "" {
		projectName = "Projeto"
	}
	e.pdf.SetFont("Arial", "", 10)
	e.pdf.SetXY(left, 14)
	e.pdf.CellFormat(pageW-left-right, 5, strings.ToUpper(removeAccents(projectName)), "", 1, "L", false, 0, "")

	e.pdf.SetFont("Arial", "", 9)
	e.pdf.SetXY(left, 20)
	e.pdf.CellFormat(pageW-left-right, 4, fmt.Sprintf("Run: %s | Gerado em: %s",
		removeAccents(runID), time.Now().Format("02/01/2006 15:04:05")), "", 1, "L", false, 0, "")

	e.pdf.SetY(bandH + 6)
	e.pdf.SetDrawColor(reportTheme.gray200.r, reportTheme.gray200.g, reportTheme.gray200.b)
	e.pdf.SetLineWidth(0.4)
	e.pdf.Line(left, e.pdf.GetY(), pageW-right, e.pdf.GetY())
	e.pdf.Ln(6)
}

func (e *PDFExporter) addBenchmarkStatus(run *models.BenchmarkRun) {
	currentY := e.pdf.GetY()
	statusColor := benchmarkStatusColor(run.Status)
	e.pdf.SetFillColor(statusColor.r, statusColor.g, statusColor.b)
	e.pdf.Rect(18, currentY, 174, 22, "F")

	e.pdf.SetTextColor(255, 255, 255)
	e.pdf.SetFont("Arial", "B", 14)
	statusText := fmt.Sprintf("STATUS: %s", translateBenchmarkStatus(run.Status))
	e.pdf.SetXY(22, currentY+3)
	e.pdf.CellFormat(166, 7, statusText, "", 1, "C", false, 0, "")

	e.pdf.SetFont("Arial", "", 10)
	e.pdf.SetXY(22, currentY+12)
	e.pdf.CellFormat(166, 6, fmt.Sprintf("Duracao: %s", formatDuration(run.StartedAt, run.EndedAt)), "", 1, "C", false, 0, "")

	e.pdf.SetY(currentY + 26)
	e.pdf.Ln(4)
}

func (e *PDFExporter) addBenchmarkGeneralInfo(projectName string, run *models.BenchmarkRun) {
	e.addSectionTitle("INFORMACOES GERAIS")

	leftX := 18
	rightX := 108
	startY := e.pdf.GetY()

	e.pdf.SetXY(float64(leftX), startY)
	e.addInfoCard("Projeto", removeAccents(projectName), 86)

	e.pdf.SetXY(float64(leftX), e.pdf.GetY())
	e.addInfoCard("Run ID", shortID(run.RunID), 86)

	e.pdf.SetXY(float64(leftX), e.pdf.GetY())
	e.addInfoCard("Status", translateBenchmarkStatus(run.Status), 86)

	e.pdf.SetXY(float64(rightX), startY)
	e.addInfoCard("Inicio", formatTimeFull(run.StartedAt), 86)

	e.pdf.SetXY(float64(rightX), e.pdf.GetY())
	e.addInfoCard("Termino", formatTimeFull(run.EndedAt), 86)

	e.pdf.SetXY(float64(rightX), e.pdf.GetY())
	e.addInfoCard("Duracao", formatDuration(run.StartedAt, run.EndedAt), 86)

	e.pdf.Ln(10)
}

func (e *PDFExporter) addBenchmarkScores(scores models.BenchmarkScores) {
	e.addSectionTitle("SCORES")

	startY := e.pdf.GetY()
	e.pdf.SetXY(18, startY)
	e.addInfoCard("Host ETL", formatScore(scores.HostETL), 56)

	e.pdf.SetXY(78, startY)
	e.addInfoCard("Origem", formatScore(scores.Origin), 56)

	e.pdf.SetXY(138, startY)
	e.addInfoCard("Destino", formatScore(scores.Destination), 56)

	e.pdf.SetY(startY + 22)
	e.pdf.Ln(6)
}

func (e *PDFExporter) addBenchmarkMetrics(metrics models.BenchmarkMetrics) {
	e.addSectionTitle("METRICAS")

	e.pdf.SetFont("Arial", "", 10)
	e.pdf.SetTextColor(reportTheme.gray700.r, reportTheme.gray700.g, reportTheme.gray700.b)

	if metrics.HostETL == nil {
		e.pdf.CellFormat(0, 6, "Host ETL: nao coletado", "", 1, "L", false, 0, "")
	} else {
		host := metrics.HostETL
		e.pdf.CellFormat(0, 6, fmt.Sprintf("Host ETL: CPU %d cores | Uso CPU %s | Memoria %s / %s",
			host.CPUCores,
			formatPercent(host.CPUUsagePct),
			formatBytes(host.MemUsedBytes),
			formatBytes(host.MemTotalBytes),
		), "", 1, "L", false, 0, "")

		if host.SwapTotalBytes > 0 {
			e.pdf.CellFormat(0, 6, fmt.Sprintf("Swap: %s / %s",
				formatBytes(host.SwapUsedBytes), formatBytes(host.SwapTotalBytes)), "", 1, "L", false, 0, "")
		}
		if host.DiskTotalBytes > 0 {
			e.pdf.CellFormat(0, 6, fmt.Sprintf("Disco: livre %s / total %s",
				formatBytes(host.DiskFreeBytes), formatBytes(host.DiskTotalBytes)), "", 1, "L", false, 0, "")
		}
	}

	e.pdf.Ln(4)

	if metrics.Origin == nil {
		e.pdf.CellFormat(0, 6, "Origem: nao coletado", "", 1, "L", false, 0, "")
	} else {
		e.addDBMetrics("Origem", metrics.Origin)
	}

	if metrics.Destination == nil {
		e.pdf.CellFormat(0, 6, "Destino: nao coletado", "", 1, "L", false, 0, "")
	} else {
		e.addDBMetrics("Destino", metrics.Destination)
	}
}

func (e *PDFExporter) addDBMetrics(label string, m *models.DBMetrics) {
	e.pdf.CellFormat(0, 6, fmt.Sprintf("%s: %s | Versao: %s", label, strings.ToUpper(m.DBType), valueOrNA(m.DBVersion)), "", 1, "L", false, 0, "")
	e.pdf.CellFormat(0, 6, fmt.Sprintf("Latencia conexao: %s | Ping: %s | QPS: %s",
		formatMillis(m.ConnLatencyMs),
		formatMillis(m.PingLatencyMs),
		formatFloat(m.ProbeQPS, 1),
	), "", 1, "L", false, 0, "")
	if m.WriteEnabled {
		e.pdf.CellFormat(0, 6, fmt.Sprintf("Write probe: %s", formatMillis(m.WriteLatencyMs)), "", 1, "L", false, 0, "")
	}
	if len(m.Errors) > 0 {
		e.pdf.CellFormat(0, 6, fmt.Sprintf("Erros: %s", strings.Join(m.Errors, "; ")), "", 1, "L", false, 0, "")
	}
	e.pdf.Ln(3)
}

func (e *PDFExporter) addBenchmarkHistoryHeader(projectName string, total int) {
	pageW, _ := e.pdf.GetPageSize()
	left, _, right, _ := e.pdf.GetMargins()
	bandH := 26.0
	e.pdf.SetFillColor(reportTheme.primary.r, reportTheme.primary.g, reportTheme.primary.b)
	e.pdf.Rect(0, 0, pageW, bandH, "F")

	e.pdf.SetTextColor(255, 255, 255)
	e.pdf.SetFont("Arial", "B", 18)
	e.pdf.SetXY(left, 6)
	e.pdf.CellFormat(pageW-left-right, 7, "HISTORICO DE BENCHMARKS", "", 1, "L", false, 0, "")

	if strings.TrimSpace(projectName) == "" {
		projectName = "Projeto"
	}
	e.pdf.SetFont("Arial", "", 10)
	e.pdf.SetXY(left, 14)
	e.pdf.CellFormat(pageW-left-right, 5, strings.ToUpper(removeAccents(projectName)), "", 1, "L", false, 0, "")

	e.pdf.SetFont("Arial", "", 9)
	e.pdf.SetXY(left, 20)
	e.pdf.CellFormat(pageW-left-right, 4, fmt.Sprintf("Total: %d | Gerado em: %s",
		total, time.Now().Format("02/01/2006 15:04:05")), "", 1, "L", false, 0, "")

	e.pdf.SetY(bandH + 6)
	e.pdf.SetDrawColor(reportTheme.gray200.r, reportTheme.gray200.g, reportTheme.gray200.b)
	e.pdf.SetLineWidth(0.4)
	e.pdf.Line(left, e.pdf.GetY(), pageW-right, e.pdf.GetY())
	e.pdf.Ln(6)
}

func (e *PDFExporter) addBenchmarkHistoryTable(summaries []models.BenchmarkSummary) {
	headers := []string{"RUN", "STATUS", "INICIO", "TERMINO", "DUR", "HOST", "ORIG", "DEST"}
	widths := []float64{28, 18, 30, 30, 14, 18, 18, 18}

	addTableHeader := func() {
		e.pdf.SetFont("Arial", "B", 9)
		e.pdf.SetFillColor(reportTheme.gray200.r, reportTheme.gray200.g, reportTheme.gray200.b)
		e.pdf.SetTextColor(reportTheme.gray900.r, reportTheme.gray900.g, reportTheme.gray900.b)
		for i, h := range headers {
			e.pdf.CellFormat(widths[i], 7, h, "1", 0, "C", true, 0, "")
		}
		e.pdf.Ln(-1)
	}

	addTableHeader()

	e.pdf.SetFont("Arial", "", 8)
	e.pdf.SetTextColor(reportTheme.gray700.r, reportTheme.gray700.g, reportTheme.gray700.b)

	for i, s := range summaries {
		if e.pdf.GetY() > 260 {
			e.pdf.AddPage()
			addTableHeader()
		}

		row := []string{
			shortID(s.RunID),
			translateBenchmarkStatus(s.Status),
			formatTimeShort(s.StartedAt),
			formatTimeShort(s.EndedAt),
			formatDuration(s.StartedAt, s.EndedAt),
			formatScore(s.Scores.HostETL),
			formatScore(s.Scores.Origin),
			formatScore(s.Scores.Destination),
		}

		for j, col := range row {
			align := "C"
			if j == 0 {
				align = "L"
			}
			e.pdf.CellFormat(widths[j], 6, col, "1", 0, align, false, 0, "")
		}
		e.pdf.Ln(-1)

		if i < len(summaries)-1 {
			e.pdf.SetDrawColor(reportTheme.gray200.r, reportTheme.gray200.g, reportTheme.gray200.b)
		}
	}
}

// -------------------- Benchmark PDF (UI layout) --------------------

func (e *PDFExporter) renderBenchmarkReportUI(projectName string, run *models.BenchmarkRun) {
	pageW, pageH := e.pdf.GetPageSize()

	e.pdf.SetFillColor(benchmarkTheme.bg.r, benchmarkTheme.bg.g, benchmarkTheme.bg.b)
	e.pdf.Rect(0, 0, pageW, pageH, "F")

	margin := 12.0
	gap := 6.0
	topY := 12.0
	topH := 72.0
	leftW := 112.0
	rightW := pageW - (2 * margin) - gap - leftW

	e.drawCard(margin, topY, leftW, topH, benchmarkTheme.card)
	e.drawCard(margin+leftW+gap, topY, rightW, topH, benchmarkTheme.cardAlt)

	e.addBenchmarkSummaryCard(margin, topY, leftW, topH, projectName, run)
	e.addBenchmarkQuickIndicators(margin+leftW+gap, topY, rightW, topH, run)

	sectionY := topY + topH + 10
	e.addBenchmarkSectionHeader(margin, sectionY, pageW-(2*margin), run.Status)

	cardY := sectionY + 18
	cardH := 120.0
	cardW := (pageW - (2 * margin) - 2*gap) / 3

	e.drawCard(margin, cardY, cardW, cardH, benchmarkTheme.card)
	e.drawCard(margin+cardW+gap, cardY, cardW, cardH, benchmarkTheme.card)
	e.drawCard(margin+(cardW+gap)*2, cardY, cardW, cardH, benchmarkTheme.card)

	e.addBenchmarkHostCard(margin, cardY, cardW, cardH, run)
	e.addBenchmarkOriginCard(margin+cardW+gap, cardY, cardW, cardH, run)
	e.addBenchmarkDestinationCard(margin+(cardW+gap)*2, cardY, cardW, cardH, run)
}

func (e *PDFExporter) drawCard(x, y, w, h float64, fill color) {
	e.pdf.SetFillColor(fill.r, fill.g, fill.b)
	e.pdf.SetDrawColor(benchmarkTheme.cardBorder.r, benchmarkTheme.cardBorder.g, benchmarkTheme.cardBorder.b)
	e.pdf.SetLineWidth(0.4)
	e.pdf.RoundedRect(x, y, w, h, 3, "1234", "FD")
}

func (e *PDFExporter) addBenchmarkSummaryCard(x, y, w, h float64, projectName string, run *models.BenchmarkRun) {
	padding := 6.0
	curX := x + padding
	curY := y + padding

	statusText, statusColor := benchmarkStatusLabel(run.Status)
	e.drawPill(x+w-padding-24, y+padding-1, 24, 6, statusText, statusColor)

	e.pdf.SetTextColor(benchmarkTheme.textMuted.r, benchmarkTheme.textMuted.g, benchmarkTheme.textMuted.b)
	e.pdf.SetFont("Arial", "B", 7)
	e.pdf.SetXY(curX, curY)
	e.pdf.CellFormat(0, 4, "ULTIMO BENCHMARK", "", 1, "L", false, 0, "")

	curY += 4
	e.pdf.SetTextColor(benchmarkTheme.textMain.r, benchmarkTheme.textMain.g, benchmarkTheme.textMain.b)
	e.pdf.SetFont("Arial", "B", 12)
	e.pdf.SetXY(curX, curY)
	e.pdf.CellFormat(0, 5, removeAccents("Resumo da execução"), "", 1, "L", false, 0, "")

	curY += 5
	e.pdf.SetFont("Arial", "", 8)
	e.pdf.SetTextColor(benchmarkTheme.textMuted.r, benchmarkTheme.textMuted.g, benchmarkTheme.textMuted.b)
	e.pdf.SetXY(curX, curY)
	e.pdf.CellFormat(0, 4, removeAccents("Acompanhe a saúde geral da infraestrutura."), "", 1, "L", false, 0, "")

	infoY := curY + 6
	colW := (w - (padding * 2)) / 3
	e.drawLabelValue(curX, infoY, colW, "INICIO", formatTimeFull(run.StartedAt))
	e.drawLabelValue(curX+colW, infoY, colW, "FIM", formatTimeFull(run.EndedAt))
	e.drawLabelValue(curX+(colW*2), infoY, colW, "DURACAO", formatDurationShort(run.StartedAt, run.EndedAt))

	runY := infoY + 10
	e.drawLabelValue(curX, runY, w-(padding*2), "RUN ID", shortID(run.RunID))

	scoreY := y + h - 26
	scoreGap := 4.0
	scoreW := (w - (padding * 2) - (scoreGap * 2)) / 3
	e.drawScoreChip(curX, scoreY, scoreW, 14, "HOST ETL", run.Scores.HostETL)
	e.drawScoreChip(curX+scoreW+scoreGap, scoreY, scoreW, 14, "ORIGEM", run.Scores.Origin)
	e.drawScoreChip(curX+(scoreW+scoreGap)*2, scoreY, scoreW, 14, "DESTINO", run.Scores.Destination)

	pillsY := y + h - 9
	pills := benchmarkPills(run)
	e.drawPills(curX, pillsY, w-(padding*2), 5, pills)
}

func (e *PDFExporter) addBenchmarkQuickIndicators(x, y, w, h float64, run *models.BenchmarkRun) {
	padding := 6.0
	curX := x + padding
	curY := y + padding

	e.pdf.SetTextColor(benchmarkTheme.textMuted.r, benchmarkTheme.textMuted.g, benchmarkTheme.textMuted.b)
	e.pdf.SetFont("Arial", "B", 7)
	e.pdf.SetXY(curX, curY)
	e.pdf.CellFormat(0, 4, "VISAO GERAL", "", 1, "L", false, 0, "")

	curY += 4
	e.pdf.SetTextColor(benchmarkTheme.textMain.r, benchmarkTheme.textMain.g, benchmarkTheme.textMain.b)
	e.pdf.SetFont("Arial", "B", 11)
	e.pdf.SetXY(curX, curY)
	e.pdf.CellFormat(0, 5, removeAccents("Indicadores rápidos"), "", 1, "L", false, 0, "")

	curY += 5
	e.pdf.SetFont("Arial", "", 8)
	e.pdf.SetTextColor(benchmarkTheme.textMuted.r, benchmarkTheme.textMuted.g, benchmarkTheme.textMuted.b)
	e.pdf.SetXY(curX, curY)
	e.pdf.CellFormat(0, 4, removeAccents("Resumo dos principais sinais de saúde."), "", 1, "L", false, 0, "")

	rowY := curY + 8
	colGap := 4.0
	colW := (w - (padding * 2) - (colGap * 2)) / 3

	host := run.Metrics.HostETL
	origin := run.Metrics.Origin
	dest := run.Metrics.Destination

	e.drawIndicator(curX, rowY, colW, "CPU", formatPercent(valueOrZeroFloat(host, func(m *models.HostMetrics) float64 { return m.CPUUsagePct })))
	e.drawIndicator(curX+colW+colGap, rowY, colW, "MEMORIA", formatBytesShort(valueOrZeroUint64(host, func(m *models.HostMetrics) uint64 { return m.MemUsedBytes })))
	e.drawIndicator(curX+(colW+colGap)*2, rowY, colW, "ORIGEM", formatDBType(origin))

	rowY += 16
	e.drawIndicator(curX, rowY, colW, "LATENCIA ORIGEM", formatMillis(averageDBLatency(origin)))
	e.drawIndicator(curX+colW+colGap, rowY, colW, "DESTINO", formatDBType(dest))
	e.drawIndicator(curX+(colW+colGap)*2, rowY, colW, "LATENCIA DESTINO", formatMillis(averageDBLatency(dest)))
}

func (e *PDFExporter) addBenchmarkSectionHeader(x, y, w float64, status string) {
	e.pdf.SetTextColor(benchmarkTheme.textMuted.r, benchmarkTheme.textMuted.g, benchmarkTheme.textMuted.b)
	e.pdf.SetFont("Arial", "B", 7)
	e.pdf.SetXY(x, y)
	e.pdf.CellFormat(0, 4, "BENCHMARK SELECIONADO", "", 1, "L", false, 0, "")

	y += 4
	e.pdf.SetFont("Arial", "B", 13)
	e.pdf.SetTextColor(benchmarkTheme.textMain.r, benchmarkTheme.textMain.g, benchmarkTheme.textMain.b)
	e.pdf.SetXY(x, y)
	e.pdf.CellFormat(0, 6, removeAccents("Detalhes do histórico"), "", 1, "L", false, 0, "")

	y += 6
	e.pdf.SetFont("Arial", "", 8)
	e.pdf.SetTextColor(benchmarkTheme.textMuted.r, benchmarkTheme.textMuted.g, benchmarkTheme.textMuted.b)
	e.pdf.SetXY(x, y)
	e.pdf.CellFormat(0, 4, removeAccents("Métricas completas para o benchmark escolhido."), "", 1, "L", false, 0, "")

	statusText, statusColor := benchmarkStatusLabel(status)
	e.drawPill(x+w-26, y-8, 24, 6, statusText, statusColor)
}

func (e *PDFExporter) addBenchmarkHostCard(x, y, w, h float64, run *models.BenchmarkRun) {
	padding := 6.0
	curX := x + padding
	curY := y + padding

	scoreLabel, scoreColor := scoreLabel(run.Scores.HostETL)
	e.drawScorePill(x+w-padding-24, y+padding-1, 22, 6, run.Scores.HostETL, scoreLabel, scoreColor)

	e.pdf.SetTextColor(benchmarkTheme.textMain.r, benchmarkTheme.textMain.g, benchmarkTheme.textMain.b)
	e.pdf.SetFont("Arial", "B", 10)
	e.pdf.SetXY(curX, curY)
	e.pdf.CellFormat(0, 5, "Host ETL", "", 1, "L", false, 0, "")

	curY += 5
	e.pdf.SetFont("Arial", "", 8)
	e.pdf.SetTextColor(benchmarkTheme.textMuted.r, benchmarkTheme.textMuted.g, benchmarkTheme.textMuted.b)
	e.pdf.SetXY(curX, curY)
	e.pdf.CellFormat(0, 4, removeAccents("Infraestrutura local de execução."), "", 1, "L", false, 0, "")

	curY += 8
	if run.Metrics.HostETL == nil {
		e.drawCardLine(curX, curY, w-(padding*2), "Nao coletado", "")
		return
	}
	host := run.Metrics.HostETL
	e.drawCardLine(curX, curY, w-(padding*2), "Cores", fmt.Sprintf("%d", host.CPUCores))
	curY += 10
	e.drawCardLine(curX, curY, w-(padding*2), "CPU", formatPercent(host.CPUUsagePct))
	curY += 10
	e.drawCardLine(curX, curY, w-(padding*2), "Memoria", formatBytesShort(host.MemUsedBytes)+"/"+formatBytesShort(host.MemTotalBytes))
	curY += 10
	if host.SwapTotalBytes > 0 {
		e.drawCardLine(curX, curY, w-(padding*2), "Swap", formatBytesShort(host.SwapUsedBytes)+"/"+formatBytesShort(host.SwapTotalBytes))
		curY += 10
	}
	if host.DiskTotalBytes > 0 {
		e.drawCardLine(curX, curY, w-(padding*2), "Disco", formatBytesShort(host.DiskFreeBytes)+" livre de "+formatBytesShort(host.DiskTotalBytes))
	}
}

func (e *PDFExporter) addBenchmarkOriginCard(x, y, w, h float64, run *models.BenchmarkRun) {
	padding := 6.0
	curX := x + padding
	curY := y + padding

	scoreLabel, scoreColor := scoreLabel(run.Scores.Origin)
	e.drawScorePill(x+w-padding-24, y+padding-1, 22, 6, run.Scores.Origin, scoreLabel, scoreColor)

	e.pdf.SetTextColor(benchmarkTheme.textMain.r, benchmarkTheme.textMain.g, benchmarkTheme.textMain.b)
	e.pdf.SetFont("Arial", "B", 10)
	e.pdf.SetXY(curX, curY)
	e.pdf.CellFormat(0, 5, "Origem", "", 1, "L", false, 0, "")

	curY += 5
	e.pdf.SetFont("Arial", "", 8)
	e.pdf.SetTextColor(benchmarkTheme.textMuted.r, benchmarkTheme.textMuted.g, benchmarkTheme.textMuted.b)
	e.pdf.SetXY(curX, curY)
	e.pdf.CellFormat(0, 4, removeAccents("Banco de dados de origem."), "", 1, "L", false, 0, "")

	curY += 8
	if run.Metrics.Origin == nil {
		e.drawCardLine(curX, curY, w-(padding*2), "Nao coletado", "")
		return
	}
	e.drawDBCard(curX, curY, w-(padding*2), run.Metrics.Origin)
}

func (e *PDFExporter) addBenchmarkDestinationCard(x, y, w, h float64, run *models.BenchmarkRun) {
	padding := 6.0
	curX := x + padding
	curY := y + padding

	scoreLabel, scoreColor := scoreLabel(run.Scores.Destination)
	e.drawScorePill(x+w-padding-24, y+padding-1, 22, 6, run.Scores.Destination, scoreLabel, scoreColor)

	e.pdf.SetTextColor(benchmarkTheme.textMain.r, benchmarkTheme.textMain.g, benchmarkTheme.textMain.b)
	e.pdf.SetFont("Arial", "B", 10)
	e.pdf.SetXY(curX, curY)
	e.pdf.CellFormat(0, 5, "Destino", "", 1, "L", false, 0, "")

	curY += 5
	e.pdf.SetFont("Arial", "", 8)
	e.pdf.SetTextColor(benchmarkTheme.textMuted.r, benchmarkTheme.textMuted.g, benchmarkTheme.textMuted.b)
	e.pdf.SetXY(curX, curY)
	e.pdf.CellFormat(0, 4, removeAccents("Banco de dados de destino."), "", 1, "L", false, 0, "")

	curY += 8
	if run.Metrics.Destination == nil {
		e.drawCardLine(curX, curY, w-(padding*2), "Nao coletado", "")
		return
	}
	e.drawDBCard(curX, curY, w-(padding*2), run.Metrics.Destination)
}

func (e *PDFExporter) drawLabelValue(x, y, w float64, label, value string) {
	e.pdf.SetFont("Arial", "B", 7)
	e.pdf.SetTextColor(benchmarkTheme.textMuted.r, benchmarkTheme.textMuted.g, benchmarkTheme.textMuted.b)
	e.pdf.SetXY(x, y)
	e.pdf.CellFormat(w, 4, removeAccents(label), "", 0, "L", false, 0, "")

	e.pdf.SetFont("Arial", "B", 9)
	e.pdf.SetTextColor(benchmarkTheme.textMain.r, benchmarkTheme.textMain.g, benchmarkTheme.textMain.b)
	e.pdf.SetXY(x, y+4)
	e.pdf.CellFormat(w, 4, removeAccents(value), "", 0, "L", false, 0, "")
}

func (e *PDFExporter) drawIndicator(x, y, w float64, label, value string) {
	e.pdf.SetFont("Arial", "B", 7)
	e.pdf.SetTextColor(benchmarkTheme.textMuted.r, benchmarkTheme.textMuted.g, benchmarkTheme.textMuted.b)
	e.pdf.SetXY(x, y)
	e.pdf.CellFormat(w, 4, removeAccents(label), "", 0, "L", false, 0, "")

	e.pdf.SetFont("Arial", "B", 9)
	e.pdf.SetTextColor(benchmarkTheme.textMain.r, benchmarkTheme.textMain.g, benchmarkTheme.textMain.b)
	e.pdf.SetXY(x, y+4)
	e.pdf.CellFormat(w, 5, removeAccents(value), "", 0, "L", false, 0, "")
}

func (e *PDFExporter) drawScoreChip(x, y, w, h float64, label string, score float64) {
	status, statusColor := scoreLabel(score)
	e.pdf.SetFillColor(benchmarkTheme.cardAlt.r, benchmarkTheme.cardAlt.g, benchmarkTheme.cardAlt.b)
	e.pdf.SetDrawColor(statusColor.r, statusColor.g, statusColor.b)
	e.pdf.RoundedRect(x, y, w, h, 2.5, "1234", "FD")

	e.pdf.SetTextColor(benchmarkTheme.textMuted.r, benchmarkTheme.textMuted.g, benchmarkTheme.textMuted.b)
	e.pdf.SetFont("Arial", "B", 6)
	e.pdf.SetXY(x+3, y+2)
	e.pdf.CellFormat(w-6, 3, removeAccents(label), "", 0, "L", false, 0, "")

	e.pdf.SetTextColor(statusColor.r, statusColor.g, statusColor.b)
	e.pdf.SetFont("Arial", "B", 10)
	e.pdf.SetXY(x+3, y+6)
	e.pdf.CellFormat(w-6, 4, formatScore(score), "", 0, "L", false, 0, "")

	e.pdf.SetTextColor(benchmarkTheme.textMuted.r, benchmarkTheme.textMuted.g, benchmarkTheme.textMuted.b)
	e.pdf.SetFont("Arial", "", 6)
	e.pdf.SetXY(x+3, y+10)
	e.pdf.CellFormat(w-6, 3, status, "", 0, "L", false, 0, "")
}

func (e *PDFExporter) drawPills(x, y, w, h float64, pills []string) {
	curX := x
	for _, pill := range pills {
		pillW := e.pdf.GetStringWidth(pill) + 6
		if curX+pillW > x+w {
			break
		}
		e.drawPill(curX, y, pillW, h, pill, benchmarkTheme.cardBorder)
		curX += pillW + 3
	}
}

func (e *PDFExporter) drawPill(x, y, w, h float64, text string, border color) {
	e.pdf.SetFillColor(benchmarkTheme.bg.r, benchmarkTheme.bg.g, benchmarkTheme.bg.b)
	e.pdf.SetDrawColor(border.r, border.g, border.b)
	e.pdf.RoundedRect(x, y, w, h, 2.5, "1234", "FD")
	e.pdf.SetTextColor(benchmarkTheme.textMain.r, benchmarkTheme.textMain.g, benchmarkTheme.textMain.b)
	e.pdf.SetFont("Arial", "B", 6)
	e.pdf.SetXY(x+2.5, y+1.3)
	e.pdf.CellFormat(w-4, h-2, removeAccents(text), "", 0, "L", false, 0, "")
}

func (e *PDFExporter) drawScorePill(x, y, w, h float64, score float64, label string, fill color) {
	e.pdf.SetFillColor(fill.r, fill.g, fill.b)
	e.pdf.SetDrawColor(fill.r, fill.g, fill.b)
	e.pdf.RoundedRect(x, y, w, h, 2.5, "1234", "FD")
	e.pdf.SetTextColor(255, 255, 255)
	e.pdf.SetFont("Arial", "B", 6)
	e.pdf.SetXY(x+1.5, y+0.8)
	e.pdf.CellFormat(w-3, 3, formatScore(score), "", 0, "C", false, 0, "")
	e.pdf.SetFont("Arial", "", 5.5)
	e.pdf.SetXY(x+1.5, y+3.0)
	e.pdf.CellFormat(w-3, 3, label, "", 0, "C", false, 0, "")
}

func (e *PDFExporter) drawCardLine(x, y, w float64, label, value string) {
	e.pdf.SetFont("Arial", "B", 7)
	e.pdf.SetTextColor(benchmarkTheme.textMuted.r, benchmarkTheme.textMuted.g, benchmarkTheme.textMuted.b)
	e.pdf.SetXY(x, y)
	e.pdf.CellFormat(w, 4, removeAccents(label), "", 1, "L", false, 0, "")

	e.pdf.SetFont("Arial", "B", 8.5)
	e.pdf.SetTextColor(benchmarkTheme.textMain.r, benchmarkTheme.textMain.g, benchmarkTheme.textMain.b)
	e.pdf.SetXY(x, y+4)
	e.pdf.CellFormat(w, 4, removeAccents(value), "", 1, "L", false, 0, "")
}

func (e *PDFExporter) drawDBCard(x, y, w float64, db *models.DBMetrics) {
	curY := y
	e.drawCardLine(x, curY, w, "Tipo", strings.ToLower(db.DBType))
	curY += 10

	e.pdf.SetFont("Arial", "B", 7)
	e.pdf.SetTextColor(benchmarkTheme.textMuted.r, benchmarkTheme.textMuted.g, benchmarkTheme.textMuted.b)
	e.pdf.SetXY(x, curY)
	e.pdf.CellFormat(w, 4, "Versao", "", 1, "L", false, 0, "")

	e.pdf.SetFont("Arial", "B", 7.5)
	e.pdf.SetTextColor(benchmarkTheme.textMain.r, benchmarkTheme.textMain.g, benchmarkTheme.textMain.b)
	e.pdf.SetXY(x, curY+4)
	e.pdf.MultiCell(w, 4, removeAccents(valueOrNA(db.DBVersion)), "", "L", false)
	curY = e.pdf.GetY() + 2

	e.drawCardLine(x, curY, w, "Conexao", formatMillis(db.ConnLatencyMs))
	curY += 10
	e.drawCardLine(x, curY, w, "Ping", formatMillis(db.PingLatencyMs))
	curY += 10
	e.drawCardLine(x, curY, w, "Probe", formatFloat(db.ProbeQPS, 1)+" qps")
	curY += 10
	writeStatus := "Inativo"
	if db.WriteEnabled {
		writeStatus = "Ativo"
	}
	e.drawCardLine(x, curY, w, "Write", writeStatus)
}

func benchmarkPills(run *models.BenchmarkRun) []string {
	pills := []string{
		fmt.Sprintf("Iteracoes: %d", run.Options.ProbeIterations),
		formatToggle("Host", run.Options.IncludeHost),
		formatToggle("Origem", run.Options.IncludeOrigin),
		formatToggle("Destino", run.Options.IncludeDestination),
	}
	writeStatus := "Inativo"
	if run.Options.EnableWriteProbe {
		writeStatus = "Ativo"
	}
	pills = append(pills, "Write probe: "+writeStatus)
	return pills
}

func formatToggle(label string, enabled bool) string {
	if enabled {
		return label + ": On"
	}
	return label + ": Off"
}

func benchmarkStatusLabel(status string) (string, color) {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "ok":
		return "SUCESSO", benchmarkTheme.green
	case "partial":
		return "PARCIAL", benchmarkTheme.orange
	case "error":
		return "ERRO", benchmarkTheme.red
	default:
		return strings.ToUpper(status), benchmarkTheme.cardBorder
	}
}

func scoreLabel(score float64) (string, color) {
	switch {
	case score >= 80:
		return "SAUDAVEL", benchmarkTheme.green
	case score >= 60:
		return "MEDIO", benchmarkTheme.orange
	case score > 0:
		return "CRITICO", benchmarkTheme.red
	default:
		return "-", benchmarkTheme.cardBorder
	}
}

func averageDBLatency(db *models.DBMetrics) float64 {
	if db == nil {
		return 0
	}
	if db.ConnLatencyMs > 0 && db.PingLatencyMs > 0 {
		return (db.ConnLatencyMs + db.PingLatencyMs) / 2
	}
	if db.ConnLatencyMs > 0 {
		return db.ConnLatencyMs
	}
	return db.PingLatencyMs
}

func formatDurationShort(start, end time.Time) string {
	if start.IsZero() || end.IsZero() || end.Before(start) {
		return "-"
	}
	dur := end.Sub(start)
	if dur < time.Minute {
		return fmt.Sprintf("%ds", int(dur.Seconds()))
	}
	if dur < time.Hour {
		return fmt.Sprintf("%dm%ds", int(dur.Minutes()), int(dur.Seconds())%60)
	}
	return formatDuration(start, end)
}

func formatDBType(db *models.DBMetrics) string {
	if db == nil {
		return "n/a"
	}
	if strings.TrimSpace(db.DBType) == "" {
		return "n/a"
	}
	return strings.ToLower(db.DBType)
}

func valueOrZeroFloat(m *models.HostMetrics, getter func(*models.HostMetrics) float64) float64 {
	if m == nil {
		return 0
	}
	return getter(m)
}

func valueOrZeroUint64(m *models.HostMetrics, getter func(*models.HostMetrics) uint64) uint64 {
	if m == nil {
		return 0
	}
	return getter(m)
}

// -------------------- Storage --------------------

func benchmarkBaseDir(projectID string) string {
	return filepath.Join("logs", "benchmarks", projectID)
}

func benchmarkFilePath(projectID, runID string) string {
	return filepath.Join(benchmarkBaseDir(projectID), fmt.Sprintf("benchmark_%s.json", runID))
}

func loadBenchmarkRun(projectID, runID string) (*models.BenchmarkRun, error) {
	path := benchmarkFilePath(projectID, runID)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var run models.BenchmarkRun
	if err := json.Unmarshal(data, &run); err != nil {
		return nil, err
	}
	return &run, nil
}

func listBenchmarkSummaries(projectID string, limit int) ([]models.BenchmarkSummary, error) {
	baseDir := benchmarkBaseDir(projectID)
	entries, err := os.ReadDir(baseDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []models.BenchmarkSummary{}, nil
		}
		return nil, err
	}

	var summaries []models.BenchmarkSummary
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		path := filepath.Join(baseDir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var run models.BenchmarkRun
		if err := json.Unmarshal(data, &run); err != nil {
			continue
		}
		summaries = append(summaries, models.BenchmarkSummary{
			RunID:     run.RunID,
			Status:    run.Status,
			StartedAt: run.StartedAt,
			EndedAt:   run.EndedAt,
			Scores:    run.Scores,
		})
	}

	sort.Slice(summaries, func(i, j int) bool {
		return summaries[i].StartedAt.After(summaries[j].StartedAt)
	})

	if limit > 0 && len(summaries) > limit {
		return summaries[:limit], nil
	}
	return summaries, nil
}

func parseBenchmarkLimit(raw string) int {
	if raw == "" {
		return 0
	}
	val, err := strconv.Atoi(raw)
	if err != nil || val <= 0 {
		return 0
	}
	return val
}

func loadProjectName(projectID string) string {
	path := filepath.Join("data", "projects", projectID, "project.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return projectID
	}
	var payload struct {
		ProjectName string `json:"projectName"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return projectID
	}
	if strings.TrimSpace(payload.ProjectName) == "" {
		return projectID
	}
	return payload.ProjectName
}

// -------------------- Formatting --------------------

func translateBenchmarkStatus(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "ok":
		return "OK"
	case "partial":
		return "PARCIAL"
	case "error":
		return "ERRO"
	default:
		return strings.ToUpper(status)
	}
}

func benchmarkStatusColor(status string) color {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "ok":
		return reportTheme.success
	case "partial":
		return reportTheme.warning
	case "error":
		return reportTheme.danger
	default:
		return reportTheme.gray500
	}
}

func formatDuration(start, end time.Time) string {
	if start.IsZero() || end.IsZero() || end.Before(start) {
		return "-"
	}
	dur := end.Sub(start)
	hours := int(dur.Hours())
	minutes := int(dur.Minutes()) % 60
	seconds := int(dur.Seconds()) % 60
	return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds)
}

func formatTimeShort(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	return t.Format("02/01 15:04")
}

func formatTimeFull(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	return t.Format("02/01/2006 15:04:05")
}

func formatScore(score float64) string {
	if score <= 0 {
		return "-"
	}
	return fmt.Sprintf("%.1f", score)
}

func formatPercent(pct float64) string {
	if pct <= 0 {
		return "n/a"
	}
	return fmt.Sprintf("%.1f%%", pct)
}

func formatMillis(ms float64) string {
	if ms <= 0 {
		return "n/a"
	}
	return fmt.Sprintf("%.0f ms", ms)
}

func formatFloat(val float64, decimals int) string {
	if val <= 0 {
		return "n/a"
	}
	format := fmt.Sprintf("%%.%df", decimals)
	return fmt.Sprintf(format, val)
}

func valueOrNA(val string) string {
	if strings.TrimSpace(val) == "" {
		return "n/a"
	}
	return val
}

func formatBytes(bytes uint64) string {
	if bytes == 0 {
		return "n/a"
	}
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := uint64(unit), 0
	for n := bytes / unit; n >= unit && exp < 4; n /= unit {
		div *= unit
		exp++
	}
	value := float64(bytes) / float64(div)
	suffix := []string{"KB", "MB", "GB", "TB", "PB"}[exp]
	return fmt.Sprintf("%.1f %s", value, suffix)
}

func formatBytesShort(bytes uint64) string {
	if bytes == 0 {
		return "n/a"
	}
	raw := formatBytes(bytes)
	return strings.Replace(raw, ".0 ", " ", 1)
}

func shortID(id string) string {
	id = strings.TrimSpace(id)
	if len(id) <= 12 {
		return id
	}
	return id[:12]
}
