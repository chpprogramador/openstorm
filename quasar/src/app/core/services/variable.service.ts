import { HttpClient } from '@angular/common/http';
import { Injectable } from '@angular/core';
import { Observable } from 'rxjs';
import { environment } from '../../../environments/environment';
import { Variable } from '../models/variable.model'; // Adjusted path

@Injectable({ providedIn: 'root' })
export class VariableService {
  private apiUrl = `${environment.apiUrl}/projects`;
  //private isBrowser: boolean;

  constructor(
    private http: HttpClient,
    //@Inject(PLATFORM_ID) private platformId: Object
  ) {
    //this.isBrowser = isPlatformBrowser(this.platformId);
  }

  /**
   * Lista todas as variáveis de um projeto
   */
  listVariables(projectId: string): Observable<Variable[]> {
     // if (this.isBrowser) {
       return this.http.get<Variable[]>(`${this.apiUrl}/${projectId}/variables`);
     // }
    // return of([]); // Return empty array during SSR
  }

  /**
   * Cria uma nova variável
   */
  createVariable(projectId: string, variable: Variable): Observable<Variable> {
    // if (this.isBrowser) {
      return this.http.post<Variable>(`${this.apiUrl}/${projectId}/variables`, variable);
    // }
    // return throwError(() => new Error('Operation not allowed during SSR'));
  }

  /**
   * Busca uma variável específica
   */
  getVariable(projectId: string, variableName: string): Observable<Variable> {
    //  if (this.isBrowser) {
      return this.http.get<Variable>(`${this.apiUrl}/${projectId}/variables/${variableName}`);
    //  }
    //  return throwError(() => new Error('Operation not allowed during SSR'));
  }

  /**
   * Atualiza uma variável existente
   */
  updateVariable(projectId: string, variableName: string, variable: Variable): Observable<Variable> {
    //  if (this.isBrowser) {
      return this.http.put<Variable>(`${this.apiUrl}/${projectId}/variables/${variableName}`, variable);
    //  }
    //  return throwError(() => new Error('Operation not allowed during SSR'));
  }

  /**
   * Exclui uma variável
   */
  deleteVariable(projectId: string, variableName: string): Observable<any> {
    //  if (this.isBrowser) {
      return this.http.delete(`${this.apiUrl}/${projectId}/variables/${variableName}`);
    //  }
    // return throwError(() => new Error('Operation not allowed during SSR'));
  }

  /**
   * Valida o valor da variável com base no tipo
   */
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

  /**
   * Formata o valor da variável para exibição
   */
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

  /**
   * Converte o valor para o tipo correto
   */
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
