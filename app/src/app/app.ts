import { ChangeDetectionStrategy, Component, inject } from '@angular/core';
import { RouterOutlet } from '@angular/router';
import { HugeiconsIconComponent } from '@hugeicons/angular';
import {
  MoonEclipseIcon,
  SunriseIcon,
} from '@hugeicons-pro/core-stroke-rounded';

import { ThemeService } from './theme.service';

@Component({
  selector: 'app-root',
  imports: [HugeiconsIconComponent, RouterOutlet],
  templateUrl: './app.html',
  styleUrl: './app.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
  host: {
    '[attr.data-theme]': 'theme.mode()',
  },
})
export class App {
  protected readonly theme = inject(ThemeService);
  protected readonly moonIcon = MoonEclipseIcon;
  protected readonly sunIcon = SunriseIcon;

  protected toggleTheme(): void {
    this.theme.toggle();
  }
}
