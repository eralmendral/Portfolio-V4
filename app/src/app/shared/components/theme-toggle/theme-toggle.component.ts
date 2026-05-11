import { ChangeDetectionStrategy, Component, inject } from '@angular/core';
import { HugeiconsIconComponent } from '@hugeicons/angular';
import {
  Moon01Icon,
  Sun01Icon,
} from '@hugeicons-pro/core-stroke-rounded';

import { ThemeService } from '../../../core/services/theme.service';

@Component({
  selector: 'app-theme-toggle',
  imports: [HugeiconsIconComponent],
  templateUrl: './theme-toggle.component.html',
  styleUrl: './theme-toggle.component.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class ThemeToggleComponent {
  protected readonly theme = inject(ThemeService);
  protected readonly moonIcon = Moon01Icon;
  protected readonly sunIcon = Sun01Icon;

  protected toggleTheme(): void {
    this.theme.toggle();
  }
}
