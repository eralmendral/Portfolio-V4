import { NgOptimizedImage } from '@angular/common';
import {
  ChangeDetectionStrategy,
  Component,
  input,
  output,
  signal,
} from '@angular/core';
import { RouterLink } from '@angular/router';
import { HugeiconsIconComponent } from '@hugeicons/angular';

import type { HeroLink, OverviewCard } from '../../models/home.models';

@Component({
  selector: 'app-hero-section',
  imports: [HugeiconsIconComponent, NgOptimizedImage, RouterLink],
  templateUrl: './hero-section.component.html',
  styleUrl: './hero-section.component.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class HeroSectionComponent {
  readonly title = input.required<string>();
  readonly introduction = input.required<string>();
  readonly logoPath = input.required<string>();
  readonly sectionLinks = input.required<OverviewCard[]>();
  readonly entertainmentToggle = input.required<OverviewCard>();
  readonly entertainmentLinks = input<OverviewCard[]>([]);
  readonly externalLinks = input.required<HeroLink[]>();
  readonly loadingExternalLinks = input(false);
  readonly externalLinksLoadFailed = input(false);
  readonly reloadExternalLinks = output<void>();
  protected readonly activeSection = signal<OverviewCard | null>(null);
  protected readonly entertainmentOpen = signal(false);

  protected reloadLinks(): void {
    this.reloadExternalLinks.emit();
  }

  protected toggleEntertainment(): void {
    this.entertainmentOpen.update((open) => !open);
  }

  protected setActiveSection(section: OverviewCard): void {
    this.activeSection.set(section);
  }
}
