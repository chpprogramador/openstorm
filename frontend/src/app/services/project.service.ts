import { HttpClient } from '@angular/common/http';
import { Injectable } from '@angular/core';
import { Observable } from 'rxjs';
import { environment } from '../../environments/environment';


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
  private apiUrl = `${environment.apiUrl}/projects`;

  constructor(private http: HttpClient) {}

  /**
   * Cria um novo projeto
   */
  createProject(project: Partial<Project>): Observable<Project> {
    return this.http.post<Project>(`${this.apiUrl}`, project);
  }

  /**
   * Busca um projeto por ID
   */
  getProject(id: string): Observable<Project> {
    return this.http.get<Project>(`${this.apiUrl}/${id}`);
  }

  /**
   * Fecha um projeto (operação fictícia por enquanto)
   */
  closeProject(id: string): Observable<any> {
    return this.http.post(`${this.apiUrl}/${id}/close`, {});
  }

  /**
   * Lista todos os projetos
   */
  listProjects(): Observable<Project[]> {
    return this.http.get<Project[]>(this.apiUrl);
  }
}
