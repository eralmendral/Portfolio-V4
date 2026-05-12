import { DOCUMENT } from '@angular/common';
import { computed, effect, inject, Injectable, signal } from '@angular/core';

export type ThemeMode = 'light' | 'dark';

@Injectable({
  providedIn: 'root',
})
export class ThemeService {
  private readonly document = inject(DOCUMENT);
  private readonly themeStorageKey = 'eric-almendral-theme';

  readonly mode = signal<ThemeMode>(this.getInitialThemeMode());
  readonly isDarkMode = computed(() => this.mode() === 'dark');
  readonly toggleLabel = computed(() =>
    this.isDarkMode() ? 'Switch to light mode' : 'Switch to dark mode',
  );
  readonly activeLabel = computed(() =>
    this.isDarkMode() ? 'Dark mode' : 'Light mode',
  );
  readonly logoPath = computed(() =>
    this.isDarkMode()
      ? '/assets/icons-set/03-transparent/full-logo-transparent-white.png'
      : '/assets/icons-set/03-transparent/full-logo-transparent-black.png',
  );
  readonly faviconPath = computed(() =>
    this.isDarkMode()
      ? '/assets/icons-set/04-favicons/favicon-transparent-32x32.png'
      : '/assets/icons-set/04-favicons/favicon-32x32.png',
  );

  constructor() {
    effect(() => {
      this.updateFavicon(this.faviconPath());
    });
  }

  toggle(): void {
    const nextMode = this.isDarkMode() ? 'light' : 'dark';
    this.mode.set(nextMode);
    this.storeThemeMode(nextMode);
  }

  private getInitialThemeMode(): ThemeMode {
    const view = this.document.defaultView;
    const storedMode = this.readStoredThemeMode();

    if (storedMode) {
      return storedMode;
    }

    if (view?.matchMedia?.('(prefers-color-scheme: dark)')?.matches) {
      return 'dark';
    }

    return 'light';
  }

  private readStoredThemeMode(): ThemeMode | null {
    try {
      const storedMode = this.document.defaultView?.localStorage.getItem(this.themeStorageKey);
      return storedMode === 'dark' || storedMode === 'light' ? storedMode : null;
    } catch {
      return null;
    }
  }

  private storeThemeMode(mode: ThemeMode): void {
    try {
      this.document.defaultView?.localStorage.setItem(this.themeStorageKey, mode);
    } catch {
      // Browsers can block storage in private or restricted contexts.
    }
  }

  private updateFavicon(path: string): void {
    const head = this.document.head;
    let icon = this.document.querySelector<HTMLLinkElement>("link[rel='icon']");

    if (!icon) {
      icon = this.document.createElement('link');
      icon.rel = 'icon';
      head.appendChild(icon);
    }

    icon.id = 'app-favicon';
    icon.type = 'image/png';
    icon.href = path;
  }
}
