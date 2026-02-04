import { HttpClient } from '@angular/common/http';
import { Injectable } from '@angular/core';
import { Observable } from 'rxjs';
import { environment } from '../../../environments/environment';

export interface ErrorInfo {
  error_type: string;
  error_code: string;
  error: string;
  error_details?: {
    suggestion?: string;
    original_error?: string;
    timestamp?: string;
  };
  started_at?: string;
  ended_at?: string;
}

export interface JobError extends ErrorInfo {
  job_id: string;
  job_name: string;
}

export interface BatchError extends ErrorInfo {
  job_id: string;
  job_name: string;
  batch_offset: number;
  batch_limit: number;
}

export interface ErrorSummary {
  pipeline_id: string;
  total_errors: number;
  error_types: { [key: string]: number };
  error_jobs: JobError[];
  error_batches: BatchError[];
}

export interface PipelineLog {
  pipeline_id: string;
  project: string;
  status: string;
  started_at: string;
  ended_at: string;
  jobs: JobLog[];
}

export interface JobLog {
  job_id: string;
  job_name: string;
  status: string;
  error?: string;
  error_type?: string;
  error_code?: string;
  error_details?: { [key: string]: any };
  stop_on_error: boolean;
  started_at: string;
  ended_at: string;
  processed: number;
  total: number;
  batches: BatchLog[];
}

export interface BatchLog {
  offset: number;
  limit: number;
  status: string;
  error?: string;
  error_type?: string;
  error_code?: string;
  rows: number;
  started_at: string;
  ended_at: string;
}

export interface PipelineStats {
  pipeline_id: string;
  project: string;
  status: string;
  started_at: string;
  ended_at: string;
  duration: string;
  total_jobs: number;
  job_stats: { [key: string]: number };
  total_batches: number;
  total_processed: number;
}

@Injectable({ providedIn: 'root' })
export class ErrorService {
  private apiUrl = `${environment.apiUrl}/api`;

  constructor(private http: HttpClient) {}

  getErrorSummary(pipelineId: string): Observable<ErrorSummary> {
    return this.http.get<ErrorSummary>(`${this.apiUrl}/pipeline/${pipelineId}/errors`);
  }

  getPipelineLog(pipelineId: string): Observable<PipelineLog> {
    return this.http.get<PipelineLog>(`${this.apiUrl}/pipeline/${pipelineId}/log`);
  }

  getPipelineStats(pipelineId: string): Observable<PipelineStats> {
    return this.http.get<PipelineStats>(`${this.apiUrl}/pipeline/${pipelineId}/stats`);
  }

  listPipelines(): Observable<string[]> {
    return this.http.get<string[]>(`${this.apiUrl}/pipelines/reports`);
  }

  getPipelineReportUrl(pipelineId: string): string {
    return `${this.apiUrl}/pipeline/${pipelineId}/report`;
  }

  getPipelineReportPreviewUrl(pipelineId: string): string {
    return `${this.apiUrl}/pipeline/${pipelineId}/report/preview`;
  }

  categorizeError(errorMessage: string): { type: string; label: string; icon: string; severity: string } {
    const lowerError = errorMessage.toLowerCase();

    if (lowerError.includes('duplicate key') || lowerError.includes('unique constraint')) {
      return {
        type: 'duplicate_key_error',
        label: 'Chave Duplicada',
        icon: 'content_copy',
        severity: 'warn'
      };
    } else if (lowerError.includes('foreign key')) {
      return {
        type: 'foreign_key_error',
        label: 'Chave Estrangeira',
        icon: 'link_off',
        severity: 'warn'
      };
    } else if (lowerError.includes('connection') || lowerError.includes('timeout')) {
      return {
        type: 'connection_error',
        label: 'Conexão',
        icon: 'wifi_off',
        severity: 'warn'
      };
    } else if (lowerError.includes('syntax error') || lowerError.includes('invalid sql')) {
      return {
        type: 'sql_syntax_error',
        label: 'Sintaxe SQL',
        icon: 'code_off',
        severity: 'warn'
      };
    } else if (lowerError.includes('permission') || lowerError.includes('access denied')) {
      return {
        type: 'permission_error',
        label: 'Permissão',
        icon: 'lock',
        severity: 'warn'
      };
    } else if (lowerError.includes('table') && lowerError.includes("doesn't exist")) {
      return {
        type: 'table_not_found',
        label: 'Tabela Não Encontrada',
        icon: 'table_chart',
        severity: 'warn'
      };
    } else {
      return {
        type: 'unknown_error',
        label: 'Erro Desconhecido',
        icon: 'help',
        severity: 'warn'
      };
    }
  }

  formatDateTime(dateString: string): string {
    if (!dateString) return '';
    const date = new Date(dateString);
    return date.toLocaleString('pt-BR');
  }

  formatTime(dateString: string): string {
    if (!dateString) return '';
    const date = new Date(dateString);
    return date.toLocaleTimeString('pt-BR');
  }

  calculateDuration(startDate: string, endDate: string): string {
    if (!startDate || !endDate) return '';

    const start = new Date(startDate);
    const end = new Date(endDate);
    const diffMs = end.getTime() - start.getTime();

    const seconds = Math.floor(diffMs / 1000);
    const minutes = Math.floor(seconds / 60);
    const hours = Math.floor(minutes / 60);
    const days = Math.floor(hours / 24);

    if (days > 0) return `${days}d ${hours % 24}h ${minutes % 60}m`;
    if (hours > 0) return `${hours}h ${minutes % 60}m ${seconds % 60}s`;
    if (minutes > 0) return `${minutes}m ${seconds % 60}s`;
    return `${seconds}s`;
  }
}
