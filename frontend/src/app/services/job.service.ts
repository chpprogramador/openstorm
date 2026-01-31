import { HttpClient } from '@angular/common/http';
import { Injectable } from '@angular/core';
import { Observable } from 'rxjs';
import { environment } from '../../environments/environment';

export interface Job {
  id: string;
  jobName: string;
  selectSql: string;
  insertSql: string;
  posInsertSql?: string;
  columns: string[];
  recordsPerPage: number;
  "type": string;
  stopOnError: boolean;
  top: number;
  left: number;
}

// type ValidateJobRequest struct {
// 	SelectSQL string `json:"selectSQL"`
// 	InsertSQL string `json:"insertSQL"`
// 	Limit     int    `json:"limit"`
// 	ProjectID string `json:"projectId"`
// }

export interface ValidateJob {
  selectSQL: string;
  insertSQL: string;
  limit: number;
  projectId: string;
}


@Injectable({ providedIn: 'root' })
export class JobService {
  private apiUrl = `${environment.apiUrl}`;

  constructor(private http: HttpClient) {}

  listJobs(projectId: string): Observable<Job[]> {
    return this.http.get<Job[]>(`${this.apiUrl}/projects/${projectId}/jobs`);
  }

  addJob(projectId: string, job: Partial<Job>): Observable<Job> {
    return this.http.post<Job>(`${this.apiUrl}/projects/${projectId}/jobs`, job);
  }

  updateJob(projectId: string, jobId: string, job: Job): Observable<Job> {
    return this.http.put<Job>(`${this.apiUrl}/projects/${projectId}/jobs/${jobId}`, job);
  }

  deleteJob(projectId: string, jobId: string): Observable<any> {
    return this.http.delete(`${this.apiUrl}/projects/${projectId}/jobs/${jobId}`);
  }

  validate(validateJob: ValidateJob): Observable<Job> {
    return this.http.post<Job>(`${this.apiUrl}/jobs/validate`, validateJob);
  }
}
