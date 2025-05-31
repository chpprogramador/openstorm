import { Component } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import { MatListModule } from '@angular/material/list';
import { RouterModule } from '@angular/router';
import { AppState } from '../../services/app-state';
import { JobService } from '../../services/job.service';

@Component({
  standalone: true,
  selector: 'app-jobs',
  imports: [
    RouterModule,
    MatListModule,
    MatFormFieldModule,
    MatInputModule,
    FormsModule
  ],
  templateUrl: './jobs.html',
  styleUrls: ['./jobs.scss']
})
export class Jobs {

  jobs: any[] = [];
  selectedJob: any;

  constructor(
    private jobservice: JobService,
    private appState: AppState
  ) {}

  ngOnInit() {
    this.jobservice.listJobs(this.appState.projectID).subscribe({
      next: (jobs) => {
        this.jobs = jobs;
        console.log('Jobs listados com sucesso:', this.jobs);
      }
      , error: (error) => {
        console.error('Erro ao listar projetos:', error);
      }
    });
  }

  job_click(job: any) {
    this.selectedJob = job;
  }


}
