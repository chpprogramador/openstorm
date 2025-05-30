import { Routes } from '@angular/router';
import { Dashboard } from './pages/dashboard/dashboard';
import { History } from './pages/history/history';
import { Jobs } from './pages/jobs/jobs';

export const routes: Routes = [
    {path: '', component: Dashboard},
    {path: 'jobs', component: Jobs},
    {path: 'history', component: History},
];
