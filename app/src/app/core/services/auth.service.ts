import { DOCUMENT } from '@angular/common';
import { computed, inject, Injectable, signal } from '@angular/core';

@Injectable({
  providedIn: 'root',
})
export class AuthService {
  private readonly document = inject(DOCUMENT);
  private readonly tokenStorageKey = 'eric-almendral-admin-token';

  readonly token = signal<string | null>(this.readToken());
  readonly isAuthenticated = computed(() => Boolean(this.token()));

  setToken(token: string): void {
    this.token.set(token);
    try {
      this.document.defaultView?.localStorage.setItem(this.tokenStorageKey, token);
    } catch {
      // Storage can be unavailable in restricted browser contexts.
    }
  }

  clearToken(): void {
    this.token.set(null);
    try {
      this.document.defaultView?.localStorage.removeItem(this.tokenStorageKey);
    } catch {
      // Storage can be unavailable in restricted browser contexts.
    }
  }

  private readToken(): string | null {
    try {
      const token = this.document.defaultView?.localStorage.getItem(this.tokenStorageKey)?.trim();
      return token || null;
    } catch {
      return null;
    }
  }
}
