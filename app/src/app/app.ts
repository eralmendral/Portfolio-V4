import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core';
import { NavigationEnd, Router, RouterOutlet } from '@angular/router';
import { filter } from 'rxjs';

import { ThemeService } from './core/services/theme.service';
import { AppNavigationComponent } from './shared/components/app-navigation/app-navigation.component';
import { BackShortcutComponent } from './shared/components/back-shortcut/back-shortcut.component';
import { ContactWidgetComponent } from './shared/components/contact-widget/contact-widget.component';
import { HomeShortcutComponent } from './shared/components/home-shortcut/home-shortcut.component';
import { ThemeToggleComponent } from './shared/components/theme-toggle/theme-toggle.component';

@Component({
  selector: 'app-root',
  imports: [
    AppNavigationComponent,
    BackShortcutComponent,
    ContactWidgetComponent,
    HomeShortcutComponent,
    RouterOutlet,
    ThemeToggleComponent,
  ],
  templateUrl: './app.html',
  styleUrl: './app.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
  host: {
    '[attr.data-theme]': 'theme.mode()',
  },
})
export class App {
  protected readonly theme = inject(ThemeService);
  private readonly router = inject(Router);
  private readonly currentUrl = signal(this.router.url);
  protected readonly isMinimalLayout = computed(() => {
    const path = this.currentUrl().split('?')[0].split('#')[0];

    return path === '/login' || path.startsWith('/admin');
  });
  protected readonly isAdminLayout = computed(() => {
    const path = this.currentUrl().split('?')[0].split('#')[0];

    return path.startsWith('/admin');
  });

  constructor() {
    this.router.events
      .pipe(filter((event): event is NavigationEnd => event instanceof NavigationEnd))
      .subscribe((event) => {
        this.currentUrl.set(event.urlAfterRedirects);
      });
  }
}
