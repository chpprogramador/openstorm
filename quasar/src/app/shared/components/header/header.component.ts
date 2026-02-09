// src/app/shared/components/header/header.component.ts
import { Component, OnInit, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterModule, Router } from '@angular/router';
import { ProjectService } from '../../../core/services/project.service';
import { ThemeService } from '../../../core/services/theme.service';
import { Project } from '../../../core/models/project.model';
import { Observable } from 'rxjs';

@Component({
    selector: 'app-header',
    standalone: true,
    imports: [CommonModule, RouterModule],
    template: `
    <header class="header">
      <div class="header-container">
        <div class="header-left">
          <button class="logo-btn" type="button" (click)="goToProjects()" aria-label="Quasar">
            <img class="logo-img logo-light" src="quasar-logo-light.png" alt="Quasar" />
            <img class="logo-img logo-dark" src="quasar-logo-dark.png" alt="Quasar" />
          </button>
          
          <nav class="nav" *ngIf="selectedProject$ | async as project">
            <a routerLink="/home" routerLinkActive="active" class="nav-link">Home</a>
            <a routerLink="/benchmark" routerLinkActive="active" class="nav-link">Benchmark</a>
            <a routerLink="/variables" routerLinkActive="active" class="nav-link">Variáveis</a>
            <a routerLink="/jobs" routerLinkActive="active" class="nav-link">Jobs</a>
            <a routerLink="/history" routerLinkActive="active" class="nav-link">Historico</a>
          </nav>
        </div>

        <div class="header-right">
          <div class="project-info" *ngIf="selectedProject$ | async as project">
            <span class="project-name">{{ project.name }}</span>
          </div>
          
          <button class="theme-toggle" (click)="toggleTheme()" [title]="(darkMode$ | async) ? 'Modo Claro' : 'Modo Escuro'">
            <svg *ngIf="!(darkMode$ | async)" xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <circle cx="12" cy="12" r="5"/>
              <line x1="12" y1="1" x2="12" y2="3"/>
              <line x1="12" y1="21" x2="12" y2="23"/>
              <line x1="4.22" y1="4.22" x2="5.64" y2="5.64"/>
              <line x1="18.36" y1="18.36" x2="19.78" y2="19.78"/>
              <line x1="1" y1="12" x2="3" y2="12"/>
              <line x1="21" y1="12" x2="23" y2="12"/>
              <line x1="4.22" y1="19.78" x2="5.64" y2="18.36"/>
              <line x1="18.36" y1="5.64" x2="19.78" y2="4.22"/>
            </svg>
            <svg *ngIf="darkMode$ | async" xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z"/>
            </svg>
          </button>
        </div>
      </div>
    </header>
  `,
    styles: [`
    .header {
      background: var(--header-bg);
      border-bottom: 1px solid var(--border-color);
      position: sticky;
      top: 0;
      z-index: 100;
      box-shadow: 0 2px 8px rgba(0, 0, 0, 0.05);
    }

    .header-container {
      max-width: 1400px;
      margin: 0 auto;
      padding: 0 2rem;
      display: flex;
      justify-content: space-between;
      align-items: center;
      height: 64px;
    }

    .header-left {
      display: flex;
      align-items: center;
      gap: 3rem;
    }

    .logo-btn {
      background: transparent;
      border: none;
      padding: 0;
      display: inline-flex;
      align-items: center;
      cursor: pointer;
    }

    .logo-btn:hover {
      opacity: 0.9;
    }

    .logo-img {
      height: 34px;
      width: auto;
      display: block;
    }

    .logo-dark {
      display: none;
    }

    :host-context(body.dark-theme) .logo-light {
      display: none;
    }

    :host-context(body.dark-theme) .logo-dark {
      display: block;
    }

    .nav {
      display: flex;
      gap: 0.5rem;
    }

    .nav-link {
      padding: 0.5rem 1rem;
      color: var(--text-secondary);
      text-decoration: none;
      border-radius: 8px;
      font-weight: 500;
      transition: all 0.2s;
      position: relative;
    }

    .nav-link:hover {
      color: var(--text-primary);
      background: var(--hover-bg);
    }

    .nav-link.active {
      color: #667eea;
      background: var(--active-bg);
    }

    .header-right {
      display: flex;
      align-items: center;
      gap: 1.5rem;
    }

    .project-info {
      display: flex;
      align-items: center;
      gap: 0.5rem;
      padding: 0.5rem 1rem;
      background: var(--card-bg);
      border-radius: 8px;
      border: 1px solid var(--border-color);
    }

    .project-name {
      font-weight: 600;
      color: var(--text-primary);
      font-size: 0.9rem;
    }

    .theme-toggle {
      background: var(--card-bg);
      border: 1px solid var(--border-color);
      border-radius: 8px;
      padding: 0.5rem;
      cursor: pointer;
      display: flex;
      align-items: center;
      justify-content: center;
      color: var(--text-primary);
      transition: all 0.2s;
    }

    .theme-toggle:hover {
      background: var(--hover-bg);
      transform: scale(1.05);
    }

    @media (max-width: 768px) {
      .header-container {
        padding: 0 1rem;
      }

      .header-left {
        gap: 1.5rem;
      }

      .nav {
        gap: 0.25rem;
      }

      .nav-link {
        padding: 0.5rem 0.75rem;
        font-size: 0.9rem;
      }

      .project-name {
        display: none;
      }
    }
  `]
})
export class HeaderComponent implements OnInit {
    private projectService = inject(ProjectService);
    private themeService = inject(ThemeService);
    private router = inject(Router);

    selectedProject$: Observable<Project | null>;
    darkMode$: Observable<boolean>;

    constructor() {
        this.selectedProject$ = this.projectService.selectedProject$;
        this.darkMode$ = this.themeService.darkMode$;
    }

    ngOnInit(): void { }

    toggleTheme(): void {
        this.themeService.toggleDarkMode();
    }

    goToProjects(): void {
        this.projectService.clearSelection();
        this.router.navigate(['/']);
    }
}
