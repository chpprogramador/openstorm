import { CommonModule, isPlatformBrowser } from '@angular/common';
import { Component, OnDestroy, OnInit, Inject, PLATFORM_ID, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { Subscription } from 'rxjs';
import { finalize } from 'rxjs/operators';
import { ProjectService } from '../../core/services/project.service';
import {
  BenchmarkRun,
  BenchmarkService,
  BenchmarkSummary
} from '../../core/services/benchmark.service';

@Component({
  selector: 'app-benchmark',
  standalone: true,
  imports: [CommonModule, FormsModule],
  templateUrl: './benchmark.component.html',
  styleUrls: ['./benchmark.component.scss']
})
export class BenchmarkComponent implements OnInit, OnDestroy {
  private selectedProjectIdSig = signal<string | null>(null);
  private historySig = signal<BenchmarkSummary[]>([]);
  private latestBenchmarkSig = signal<BenchmarkRun | null>(null);
  private selectedBenchmarkSig = signal<BenchmarkRun | null>(null);
  private selectedRunIdSig = signal<string>('');
  private isLoadingSig = signal<boolean>(false);
  private isLoadingDetailsSig = signal<boolean>(false);
  private isRunningSig = signal<boolean>(false);
  private loadErrorSig = signal<string | null>(null);

  get selectedProjectId(): string | null {
    return this.selectedProjectIdSig();
  }
  set selectedProjectId(value: string | null) {
    this.selectedProjectIdSig.set(value);
  }

  get history(): BenchmarkSummary[] {
    return this.historySig();
  }
  set history(value: BenchmarkSummary[]) {
    this.historySig.set(value);
  }

  get latestBenchmark(): BenchmarkRun | null {
    return this.latestBenchmarkSig();
  }
  set latestBenchmark(value: BenchmarkRun | null) {
    this.latestBenchmarkSig.set(value);
  }

  get selectedBenchmark(): BenchmarkRun | null {
    return this.selectedBenchmarkSig();
  }
  set selectedBenchmark(value: BenchmarkRun | null) {
    this.selectedBenchmarkSig.set(value);
  }

  get selectedRunId(): string {
    return this.selectedRunIdSig();
  }
  set selectedRunId(value: string) {
    this.selectedRunIdSig.set(value);
  }

  get isLoading(): boolean {
    return this.isLoadingSig();
  }
  set isLoading(value: boolean) {
    this.isLoadingSig.set(value);
  }

  get isLoadingDetails(): boolean {
    return this.isLoadingDetailsSig();
  }
  set isLoadingDetails(value: boolean) {
    this.isLoadingDetailsSig.set(value);
  }

  get isRunning(): boolean {
    return this.isRunningSig();
  }
  set isRunning(value: boolean) {
    this.isRunningSig.set(value);
  }

  get loadError(): string | null {
    return this.loadErrorSig();
  }
  set loadError(value: string | null) {
    this.loadErrorSig.set(value);
  }
  private projectSub?: Subscription;
  private isBrowser: boolean;
  private runFallbackTimer: ReturnType<typeof setTimeout> | null = null;
  private runPollTimer: ReturnType<typeof setInterval> | null = null;

  constructor(
    @Inject(PLATFORM_ID) platformId: any,
    private projectService: ProjectService,
    private benchmarkService: BenchmarkService
  ) {
    this.isBrowser = isPlatformBrowser(platformId);
  }

  ngOnInit(): void {
    this.projectSub = this.projectService.selectedProject$.subscribe((project) => {
      this.selectedProjectId = project?.id ?? null;
      this.resetState();
      if (this.selectedProjectId) {
        this.loadHistory(this.selectedProjectId);
      }
      this.syncView();
    });
  }

  ngOnDestroy(): void {
    this.projectSub?.unsubscribe();
    this.clearRunFallback();
    this.clearRunPoll();
  }

  runBenchmark(): void {
    if (!this.selectedProjectId || this.isRunning) return;
    const previousLatestId = this.history[0]?.run_id;
    this.isRunning = true;
    this.loadError = null;
    this.startRunFallback(previousLatestId);
    this.syncView();
    this.benchmarkService
      .runBenchmark(this.selectedProjectId)
      .pipe(finalize(() => {
        this.isRunning = false;
        this.clearRunFallback();
        this.syncView();
      }))
      .subscribe({
        next: (run) => {
          const payload: any = run;
          const resolved: BenchmarkRun = payload?.data ?? payload?.benchmark ?? payload;
          if (!resolved?.run_id) {
            this.loadError = 'Benchmark retornou um payload inesperado.';
            this.syncView();
            return;
          }
          this.latestBenchmark = resolved;
          this.selectedBenchmark = resolved;
          this.selectedRunId = resolved.run_id;
          this.upsertHistory(resolved);
          this.saveCache();
          this.clearRunPoll();
          this.loadError = null;
          this.syncView();
        },
        error: () => {
          this.loadError = 'Erro ao executar benchmark. Tente novamente.';
          this.syncView();
        }
      });
  }

  onHistoryChange(runId: string): void {
    this.selectedRunId = runId;
    if (!this.selectedRunId) {
      this.selectedBenchmark = null;
      this.syncView();
      return;
    }
    const summary = this.history.find(item => item.run_id === this.selectedRunId);
    if (summary && this.selectedProjectId) {
      this.selectedBenchmark = this.buildPlaceholderRun(summary, this.selectedProjectId);
      this.syncView();
    }
    if (this.latestBenchmark?.run_id === this.selectedRunId) {
      this.selectedBenchmark = this.latestBenchmark;
      this.syncView();
      return;
    }
    this.loadBenchmark(this.selectedRunId, 'selected');
  }

  downloadSelectedPdf(): void {
    if (!this.selectedProjectId || !this.selectedRunId) return;
    const url = this.benchmarkService.getBenchmarkReportUrl(this.selectedProjectId, this.selectedRunId);
    window.open(url, '_blank');
  }

  formatDateTime(dateString?: string | null): string {
    if (!dateString) return '—';
    const date = new Date(dateString);
    if (Number.isNaN(date.getTime())) return '—';
    return date.toLocaleString('pt-BR');
  }

  formatDuration(startedAt?: string | null, endedAt?: string | null): string {
    if (!startedAt || !endedAt) return '—';
    const start = new Date(startedAt).getTime();
    const end = new Date(endedAt).getTime();
    if (Number.isNaN(start) || Number.isNaN(end) || end < start) return '—';
    let diff = Math.floor((end - start) / 1000);
    const days = Math.floor(diff / 86400);
    diff -= days * 86400;
    const hours = Math.floor(diff / 3600);
    diff -= hours * 3600;
    const minutes = Math.floor(diff / 60);
    const seconds = diff - minutes * 60;
    if (days > 0) return `${days}d ${hours}h ${minutes}m`;
    if (hours > 0) return `${hours}h ${minutes}m ${seconds}s`;
    if (minutes > 0) return `${minutes}m ${seconds}s`;
    return `${seconds}s`;
  }

  formatBytes(value?: number | null): string {
    if (value === null || value === undefined || Number.isNaN(value)) return '—';
    const units = ['B', 'KB', 'MB', 'GB', 'TB'];
    let size = value;
    let unitIndex = 0;
    while (size >= 1024 && unitIndex < units.length - 1) {
      size /= 1024;
      unitIndex += 1;
    }
    return `${size.toFixed(size >= 10 || unitIndex === 0 ? 0 : 1)} ${units[unitIndex]}`;
  }

  formatScore(score?: number | null): string {
    if (score === null || score === undefined || Number.isNaN(score)) return '—';
    return score.toFixed(1);
  }

  formatLatency(value?: number | null): string {
    if (value === null || value === undefined || Number.isNaN(value)) return '—';
    return `${value.toFixed(1)} ms`;
  }

  formatQps(value?: number | null): string {
    if (value === null || value === undefined || Number.isNaN(value)) return '—';
    return `${value.toFixed(1)} qps`;
  }

  statusLabel(status?: string | null): string {
    if (!status) return 'Sem dados';
    if (status === 'ok') return 'Sucesso';
    if (status === 'partial') return 'Parcial';
    if (status === 'error') return 'Erro';
    return status;
  }

  statusClass(status?: string | null): string {
    if (!status) return 'status-neutral';
    if (status === 'ok') return 'status-ok';
    if (status === 'partial') return 'status-partial';
    if (status === 'error') return 'status-error';
    return 'status-neutral';
  }

  scoreClass(score?: number | null): string {
    if (score === null || score === undefined || Number.isNaN(score)) return 'score-neutral';
    if (score >= 80) return 'score-healthy';
    if (score >= 60) return 'score-attention';
    if (score >= 40) return 'score-risk';
    return 'score-critical';
  }

  scoreLabel(score?: number | null): string {
    if (score === null || score === undefined || Number.isNaN(score)) return 'Sem score';
    if (score >= 80) return 'Saudável';
    if (score >= 60) return 'Médio';
    if (score >= 40) return 'Ruim';
    return 'Crítico';
  }

  summaryLabel(summary: BenchmarkSummary): string {
    const date = this.formatDateTime(summary.started_at);
    const status = this.statusLabel(summary.status);
    return `${date} • ${status}`;
  }

  shortId(id?: string | null): string {
    if (!id) return '—';
    return id.length > 8 ? `${id.slice(0, 8)}...` : id;
  }

  get activeBenchmark(): BenchmarkRun | null {
    return this.selectedBenchmark ?? this.latestBenchmark;
  }

  isLatestActive(): boolean {
    return !!this.activeBenchmark && !!this.latestBenchmark && this.activeBenchmark.run_id === this.latestBenchmark.run_id;
  }

  private resetState(): void {
    this.history = [];
    this.latestBenchmark = null;
    this.selectedBenchmark = null;
    this.selectedRunId = '';
    this.isLoading = false;
    this.isLoadingDetails = false;
    this.isRunning = false;
    this.loadError = null;
  }

  private loadHistory(projectId: string): void {
    this.isLoading = true;
    this.loadError = null;
    this.restoreCache(projectId);
    this.syncView();
    this.benchmarkService
      .listBenchmarks(projectId, 10)
      .pipe(finalize(() => {
        this.isLoading = false;
        this.syncView();
      }))
      .subscribe({
        next: (items) => {
          const anyItems: any = items;
          const list = Array.isArray(anyItems)
            ? anyItems
            : Array.isArray(anyItems?.items)
              ? anyItems.items
              : Array.isArray(anyItems?.data)
                ? anyItems.data
                : Array.isArray(anyItems?.benchmarks)
                  ? anyItems.benchmarks
                  : [];
          const sorted = [...list].sort((a, b) => {
            return new Date(b.started_at).getTime() - new Date(a.started_at).getTime();
          });
          this.history = sorted;
          const latestId = this.history[0]?.run_id;
          if (!latestId) {
            this.latestBenchmark = null;
            this.selectedBenchmark = null;
            this.selectedRunId = '';
            this.saveCache(projectId);
            this.syncView();
            return;
          }

          if (!this.selectedRunId || !this.history.some(item => item.run_id === this.selectedRunId)) {
            this.selectedRunId = latestId;
          }

          if (!this.latestBenchmark || this.latestBenchmark.run_id !== latestId) {
            const summary = this.history.find(item => item.run_id === latestId);
            if (summary) {
              this.latestBenchmark = this.buildPlaceholderRun(summary, projectId);
              if (this.selectedRunId === latestId) {
                this.selectedBenchmark = this.latestBenchmark;
              }
            }
          }

          this.loadBenchmark(latestId, 'latest');
          if (this.selectedRunId && this.selectedRunId !== latestId) {
            this.loadBenchmark(this.selectedRunId, 'selected');
          }

          this.saveCache(projectId);
          this.syncView();
        },
        error: () => {
          this.history = [];
          this.latestBenchmark = null;
          this.selectedBenchmark = null;
          this.selectedRunId = '';
          this.loadError = 'Erro ao carregar histórico de benchmarks.';
          this.saveCache(projectId);
          this.syncView();
        }
      });
  }

  private loadBenchmark(runId: string, target: 'latest' | 'selected'): void {
    if (!this.selectedProjectId || !runId) return;
    this.isLoadingDetails = true;
    this.syncView();
    this.benchmarkService.getBenchmark(this.selectedProjectId, runId).subscribe({
      next: (run) => {
        const payload: any = run;
        const resolved: BenchmarkRun = payload?.data ?? payload?.benchmark ?? payload;
        if (target === 'selected' && resolved.run_id !== this.selectedRunId) {
          this.isLoadingDetails = false;
          this.syncView();
          return;
        }
        if (target === 'latest') {
          this.latestBenchmark = resolved;
          if (this.selectedRunId === resolved.run_id) {
            this.selectedBenchmark = resolved;
          }
        } else {
          this.selectedBenchmark = resolved;
        }
        this.isLoadingDetails = false;
        this.saveCache();
        this.syncView();
      },
      error: () => {
        this.isLoadingDetails = false;
        this.syncView();
      }
    });
  }

  private upsertHistory(run: BenchmarkRun): void {
    const summary: BenchmarkSummary = {
      run_id: run.run_id,
      status: run.status,
      started_at: run.started_at,
      ended_at: run.ended_at,
      scores: run.scores
    };
    this.history = [summary, ...this.history.filter(item => item.run_id !== run.run_id)];
    this.saveCache();
    this.syncView();
  }

  private buildPlaceholderRun(summary: BenchmarkSummary, projectId: string): BenchmarkRun {
    return {
      run_id: summary.run_id,
      project_id: projectId,
      status: summary.status,
      started_at: summary.started_at,
      ended_at: summary.ended_at,
      options: {
        probe_iterations: 0,
        enable_write_probe: false,
        include_host: true,
        include_origin: true,
        include_destination: true
      },
      metrics: {},
      scores: summary.scores || {}
    };
  }

  private cacheKey(projectId: string): string {
    return `quasar_benchmark_cache_${projectId}`;
  }

  private restoreCache(projectId: string): void {
    if (!this.isBrowser) return;
    const raw = localStorage.getItem(this.cacheKey(projectId));
    if (!raw) return;
    try {
      const parsed = JSON.parse(raw);
      const history = Array.isArray(parsed?.history) ? parsed.history : [];
      this.history = history;
      this.latestBenchmark = parsed?.latestBenchmark ?? null;
      this.selectedRunId = typeof parsed?.selectedRunId === 'string' ? parsed.selectedRunId : '';
      if (this.latestBenchmark && !this.selectedRunId) {
        this.selectedRunId = this.latestBenchmark.run_id;
      }
      if (this.latestBenchmark && this.selectedRunId === this.latestBenchmark.run_id) {
        this.selectedBenchmark = this.latestBenchmark;
      }
    } catch {
      localStorage.removeItem(this.cacheKey(projectId));
    }
  }

  private saveCache(explicitProjectId?: string): void {
    if (!this.isBrowser) return;
    const projectId = explicitProjectId ?? this.selectedProjectId;
    if (!projectId) return;
    const payload = {
      history: this.history,
      latestBenchmark: this.latestBenchmark,
      selectedRunId: this.selectedRunId
    };
    localStorage.setItem(this.cacheKey(projectId), JSON.stringify(payload));
  }

  private startRunFallback(previousLatestId?: string): void {
    this.clearRunFallback();
    this.runFallbackTimer = setTimeout(() => {
      this.isRunning = false;
      this.loadError = 'Benchmark em execução no servidor. Atualizando histórico...';
      if (this.selectedProjectId) {
        this.pollForLatest(this.selectedProjectId, previousLatestId);
      }
      this.syncView();
    }, 15000);
  }

  private syncView(): void {
    // no-op: signals trigger change detection in zoneless mode
  }

  private clearRunFallback(): void {
    if (this.runFallbackTimer) {
      clearTimeout(this.runFallbackTimer);
      this.runFallbackTimer = null;
    }
  }

  private pollForLatest(projectId: string, previousLatestId?: string): void {
    this.clearRunPoll();
    let attempts = 0;
    this.runPollTimer = setInterval(() => {
      attempts += 1;
      this.benchmarkService.listBenchmarks(projectId, 1).subscribe({
        next: (items) => {
          const list = Array.isArray(items) ? items : [];
          const latest = list[0];
          if (latest?.run_id && latest.run_id !== previousLatestId) {
            this.history = [latest, ...this.history.filter(item => item.run_id !== latest.run_id)];
            this.selectedRunId = latest.run_id;
            this.selectedBenchmark = this.buildPlaceholderRun(latest, projectId);
            this.loadBenchmark(latest.run_id, 'latest');
            this.loadError = null;
            this.clearRunPoll();
          }
        },
        error: () => {
          // keep polling a few times
        }
      });

      if (attempts >= 8) {
        this.clearRunPoll();
      }
    }, 5000);
  }

  private clearRunPoll(): void {
    if (this.runPollTimer) {
      clearInterval(this.runPollTimer);
      this.runPollTimer = null;
    }
  }
}
