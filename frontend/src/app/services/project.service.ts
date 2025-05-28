import { HttpClient } from '@angular/common/http';
import { Injectable } from '@angular/core';
import { Observable } from 'rxjs';

export interface DatabaseConfig {
  type: string;
  host: string;
  port: number;
  user: string;
  password: string;
  database: string;
}

export interface Project {
  id: string;
  projectName: string;
  jobs: string[];
  sourceDatabase: DatabaseConfig;
  destinationDatabase: DatabaseConfig;
}

@Injectable({ providedIn: 'root' })
export class ProjectService {
  private apiUrl = 'http://localhost:8080/projects';

  constructor(private http: HttpClient) {}

  createProject(project: Partial<Project>): Observable<Project> {
    return this.http.post<Project>(`${this.apiUrl}`, project);
  }

  getProject(id: string): Observable<Project> {
    return this.http.get<Project>(`${this.apiUrl}/${id}`);
  }

  closeProject(id: string): Observable<any> {
    return this.http.post(`${this.apiUrl}/${id}/close`, {});
  }
}
