import { Injectable } from '@angular/core';
import { Project } from '../models/project.model'; // Adjusted path

@Injectable({
  providedIn: 'root'
})
export class AppState {

  public project: Project | null = null;

  constructor() { }
}
