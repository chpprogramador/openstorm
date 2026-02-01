import { CommonModule } from '@angular/common';
import { Component, OnInit } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { MatButtonModule } from '@angular/material/button';
import { MatCardModule } from '@angular/material/card';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatIconModule } from '@angular/material/icon';
import { MatInputModule } from '@angular/material/input';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { MatSelectModule } from '@angular/material/select';
import { MatSnackBar, MatSnackBarModule } from '@angular/material/snack-bar';
import { forkJoin } from 'rxjs';
import { ErrorViewerComponent } from '../../components/error-viewer/error-viewer.component';
import { ErrorService, ErrorSummary, PipelineLog, PipelineStats } from '../../services/error.service';

@Component({
  selector: 'app-errors',
  standalone: true,
  imports: [
    CommonModule,
    MatCardModule,
    MatButtonModule,
    MatIconModule,
    MatFormFieldModule,
    MatSelectModule,
    MatInputModule,
    MatProgressSpinnerModule,
    MatSnackBarModule,
    FormsModule,
    ErrorViewerComponent
  ],
  template: `
    <div class="errors-page-container">
      <mat-card class="page-header">
        <mat-card-header>
          <mat-card-title>
            <mat-icon color="warn">history</mat-icon>
            Histórico & Erros
          </mat-card-title>
          <mat-card-subtitle>
            Timeline de execuções com estatísticas e detalhes de erro
          </mat-card-subtitle>
        </mat-card-header>
      </mat-card>

      <mat-card class="pipeline-selector">
        <mat-card-content>
          <div class="selector-row">
            <mat-form-field appearance="outline" class="pipeline-select">
              <mat-label>Selecione um Pipeline</mat-label>
              <mat-select [(value)]="selectedPipelineId" (selectionChange)="onPipelineChange($event.value)">
                <mat-option *ngFor="let pipeline of availablePipelines" [value]="pipeline">
                  {{ pipeline }}
                </mat-option>
              </mat-select>
            </mat-form-field>
            
            <button mat-raised-button 
                    color="primary" 
                    (click)="loadPipelineData()"
                    [disabled]="!selectedPipelineId || isLoading">
              <mat-icon>refresh</mat-icon>
              Carregar
            </button>
          </div>
        </mat-card-content>
      </mat-card>

      <!-- Loading Spinner -->
      <div class="loading-container" *ngIf="isLoading">
        <mat-spinner></mat-spinner>
        <p>Carregando dados do pipeline...</p>
      </div>

      <!-- Pipeline Stats -->
      <mat-card class="pipeline-stats" *ngIf="pipelineStats && !isLoading">
        <mat-card-header>
          <mat-card-title>
            <mat-icon>analytics</mat-icon>
            Estatísticas do Pipeline
          </mat-card-title>
        </mat-card-header>
        <mat-card-content>
          <div class="stats-summary">
            <div class="summary-item">
              <span class="summary-label">Pipeline</span>
              <span class="summary-value">{{ pipelineStats.project || pipelineStats.pipeline_id }}</span>
            </div>
            <div class="summary-item">
              <span class="summary-label">Início</span>
              <span class="summary-value">{{ formatDateTime(pipelineStats.started_at) }}</span>
            </div>
            <div class="summary-item">
              <span class="summary-label">Fim</span>
              <span class="summary-value">{{ formatDateTime(pipelineStats.ended_at) }}</span>
            </div>
            <div class="summary-item">
              <span class="summary-label">Duração</span>
              <span class="summary-value">{{ pipelineStats.duration }}</span>
            </div>
          </div>

          <div class="stats-grid">
            <div class="stat-item">
              <div class="stat-value">{{ pipelineStats.total_jobs }}</div>
              <div class="stat-label">Total de Jobs</div>
            </div>
            <div class="stat-item">
              <div class="stat-value">{{ pipelineStats.total_processed }}</div>
              <div class="stat-label">Registros Processados</div>
            </div>
            <div class="stat-item">
              <div class="stat-value">{{ pipelineStats.duration }}</div>
              <div class="stat-label">Duração</div>
            </div>
            <div class="stat-item">
              <div class="stat-value" [class]="getStatusClass(pipelineStats.status)">
                {{ getStatusLabel(pipelineStats.status) }}
              </div>
              <div class="stat-label">Status</div>
            </div>
          </div>
          
          <div class="job-stats" *ngIf="pipelineStats.job_stats">
            <h4>Status dos Jobs:</h4>
            <div class="job-stats-chips">
              <span class="job-stat-chip done" *ngIf="pipelineStats.job_stats['done']">
                <mat-icon>check_circle</mat-icon>
                Concluídos: {{ pipelineStats.job_stats['done'] }}
              </span>
              <span class="job-stat-chip error" *ngIf="pipelineStats.job_stats['error']">
                <mat-icon>error</mat-icon>
                Com Erro: {{ pipelineStats.job_stats['error'] }}
              </span>
              <span class="job-stat-chip running" *ngIf="pipelineStats.job_stats['running']">
                <mat-icon>play_circle</mat-icon>
                Executando: {{ pipelineStats.job_stats['running'] }}
              </span>
              <span class="job-stat-chip pending" *ngIf="pipelineStats.job_stats['pending']">
                <mat-icon>schedule</mat-icon>
                Pendentes: {{ pipelineStats.job_stats['pending'] }}
              </span>
            </div>
          </div>
        </mat-card-content>
      </mat-card>

      <!-- Timeline -->
      <mat-card class="timeline-card" *ngIf="pipelineLog && !isLoading">
        <mat-card-header>
          <mat-card-title>
            <mat-icon>timeline</mat-icon>
            Timeline de Execuções
          </mat-card-title>
          <mat-card-subtitle>
            {{ pipelineLog.jobs.length || 0 }} jobs nesta execução
          </mat-card-subtitle>
        </mat-card-header>
        <mat-card-content>
          <div class="timeline">
            <div class="timeline-item" *ngFor="let job of pipelineLog.jobs; let i = index">
                <div class="timeline-rail">
                <div class="timeline-node" [ngClass]="getStatusClass(job.status)"></div>
                <div class="timeline-line" *ngIf="i < (pipelineLog.jobs.length - 1)"></div>
              </div>
              <div class="timeline-content">
                <div class="timeline-header">
                  <div class="job-title">
                    <mat-icon [ngClass]="getStatusClass(job.status)">
                      {{ getStatusIcon(job.status) }}
                    </mat-icon>
                    <span>{{ job.job_name }}</span>
                  </div>
                  <div class="job-meta">
                    <span class="meta-chip" [ngClass]="getStatusClass(job.status)">
                      {{ getStatusLabel(job.status) }}
                    </span>
                    <span class="meta-chip neutral">
                      {{ job.processed }} / {{ job.total }}
                    </span>
                    <span class="meta-chip neutral">
                      {{ getDuration(job.started_at, job.ended_at) }}
                    </span>
                  </div>
                </div>

                <div class="timeline-body">
                  <div class="time-range">
                    <span>Início: {{ formatDateTime(job.started_at) }}</span>
                    <span>Fim: {{ formatDateTime(job.ended_at) }}</span>
                  </div>

                  <div class="error-block" *ngIf="job.status === 'error'">
                    <div class="error-title">
                      <mat-icon color="warn">error_outline</mat-icon>
                      <span>Erro</span>
                    </div>
                    <div class="error-message">{{ job.error }}</div>
                    <div class="error-details" *ngIf="job.error_details || job.error_type || job.error_code">
                      <div *ngIf="job.error_type"><strong>Tipo:</strong> {{ job.error_type }}</div>
                      <div *ngIf="job.error_code"><strong>Código:</strong> {{ job.error_code }}</div>
                      <div *ngIf="job.error_details?.['suggestion']"><strong>Sugestão:</strong> {{ job.error_details?.['suggestion'] }}</div>
                      <div *ngIf="job.error_details?.['original_error']"><strong>Detalhe:</strong> {{ job.error_details?.['original_error'] }}</div>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </mat-card-content>
      </mat-card>

      <!-- Error Viewer -->
      <app-error-viewer 
        *ngIf="errorSummary && !isLoading"
        [pipelineId]="selectedPipelineId"
        [errorSummary]="errorSummary">
      </app-error-viewer>

      <!-- No Pipeline Selected -->
      <mat-card class="no-selection" *ngIf="!selectedPipelineId && !isLoading">
        <mat-card-content>
          <div class="no-selection-content">
            <mat-icon color="primary">info</mat-icon>
            <h3>Selecione um Pipeline</h3>
            <p>Escolha um pipeline na lista acima para visualizar seus erros e estatísticas.</p>
          </div>
        </mat-card-content>
      </mat-card>
    </div>
  `,
  styleUrls: ['./errors.component.scss']
})
export class ErrorsComponent implements OnInit {
  selectedPipelineId: string = '';
  availablePipelines: string[] = [];
  errorSummary: ErrorSummary | null = null;
  pipelineStats: PipelineStats | null = null;
  pipelineLog: PipelineLog | null = null;
  isLoading: boolean = false;

  constructor(
    private errorService: ErrorService,
    private snackBar: MatSnackBar
  ) {}

  ngOnInit() {
    this.loadAvailablePipelines();
  }

  loadAvailablePipelines() {
    this.isLoading = true;
    this.errorService.listPipelines().subscribe({
      next: (pipelines) => {
        console.log('Pipelines disponíveis:', pipelines);
        this.availablePipelines = pipelines;
        this.isLoading = false;
      },
      error: (error) => {
        console.error('Erro ao carregar pipelines:', error);
        this.snackBar.open('Erro ao carregar lista de pipelines', 'Fechar', {
          duration: 3000,
          panelClass: ['error-snackbar']
        });
        this.isLoading = false;
      }
    });
  }

  onPipelineChange(pipelineId: string) {
    this.selectedPipelineId = pipelineId;
    this.errorSummary = null;
    this.pipelineStats = null;
    this.pipelineLog = null;
  }

  loadPipelineData() {
    if (!this.selectedPipelineId) return;

    this.isLoading = true;

    forkJoin({
      stats: this.errorService.getPipelineStats(this.selectedPipelineId),
      errors: this.errorService.getErrorSummary(this.selectedPipelineId),
      log: this.errorService.getPipelineLog(this.selectedPipelineId)
    }).subscribe({
      next: ({ stats, errors, log }) => {
        this.pipelineStats = stats;
        this.errorSummary = errors;
        this.pipelineLog = log;
        this.isLoading = false;

        if (errors.total_errors > 0) {
          this.snackBar.open(
            `Pipeline carregado com ${errors.total_errors} erro(s) encontrado(s)`, 
            'Fechar', 
            { duration: 3000 }
          );
        } else {
          this.snackBar.open('Pipeline carregado sem erros!', 'Fechar', {
            duration: 3000,
            panelClass: ['success-snackbar']
          });
        }
      },
      error: (error) => {
        console.error('Erro ao carregar dados do pipeline:', error);
        this.snackBar.open('Erro ao carregar dados do pipeline', 'Fechar', {
          duration: 3000,
          panelClass: ['error-snackbar']
        });
        this.isLoading = false;
      }
    });
  }

  getStatusClass(status: string): string {
    const classes: { [key: string]: string } = {
      'done': 'status-done',
      'error': 'status-error',
      'running': 'status-running',
      'pending': 'status-pending'
    };
    return classes[status] || 'status-unknown';
  }

  getStatusLabel(status: string): string {
    const labels: { [key: string]: string } = {
      'done': 'Concluído',
      'error': 'Com Erro',
      'running': 'Executando',
      'pending': 'Pendente'
    };
    return labels[status] || 'Desconhecido';
  }

  getStatusIcon(status: string): string {
    const icons: { [key: string]: string } = {
      'done': 'check_circle',
      'error': 'error',
      'running': 'play_circle',
      'pending': 'schedule'
    };
    return icons[status] || 'help';
  }

  formatDateTime(dateString: string): string {
    return this.errorService.formatDateTime(dateString);
  }

  getDuration(startedAt: string, endedAt: string): string {
    if (!startedAt || !endedAt) return '';
    const start = new Date(startedAt).getTime();
    const end = new Date(endedAt).getTime();
    if (Number.isNaN(start) || Number.isNaN(end)) return '';
    const diffMs = Math.max(0, end - start);
    const totalSeconds = Math.floor(diffMs / 1000);
    const minutes = Math.floor(totalSeconds / 60);
    const seconds = totalSeconds % 60;
    return `${minutes}m ${seconds}s`;
  }
}
