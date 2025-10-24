// src/app/features/jobs/jobs.component.ts
import { Component } from '@angular/core';
import { CommonModule } from '@angular/common';

@Component({
    selector: 'app-jobs',
    standalone: true,
    imports: [CommonModule],
    template: `
    <div class="container">
      <div class="page-header">
        <div>
          <h1 class="page-title">Jobs</h1>
          <p class="page-subtitle">Gerencie e monitore seus jobs</p>
        </div>
        <button class="btn-primary">
          <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <line x1="12" y1="5" x2="12" y2="19"/>
            <line x1="5" y1="12" x2="19" y2="12"/>
          </svg>
          Novo Job
        </button>
      </div>

      <div class="jobs-grid">
        <div class="job-card">
          <div class="job-header">
            <div>
              <h3 class="job-title">Deploy Production</h3>
              <p class="job-description">Deploy automático para ambiente de produção</p>
            </div>
            <span class="status-badge success">Ativo</span>
          </div>
          <div class="job-meta">
            <div class="meta-item">
              <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <circle cx="12" cy="12" r="10"/>
                <polyline points="12 6 12 12 16 14"/>
              </svg>
              <span>Última execução: há 2 horas</span>
            </div>
            <div class="meta-item">
              <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <polyline points="22 12 18 12 15 21 9 3 6 12 2 12"/>
              </svg>
              <span>156 execuções</span>
            </div>
          </div>
          <div class="job-actions">
            <button class="btn-secondary btn-sm">
              <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <polygon points="5 3 19 12 5 21 5 3"/>
              </svg>
              Executar
            </button>
            <button class="btn-icon">
              <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/>
                <path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/>
              </svg>
            </button>
          </div>
        </div>

        <div class="job-card">
          <div class="job-header">
            <div>
              <h3 class="job-title">Database Backup</h3>
              <p class="job-description">Backup diário do banco de dados</p>
            </div>
            <span class="status-badge success">Ativo</span>
          </div>
          <div class="job-meta">
            <div class="meta-item">
              <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <circle cx="12" cy="12" r="10"/>
                <polyline points="12 6 12 12 16 14"/>
              </svg>
              <span>Última execução: há 8 horas</span>
            </div>
            <div class="meta-item">
              <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <polyline points="22 12 18 12 15 21 9 3 6 12 2 12"/>
              </svg>
              <span>89 execuções</span>
            </div>
          </div>
          <div class="job-actions">
            <button class="btn-secondary btn-sm">
              <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <polygon points="5 3 19 12 5 21 5 3"/>
              </svg>
              Executar
            </button>
            <button class="btn-icon">
              <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/>
                <path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/>
              </svg>
            </button>
          </div>
        </div>

        <div class="job-card">
          <div class="job-header">
            <div>
              <h3 class="job-title">Email Notifications</h3>
              <p class="job-description">Envio de notificações por email</p>
            </div>
            <span class="status-badge inactive">Inativo</span>
          </div>
          <div class="job-meta">
            <div class="meta-item">
              <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <circle cx="12" cy="12" r="10"/>
                <polyline points="12 6 12 12 16 14"/>
              </svg>
              <span>Última execução: há 3 dias</span>
            </div>
            <div class="meta-item">
              <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <polyline points="22 12 18 12 15 21 9 3 6 12 2 12"/>
              </svg>
              <span>42 execuções</span>
            </div>
          </div>
          <div class="job-actions">
            <button class="btn-secondary btn-sm">
              <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <polygon points="5 3 19 12 5 21 5 3"/>
              </svg>
              Executar
            </button>
            <button class="btn-icon">
              <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/>
                <path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/>
              </svg>
            </button>
          </div>
        </div>
      </div>
    </div>
  `,
    styles: [`
    .container {
      max-width: 1400px;
      margin: 0 auto;
      padding: 3rem 2rem;
    }

    .page-header {
      display: flex;
      justify-content: space-between;
      align-items: flex-start;
      margin-bottom: 2.5rem;
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

    .jobs-grid {
      display: grid;
      grid-template-columns: repeat(auto-fill, minmax(350px, 1fr));
      gap: 1.5rem;
    }

    .job-card {
      background: var(--card-bg);
      border: 1px solid var(--border-color);
      border-radius: 12px;
      padding: 1.5rem;
      transition: all 0.3s;
      display: flex;
      flex-direction: column;
      gap: 1.5rem;
    }

    .job-card:hover {
      transform: translateY(-4px);
      box-shadow: 0 8px 24px rgba(0, 0, 0, 0.1);
    }

    .job-header {
      display: flex;
      justify-content: space-between;
      align-items: flex-start;
      gap: 1rem;
    }

    .job-title {
      font-size: 1.25rem;
      font-weight: 600;
      color: var(--text-primary);
      margin: 0 0 0.5rem 0;
    }

    .job-description {
      color: var(--text-secondary);
      margin: 0;
      font-size: 0.9rem;
    }

    .status-badge {
      padding: 0.375rem 0.75rem;
      border-radius: 6px;
      font-size: 0.75rem;
      font-weight: 600;
      text-transform: uppercase;
      flex-shrink: 0;
    }

    .status-badge.success {
      background: rgba(34, 197, 94, 0.1);
      color: #22c55e;
    }

    .status-badge.inactive {
      background: rgba(156, 163, 175, 0.1);
      color: #9ca3af;
    }

    .job-meta {
      display: flex;
      flex-direction: column;
      gap: 0.75rem;
      padding: 1rem;
      background: var(--hover-bg);
      border-radius: 8px;
    }

    .meta-item {
      display: flex;
      align-items: center;
      gap: 0.5rem;
      color: var(--text-secondary);
      font-size: 0.875rem;
    }

    .meta-item svg {
      flex-shrink: 0;
    }

    .job-actions {
      display: flex;
      gap: 0.75rem;
      align-items: center;
    }

    .btn-primary {
      background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
      color: white;
      border: none;
      padding: 0.75rem 1.5rem;
      border-radius: 8px;
      font-weight: 600;
      cursor: pointer;
      display: flex;
      align-items: center;
      gap: 0.5rem;
      transition: all 0.2s;
    }

    .btn-primary:hover {
      transform: translateY(-2px);
      box-shadow: 0 4px 12px rgba(102, 126, 234, 0.4);
    }

    .btn-secondary {
      background: var(--card-bg);
      color: var(--text-primary);
      border: 1px solid var(--border-color);
      padding: 0.75rem 1.5rem;
      border-radius: 8px;
      font-weight: 600;
      cursor: pointer;
      transition: all 0.2s;
      display: flex;
      align-items: center;
      gap: 0.5rem;
      flex: 1;
      justify-content: center;
    }

    .btn-secondary:hover {
      background: var(--hover-bg);
    }

    .btn-sm {
      padding: 0.5rem 1rem;
      font-size: 0.875rem;
    }

    .btn-icon {
      background: transparent;
      border: none;
      padding: 0.5rem;
      cursor: pointer;
      border-radius: 6px;
      color: var(--text-secondary);
      display: flex;
      align-items: center;
      justify-content: center;
      transition: all 0.2s;
    }

    .btn-icon:hover {
      background: var(--hover-bg);
      color: var(--text-primary);
    }

    @media (max-width: 768px) {
      .container {
        padding: 2rem 1rem;
      }

      .page-header {
        flex-direction: column;
        gap: 1rem;
      }

      .jobs-grid {
        grid-template-columns: 1fr;
      }
    }
  `]
})
export class JobsComponent { }