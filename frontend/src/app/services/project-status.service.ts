import { Injectable } from '@angular/core';
import { Observable } from 'rxjs';
import { webSocket, WebSocketSubject } from 'rxjs/webSocket';
import { environment } from '../../environments/environment';

export interface ProjectStatus {
  status: 'running' | 'stop';
}

@Injectable({ providedIn: 'root' })
export class ProjectStatusService {
  private socket$: WebSocketSubject<ProjectStatus>;

  constructor() {
    this.socket$ = webSocket(`${environment.ws}/ws/project-status`);
  }

  listen(): Observable<ProjectStatus> {
    return this.socket$;
  }
}
