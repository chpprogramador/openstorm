import { Injectable, PLATFORM_ID, inject } from '@angular/core';
import { Observable, defer, EMPTY, timer } from 'rxjs';
import { webSocket, WebSocketSubject } from 'rxjs/webSocket';
import { environment } from '../../../environments/environment';
import { isPlatformBrowser } from '@angular/common';
import { retry } from 'rxjs/operators';

export interface LogEntry {
  timestamp: string;
  message: string;
}

@Injectable({ providedIn: 'root' })
export class LogStatusService {
  private platformId = inject(PLATFORM_ID);
  private socket$: WebSocketSubject<LogEntry[]>;

  constructor() {
    this.socket$ = webSocket(`${environment.ws}/ws/logs`);
  }

  listen(): Observable<LogEntry[]> {
    if (!isPlatformBrowser(this.platformId)) {
      return EMPTY;
    }
    return defer(() => {
      this.socket$ = webSocket(`${environment.ws}/ws/logs`);
      return this.socket$;
    }).pipe(
      retry({
        delay: (_error, retryCount) => timer(Math.min(2000 * retryCount, 10000))
      })
    );
  }

  clearLogs() {
    this.socket$ = webSocket(`${environment.ws}/ws/logs`);
  }
}
