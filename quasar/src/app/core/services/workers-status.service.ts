import { Injectable, PLATFORM_ID, inject } from '@angular/core';
import { isPlatformBrowser } from '@angular/common';
import { Observable, defer, EMPTY, timer } from 'rxjs';
import { webSocket, WebSocketSubject } from 'rxjs/webSocket';
import { retry } from 'rxjs/operators';
import { environment } from '../../../environments/environment';

export interface WorkersUsage {
  readActive: number;
  readTotal: number;
  writeActive: number;
  writeTotal: number;
}

@Injectable({ providedIn: 'root' })
export class WorkersStatusService {
  private platformId = inject(PLATFORM_ID);
  private socket$: WebSocketSubject<WorkersUsage>;

  constructor() {
    this.socket$ = webSocket(`${environment.ws}/ws/workers`);
  }

  listen(): Observable<WorkersUsage> {
    if (!isPlatformBrowser(this.platformId)) {
      return EMPTY;
    }
    return defer(() => {
      this.socket$ = webSocket(`${environment.ws}/ws/workers`);
      return this.socket$;
    }).pipe(
      retry({
        delay: (_error, retryCount) => timer(Math.min(2000 * retryCount, 10000))
      })
    );
  }
}
