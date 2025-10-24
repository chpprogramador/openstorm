// src/app/features/home/home.component.ts
import { Component, OnInit, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ProjectService } from '../../core/services/project.service';
import { Project } from '../../core/models/project.model';

@Component({
    selector: 'app-home',
    standalone: true,
    imports: [CommonModule],
    template: `
    <div class="container">
      <div class="page-header">
        <h1 class="page-title">Dashboard</h1>
        <p class="page-subtitle">Visão geral do projeto</p>
      </div>

      <div class="stats-grid">
        <div class="stat-card">
          <div class="stat-icon" style="background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);">
            <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="white" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <path d="M12 2v20M17 5H9.5a3.5 3.5 0 0 0 0 7h5a3.5 3.5 0 0 1 0 7H6"/>
            </svg>
          </div>
          <div class="stat-content">
            <div class="stat-value">24</div>
            <div class="stat-label">Variáveis</div>
          </div>
        </div>

        <div class="stat-card">
          <div class="stat-icon" style="background: linear-gradient(135deg, #f093fb 0%, #f5576c 100%);">
            <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="white" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <rect x="2" y="7" width="20" height="14" rx="2" ry="2"/>
              <path d="M16 21V5a2 2 0 0 0-2-2h-4a2 2 0 0 0-2 2v16"/>
            </svg>
          </div>
          <div class="stat-content">
            <div class="stat-value">12</div>
            <div class="stat-label">Jobs Ativos</div>
          </div>
        </div>

        <div class="stat-card">
          <div class="stat-icon" style="background: linear-gradient(135deg, #4facfe 0%, #00f2fe 100%);">
            <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="white" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <polyline points="22 12 18 12 15 21 9 3 6 12 2 12"/>
            </svg>
          </div>
          <div class="stat-content">
            <div class="stat-value">156</div>
            <div class="stat-label">Execuções</div>
          </div>
        </div>

        <div class="stat-card">
          <div class="stat-icon" style="background: linear-gradient(135deg, #43e97b 0%, #38f9d7 100%);">
            <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="white" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <polyline points="20 6 9 17 4 12"/>
            </svg>
          </div>
          <div class="stat-content">
            <div class="stat-value">98%</div>
            <div class="stat-label">Taxa de Sucesso</div>
          </div>
        </div>
      </div>

      <div class="content-grid">
        <div class="card">
          <div class="card-header">
            <h3>Atividade Recente</h3>
          </div>
          <div class="activity-list">
            <div class="activity-item">
              <div class="activity-icon success">
                <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                  <polyline points="20 6 9 17 4 12"/>
                </svg>
              </div>
              <div class="activity-content">
                <div class="activity-title">Job "Deploy Production" concluído</div>
                <div class="activity-time">há 5 minutos</div>
              </div>
            </div>

            <div class="activity-item">
              <div class="activity-icon info">
                <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                  <circle cx="12" cy="12" r="10"/>
                  <line x1="12" y1="16" x2="12" y2="12"/>
                  <line x1="12" y1="8" x2="12.01" y2="8"/>
                </svg>
              </div>
              <div class="activity-content">
                <div class="activity-title">Variável "API_KEY" atualizada</div>
                <div class="activity-time">há 15 minutos</div>
              </div>
            </div>

            <div class="activity-item">
              <div class="activity-icon warning">
                <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                  <path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"/>
                  <line x1="12" y1="9" x2="12" y2="13"/>
                  <line x1="12" y1="17" x2="12.01" y2="17"/>
                </svg>
              </div>
              <div class="activity-content">
                <div class="activity-title">Job "Database Backup" demorou mais que o esperado</div>
                <div class="activity-time">há 1 hora</div>
              </div>
            </div>

            <div class="activity-item">
              <div class="activity-icon success">
                <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                  <polyline points="20 6 9 17 4 12"/>
                </svg>
              </div>
              <div class="activity-content">
                <div class="activity-title">Novo job "Email Notifications" criado</div>
                <div class="activity-time">há 2 horas</div>
              </div>
            </div>
          </div>
        </div>

        <div class="card">
          <div class="card-header">
            <h3>Jobs em Execução</h3>
          </div>
          <div class="jobs-list">
            <div class="job-item">
              <div class="job-info">
                <div class="job-name">Data Processing</div>
                <div class="job-status">Processando...</div>
              </div>
              <div class="progress-bar">
                <div class="progress-fill" style="width: 65%"></div>
              </div>
            </div>

            <div class="job-item">
              <div class="job-info">
                <div class="job-name">Report Generation</div>
                <div class="job-status">Inicializando...</div>
              </div>
              <div class="progress-bar">
                <div class="progress-fill" style="width: 15%"></div>
              </div>
            </div>
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

    .stats-grid {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
      gap: 1.5rem;
      margin-bottom: 2rem;
    }

    .stat-card {
      background: var(--card-bg);
      border: 1px solid var(--border-color);
      border-radius: 12px;
      padding: 1.5rem;
      display: flex;
      align-items: center;
      gap: 1rem;
      transition: all 0.3s;
    }

    .stat-card:hover {
      transform: translateY(-4px);
      box-shadow: 0 8px 24px rgba(0, 0, 0, 0.1);
    }

    .stat-icon {
      width: 56px;
      height: 56px;
      border-radius: 12px;
      display: flex;
      align-items: center;
      justify-content: center;
      flex-shrink: 0;
    }

    .stat-content {
      flex: 1;
    }

    .stat-value {
      font-size: 1.75rem;
      font-weight: 700;
      color: var(--text-primary);
      line-height: 1;
      margin-bottom: 0.25rem;
    }

    .stat-label {
      font-size: 0.875rem;
      color: var(--text-secondary);
    }

    .content-grid {
      display: grid;
      grid-template-columns: 1fr 1fr;
      gap: 1.5rem;
    }

    .card {
      background: var(--card-bg);
      border: 1px solid var(--border-color);
      border-radius: 12px;
      overflow: hidden;
    }

    .card-header {
      padding: 1.5rem;
      border-bottom: 1px solid var(--border-color);
    }

    .card-header h3 {
      margin: 0;
      font-size: 1.125rem;
      font-weight: 600;
      color: var(--text-primary);
    }

    .activity-list {
      padding: 1rem;
    }

    .activity-item {
      display: flex;
      gap: 1rem;
      padding: 1rem;
      border-radius: 8px;
      transition: background 0.2s;
    }

    .activity-item:hover {
      background: var(--hover-bg);
    }

    .activity-icon {
      width: 36px;
      height: 36px;
      border-radius: 8px;
      display: flex;
      align-items: center;
      justify-content: center;
      flex-shrink: 0;
    }

    .activity-icon.success {
      background: rgba(67, 233, 123, 0.1);
      color: #43e97b;
    }

    .activity-icon.info {
      background: rgba(79, 172, 254, 0.1);
      color: #4facfe;
    }

    .activity-icon.warning {
      background: rgba(251, 191, 36, 0.1);
      color: #fbbf24;
    }

    .activity-content {
      flex: 1;
    }

    .activity-title {
      font-weight: 500;
      color: var(--text-primary);
      margin-bottom: 0.25rem;
    }

    .activity-time {
      font-size: 0.875rem;
      color: var(--text-secondary);
    }

    .jobs-list {
      padding: 1.5rem;
      display: flex;
      flex-direction: column;
      gap: 1.5rem;
    }

    .job-item {
      display: flex;
      flex-direction: column;
      gap: 0.75rem;
    }

    .job-info {
      display: flex;
      justify-content: space-between;
      align-items: center;
    }

    .job-name {
      font-weight: 600;
      color: var(--text-primary);
    }

    .job-status {
      font-size: 0.875rem;
      color: var(--text-secondary);
    }

    .progress-bar {
      height: 8px;
      background: var(--hover-bg);
      border-radius: 4px;
      overflow: hidden;
    }

    .progress-fill {
      height: 100%;
      background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
      border-radius: 4px;
      transition: width 0.3s ease;
    }

    @media (max-width: 1024px) {
      .content-grid {
        grid-template-columns: 1fr;
      }
    }

    @media (max-width: 768px) {
      .container {
        padding: 2rem 1rem;
      }

      .stats-grid {
        grid-template-columns: 1fr;
      }
    }
  `]
})
export class HomeComponent implements OnInit {
    private projectService = inject(ProjectService);
    project: Project | null = null;

    ngOnInit(): void {
        this.project = this.projectService.getSelectedProject();
    }
}