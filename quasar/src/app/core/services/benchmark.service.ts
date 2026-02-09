import { HttpClient, HttpParams } from '@angular/common/http';
import { Injectable } from '@angular/core';
import { Observable } from 'rxjs';
import { timeout } from 'rxjs/operators';
import { environment } from '../../../environments/environment';

export interface BenchmarkOptions {
  probe_iterations: number;
  enable_write_probe: boolean;
  include_host: boolean;
  include_origin: boolean;
  include_destination: boolean;
}

export interface BenchmarkHostMetrics {
  cpu_cores: number;
  cpu_usage_pct: number;
  mem_total_bytes: number;
  mem_used_bytes: number;
  swap_total_bytes?: number;
  swap_used_bytes?: number;
  disk_total_bytes?: number;
  disk_free_bytes?: number;
}

export interface BenchmarkDbMetrics {
  db_type: string;
  db_version?: string;
  conn_latency_ms?: number;
  ping_latency_ms?: number;
  probe_iterations?: number;
  probe_qps?: number;
  write_enabled: boolean;
  write_latency_ms?: number;
  errors?: string[];
}

export interface BenchmarkMetrics {
  host_etl?: BenchmarkHostMetrics;
  origin?: BenchmarkDbMetrics;
  destination?: BenchmarkDbMetrics;
}

export interface BenchmarkScores {
  host_etl?: number;
  origin?: number;
  destination?: number;
}

export interface BenchmarkRun {
  run_id: string;
  project_id: string;
  status: 'ok' | 'partial' | 'error' | string;
  error?: string;
  started_at: string;
  ended_at: string;
  options: BenchmarkOptions;
  metrics: BenchmarkMetrics;
  scores: BenchmarkScores;
}

export interface BenchmarkSummary {
  run_id: string;
  status: 'ok' | 'partial' | 'error' | string;
  started_at: string;
  ended_at: string;
  scores: BenchmarkScores;
}

export interface BenchmarkRunRequest {
  probeIterations?: number;
  enableWriteProbe?: boolean;
  includeHost?: boolean;
  includeOrigin?: boolean;
  includeDestination?: boolean;
}

@Injectable({ providedIn: 'root' })
export class BenchmarkService {
  private apiUrl = `${environment.apiUrl}/api/projects`;

  constructor(private http: HttpClient) {}

  runBenchmark(projectId: string, options?: BenchmarkRunRequest): Observable<BenchmarkRun> {
    return this.http
      .post<BenchmarkRun>(`${this.apiUrl}/${projectId}/benchmarks/run`, options ?? {})
      .pipe(timeout(60000));
  }

  listBenchmarks(projectId: string, limit = 10): Observable<BenchmarkSummary[]> {
    const params = new HttpParams().set('limit', limit.toString());
    return this.http
      .get<BenchmarkSummary[]>(`${this.apiUrl}/${projectId}/benchmarks`, { params })
      .pipe(timeout(12000));
  }

  getBenchmark(projectId: string, runId: string): Observable<BenchmarkRun> {
    return this.http
      .get<BenchmarkRun>(`${this.apiUrl}/${projectId}/benchmarks/${runId}`)
      .pipe(timeout(12000));
  }

  getBenchmarkReportUrl(projectId: string, runId: string): string {
    return `${this.apiUrl}/${projectId}/benchmarks/${runId}/report`;
  }

  getBenchmarkHistoryReportUrl(projectId: string, limit?: number): string {
    if (typeof limit === 'number') {
      return `${this.apiUrl}/${projectId}/benchmarks/report?limit=${limit}`;
    }
    return `${this.apiUrl}/${projectId}/benchmarks/report`;
  }
}
