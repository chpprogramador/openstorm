// src/app/features/history/history.component.ts
import { Component, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { ErrorService } from '../../core/services/error.service';

@Component({
  selector: 'app-history',
  standalone: true,
  imports: [CommonModule, FormsModule],
  template: `
    <div class="container">
      <div class="page-header">
        <div>
          <h1 class="page-title">Histórico</h1>
          <p class="page-subtitle">Visualize o histórico de execuções</p>
        </div>
      </div>

      <div class="report-toolbar">
        <div class="report-controls">
          <label class="report-label">Pipeline</label>
          <select class="report-select" [(ngModel)]="selectedPipelineId">
            <option value="" disabled>Selecione uma pipeline</option>
            <option *ngFor="let pipeline of availablePipelines" [value]="pipeline">
              {{ pipeline }}
            </option>
          </select>
        </div>
        <div class="report-actions">
          <button class="btn-primary" (click)="openPreview()" [disabled]="!selectedPipelineId">
            Visualizar PDF
          </button>
          <button class="btn-secondary" (click)="downloadReport()" [disabled]="!selectedPipelineId">
            Baixar PDF
          </button>
        </div>
      </div>

      <div class="history-timeline">
        <div class="timeline-item">
          <div class="timeline-marker success"></div>
          <div class="timeline-content">
            <div class="timeline-header">
              <h3>Deploy Production</h3>
              <span class="timeline-time">há 5 minutos</span>
            </div>
            <p class="timeline-description">Deployment concluído com sucesso</p>
            <div class="timeline-meta">
              <span class="meta-tag">Duração: 2m 34s</span>
              <span class="meta-tag">Branch: main</span>
            </div>
          </div>
        </div>

        <div class="timeline-item">
          <div class="timeline-marker success"></div>
          <div class="timeline-content">
            <div class="timeline-header">
              <h3>Database Backup</h3>
              <span class="timeline-time">há 2 horas</span>
            </div>
            <p class="timeline-description">Backup realizado com sucesso</p>
            <div class="timeline-meta">
              <span class="meta-tag">Duração: 1m 12s</span>
              <span class="meta-tag">Tamanho: 45.2 MB</span>
            </div>
          </div>
        </div>

        <div class="timeline-item">
          <div class="timeline-marker warning"></div>
          <div class="timeline-content">
            <div class="timeline-header">
              <h3>Data Processing</h3>
              <span class="timeline-time">há 4 horas</span>
            </div>
            <p class="timeline-description">Completado com avisos</p>
            <div class="timeline-meta">
              <span class="meta-tag">Duração: 5m 48s</span>
              <span class="meta-tag">Avisos: 3</span>
            </div>
          </div>
        </div>

        <div class="timeline-item">
          <div class="timeline-marker error"></div>
          <div class="timeline-content">
            <div class="timeline-header">
              <h3>Email Notifications</h3>
              <span class="timeline-time">há 6 horas</span>
            </div>
            <p class="timeline-description">Falha ao enviar notificações</p>
            <div class="timeline-meta">
              <span class="meta-tag">Erro: Connection timeout</span>
            </div>
          </div>
        </div>

        <div class="timeline-item">
          <div class="timeline-marker success"></div>
          <div class="timeline-content">
            <div class="timeline-header">
              <h3>Deploy Staging</h3>
              <span class="timeline-time">há 1 dia</span>
            </div>
            <p class="timeline-description">Deploy no ambiente de staging concluído</p>
            <div class="timeline-meta">
              <span class="meta-tag">Duração: 3m 21s</span>
              <span class="meta-tag">Branch: develop</span>
            </div>
          </div>
        </div>

        <div class="timeline-item">
          <div class="timeline-marker success"></div>
          <div class="timeline-content">
            <div class="timeline-header">
              <h3>Database Backup</h3>
              <span class="timeline-time">há 1 dia</span>
            </div>
            <p class="timeline-description">Backup realizado com sucesso</p>
            <div class="timeline-meta">
              <span class="meta-tag">Duração: 58s</span>
              <span class="meta-tag">Tamanho: 44.8 MB</span>
            </div>
          </div>
        </div>

        <div class="timeline-item">
          <div class="timeline-marker success"></div>
          <div class="timeline-content">
            <div class="timeline-header">
              <h3>Code Quality Check</h3>
              <span class="timeline-time">há 2 dias</span>
            </div>
            <p class="timeline-description">Análise de código concluída</p>
            <div class="timeline-meta">
              <span class="meta-tag">Duração: 4m 15s</span>
              <span class="meta-tag">Score: 98/100</span>
            </div>
          </div>
        </div>

        <div class="timeline-item">
          <div class="timeline-marker warning"></div>
          <div class="timeline-content">
            <div class="timeline-header">
              <h3>Security Scan</h3>
              <span class="timeline-time">há 3 dias</span>
            </div>
            <p class="timeline-description">Scan completo com alertas menores</p>
            <div class="timeline-meta">
              <span class="meta-tag">Duração: 8m 32s</span>
              <span class="meta-tag">Vulnerabilidades: 2 low</span>
            </div>
          </div>
        </div>
      </div>
    </div>
  `,
  styles: [`
    .container {
      max-width: 1000px;
      margin: 0 auto;
      padding: 3rem 2rem;
    }

    .page-header {
      margin-bottom: 2rem;
    }

    .page-title {
      font-size: 2rem;
      font-weight: 700;
      color: var(--text-primary);
      margin: 0 0 0.5rem 0;
    }

    .page-subtitle {
      color: var(--text-secondary);
      margin: 0;
    }

    .report-toolbar {
      display: flex;
      flex-wrap: wrap;
      gap: 1rem 2rem;
      align-items: flex-end;
      justify-content: space-between;
      margin-bottom: 2rem;
      padding: 1rem 1.5rem;
      background: var(--card-bg);
      border: 1px solid var(--border-color);
      border-radius: 12px;
    }

    .report-controls {
      display: flex;
      flex-direction: column;
      gap: 0.4rem;
      min-width: 260px;
    }

    .report-label {
      font-size: 0.9rem;
      color: var(--text-secondary);
    }

    .report-select {
      background: transparent;
      color: var(--text-primary);
      border: 1px solid var(--border-color);
      border-radius: 10px;
      padding: 0.6rem 0.8rem;
      font-size: 0.95rem;
    }

    .report-actions {
      display: flex;
      gap: 0.75rem;
    }

    .btn-primary,
    .btn-secondary {
      border: none;
      border-radius: 999px;
      padding: 0.55rem 1.2rem;
      font-weight: 600;
      cursor: pointer;
    }

    .btn-primary {
      background: #22c55e;
      color: #04100a;
    }

    .btn-secondary {
      background: transparent;
      border: 1px solid var(--border-color);
      color: var(--text-primary);
    }

    .history-timeline {
      position: relative;
      padding-left: 2rem;
    }

    .history-timeline::before {
      content: '';
      position: absolute;
      left: 0.5rem;
      top: 0;
      bottom: 0;
      width: 2px;
      background: var(--border-color);
    }

    .timeline-item {
      position: relative;
      padding-bottom: 2.5rem;
      display: flex;
      gap: 1.5rem;
    }

    .timeline-item:last-child {
      padding-bottom: 0;
    }

    .timeline-marker {
      position: absolute;
      left: -1.5rem;
      width: 1rem;
      height: 1rem;
      border-radius: 50%;
      border: 3px solid var(--bg-primary);
      z-index: 1;
    }

    .timeline-marker.success {
      background: #22c55e;
    }

    .timeline-marker.warning {
      background: #fbbf24;
    }

    .timeline-marker.error {
      background: #ef4444;
    }

    .timeline-content {
      background: var(--card-bg);
      border: 1px solid var(--border-color);
      border-radius: 12px;
      padding: 1.5rem;
      flex: 1;
      transition: all 0.3s;
    }

    .timeline-content:hover {
      transform: translateY(-2px);
      box-shadow: 0 8px 24px rgba(15, 23, 42, 0.2);
    }

    .timeline-header {
      display: flex;
      align-items: center;
      justify-content: space-between;
      margin-bottom: 0.5rem;
      gap: 1rem;
    }

    .timeline-header h3 {
      margin: 0;
      font-size: 1.1rem;
      color: var(--text-primary);
    }

    .timeline-time {
      font-size: 0.85rem;
      color: var(--text-secondary);
    }

    .timeline-description {
      color: var(--text-secondary);
      margin: 0 0 1rem 0;
    }

    .timeline-meta {
      display: flex;
      flex-wrap: wrap;
      gap: 0.5rem;
    }

    .meta-tag {
      background: rgba(148, 163, 184, 0.15);
      color: var(--text-secondary);
      padding: 0.35rem 0.6rem;
      border-radius: 999px;
      font-size: 0.8rem;
    }
  `]
})
export class HistoryComponent implements OnInit {
  availablePipelines: string[] = [];
  selectedPipelineId = '';

  constructor(private errorService: ErrorService) {}

  ngOnInit() {
    this.errorService.listPipelines().subscribe({
      next: (pipelines) => {
        this.availablePipelines = pipelines || [];
        if (!this.selectedPipelineId && this.availablePipelines.length > 0) {
          this.selectedPipelineId = this.availablePipelines[0];
        }
      },
      error: () => {
        this.availablePipelines = [];
      }
    });
  }

  openPreview() {
    if (!this.selectedPipelineId) return;
    const url = this.errorService.getPipelineReportPreviewUrl(this.selectedPipelineId);
    window.open(url, '_blank');
  }

  downloadReport() {
    if (!this.selectedPipelineId) return;
    const url = this.errorService.getPipelineReportUrl(this.selectedPipelineId);
    window.open(url, '_blank');
  }
}
