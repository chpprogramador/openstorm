package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"etl/models"
	"fmt"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql" // MySQL
	"github.com/google/uuid"
	_ "github.com/lib/pq" // Postgres
)

const (
	defaultProbeIterations = 5
	maxProbeIterations     = 100
	defaultProbeTimeout    = 5 * time.Second
)

var benchmarkFileMu sync.Mutex

// -------------------- Run Benchmark --------------------
func RunBenchmark(c *gin.Context) {
	projectID := c.Param("id")

	var req models.BenchmarkRunRequest
	_ = c.ShouldBindJSON(&req)

	opts := normalizeBenchmarkOptions(req)

	projectPath := filepath.Join("data", "projects", projectID, "project.json")
	project, err := loadProjectFile(projectPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao ler project.json"})
		return
	}

	decryptProjectFields(&project)

	run := models.BenchmarkRun{
		RunID:     uuid.New().String(),
		ProjectID: projectID,
		Status:    "running",
		StartedAt: time.Now().UTC(),
		Options: models.BenchmarkOptions{
			ProbeIterations:    opts.ProbeIterations,
			EnableWriteProbe:   derefBool(opts.EnableWriteProbe),
			IncludeHost:        opts.IncludeHost,
			IncludeOrigin:      opts.IncludeOrigin,
			IncludeDestination: opts.IncludeDestination,
		},
	}

	var targetCount int
	var failedTargets int

	metrics := models.BenchmarkMetrics{}

	if opts.IncludeHost {
		targetCount++
		hostMetrics, hostErr := collectHostMetrics()
		if hostErr != nil {
			failedTargets++
			run.Error = appendError(run.Error, fmt.Sprintf("host_etl: %s", hostErr.Error()))
		} else {
			metrics.HostETL = hostMetrics
		}
	}

	if opts.IncludeOrigin {
		targetCount++
		originMetrics, ok := collectDBMetrics(project.SourceDatabase, opts.ProbeIterations, derefBool(opts.EnableWriteProbe))
		if !ok {
			failedTargets++
			run.Error = appendError(run.Error, fmt.Sprintf("origin: %s", strings.Join(originMetrics.Errors, "; ")))
		}
		metrics.Origin = originMetrics
	}

	if opts.IncludeDestination {
		targetCount++
		destMetrics, ok := collectDBMetrics(project.DestinationDatabase, opts.ProbeIterations, derefBool(opts.EnableWriteProbe))
		if !ok {
			failedTargets++
			run.Error = appendError(run.Error, fmt.Sprintf("destination: %s", strings.Join(destMetrics.Errors, "; ")))
		}
		metrics.Destination = destMetrics
	}

	run.Metrics = metrics
	run.Scores = computeBenchmarkScores(metrics)
	run.EndedAt = time.Now().UTC()

	switch {
	case targetCount == 0:
		run.Status = "error"
		run.Error = appendError(run.Error, "nenhum alvo selecionado")
	case failedTargets == 0:
		run.Status = "ok"
	case failedTargets == targetCount:
		run.Status = "error"
	default:
		run.Status = "partial"
	}

	if err := saveBenchmarkRun(projectID, &run); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Erro ao salvar benchmark: %v", err)})
		return
	}

	c.JSON(http.StatusOK, run)
}

// -------------------- List Benchmarks --------------------
func ListBenchmarks(c *gin.Context) {
	projectID := c.Param("id")
	limit := parseLimit(c.Query("limit"))

	summaries, err := listBenchmarkRuns(projectID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Erro ao listar benchmarks: %v", err)})
		return
	}
	c.JSON(http.StatusOK, summaries)
}

// -------------------- Get Benchmark by Run ID --------------------
func GetBenchmark(c *gin.Context) {
	projectID := c.Param("id")
	runID := c.Param("runId")

	run, err := loadBenchmarkRun(projectID, runID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Benchmark não encontrado"})
		return
	}

	c.JSON(http.StatusOK, run)
}

// -------------------- Options --------------------
func normalizeBenchmarkOptions(req models.BenchmarkRunRequest) models.BenchmarkRunRequest {
	opts := req
	if opts.ProbeIterations <= 0 {
		opts.ProbeIterations = defaultProbeIterations
	}
	if opts.ProbeIterations > maxProbeIterations {
		opts.ProbeIterations = maxProbeIterations
	}
	if opts.EnableWriteProbe == nil {
		defaultWrite := true
		opts.EnableWriteProbe = &defaultWrite
	}
	if !opts.IncludeHost && !opts.IncludeOrigin && !opts.IncludeDestination {
		opts.IncludeHost = true
		opts.IncludeOrigin = true
		opts.IncludeDestination = true
	}
	return opts
}

// -------------------- DB Metrics --------------------

type dbProbeSpec struct {
	VersionQuery  string
	PingQuery     string
	WriteSetup    []string
	WriteProbe    string
	WriteTeardown []string
}

func collectDBMetrics(cfg models.DatabaseConfig, probeIterations int, enableWrite bool) (*models.DBMetrics, bool) {
	metrics := &models.DBMetrics{DBType: cfg.Type, WriteEnabled: false}

	spec, err := getDBProbeSpec(cfg.Type)
	if err != nil {
		metrics.Errors = append(metrics.Errors, err.Error())
		return metrics, false
	}

	dsn := buildDSN(cfg)
	if dsn == "" {
		metrics.Errors = append(metrics.Errors, "DSN inválido")
		return metrics, false
	}

	db, err := sql.Open(cfg.Type, dsn)
	if err != nil {
		metrics.Errors = append(metrics.Errors, fmt.Sprintf("erro ao abrir conexão: %v", err))
		return metrics, false
	}
	defer db.Close()

	connLatency, err := measureDBPing(db)
	if err != nil {
		metrics.Errors = append(metrics.Errors, fmt.Sprintf("erro no ping: %v", err))
		return metrics, false
	}
	metrics.ConnLatencyMs = connLatency

	if spec.VersionQuery != "" {
		if version, err := queryString(db, spec.VersionQuery); err == nil {
			metrics.DBVersion = version
		} else {
			metrics.Errors = append(metrics.Errors, fmt.Sprintf("erro ao obter versão: %v", err))
		}
	}

	pingLatency, err := measureQueryLatency(db, spec.PingQuery)
	if err == nil {
		metrics.PingLatencyMs = pingLatency
	} else {
		metrics.Errors = append(metrics.Errors, fmt.Sprintf("erro no ping query: %v", err))
	}

	if probeIterations > 0 {
		qps, err := measureQueryQPS(db, spec.PingQuery, probeIterations)
		if err == nil {
			metrics.ProbeIterations = probeIterations
			metrics.ProbeQPS = qps
		} else {
			metrics.Errors = append(metrics.Errors, fmt.Sprintf("erro no probe: %v", err))
		}
	}

	if enableWrite && spec.WriteProbe != "" {
		readOnly, roErr := isDBReadOnly(db, cfg.Type)
		if roErr != nil {
			metrics.Errors = append(metrics.Errors, fmt.Sprintf("erro ao checar modo somente leitura: %v", roErr))
		}
		canWrite := true
		if strings.ToLower(cfg.Type) == "postgres" {
			var err error
			canWrite, err = canWritePostgres(db)
			if err != nil {
				metrics.Errors = append(metrics.Errors, fmt.Sprintf("erro ao checar privilegios de escrita: %v", err))
			}
		}
		if !readOnly && canWrite {
			if latency, err := runWriteProbe(db, spec); err == nil {
				metrics.WriteLatencyMs = latency
				metrics.WriteEnabled = true
			} else {
				metrics.Errors = append(metrics.Errors, fmt.Sprintf("erro no write probe: %v", err))
			}
		}
	}

	return metrics, true
}

func getDBProbeSpec(dbType string) (dbProbeSpec, error) {
	switch strings.ToLower(dbType) {
	case "postgres":
		return dbProbeSpec{
			VersionQuery:  "SELECT version()",
			PingQuery:     "SELECT 1",
			WriteSetup:    []string{"CREATE TEMP TABLE IF NOT EXISTS etl_bench_probe (id INT)"},
			WriteProbe:    "INSERT INTO etl_bench_probe (id) VALUES (1)",
			WriteTeardown: []string{"DROP TABLE IF EXISTS etl_bench_probe"},
		}, nil
	case "mysql":
		return dbProbeSpec{
			VersionQuery:  "SELECT VERSION()",
			PingQuery:     "SELECT 1",
			WriteSetup:    []string{"CREATE TEMPORARY TABLE IF NOT EXISTS etl_bench_probe (id INT)"},
			WriteProbe:    "INSERT INTO etl_bench_probe (id) VALUES (1)",
			WriteTeardown: []string{"DROP TEMPORARY TABLE IF EXISTS etl_bench_probe"},
		}, nil
	default:
		return dbProbeSpec{}, fmt.Errorf("dialeto não suportado para benchmark: %s", dbType)
	}
}

func measureDBPing(db *sql.DB) (float64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultProbeTimeout)
	defer cancel()
	start := time.Now()
	if err := db.PingContext(ctx); err != nil {
		return 0, err
	}
	return float64(time.Since(start).Milliseconds()), nil
}

func measureQueryLatency(db *sql.DB, query string) (float64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultProbeTimeout)
	defer cancel()
	start := time.Now()
	row := db.QueryRowContext(ctx, query)
	var tmp interface{}
	if err := row.Scan(&tmp); err != nil {
		return 0, err
	}
	return float64(time.Since(start).Milliseconds()), nil
}

func measureQueryQPS(db *sql.DB, query string, iterations int) (float64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultProbeTimeout)
	defer cancel()
	start := time.Now()
	for i := 0; i < iterations; i++ {
		row := db.QueryRowContext(ctx, query)
		var tmp interface{}
		if err := row.Scan(&tmp); err != nil {
			return 0, err
		}
	}
	dur := time.Since(start)
	if dur <= 0 {
		return 0, errors.New("duração inválida")
	}
	return float64(iterations) / dur.Seconds(), nil
}

func runWriteProbe(db *sql.DB, spec dbProbeSpec) (float64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultProbeTimeout)
	defer cancel()

	conn, err := db.Conn(ctx)
	if err != nil {
		return 0, err
	}
	defer conn.Close()

	for _, stmt := range spec.WriteSetup {
		if _, err := conn.ExecContext(ctx, stmt); err != nil {
			return 0, err
		}
	}

	start := time.Now()
	if _, err := conn.ExecContext(ctx, spec.WriteProbe); err != nil {
		return 0, err
	}
	latency := float64(time.Since(start).Milliseconds())

	for _, stmt := range spec.WriteTeardown {
		_, _ = conn.ExecContext(ctx, stmt)
	}

	return latency, nil
}

func queryString(db *sql.DB, query string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultProbeTimeout)
	defer cancel()
	row := db.QueryRowContext(ctx, query)
	var out string
	if err := row.Scan(&out); err != nil {
		return "", err
	}
	return out, nil
}

func queryBool(db *sql.DB, query string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultProbeTimeout)
	defer cancel()
	row := db.QueryRowContext(ctx, query)
	var out bool
	if err := row.Scan(&out); err != nil {
		return false, err
	}
	return out, nil
}

func isDBReadOnly(db *sql.DB, dbType string) (bool, error) {
	switch strings.ToLower(dbType) {
	case "postgres":
		val, err := queryString(db, "SHOW transaction_read_only")
		if err != nil {
			return false, err
		}
		val = strings.ToLower(strings.TrimSpace(val))
		return val == "on" || val == "true" || val == "1", nil
	case "mysql":
		val, err := queryString(db, "SELECT @@read_only")
		if err != nil {
			return false, err
		}
		val = strings.TrimSpace(val)
		parsed, err := strconv.Atoi(val)
		if err != nil {
			return false, err
		}
		return parsed == 1, nil
	default:
		return false, nil
	}
}

func canWritePostgres(db *sql.DB) (bool, error) {
	// Verifica se o usuario atual tem privilegio de escrita em alguma tabela do schema do usuario.
	query := `
SELECT EXISTS (
  SELECT 1
  FROM pg_tables
  WHERE schemaname NOT IN ('pg_catalog', 'information_schema')
    AND (
      has_table_privilege(current_user, quote_ident(schemaname)||'.'||quote_ident(tablename), 'INSERT')
      OR has_table_privilege(current_user, quote_ident(schemaname)||'.'||quote_ident(tablename), 'UPDATE')
      OR has_table_privilege(current_user, quote_ident(schemaname)||'.'||quote_ident(tablename), 'DELETE')
    )
);`
	return queryBool(db, query)
}

// -------------------- Scores --------------------
func computeBenchmarkScores(metrics models.BenchmarkMetrics) models.BenchmarkScores {
	scores := models.BenchmarkScores{}
	if metrics.HostETL != nil {
		scores.HostETL = scoreHost(*metrics.HostETL)
	}
	if metrics.Origin != nil {
		scores.Origin = scoreDB(*metrics.Origin)
	}
	if metrics.Destination != nil {
		scores.Destination = scoreDB(*metrics.Destination)
	}
	return scores
}

func scoreHost(m models.HostMetrics) float64 {
	cpuScore := scoreInversePercent(m.CPUUsagePct, 20, 50, 70, 85, 95)
	memScore := 0.0
	if m.MemTotalBytes > 0 {
		memUsedPct := (float64(m.MemUsedBytes) / float64(m.MemTotalBytes)) * 100
		memScore = scoreInversePercent(memUsedPct, 40, 60, 75, 90, 97)
	}
	swapScore := 100.0
	if m.SwapTotalBytes > 0 {
		swapUsedPct := (float64(m.SwapUsedBytes) / float64(m.SwapTotalBytes)) * 100
		swapScore = scoreInversePercent(swapUsedPct, 10, 25, 50, 75, 90)
	}

	score := (cpuScore * 0.5) + (memScore * 0.4) + (swapScore * 0.1)
	return clampScore(score)
}

func scoreDB(m models.DBMetrics) float64 {
	latency := averageLatency(m.ConnLatencyMs, m.PingLatencyMs)
	latencyScore := scoreLatency(latency)
	qpsScore := scoreQPS(m.ProbeQPS)
	penalty := 0.0
	if len(m.Errors) > 0 {
		penalty = 20
	}
	score := (latencyScore * 0.5) + (qpsScore * 0.5) - penalty
	return clampScore(score)
}

func averageLatency(a, b float64) float64 {
	if a > 0 && b > 0 {
		return (a + b) / 2
	}
	if a > 0 {
		return a
	}
	return b
}

func scoreLatency(ms float64) float64 {
	switch {
	case ms <= 20:
		return 100
	case ms <= 50:
		return 85
	case ms <= 100:
		return 70
	case ms <= 200:
		return 50
	case ms <= 500:
		return 30
	default:
		return 10
	}
}

func scoreQPS(qps float64) float64 {
	switch {
	case qps >= 100:
		return 100
	case qps >= 50:
		return 85
	case qps >= 20:
		return 70
	case qps >= 10:
		return 50
	case qps >= 5:
		return 30
	default:
		return 10
	}
}

func scoreInversePercent(pct float64, t1, t2, t3, t4, t5 float64) float64 {
	switch {
	case pct <= t1:
		return 100
	case pct <= t2:
		return 85
	case pct <= t3:
		return 70
	case pct <= t4:
		return 50
	case pct <= t5:
		return 30
	default:
		return 10
	}
}

func clampScore(score float64) float64 {
	if math.IsNaN(score) || math.IsInf(score, 0) {
		return 0
	}
	if score < 0 {
		return 0
	}
	if score > 100 {
		return 100
	}
	return math.Round(score*10) / 10
}

// -------------------- Storage --------------------
func benchmarkBaseDir(projectID string) string {
	return filepath.Join("logs", "benchmarks", projectID)
}

func benchmarkFilePath(projectID, runID string) string {
	return filepath.Join(benchmarkBaseDir(projectID), fmt.Sprintf("benchmark_%s.json", runID))
}

func saveBenchmarkRun(projectID string, run *models.BenchmarkRun) error {
	benchmarkFileMu.Lock()
	defer benchmarkFileMu.Unlock()

	baseDir := benchmarkBaseDir(projectID)
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(run, "", "  ")
	if err != nil {
		return err
	}

	path := benchmarkFilePath(projectID, run.RunID)
	return os.WriteFile(path, data, 0644)
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

func listBenchmarkRuns(projectID string, limit int) ([]models.BenchmarkSummary, error) {
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

func parseLimit(raw string) int {
	if raw == "" {
		return 0
	}
	val, err := strconv.Atoi(raw)
	if err != nil || val <= 0 {
		return 0
	}
	return val
}

func derefBool(val *bool) bool {
	if val == nil {
		return false
	}
	return *val
}

func appendError(current, next string) string {
	if strings.TrimSpace(next) == "" {
		return current
	}
	if current == "" {
		return next
	}
	return current + "; " + next
}
