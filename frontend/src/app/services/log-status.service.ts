import { Injectable } from '@angular/core';
import { Observable } from 'rxjs';
import { webSocket, WebSocketSubject } from 'rxjs/webSocket';
import { environment } from '../../environments/environment';

export interface LogEntry {
  timestamp: string;
  message: string;
}

@Injectable({ providedIn: 'root' })
export class LogStatusService {
  private socket$: WebSocketSubject<LogEntry[]>;

  constructor() {
    this.socket$ = webSocket(`${environment.ws}/ws/logs`);
  }

  listen(): Observable<LogEntry[]> {
    return this.socket$;
  }
}
