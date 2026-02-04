import { Component } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import { MatListModule } from '@angular/material/list';
import { RouterModule } from '@angular/router';
import { AppState } from '../../core/services/app-state';
import { Job, JobService } from '../../core/services/job.service';
import { ProjectStatusService } from '../../core/services/project-status.service';
import { StatusService } from '../../core/services/status.service';
import { isRunning_, jobs_, updateJobsWithStatus } from '../../core/services/job-state.service';
import { Diagram } from './diagram/diagram.component';

export interface JobExtended extends Job {
  total?: number;
  processed?: number;
  progress?: number;
  status?: 'pending' | 'running' | 'done' | 'error';
  startedAt?: string;
  endedAt?: string;
  error?: string;
}

@Component({
  standalone: true,
  selector: 'app-jobs',
  imports: [
    RouterModule,
    MatListModule,
    MatFormFieldModule,
    MatInputModule,
    FormsModule,
    Diagram
  ],
  template: `<app-diagram [jobs]="jobs" [project]="appState.project" [isRunning]="isRunning"></app-diagram>`,
  styleUrls: ['./jobs.component.scss']
})
export class JobsComponent {
  jobs: JobExtended[] = jobs_;
  isRunning = isRunning_;
  selectedJob: JobExtended | null = null;

  constructor(
    private jobservice: JobService,
    public appState: AppState,
    public statusService: StatusService,
    private projectStatusService: ProjectStatusService
  ) {}

  ngOnInit() {
    this.jobs = jobs_;
    this.isRunning = isRunning_;

    const projectId = this.appState.project?.id;
    if (projectId) {
      this.jobservice.listJobs(projectId).subscribe({
        next: (jobs) => {
          if (!Array.isArray(jobs) || jobs.length === 0) {
            jobs_.length = 0;
            return;
          }

          updateJobsWithStatus(jobs);
        },
        error: (error) => {
          console.error('Erro ao listar projetos:', error);
        }
      });
    }

    this.statusService.listen().subscribe(statuses => {
      interface JobStatus {
        id: string;
        progress?: number;
        status?: 'pending' | 'running' | 'done' | 'error';
        startedAt?: string;
        endedAt?: string;
        error?: string;
        total?: number;
        processed?: number;
      }

      jobs_.forEach((job: JobExtended) => {
        const status: JobStatus | undefined = statuses.find((s: JobStatus) => s.id === job.id);
        if (status) {
          job.progress = status.progress;
          job.status = status.status;
          job.startedAt = status.startedAt;
          job.endedAt = status.endedAt;
          job.error = status.error;
          job.total = status.total;
          job.processed = status.processed;
        }
      });

      const hasRunning = jobs_.some((job) => job.status === 'running');
      this.isRunning = hasRunning;
    });

    this.projectStatusService.listen().subscribe(projectStatuses => {
      if (projectStatuses.status === 'running') {
        this.isRunning = true;
      } else {
        this.isRunning = false;
      }
    });
  }

  job_click(job: any) {
    this.selectedJob = job;
  }
}
