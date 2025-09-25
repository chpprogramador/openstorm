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
import { ErrorViewerComponent } from '../../components/error-viewer/error-viewer.component';
import { ErrorService, ErrorSummary, PipelineStats } from '../../services/error.service';

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
            <mat-icon color="warn">bug_report</mat-icon>
            Visualizador de Erros
          </mat-card-title>
          <mat-card-subtitle>
            Analise e diagnostique erros de execução de pipelines
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
  }

  loadPipelineData() {
    if (!this.selectedPipelineId) return;

    this.isLoading = true;
    
    // Carrega estatísticas e erros em paralelo
    const statsRequest = this.errorService.getPipelineStats(this.selectedPipelineId);
    const errorsRequest = this.errorService.getErrorSummary(this.selectedPipelineId);

    // Carrega estatísticas
    statsRequest.subscribe({
      next: (stats) => {
        this.pipelineStats = stats;
      },
      error: (error) => {
        console.error('Erro ao carregar estatísticas:', error);
        this.snackBar.open('Erro ao carregar estatísticas do pipeline', 'Fechar', {
          duration: 3000,
          panelClass: ['error-snackbar']
        });
      }
    });

    // Carrega resumo de erros
    errorsRequest.subscribe({
      next: (errorSummary) => {
        this.errorSummary = errorSummary;
        this.isLoading = false;
        
        if (errorSummary.total_errors > 0) {
          this.snackBar.open(
            `Pipeline carregado com ${errorSummary.total_errors} erro(s) encontrado(s)`, 
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
        console.error('Erro ao carregar resumo de erros:', error);
        this.snackBar.open('Erro ao carregar resumo de erros', 'Fechar', {
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
}
