// src/app/core/services/theme.service.ts
import { Injectable, PLATFORM_ID, inject } from '@angular/core';
import { isPlatformBrowser } from '@angular/common';
import { BehaviorSubject } from 'rxjs';

@Injectable({
    providedIn: 'root'
})
export class ThemeService {
    private platformId = inject(PLATFORM_ID);
    private isBrowser: boolean;

    private darkModeSubject = new BehaviorSubject<boolean>(false);
    public darkMode$ = this.darkModeSubject.asObservable();

    constructor() {
        this.isBrowser = isPlatformBrowser(this.platformId);
        this.loadTheme();
    }

    private loadTheme(): void {
        if (!this.isBrowser) {
            return;
        }

        const stored = localStorage.getItem('quasar_dark_mode');
        const isDark = stored === 'true';
        this.setDarkMode(isDark);
    }

    toggleDarkMode(): void {
        const newValue = !this.darkModeSubject.value;
        this.setDarkMode(newValue);
    }

    private setDarkMode(isDark: boolean): void {
        this.darkModeSubject.next(isDark);

        if (!this.isBrowser) {
            return;
        }

        localStorage.setItem('quasar_dark_mode', isDark.toString());

        if (isDark) {
            document.body.classList.add('dark-theme');
        } else {
            document.body.classList.remove('dark-theme');
        }
    }

    isDarkMode(): boolean {
        return this.darkModeSubject.value;
    }
}