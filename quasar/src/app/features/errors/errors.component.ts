import { CommonModule } from '@angular/common';
import { ChangeDetectorRef, Component, OnInit } from '@angular/core';
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
import { ErrorViewerComponent } from '../../shared/components/error-viewer/error-viewer.component';
import { ErrorService, ErrorSummary, PipelineLog, PipelineStats } from '../../core/services/error.service';

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
  templateUrl: './errors.component.html',
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
    private snackBar: MatSnackBar,
    private cdr: ChangeDetectorRef
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
        this.cdr.detectChanges();
      },
      error: () => {
        this.snackBar.open('Erro ao carregar lista de pipelines', 'Fechar', {
          duration: 3000,
          panelClass: ['error-snackbar']
        });
        this.isLoading = false;
        this.cdr.detectChanges();
      }
    });
  }

  onPipelineChange(pipelineId: string) {
    this.selectedPipelineId = pipelineId;
    this.errorSummary = null;
    this.pipelineStats = null;
    this.pipelineLog = null;
    if (this.selectedPipelineId) {
      this.loadPipelineData();
    }
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
        this.cdr.detectChanges();
      },
      error: () => {
        this.snackBar.open('Erro ao carregar dados do pipeline', 'Fechar', {
          duration: 3000,
          panelClass: ['error-snackbar']
        });
        this.isLoading = false;
        this.cdr.detectChanges();
      }
    });
  }

  getStatusClass(status: string): string {
    const classes: { [key: string]: string } = {
      done: 'status-done',
      error: 'status-error',
      running: 'status-running',
      pending: 'status-pending'
    };
    return classes[status] || 'status-unknown';
  }

  getStatusLabel(status: string): string {
    const labels: { [key: string]: string } = {
      done: 'Concluido',
      error: 'Com Erro',
      running: 'Executando',
      pending: 'Pendente'
    };
    return labels[status] || 'Desconhecido';
  }

  getStatusIcon(status: string): string {
    const icons: { [key: string]: string } = {
      done: 'check_circle',
      error: 'error',
      running: 'play_circle',
      pending: 'schedule'
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
