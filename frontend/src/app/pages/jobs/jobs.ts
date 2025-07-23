import { Component } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import { MatListModule } from '@angular/material/list';
import { RouterModule } from '@angular/router';
import { AppState } from '../../services/app-state';
import { Job, JobService } from '../../services/job.service';
import { ProjectStatusService } from '../../services/project-status.service';
import { StatusService } from '../../services/status.service';
// Update the import path to the correct location if the file was moved or renamed
import { isRunning_, jobs_, updateJobsWithStatus } from '../../services/job-state.service';
import { Diagram } from "./diagram/diagram";

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
  templateUrl: './jobs.html',
  styleUrls: ['./jobs.scss']
})
export class Jobs {

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
    // Sempre aponta para as variáveis globais
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
          
          // Usa a nova função que preserva status
          updateJobsWithStatus(jobs);
          
          console.log('Jobs listados com sucesso:', jobs_);
        },
        error: (error) => {
          console.error('Erro ao listar projetos:', error);
        }
      });
    }

    this.statusService.listen().subscribe(statuses => {
      console.log('Status dos jobs recebidos:', statuses);
      
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
        
        if (job.status === 'done' || job.status === 'error') {
          this.isRunning = false;
          console.log(`Job ${job.jobName} concluído com status: ${job.status}`);
        } else if (job.status === 'running') {
          this.isRunning = true;
          console.log(`Job ${job.jobName} ainda está em execução.`);
        }
      });
    });

    this.projectStatusService.listen().subscribe(projectStatuses => {
      console.log('Status do projeto recebido:', projectStatuses);

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