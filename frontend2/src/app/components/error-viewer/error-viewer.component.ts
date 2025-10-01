import { CommonModule } from '@angular/common';
import { Component, Input, OnInit } from '@angular/core';
import { MatBadgeModule } from '@angular/material/badge';
import { MatButtonModule } from '@angular/material/button';
import { MatCardModule } from '@angular/material/card';
import { MatChipsModule } from '@angular/material/chips';
import { MatDividerModule } from '@angular/material/divider';
import { MatExpansionModule } from '@angular/material/expansion';
import { MatIconModule } from '@angular/material/icon';
import { MatListModule } from '@angular/material/list';
import { MatTooltipModule } from '@angular/material/tooltip';

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

@Component({
  selector: 'app-error-viewer',
  standalone: true,
  imports: [
    CommonModule,
    MatCardModule,
    MatExpansionModule,
    MatIconModule,
    MatChipsModule,
    MatButtonModule,
    MatTooltipModule,
    MatDividerModule,
    MatListModule,
    MatBadgeModule
  ],
  template: `
    <div class="error-viewer-container">
      <mat-card class="error-summary-card" *ngIf="errorSummary">
        <mat-card-header>
          <mat-card-title>
            <mat-icon color="warn">error</mat-icon>
            Resumo de Erros - {{ errorSummary.pipeline_id }}
          </mat-card-title>
          <mat-card-subtitle>
            Total de erros: {{ errorSummary.total_errors }}
          </mat-card-subtitle>
        </mat-card-header>
        
        <mat-card-content>
          <!-- Tipos de Erro -->
          <div class="error-types-section" *ngIf="errorSummary.error_types && (errorSummary.error_types | keyvalue).length > 0">
            <h4>Tipos de Erro:</h4>
            <div class="error-types-chips">
              <mat-chip-set>
                <mat-chip *ngFor="let errorType of errorSummary.error_types | keyvalue" 
                          [class]="getErrorTypeClass(errorType.key)">
                  <mat-icon>{{ getErrorTypeIcon(errorType.key) }}</mat-icon>
                  {{ getErrorTypeLabel(errorType.key) }} ({{ errorType.value }})
                </mat-chip>
              </mat-chip-set>
            </div>
          </div>

          <mat-divider></mat-divider>

          <!-- Erros de Jobs -->
          <div class="job-errors-section" *ngIf="jobErrors.length > 0">
            <h4>
              <mat-icon color="warn">work</mat-icon>
              Erros de Jobs ({{ jobErrors.length }})
            </h4>
            
            <mat-accordion>
              <mat-expansion-panel *ngFor="let jobError of jobErrors; let i = index">
                <mat-expansion-panel-header>
                  <mat-panel-title>
                    <mat-icon [color]="getErrorSeverity(jobError.error_type)">{{ getErrorTypeIcon(jobError.error_type) }}</mat-icon>
                    {{ jobError.job_name }}
                  </mat-panel-title>
                  <mat-panel-description>
                    <mat-chip [class]="getErrorTypeClass(jobError.error_type)">
                      {{ getErrorTypeLabel(jobError.error_type) }}
                    </mat-chip>
                    <span class="error-time">{{ formatTimeSafe(jobError.ended_at) }}</span>
                  </mat-panel-description>
                </mat-expansion-panel-header>
                
                <div class="error-details">
                  <div class="error-message">
                    <strong>Erro:</strong>
                    <pre class="error-text">{{ jobError.error }}</pre>
                  </div>
                  
                  <div class="error-info" *ngIf="jobError.error_details">
                    <div class="error-suggestion" *ngIf="jobError.error_details.suggestion">
                      <mat-icon color="primary">lightbulb</mat-icon>
                      <strong>Sugestão:</strong> {{ jobError.error_details.suggestion }}
                    </div>
                  </div>
                  
                  <div class="error-metadata">
                    <div class="metadata-item">
                      <strong>Job ID:</strong> {{ jobError.job_id }}
                    </div>
                    <div class="metadata-item">
                      <strong>Código do Erro:</strong> {{ jobError.error_code }}
                    </div>
                    <div class="metadata-item" *ngIf="jobError.started_at">
                      <strong>Iniciado em:</strong> {{ formatDateTime(jobError.started_at) }}
                    </div>
                    <div class="metadata-item" *ngIf="jobError.ended_at">
                      <strong>Finalizado em:</strong> {{ formatDateTime(jobError.ended_at) }}
                    </div>
                  </div>
                </div>
              </mat-expansion-panel>
            </mat-accordion>
          </div>

          <mat-divider *ngIf="jobErrors.length > 0 && batchErrors.length > 0"></mat-divider>

          <!-- Erros de Batches -->
          <div class="batch-errors-section" *ngIf="batchErrors.length > 0">
            <h4>
              <mat-icon color="warn">view_list</mat-icon>
              Erros de Batches ({{ batchErrors.length }})
            </h4>
            
            <mat-accordion>
              <mat-expansion-panel *ngFor="let batchError of batchErrors; let i = index">
                <mat-expansion-panel-header>
                  <mat-panel-title>
                    <mat-icon [color]="getErrorSeverity(batchError.error_type)">{{ getErrorTypeIcon(batchError.error_type) }}</mat-icon>
                    {{ batchError.job_name }} - Batch {{ batchError.batch_offset }}-{{ batchError.batch_offset + batchError.batch_limit }}
                  </mat-panel-title>
                  <mat-panel-description>
                    <mat-chip [class]="getErrorTypeClass(batchError.error_type)">
                      {{ getErrorTypeLabel(batchError.error_type) }}
                    </mat-chip>
    <span class="error-time">{{ formatTimeSafe(batchError.ended_at) }}</span>
  </mat-panel-description>
                </mat-expansion-panel-header>
                
                <div class="error-details">
                  <div class="error-message">
                    <strong>Erro:</strong>
                    <pre class="error-text">{{ batchError.error }}</pre>
                  </div>
                  
                  <div class="error-info" *ngIf="batchError.error_details">
                    <div class="error-suggestion" *ngIf="batchError.error_details.suggestion">
                      <mat-icon color="primary">lightbulb</mat-icon>
                      <strong>Sugestão:</strong> {{ batchError.error_details.suggestion }}
                    </div>
                  </div>
                  
                  <div class="error-metadata">
                    <div class="metadata-item">
                      <strong>Job ID:</strong> {{ batchError.job_id }}
                    </div>
                    <div class="metadata-item">
                      <strong>Offset:</strong> {{ batchError.batch_offset }}
                    </div>
                    <div class="metadata-item">
                      <strong>Limit:</strong> {{ batchError.batch_limit }}
                    </div>
                    <div class="metadata-item">
                      <strong>Código do Erro:</strong> {{ batchError.error_code }}
                    </div>
                    <div class="metadata-item" *ngIf="batchError.started_at">
                      <strong>Iniciado em:</strong> {{ formatDateTime(batchError.started_at) }}
                    </div>
                    <div class="metadata-item" *ngIf="batchError.ended_at">
                      <strong>Finalizado em:</strong> {{ formatDateTime(batchError.ended_at) }}
                    </div>
                  </div>
                </div>
              </mat-expansion-panel>
            </mat-accordion>
          </div>
        </mat-card-content>
      </mat-card>

      <div class="no-errors-message" *ngIf="!errorSummary || errorSummary.total_errors === 0">
        <mat-icon color="primary">check_circle</mat-icon>
        <h3>Nenhum erro encontrado!</h3>
        <p>O pipeline foi executado com sucesso sem erros.</p>
      </div>
    </div>
  `,
  styleUrls: ['./error-viewer.component.scss']
})
export class ErrorViewerComponent implements OnInit {
  formatTimeSafe(dateString?: string): string {
    if (!dateString) return '';
    const date = new Date(dateString);
    if (isNaN(date.getTime())) return '';
    return date.toLocaleTimeString('pt-BR');
  }
  @Input() pipelineId: string = '';
  @Input() errorSummary: ErrorSummary | null = null;

  get jobErrors() {
    return Array.isArray(this.errorSummary?.error_jobs) ? this.errorSummary!.error_jobs : [];
  }

  get batchErrors() {
    return Array.isArray(this.errorSummary?.error_batches) ? this.errorSummary!.error_batches : [];
  }

  ngOnInit() {
    // Componente pode ser inicializado com dados ou buscar via API
  }

  getErrorTypes() {
    if (!this.errorSummary?.error_types) return [];
    return Object.entries(this.errorSummary.error_types).map(([type, count]) => ({
      type,
      count
    }));
  }

  getErrorTypeClass(errorType: string | undefined): string {
    const classes: { [key: string]: string } = {
      'duplicate_key_error': 'error-chip-duplicate',
      'foreign_key_error': 'error-chip-foreign-key',
      'connection_error': 'error-chip-connection',
      'sql_syntax_error': 'error-chip-syntax',
      'permission_error': 'error-chip-permission',
      'table_not_found': 'error-chip-table',
      'unknown_error': 'error-chip-unknown'
    };
  if (!errorType) return 'error-chip-unknown';
  return classes[errorType] || 'error-chip-unknown';
  }

  getErrorTypeIcon(errorType: string | undefined): string {
    const icons: { [key: string]: string } = {
      'duplicate_key_error': 'content_copy',
      'foreign_key_error': 'link_off',
      'connection_error': 'wifi_off',
      'sql_syntax_error': 'code_off',
      'permission_error': 'lock',
      'table_not_found': 'table_chart',
      'unknown_error': 'help'
    };
  if (!errorType) return 'help';
  return icons[errorType] || 'help';
  }

  getErrorTypeLabel(errorType: string | undefined): string {
    const labels: { [key: string]: string } = {
      'duplicate_key_error': 'Chave Duplicada',
      'foreign_key_error': 'Chave Estrangeira',
      'connection_error': 'Conexão',
      'sql_syntax_error': 'Sintaxe SQL',
      'permission_error': 'Permissão',
      'table_not_found': 'Tabela Não Encontrada',
      'unknown_error': 'Erro Desconhecido'
    };
  if (!errorType) return 'Erro Desconhecido';
  return labels[errorType] || 'Erro Desconhecido';
  }

  getErrorSeverity(errorType: string | undefined): string {
    const severities: { [key: string]: string } = {
      'duplicate_key_error': 'warn',
      'foreign_key_error': 'warn',
      'connection_error': 'warn',
      'sql_syntax_error': 'warn',
      'permission_error': 'warn',
      'table_not_found': 'warn',
      'unknown_error': 'warn'
    };
  if (!errorType) return 'warn';
  return severities[errorType] || 'warn';
  }

  formatDateTime(dateString: string): string {
  if (!dateString) return '';
  const date = new Date(dateString);
  if (isNaN(date.getTime())) return '';
  return date.toLocaleString('pt-BR');
  }

  formatTime(dateString: string): string {
  if (!dateString) return '';
  const date = new Date(dateString);
  if (isNaN(date.getTime())) return '';
  return date.toLocaleTimeString('pt-BR');
  }
}
