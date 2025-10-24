// src/app/features/history/history.component.ts
import { Component } from '@angular/core';
import { CommonModule } from '@angular/common';

@Component({
  selector: 'app-history',
  standalone: true,
  imports: [CommonModule],
  template: `
    <div class="container">
      <div class="page-header">
        <div>
          <h1 class="page-title">Histórico</h1>
          <p class="page-subtitle">Visualize o histórico de execuções</p>
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
      margin-bottom: 3rem;
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
      transform: translateX(4px);
      box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
    }

    .timeline-header {
      display: flex;
      justify-content: space-between;
      align-items: center;
      margin-bottom: 0.5rem;
    }

    .timeline-header h3 {
      font-size: 1.125rem;
      font-weight: 600;
      color: var(--text-primary);
      margin: 0;
    }

    .timeline-time {
      font-size: 0.875rem;
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
      background: var(--hover-bg);
      padding: 0.25rem 0.75rem;
      border-radius: 6px;
      font-size: 0.8rem;
      color: var(--text-secondary);
    }

    @media (max-width: 768px) {
      .container {
        padding: 2rem 1rem;
      }

      .timeline-header {
        flex-direction: column;
        align-items: flex-start;
        gap: 0.25rem;
      }
    }
  `]
})
export class HistoryComponent { }