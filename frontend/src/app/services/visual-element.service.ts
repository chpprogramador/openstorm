import { HttpClient } from '@angular/common/http';
import { Injectable } from '@angular/core';
import { Observable } from 'rxjs';
import { environment } from '../../environments/environment';

export type VisualElementType = 'rect' | 'circle' | 'line' | 'text';

export interface VisualElement {
  id?: string;
  elementId?: string;
  type: VisualElementType;
  x: number;
  y: number;
  width?: number;
  height?: number;
  x2?: number;
  y2?: number;
  radius?: number;
  fillColor?: string;
  borderColor?: string;
  borderWidth?: number;
  text?: string;
  textColor?: string;
  fontSize?: number;
  fontFamily?: string;
  textAlign?:
    | 'center'
    | 'top-center'
    | 'bottom-center'
    | 'left'
    | 'center-left'
    | 'right'
    | 'center-right'
    | 'top-left'
    | 'top-right'
    | 'bottom-left'
    | 'bottom-right';
  cornerRadius?: number;
}

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
