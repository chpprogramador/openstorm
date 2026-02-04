import { HttpClient } from '@angular/common/http';
import { Injectable } from '@angular/core';
import { Observable } from 'rxjs';
import { environment } from '../../../environments/environment';
import { VisualElement } from '../models/visual-element.model';

@Injectable({ providedIn: 'root' })
export class VisualElementService {
  private apiUrl = `${environment.apiUrl}`;

  constructor(private http: HttpClient) {}

  list(projectId: string): Observable<VisualElement[]> {
    return this.http.get<VisualElement[]>(`${this.apiUrl}/projects/${projectId}/visual-elements`);
  }

  get(projectId: string, elementId: string): Observable<VisualElement> {
    return this.http.get<VisualElement>(`${this.apiUrl}/projects/${projectId}/visual-elements/${elementId}`);
  }

  create(projectId: string, element: VisualElement): Observable<VisualElement> {
    return this.http.post<VisualElement>(`${this.apiUrl}/projects/${projectId}/visual-elements`, element);
  }

  update(projectId: string, elementId: string, element: VisualElement): Observable<VisualElement> {
    return this.http.put<VisualElement>(`${this.apiUrl}/projects/${projectId}/visual-elements/${elementId}`, element);
  }

  remove(projectId: string, elementId: string): Observable<void> {
    return this.http.delete<void>(`${this.apiUrl}/projects/${projectId}/visual-elements/${elementId}`);
  }
}
