// src/app/app.routes.ts
import { Routes } from '@angular/router';
import { ProjectSelectionComponent } from './features/project-selection/project-selection.component';
import { HomeComponent } from './features/home/home.component';

export const routes: Routes = [
    {
        path: '',
        component: ProjectSelectionComponent
    },
    {
        path: 'home',
        component: HomeComponent
    },
    {
        path: 'variables',
        loadComponent: () => import('./features/variables/variables').then(m => m.Variables)
    },
    {
        path: 'jobs',
        loadComponent: () => import('./features/jobs/jobs.component').then(m => m.JobsComponent)
    },
    {
        path: 'history',
        loadComponent: () => import('./features/history/history.component').then(m => m.HistoryComponent)
    },
    {
        path: '**',
        redirectTo: ''
    }
];