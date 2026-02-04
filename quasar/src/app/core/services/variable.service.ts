import { HttpClient } from '@angular/common/http';
import { Injectable } from '@angular/core';
import { Observable } from 'rxjs';
import { environment } from '../../../environments/environment';
import { Variable } from '../models/project.model';

@Injectable({ providedIn: 'root' })
export class VariableService {
  private apiUrl = `${environment.apiUrl}/projects`;

  constructor(private http: HttpClient) {}

  listVariables(projectId: string): Observable<Variable[]> {
    return this.http.get<Variable[]>(`${this.apiUrl}/${projectId}/variables`);
  }

  createVariable(projectId: string, variable: Variable): Observable<Variable> {
    return this.http.post<Variable>(`${this.apiUrl}/${projectId}/variables`, variable);
  }

  getVariable(projectId: string, variableName: string): Observable<Variable> {
    return this.http.get<Variable>(`${this.apiUrl}/${projectId}/variables/${variableName}`);
  }

  updateVariable(projectId: string, variableName: string, variable: Variable): Observable<Variable> {
    return this.http.put<Variable>(`${this.apiUrl}/${projectId}/variables/${variableName}`, variable);
  }

  deleteVariable(projectId: string, variableName: string): Observable<any> {
    return this.http.delete(`${this.apiUrl}/${projectId}/variables/${variableName}`);
  }

  validateVariableValue(value: string, type: string): boolean {
    switch (type) {
      case 'number':
        return !isNaN(Number(value)) && value !== '';
      case 'boolean':
        return value.toLowerCase() === 'true' || value.toLowerCase() === 'false';
      case 'date':
        return !isNaN(Date.parse(value));
      case 'string':
      default:
        return true;
    }
  }

  formatVariableValue(value: string, type: string): string {
    switch (type) {
      case 'boolean':
        return value.toLowerCase() === 'true' ? 'Verdadeiro' : 'Falso';
      case 'date':
        const date = new Date(value);
        return date.toLocaleDateString('pt-BR');
      default:
        return value;
    }
  }

  convertValue(value: string, type: string): any {
    switch (type) {
      case 'number':
        return Number(value);
      case 'boolean':
        return value.toLowerCase() === 'true';
      case 'date':
        return new Date(value).toISOString().split('T')[0];
      default:
        return value;
    }
  }
}
