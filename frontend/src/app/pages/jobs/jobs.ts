import { Component } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import { MatListModule } from '@angular/material/list';
import { RouterModule } from '@angular/router';
import { AppState } from '../../services/app-state';
import { Job, JobService } from '../../services/job.service';
import { Diagram } from "./diagram/diagram";

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

  jobs: Job[] = [];
  selectedJob: Job | null = null;

  constructor(
    private jobservice: JobService,
    public appState: AppState
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
    } //else {
      //console.error('Project ID is undefined. Cannot list jobs.');
    //}
  }

  job_click(job: any) {
    this.selectedJob = job;
  }


}
