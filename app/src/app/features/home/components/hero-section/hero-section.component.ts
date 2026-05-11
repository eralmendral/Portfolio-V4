import { NgOptimizedImage } from '@angular/common';
import {
  ChangeDetectionStrategy,
  Component,
  input,
  output,
} from '@angular/core';
import { HugeiconsIconComponent } from '@hugeicons/angular';

import type { HeroLink } from '../../models/home.models';

@Component({
  selector: 'app-hero-section',
  imports: [HugeiconsIconComponent, NgOptimizedImage],
  templateUrl: './hero-section.component.html',
  styleUrl: './hero-section.component.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class HeroSectionComponent {
  readonly title = input.required<string>();
  readonly introduction = input.required<string>();
  readonly logoPath = input.required<string>();
  readonly externalLinks = input.required<HeroLink[]>();
  readonly loadingExternalLinks = input(false);
  readonly externalLinksLoadFailed = input(false);
  readonly reloadExternalLinks = output<void>();

  protected reloadLinks(): void {
    this.reloadExternalLinks.emit();
  }
}
