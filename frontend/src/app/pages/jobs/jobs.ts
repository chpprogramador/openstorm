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

  

  jobs: JobExtended[] = [];
  isRunning = false;
  selectedJob: JobExtended | null = null;


  constructor(
    private jobservice: JobService,
    public appState: AppState,
    public statusService: StatusService,
    private projectStatusService: ProjectStatusService
  ) {}

  ngOnInit() {
    const projectId = this.appState.project?.id;
    if (projectId) {
      this.jobservice.listJobs(projectId).subscribe({
        next: (jobs) => {
          if (!Array.isArray(jobs) || jobs.length === 0) {
            this.jobs = [];
            return;
          }
          this.jobs = jobs;

          console.log('Jobs listados com sucesso:', this.jobs);
        },
        error: (error) => {
          console.error('Erro ao listar projetos:', error);
        }
      });
    }

    

    this.statusService.listen().subscribe(statuses => {
      console.log('Status dos jobs recebidos:', statuses);
      this.jobs.forEach(job => {
        const status = statuses.find(s => s.id === job.id);
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
        } else {
          this.isRunning = true;
          console.log(`Job ${job.jobName} ainda está em execução.`);
        }

      });

      this.jobs.forEach(job => {
        if (job.status === 'running') {
          console.log(`Job ${job.jobName} está em execução.`);
        } else if (job.status === 'done') {
          console.log(`Job ${job.jobName} foi concluído com sucesso.`);
        } else if (job.status === 'error') {
          console.error(`Job ${job.jobName} falhou: ${job.error}`);
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
