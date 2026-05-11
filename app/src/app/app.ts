import { ChangeDetectionStrategy, Component, inject } from '@angular/core';
import { RouterOutlet } from '@angular/router';

import { ThemeService } from './core/services/theme.service';
import { ContactWidgetComponent } from './shared/components/contact-widget/contact-widget.component';
import { HomeShortcutComponent } from './shared/components/home-shortcut/home-shortcut.component';
import { ThemeToggleComponent } from './shared/components/theme-toggle/theme-toggle.component';

@Component({
  selector: 'app-root',
  imports: [
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
}
