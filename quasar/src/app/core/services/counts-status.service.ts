import { Injectable, PLATFORM_ID, inject } from '@angular/core';
import { isPlatformBrowser } from '@angular/common';
import { Observable, defer, EMPTY, timer } from 'rxjs';
import { webSocket, WebSocketSubject } from 'rxjs/webSocket';
import { retry } from 'rxjs/operators';
import { environment } from '../../../environments/environment';

export interface CountsProgress {
  done: number;
  total: number;
}

@Injectable({ providedIn: 'root' })
export class CountsStatusService {
  private platformId = inject(PLATFORM_ID);
  private socket$: WebSocketSubject<CountsProgress>;

  constructor() {
    this.socket$ = webSocket(`${environment.ws}/ws/counts`);
  }

  listen(): Observable<CountsProgress> {
    if (!isPlatformBrowser(this.platformId)) {
      return EMPTY;
    }
    return defer(() => {
      this.socket$ = webSocket(`${environment.ws}/ws/counts`);
      return this.socket$;
    }).pipe(
      retry({
        delay: (_error, retryCount) => timer(Math.min(2000 * retryCount, 10000))
      })
    );
  }
}
