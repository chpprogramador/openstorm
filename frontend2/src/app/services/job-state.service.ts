import { Injectable } from '@angular/core';
import { BehaviorSubject } from 'rxjs';
import { Job } from './job.service';
// import { JobExtended } from "../pages/jobs/jobs";

export var jobs_: JobExtended[] = [];
export var isRunning_ = false;

// Estende Job com propriedades adicionais para status e posicionamento
export interface JobExtended extends Job {
  // Propriedades para posicionamento no diagrama
  left: number;
  top: number;
  
  // Propriedades para status de execução
  total?: number;
  processed?: number;
  progress?: number;
  status?: 'pending' | 'running' | 'done' | 'error';
  startedAt?: string;
  endedAt?: string;
  error?: string;
}

@Injectable({
  providedIn: 'root'
})
export class JobStateService {
  private jobsSubject = new BehaviorSubject<JobExtended[]>([]);
  private isRunningSubject = new BehaviorSubject<boolean>(false);

  jobs$ = this.jobsSubject.asObservable();
  isRunning$ = this.isRunningSubject.asObservable();

  updateJobs(jobs: Job[]) {
    // Converte Job[] para JobExtended[] preservando as propriedades originais
    const extendedJobs: JobExtended[] = jobs.map(job => ({
      ...job,
      total: undefined,
      processed: undefined,
      progress: undefined,
      status: 'pending' as const,
      startedAt: undefined,
      endedAt: undefined,
      error: undefined
    }));
    this.jobsSubject.next(extendedJobs);
  }

  updateJob(jobId: string, updates: Partial<JobExtended>) {
    const currentJobs = this.jobsSubject.value;
    const updatedJobs = currentJobs.map(job => 
      job.id === jobId ? { ...job, ...updates } : job
    );
    this.jobsSubject.next(updatedJobs);
  }

  updateRunningState(isRunning: boolean) {
    this.isRunningSubject.next(isRunning);
  }

  addJob(job: Job) {
    const currentJobs = this.jobsSubject.value;
    const extendedJob: JobExtended = {
      ...job,
      total: undefined,
      processed: undefined,
      progress: undefined,
      status: 'pending' as const,
      startedAt: undefined,
      endedAt: undefined,
      error: undefined
    };
    this.jobsSubject.next([...currentJobs, extendedJob]);
  }

  removeJob(jobId: string) {
    const currentJobs = this.jobsSubject.value;
    const filteredJobs = currentJobs.filter(job => job.id !== jobId);
    this.jobsSubject.next(filteredJobs);
  }

  get jobs() { 
    return this.jobsSubject.value; 
  }

  get isRunning() { 
    return this.isRunningSubject.value; 
  }

  clearJobs() {
    this.jobsSubject.next([]);
  }

  
}

export function updateJobsWithStatus(newJobs: any[]) {
  const updatedJobs = newJobs.map(newJob => {
    const existingJob = jobs_.find(j => j.id === newJob.id);
    if (existingJob) {
      // Preserva status do job existente
      return {
        ...newJob,
        total: existingJob.total,
        processed: existingJob.processed,
        progress: existingJob.progress,
        status: existingJob.status,
        startedAt: existingJob.startedAt,
        endedAt: existingJob.endedAt,
        error: existingJob.error
      };
    }
    // Job novo, sem status ainda
    return {
      ...newJob,
      status: 'pending'
    };
  });
  
  jobs_.length = 0;
  jobs_.push(...updatedJobs);
}