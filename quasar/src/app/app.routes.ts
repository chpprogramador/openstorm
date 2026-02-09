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
        path: 'benchmark',
        loadComponent: () => import('./features/benchmark/benchmark.component').then(m => m.BenchmarkComponent)
    },
    {
        path: 'variables',
        loadComponent: () => import('./features/variables/variables.component').then(m => m.VariablesComponent)
    },
    {
        path: 'jobs',
        loadComponent: () => import('./features/jobs/jobs.component').then(m => m.JobsComponent)
    },
    {
        path: 'history',
        loadComponent: () => import('./features/errors/errors.component').then(m => m.ErrorsComponent)
    },
    {
        path: 'errors',
        loadComponent: () => import('./features/errors/errors.component').then(m => m.ErrorsComponent)
    },
    {
        path: '**',
        redirectTo: ''
    }
];
