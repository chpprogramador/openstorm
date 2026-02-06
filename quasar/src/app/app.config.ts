// src/app/app.config.ts
import { ApplicationConfig, provideZonelessChangeDetection } from '@angular/core';
import { provideRouter, withHashLocation } from '@angular/router';
import { provideClientHydration } from '@angular/platform-browser';
import { provideAnimations } from '@angular/platform-browser/animations';
import { provideHttpClient, withFetch } from '@angular/common/http';
import { routes } from './app.routes';
import { environment } from '../environments/environment';

const routerFeatures = environment.production ? [] : [withHashLocation()];

export const appConfig: ApplicationConfig = {
  providers: [
    provideZonelessChangeDetection(),
    provideRouter(routes, ...routerFeatures),
    provideClientHydration(),
    provideAnimations(),
    provideHttpClient(withFetch())
  ]
};
