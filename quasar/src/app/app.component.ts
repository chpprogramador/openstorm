// src/app/app.component.ts
import { Component, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterOutlet, Router, NavigationEnd } from '@angular/router';
import { HeaderComponent } from './shared/components/header/header.component';
import { filter } from 'rxjs/operators';

@Component({
  selector: 'app-root',
  standalone: true,
  imports: [CommonModule, RouterOutlet, HeaderComponent],
  template: `
    <div class="app-container">
      <app-header *ngIf="showHeader"></app-header>
      <main class="main-content">
        <router-outlet></router-outlet>
      </main>
    </div>
  `,
  styles: [`
    .app-container {
      min-height: 100vh;
      background: var(--bg-primary);
      display: flex;
      flex-direction: column;
    }

    .main-content {
      flex: 1;
    }
  `]
})
export class AppComponent {
  private router = inject(Router);
  showHeader = true;

  constructor() {
    this.router.events
      .pipe(filter(event => event instanceof NavigationEnd))
      .subscribe((event: any) => {
        this.showHeader = event.url !== '/';
      });
  }
}