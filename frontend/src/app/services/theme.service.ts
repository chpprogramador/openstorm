import { Injectable, Inject, PLATFORM_ID } from '@angular/core';
import { isPlatformBrowser } from '@angular/common';
import { BehaviorSubject } from 'rxjs';

export type Theme = 'dark' | 'light';

@Injectable({
  providedIn: 'root'
})
export class ThemeService {
  private currentThemeSubject = new BehaviorSubject<Theme>('dark');
  public currentTheme$ = this.currentThemeSubject.asObservable();
  private isBrowser: boolean;

  constructor(@Inject(PLATFORM_ID) private platformId: Object) {
    this.isBrowser = isPlatformBrowser(this.platformId);
    
    if (this.isBrowser) {
      // Carrega o tema salvo no localStorage ou usa 'dark' como padrão
      const savedTheme = localStorage.getItem('theme') as Theme;
      if (savedTheme) {
        this.setTheme(savedTheme);
      } else {
        this.setTheme('dark');
      }
    }
  }

  setTheme(theme: Theme): void {
    this.currentThemeSubject.next(theme);
    
    if (this.isBrowser) {
      localStorage.setItem('theme', theme);
      
      // Remove todas as classes de tema do body
      document.body.classList.remove('dark-theme', 'light-theme');
      
      // Adiciona a nova classe de tema
      document.body.classList.add(`${theme}-theme`);
      
      // Atualiza a propriedade CSS customizada
      document.documentElement.setAttribute('data-theme', theme);
    }
  }

  toggleTheme(): void {
    const currentTheme = this.currentThemeSubject.value;
    const newTheme: Theme = currentTheme === 'dark' ? 'light' : 'dark';
    this.setTheme(newTheme);
  }

  getCurrentTheme(): Theme {
    return this.currentThemeSubject.value;
  }

  isDark(): boolean {
    return this.currentThemeSubject.value === 'dark';
  }

  isLight(): boolean {
    return this.currentThemeSubject.value === 'light';
  }
}
