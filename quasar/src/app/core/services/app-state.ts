import { Injectable, inject } from '@angular/core';
import { ProjectService } from './project.service';
import { Project } from '../models/project.model';

@Injectable({
  providedIn: 'root'
})
export class AppState {
  private projectService = inject(ProjectService);
  public project: Project | null = null;

  constructor() {
    this.projectService.selectedProject$.subscribe(project => {
      this.project = project;
    });
  }
}
