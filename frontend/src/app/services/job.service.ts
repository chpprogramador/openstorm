import { HttpClient } from '@angular/common/http';
import { Injectable } from '@angular/core';
import { Observable } from 'rxjs';
import { environment } from '../../environments/environment';

export interface Job {
  id: string;
  jobName: string;
  selectSql: string;
  insertSql: string;
  recordsPerPage: number;
  concurrency: number;
}

@Injectable({ providedIn: 'root' })
export class JobService {
  private apiUrl = `${environment.apiUrl}/projects`;

  constructor(private http: HttpClient) {}

  listJobs(projectId: string): Observable<Job[]> {
    return this.http.get<Job[]>(`${this.apiUrl}/${projectId}/jobs`);
  }

  addJob(projectId: string, job: Partial<Job>): Observable<Job> {
    return this.http.post<Job>(`${this.apiUrl}/${projectId}/jobs`, job);
  }

  updateJob(projectId: string, jobId: string, job: Job): Observable<Job> {
    return this.http.put<Job>(`${this.apiUrl}/${projectId}/jobs/${jobId}`, job);
  }

  deleteJob(projectId: string, jobId: string): Observable<any> {
    return this.http.delete(`${this.apiUrl}/${projectId}/jobs/${jobId}`);
  }
}
