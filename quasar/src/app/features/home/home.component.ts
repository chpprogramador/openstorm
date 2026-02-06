import { ChangeDetectorRef, Component, OnDestroy, OnInit, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ProjectService } from '../../core/services/project.service';
import { Project } from '../../core/models/project.model';
import { ErrorService, ErrorSummary, PipelineLog, PipelineStats } from '../../core/services/error.service';
import { Subscription, forkJoin, of } from 'rxjs';
import { catchError, switchMap, tap } from 'rxjs/operators';

@Component({
    selector: 'app-home',
    standalone: true,
    imports: [CommonModule],
    template: `
    <div class="container">
      <div class="page-header">
        <div>
          <p class="kicker">Projeto</p>
          <h1 class="page-title">Dashboard</h1>
          <p class="page-subtitle">Visão geral do projeto com foco em resultados e qualidade</p>
        </div>
      </div>

      <div class="summary-card" *ngIf="!project">
        <div class="summary-empty">
          <h3>Nenhum projeto selecionado</h3>
          <p>Volte para a tela inicial e selecione um projeto para ver os indicadores.</p>
        </div>
      </div>

      <ng-container *ngIf="project">
        <div class="stats-grid" *ngIf="isLoading">
          <div class="stat-card skeleton"></div>
          <div class="stat-card skeleton"></div>
          <div class="stat-card skeleton"></div>
          <div class="stat-card skeleton"></div>
        </div>

        <div class="stats-grid" *ngIf="!isLoading">
          <div class="stat-card highlight">
            <div class="stat-label"><span class="icon-badge"></span>Taxa de Sucesso</div>
            <div class="stat-value">{{ successRate }}%</div>
            <div class="stat-meta">
              {{ doneJobs }} de {{ totalJobs }} jobs concluídos
            </div>
          </div>

          <div class="stat-card">
            <div class="stat-label"><span class="icon-badge warm"></span>Taxa de Erro</div>
            <div class="stat-value">{{ errorRate }}%</div>
            <div class="stat-meta">
              {{ errorJobsCount }} jobs com erro
            </div>
          </div>

          <div class="stat-card">
            <div class="stat-label"><span class="icon-badge cool"></span>Registros Processados</div>
            <div class="stat-value">{{ totalProcessed }}</div>
            <div class="stat-meta">Última execução</div>
          </div>

          <div class="stat-card">
            <div class="stat-label"><span class="icon-badge soft"></span>Erros Detectados</div>
            <div class="stat-value">{{ totalErrors }}</div>
            <div class="stat-meta">Última execução</div>
          </div>
        </div>

        <div class="content-grid">
          <div class="card">
            <div class="card-header">
              <h3><span class="title-icon"></span>Qualidade da Execução</h3>
              <span class="pill muted">Última execução</span>
            </div>
            <div class="card-body">
              <div class="chart-row">
                <div class="donut" [style.background]="donutStyle">
                  <div class="donut-center">
                    <strong>{{ successRate }}%</strong>
                    <span>Sucesso</span>
                  </div>
                </div>
                <div class="legend">
                  <div class="legend-item success">
                    <span class="legend-dot"></span>
                    <span>Sucesso: {{ doneJobs }}</span>
                  </div>
                  <div class="legend-item error">
                    <span class="legend-dot"></span>
                    <span>Erro: {{ errorJobsCount }}</span>
                  </div>
                  <div class="legend-item pending">
                    <span class="legend-dot"></span>
                    <span>Pendente: {{ pendingJobsCount }}</span>
                  </div>
                </div>
              </div>
              <div class="info-row">
                <span>Pipeline</span>
                <strong>{{ latestPipelineLabel || '—' }}</strong>
              </div>
              <div class="info-row">
                <span>Início</span>
                <strong>{{ latestStats ? formatDateTime(latestStats.started_at) : '—' }}</strong>
              </div>
              <div class="info-row">
                <span>Duração</span>
                <strong>{{ formatDuration(latestStats?.duration, latestStats?.started_at, latestStats?.ended_at) }}</strong>
              </div>
            </div>
          </div>

          <div class="card">
            <div class="card-header">
              <h3><span class="title-icon warn"></span>Jobs com Erro</h3>
              <span class="pill muted">{{ errorJobs.length }} itens</span>
            </div>
            <div class="card-body">
              <div class="empty" *ngIf="errorJobs.length === 0">
                Nenhum erro registrado na última execução.
              </div>
              <ul class="error-list" *ngIf="errorJobs.length > 0">
                <li *ngFor="let job of errorJobs">
                  <span class="dot error"></span>
                  <div>
                    <strong>{{ job.job_name }}</strong>
                    <p>{{ job.error || 'Erro não detalhado' }}</p>
                  </div>
                </li>
              </ul>
            </div>
          </div>

          <div class="card full">
            <div class="card-header">
              <h3><span class="title-icon cool"></span>Execuções Recentes</h3>
              <span class="pill muted">{{ trendRuns.length }} execuções</span>
            </div>
            <div class="card-body">
              <div class="empty" *ngIf="trendRuns.length === 0">Sem histórico suficiente.</div>
              <ul class="runs-list" *ngIf="trendRuns.length > 0">
                <li *ngFor="let run of trendRuns">
                  <div class="run-main">
                    <div>
                      <div class="run-title">{{ run.label }}</div>
                      <div class="run-meta">{{ run.dateLabel }}</div>
                    </div>
                    <div class="run-stats">
                      <span class="stat-pill success">Sucesso {{ run.successRate }}%</span>
                      <span class="stat-pill error" *ngIf="run.errorRate > 0">Erro {{ run.errorRate }}%</span>
                      <span class="stat-pill" *ngIf="run.totalProcessed !== null">{{ run.totalProcessed }} registros</span>
                      <span class="stat-pill" *ngIf="run.duration">{{ formatDuration(run.duration, run.started_at, run.ended_at) }}</span>
                      <span class="stat-pill neutral">{{ run.statusLabel }}</span>
                    </div>
                  </div>
                  <div class="trend-bar">
                    <div class="trend-fill" [class.error]="run.errorRate > 0" [style.width.%]="run.successRate"></div>
                  </div>
                </li>
              </ul>
            </div>
          </div>
        </div>
      </ng-container>
    </div>
  `,
    styles: [`
    .container {
      max-width: 1400px;
      margin: 0 auto;
      padding: 3rem 2rem;
    }

    .page-header {
      margin-bottom: 2rem;
      display: flex;
      justify-content: space-between;
      align-items: flex-start;
      gap: 2rem;
    }

    .kicker {
      text-transform: uppercase;
      letter-spacing: 0.24em;
      color: var(--text-muted);
      font-size: 0.7rem;
      font-weight: 600;
      margin-bottom: 0.5rem;
    }

    .page-title {
      font-size: 2.2rem;
      font-weight: 700;
      color: var(--text-primary);
      margin: 0 0 0.5rem 0;
    }

    .page-subtitle {
      color: var(--text-secondary);
      margin: 0;
    }

    .summary-card {
      background: var(--card-bg);
      border: 1px solid var(--border-color);
      border-radius: 16px;
      padding: 2rem;
      text-align: center;
      color: var(--text-secondary);
    }

    .stats-grid {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(240px, 1fr));
      gap: 1.25rem;
      margin-bottom: 1.75rem;
    }

    .stat-card {
      background: var(--card-bg);
      border: 1px solid var(--border-color);
      border-radius: 16px;
      padding: 1.5rem;
      display: flex;
      flex-direction: column;
      gap: 0.75rem;
      box-shadow: var(--shadow-sm);
    }

    .stat-card.highlight {
      background: linear-gradient(135deg, rgba(34, 211, 238, 0.16), rgba(99, 102, 241, 0.12));
      border-color: rgba(34, 211, 238, 0.35);
    }

    .stat-card.skeleton {
      min-height: 140px;
      background: linear-gradient(120deg, rgba(148, 163, 184, 0.08), rgba(148, 163, 184, 0.18), rgba(148, 163, 184, 0.08));
      background-size: 200% 100%;
      animation: shimmer 1.4s ease infinite;
    }

    .stat-label {
      font-size: 0.8rem;
      text-transform: uppercase;
      letter-spacing: 0.12em;
      color: var(--text-muted);
      display: inline-flex;
      align-items: center;
      gap: 0.5rem;
    }

    .icon-badge {
      width: 10px;
      height: 10px;
      border-radius: 999px;
      background: rgba(34, 211, 238, 0.9);
      box-shadow: 0 0 8px rgba(34, 211, 238, 0.5);
    }

    .icon-badge.warm {
      background: rgba(248, 113, 113, 0.9);
      box-shadow: 0 0 8px rgba(248, 113, 113, 0.5);
    }

    .icon-badge.cool {
      background: rgba(96, 165, 250, 0.9);
      box-shadow: 0 0 8px rgba(96, 165, 250, 0.5);
    }

    .icon-badge.soft {
      background: rgba(148, 163, 184, 0.9);
      box-shadow: 0 0 8px rgba(148, 163, 184, 0.4);
    }

    .stat-value {
      font-size: 1.8rem;
      font-weight: 700;
      color: var(--text-primary);
    }

    .stat-meta {
      font-size: 0.85rem;
      color: var(--text-secondary);
    }

    .content-grid {
      display: grid;
      grid-template-columns: repeat(2, minmax(0, 1fr));
      gap: 1.5rem;
    }

    .card {
      background: var(--card-bg);
      border: 1px solid var(--border-color);
      border-radius: 16px;
      overflow: hidden;
      box-shadow: var(--shadow-sm);
    }

    .card.full {
      grid-column: 1 / -1;
    }

    .card-header {
      padding: 1.25rem 1.5rem;
      border-bottom: 1px solid var(--border-color);
      display: flex;
      align-items: center;
      justify-content: space-between;
      gap: 1rem;
    }

    .card-header h3 {
      margin: 0;
      font-size: 1.125rem;
      font-weight: 600;
      color: var(--text-primary);
      display: inline-flex;
      align-items: center;
      gap: 0.5rem;
    }

    .title-icon {
      width: 10px;
      height: 10px;
      border-radius: 999px;
      background: rgba(99, 102, 241, 0.8);
      box-shadow: 0 0 8px rgba(99, 102, 241, 0.5);
    }

    .title-icon.warn {
      background: rgba(248, 113, 113, 0.9);
      box-shadow: 0 0 8px rgba(248, 113, 113, 0.5);
    }

    .title-icon.cool {
      background: rgba(34, 211, 238, 0.8);
      box-shadow: 0 0 8px rgba(34, 211, 238, 0.5);
    }

    .card-body {
      padding: 1.5rem;
      display: flex;
      flex-direction: column;
      gap: 1rem;
    }

    .pill {
      padding: 0.35rem 0.75rem;
      border-radius: 999px;
      font-size: 0.75rem;
      font-weight: 600;
      background: rgba(148, 163, 184, 0.2);
      color: var(--text-primary);
    }

    .pill.muted {
      color: var(--text-secondary);
      background: rgba(148, 163, 184, 0.15);
    }

    .info-row {
      display: flex;
      justify-content: space-between;
      gap: 1rem;
      color: var(--text-secondary);
    }

    .info-row strong {
      color: var(--text-primary);
      font-weight: 600;
    }

    .chart-row {
      display: grid;
      grid-template-columns: 140px 1fr;
      gap: 1.25rem;
      align-items: center;
    }

    .donut {
      width: 140px;
      height: 140px;
      border-radius: 50%;
      display: grid;
      place-items: center;
      position: relative;
      background: conic-gradient(#22c55e 0% 60%, rgba(248, 113, 113, 0.85) 60% 80%, rgba(59, 130, 246, 0.6) 80% 100%);
      border: 1px solid rgba(148, 163, 184, 0.25);
      box-shadow: inset 0 0 20px rgba(0,0,0,0.2);
    }

    .donut::after {
      content: '';
      position: absolute;
      inset: 18px;
      background: var(--card-bg);
      border-radius: 50%;
    }

    .donut-center {
      position: relative;
      z-index: 1;
      text-align: center;
      color: var(--text-primary);
      display: grid;
      gap: 0.25rem;
    }

    .donut-center strong {
      font-size: 1.4rem;
    }

    .donut-center span {
      font-size: 0.75rem;
      color: var(--text-secondary);
    }

    .legend {
      display: grid;
      gap: 0.5rem;
      color: var(--text-secondary);
    }

    .legend-item {
      display: inline-flex;
      align-items: center;
      gap: 0.5rem;
      font-size: 0.85rem;
    }

    .legend-dot {
      width: 10px;
      height: 10px;
      border-radius: 999px;
      background: rgba(148, 163, 184, 0.5);
    }

    .legend-item.success .legend-dot {
      background: #22c55e;
    }

    .legend-item.error .legend-dot {
      background: #f87171;
    }

    .legend-item.pending .legend-dot {
      background: #60a5fa;
    }

    .error-list {
      list-style: none;
      padding: 0;
      margin: 0;
      display: grid;
      gap: 0.75rem;
    }

    .error-list li {
      display: grid;
      grid-template-columns: auto 1fr;
      gap: 0.75rem;
      align-items: start;
      padding: 0.75rem 1rem;
      border-radius: 12px;
      background: rgba(248, 113, 113, 0.08);
      border: 1px solid rgba(248, 113, 113, 0.2);
    }

    .error-list p {
      margin: 0.25rem 0 0;
      color: var(--text-secondary);
      font-size: 0.85rem;
    }

    .dot {
      width: 10px;
      height: 10px;
      border-radius: 999px;
      margin-top: 0.35rem;
    }

    .dot.error {
      background: #f87171;
      box-shadow: 0 0 8px rgba(248, 113, 113, 0.5);
    }

    .runs-list {
      list-style: none;
      padding: 0;
      margin: 0;
      display: grid;
      gap: 0.9rem;
    }

    .runs-list li {
      padding: 1rem;
      border-radius: 14px;
      border: 1px solid var(--border-color);
      background: rgba(15, 23, 42, 0.35);
      display: grid;
      gap: 0.75rem;
    }

    .run-main {
      display: flex;
      justify-content: space-between;
      gap: 1rem;
      align-items: center;
    }

    .run-stats {
      display: flex;
      flex-wrap: wrap;
      gap: 0.5rem;
      justify-content: flex-end;
    }

    .stat-pill {
      padding: 0.25rem 0.6rem;
      border-radius: 999px;
      font-size: 0.72rem;
      background: rgba(148, 163, 184, 0.2);
      color: var(--text-primary);
      font-weight: 600;
    }

    .stat-pill.success {
      background: rgba(34, 197, 94, 0.2);
      color: #22c55e;
    }

    .stat-pill.error {
      background: rgba(248, 113, 113, 0.2);
      color: #f87171;
    }

    .stat-pill.neutral {
      background: rgba(99, 102, 241, 0.18);
      color: #a5b4fc;
    }

    .run-title {
      font-weight: 600;
      color: var(--text-primary);
    }

    .run-meta {
      font-size: 0.85rem;
      color: var(--text-secondary);
    }

    .empty {
      color: var(--text-secondary);
      font-size: 0.9rem;
    }

    .trend-bar {
      height: 8px;
      background: rgba(148, 163, 184, 0.2);
      border-radius: 999px;
      overflow: hidden;
    }

    .trend-fill {
      height: 100%;
      border-radius: 999px;
      background: rgba(34, 197, 94, 0.85);
      transition: width 0.3s ease;
    }

    .trend-fill.error {
      background: rgba(248, 113, 113, 0.85);
    }

    .summary-empty h3 {
      margin: 0 0 0.5rem 0;
      color: var(--text-primary);
    }

    @keyframes shimmer {
      0% { background-position: 0% 0%; }
      100% { background-position: -200% 0%; }
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

      .page-header {
        flex-direction: column;
        align-items: flex-start;
      }

      .stats-grid {
        grid-template-columns: 1fr;
      }

      .chart-row {
        grid-template-columns: 1fr;
      }

      .run-main {
        flex-direction: column;
        align-items: flex-start;
      }

      .run-stats {
        justify-content: flex-start;
      }
    }
  `]
})
export class HomeComponent implements OnInit, OnDestroy {
    private projectService = inject(ProjectService);
    private errorService = inject(ErrorService);
    private cdr = inject(ChangeDetectorRef);

    project: Project | null = null;
    latestStats: PipelineStats | null = null;
    latestLog: PipelineLog | null = null;
    latestErrors: ErrorSummary | null = null;
    latestPipelineLabel = '';
    recentRuns: Array<{ id: string; label: string; dateLabel: string; sortKey: number }> = [];
    trendRuns: Array<{ id: string; label: string; dateLabel: string; successRate: number; errorRate: number; totalProcessed: number | null; duration: string | null; statusLabel: string; started_at?: string | null; ended_at?: string | null }> = [];
    doneJobs = 0;
    totalJobs = 0;
    successRate = 0;
    errorRate = 0;
    totalProcessed = 0;
    totalErrors = 0;
    errorJobsCount = 0;
    pendingJobsCount = 0;
    errorJobs: Array<{ job_name: string; error?: string }> = [];
    isLoading = false;
    donutStyle = 'conic-gradient(#22c55e 0% 100%)';
    private projectSub?: Subscription;

    ngOnInit(): void {
        this.projectSub = this.projectService.selectedProject$
          .pipe(
            tap((project) => {
              this.project = project;
              this.resetDashboard();
              this.cdr.detectChanges();
            }),
            switchMap((project) => {
              if (!project?.id) {
                return of(null);
              }
              this.isLoading = true;
              return this.errorService.listPipelines(project.id).pipe(
                catchError(() => of([]))
              );
            })
          )
          .subscribe((pipelines) => {
            if (!this.project?.id) return;
            const runs = this.normalizePipelines(pipelines || []);
            this.recentRuns = runs.slice(0, 6);
            this.loadTrend(runs.slice(0, 8));
            const latest = runs[0]?.id;
            if (!latest) {
              this.isLoading = false;
              this.cdr.detectChanges();
              return;
            }
            this.latestPipelineLabel = runs[0].label;
            forkJoin({
              stats: this.errorService.getPipelineStats(latest).pipe(catchError(() => of(null))),
              log: this.errorService.getPipelineLog(latest).pipe(catchError(() => of(null))),
              errors: this.errorService.getErrorSummary(latest).pipe(catchError(() => of(null)))
            }).subscribe(({ stats, log, errors }) => {
              this.latestStats = stats as PipelineStats | null;
              this.latestLog = log as PipelineLog | null;
              this.latestErrors = errors as ErrorSummary | null;
              this.hydrateMetrics();
              this.isLoading = false;
              this.cdr.detectChanges();
            });
          });
    }

    ngOnDestroy(): void {
      this.projectSub?.unsubscribe();
    }

    private resetDashboard() {
      this.latestStats = null;
      this.latestLog = null;
      this.latestErrors = null;
      this.latestPipelineLabel = '';
      this.recentRuns = [];
      this.trendRuns = [];
      this.doneJobs = 0;
      this.totalJobs = 0;
      this.successRate = 0;
      this.errorRate = 0;
      this.totalProcessed = 0;
      this.totalErrors = 0;
      this.errorJobsCount = 0;
      this.pendingJobsCount = 0;
      this.errorJobs = [];
      this.isLoading = false;
      this.donutStyle = 'conic-gradient(#22c55e 0% 100%)';
    }

    private hydrateMetrics() {
      const stats = this.latestStats;
      if (stats) {
        this.totalJobs = stats.total_jobs || 0;
        const done = stats.job_stats?.['done'] || 0;
        const error = stats.job_stats?.['error'] || 0;
        const pending = stats.job_stats?.['pending'] || 0;
        this.doneJobs = done;
        this.errorJobsCount = error;
        this.pendingJobsCount = pending;
        this.successRate = this.totalJobs > 0 ? Math.round((done / this.totalJobs) * 100) : 0;
        this.errorRate = this.totalJobs > 0 ? Math.round((error / this.totalJobs) * 100) : 0;
        this.totalProcessed = stats.total_processed || 0;
        const successAngle = this.successRate;
        const errorAngle = this.errorRate;
        this.donutStyle = `conic-gradient(#22c55e 0% ${successAngle}%, rgba(248, 113, 113, 0.85) ${successAngle}% ${successAngle + errorAngle}%, rgba(59, 130, 246, 0.6) ${successAngle + errorAngle}% 100%)`;
      }
      const errors = this.latestErrors;
      this.totalErrors = errors?.total_errors || 0;
      this.errorJobs = (errors?.error_jobs || []).slice(0, 5).map(job => ({
        job_name: job.job_name,
        error: job.error
      }));
    }

    private loadTrend(runs: Array<{ id: string; label: string; dateLabel: string }>) {
      if (!runs.length) {
        this.trendRuns = [];
        return;
      }
      forkJoin(
        runs.map(run =>
          this.errorService.getPipelineStats(run.id).pipe(catchError(() => of(null)))
        )
      ).subscribe((statsList) => {
        this.trendRuns = runs.map((run, index) => {
          const stats = statsList[index] as PipelineStats | null;
          const totalJobs = stats?.total_jobs || 0;
          const done = stats?.job_stats?.['done'] || 0;
          const error = stats?.job_stats?.['error'] || 0;
          const successRate = totalJobs > 0 ? Math.round((done / totalJobs) * 100) : 0;
          const errorRate = totalJobs > 0 ? Math.round((error / totalJobs) * 100) : 0;
          return {
            id: run.id,
            label: run.label,
            dateLabel: run.dateLabel,
            successRate,
            errorRate,
            totalProcessed: stats?.total_processed ?? null,
            duration: stats?.duration ?? null,
            statusLabel: this.mapStatus(stats?.status || ''),
            started_at: stats?.started_at ?? null,
            ended_at: stats?.ended_at ?? null
          };
        });
        this.cdr.detectChanges();
      });
    }

    private normalizePipelines(pipelines: string[]) {
      return (pipelines || [])
        .map((id) => {
          const match = id.match(/(\d{4})-(\d{2})-(\d{2})_(\d{2})-(\d{2})-(\d{2})/);
          const sortKey = match ? Date.parse(`${match[1]}-${match[2]}-${match[3]}T${match[4]}:${match[5]}:${match[6]}`) : 0;
          const cleaned = id
            .replace(/^pipeline[_-]?/i, '')
            .replace(/_/g, ' ')
            .replace(/\s+/g, ' ')
            .trim();
          const label = match
            ? `${cleaned.replace(match[0].replace(/_/g, ' '), '').trim()}`
            : cleaned;
          const dateLabel = match
            ? `${match[3]}/${match[2]}/${match[1]} ${match[4]}:${match[5]}`
            : '';
          return { id, label: label || id, dateLabel, sortKey };
        })
        .sort((a, b) => b.sortKey - a.sortKey);
    }

    private mapStatus(status: string): string {
      switch (status) {
        case 'done':
          return 'OK';
        case 'error':
          return 'Falha';
        case 'running':
          return 'Ativo';
        case 'pending':
          return 'Pendente';
        default:
          return 'Indefinido';
      }
    }

    formatDuration(value?: string | null, startedAt?: string | null, endedAt?: string | null): string {
      const resolved = this.computeDurationFromDates(startedAt, endedAt);
      if (resolved !== null) {
        return resolved;
      }
      if (!value) return '—';
      if (value.includes(':')) {
        const parts = value.split(':').map(p => p.trim());
        if (parts.length === 3) {
          return parts.map(p => p.padStart(2, '0')).join(':');
        }
      }
      const h = /([0-9]+)h/.exec(value);
      const m = /([0-9]+)m/.exec(value);
      const s = /([0-9]+(?:\.[0-9]+)?)s/.exec(value);
      const hours = h ? parseInt(h[1], 10) : 0;
      const minutes = m ? parseInt(m[1], 10) : 0;
      const seconds = s ? Math.floor(parseFloat(s[1])) : 0;
      const totalSeconds = hours * 3600 + minutes * 60 + seconds;
      return this.formatSeconds(totalSeconds);
    }

    private computeDurationFromDates(startedAt?: string | null, endedAt?: string | null): string | null {
      if (!startedAt || !endedAt) return null;
      const start = new Date(startedAt).getTime();
      const end = new Date(endedAt).getTime();
      if (Number.isNaN(start) || Number.isNaN(end) || end < start) return null;
      const totalSeconds = Math.floor((end - start) / 1000);
      return this.formatSeconds(totalSeconds);
    }

    private formatSeconds(totalSeconds: number): string {
      const safeSeconds = Math.max(0, totalSeconds);
      const hh = Math.floor(safeSeconds / 3600).toString().padStart(2, '0');
      const mm = Math.floor((safeSeconds % 3600) / 60).toString().padStart(2, '0');
      const ss = Math.floor(safeSeconds % 60).toString().padStart(2, '0');
      return `${hh}:${mm}:${ss}`;
    }

    formatDateTime(value: string): string {
      if (!value) return '—';
      const date = new Date(value);
      return date.toLocaleString('pt-BR');
    }
}
