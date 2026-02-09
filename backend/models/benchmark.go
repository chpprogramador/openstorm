package models

import "time"

type BenchmarkRunRequest struct {
	ProbeIterations    int   `json:"probeIterations"`
	EnableWriteProbe   *bool `json:"enableWriteProbe"`
	IncludeHost        bool  `json:"includeHost"`
	IncludeOrigin      bool  `json:"includeOrigin"`
	IncludeDestination bool  `json:"includeDestination"`
}

type BenchmarkRun struct {
	RunID     string           `json:"run_id"`
	ProjectID string           `json:"project_id"`
	Status    string           `json:"status"`
	Error     string           `json:"error,omitempty"`
	StartedAt time.Time        `json:"started_at"`
	EndedAt   time.Time        `json:"ended_at"`
	Options   BenchmarkOptions `json:"options"`
	Metrics   BenchmarkMetrics `json:"metrics"`
	Scores    BenchmarkScores  `json:"scores"`
}

type BenchmarkOptions struct {
	ProbeIterations    int  `json:"probe_iterations"`
	EnableWriteProbe   bool `json:"enable_write_probe"`
	IncludeHost        bool `json:"include_host"`
	IncludeOrigin      bool `json:"include_origin"`
	IncludeDestination bool `json:"include_destination"`
}

type BenchmarkMetrics struct {
	HostETL     *HostMetrics `json:"host_etl,omitempty"`
	Origin      *DBMetrics   `json:"origin,omitempty"`
	Destination *DBMetrics   `json:"destination,omitempty"`
}

type BenchmarkScores struct {
	HostETL     float64 `json:"host_etl,omitempty"`
	Origin      float64 `json:"origin,omitempty"`
	Destination float64 `json:"destination,omitempty"`
}

type HostMetrics struct {
	CPUCores       int     `json:"cpu_cores"`
	CPUUsagePct    float64 `json:"cpu_usage_pct"`
	MemTotalBytes  uint64  `json:"mem_total_bytes"`
	MemUsedBytes   uint64  `json:"mem_used_bytes"`
	SwapTotalBytes uint64  `json:"swap_total_bytes,omitempty"`
	SwapUsedBytes  uint64  `json:"swap_used_bytes,omitempty"`
	DiskTotalBytes uint64  `json:"disk_total_bytes,omitempty"`
	DiskFreeBytes  uint64  `json:"disk_free_bytes,omitempty"`
}

type DBMetrics struct {
	DBType          string   `json:"db_type"`
	DBVersion       string   `json:"db_version,omitempty"`
	ConnLatencyMs   float64  `json:"conn_latency_ms,omitempty"`
	PingLatencyMs   float64  `json:"ping_latency_ms,omitempty"`
	ProbeIterations int      `json:"probe_iterations,omitempty"`
	ProbeQPS        float64  `json:"probe_qps,omitempty"`
	WriteEnabled    bool     `json:"write_enabled"`
	WriteLatencyMs  float64  `json:"write_latency_ms,omitempty"`
	Errors          []string `json:"errors,omitempty"`
}

type BenchmarkSummary struct {
	RunID     string          `json:"run_id"`
	Status    string          `json:"status"`
	StartedAt time.Time       `json:"started_at"`
	EndedAt   time.Time       `json:"ended_at"`
	Scores    BenchmarkScores `json:"scores"`
}
