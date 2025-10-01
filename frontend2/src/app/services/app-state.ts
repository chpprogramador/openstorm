import { Injectable } from '@angular/core';
import { Project } from './project.service';

@Injectable({
  providedIn: 'root'
})
export class AppState {

  public project: Project | null = null;

  constructor() { }
}
