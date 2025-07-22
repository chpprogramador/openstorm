import { Injectable } from '@angular/core';
import { Observable } from 'rxjs';
import { webSocket, WebSocketSubject } from 'rxjs/webSocket';
import { environment } from '../../environments/environment';

export interface JobStatus {
  id: string;
  name: string;
  total: number;
  processed: number;
  progress: number;
  status: 'pending' | 'running' | 'done' | 'error';
  startedAt?: string;
  endedAt?: string;
  error?: string;
}


@Injectable({ providedIn: 'root' })
export class StatusService {
  private socket$: WebSocketSubject<JobStatus[]>;

  constructor() {
    this.socket$ = webSocket(`${environment.ws}/ws/status`);
  }

  listen(): Observable<JobStatus[]> {
    return this.socket$;
  }
}
