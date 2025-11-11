// src/app/app.config.ts
import { ApplicationConfig, provideZonelessChangeDetection } from '@angular/core';
import { provideRouter } from '@angular/router';
import { provideClientHydration } from '@angular/platform-browser';
import { provideHttpClient, withFetch } from '@angular/common/http';
import { provideAnimations, BrowserAnimationsModule } from '@angular/platform-browser/animations'; // Import provideAnimations and BrowserAnimationsModule
import { routes } from './app.routes';

export const appConfig: ApplicationConfig = {
  providers: [
    provideZonelessChangeDetection(),
    provideRouter(routes),
    provideClientHydration(),
    provideHttpClient(withFetch()),
    provideAnimations() // Keep provideAnimations
  ],
  // Add BrowserAnimationsModule to the imports array if it's a module, or to providers if it's a function
  // For standalone components, provideAnimations() is usually sufficient.
  // If BrowserAnimationsModule is needed, it's typically imported in app.module.ts for non-standalone apps.
  // Since this is a standalone app, let's stick to provideAnimations() and investigate further if the error persists.
  // The error message "Could not resolve "@angular/animations/browser"" is more about the build system not finding the package.
  // Let's try to remove the `provideAnimations` and add `importProvidersFrom(BrowserAnimationsModule)`
  // This is a common pattern for standalone components that need modules.
};