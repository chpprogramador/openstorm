import { Routes } from '@angular/router';
import { Dashboard } from './pages/dashboard/dashboard';
import { ErrorsComponent } from './pages/errors/errors.component';
import { History } from './pages/history/history';
import { Jobs } from './pages/jobs/jobs';
import { Variables } from './pages/variables/variables';

export const routes: Routes = [
    {path: '', component: Dashboard},
    {path: 'jobs', component: Jobs},
    {path: 'history', component: History},
    {path: 'variables', component: Variables},
    {path: 'errors', component: ErrorsComponent},
];
