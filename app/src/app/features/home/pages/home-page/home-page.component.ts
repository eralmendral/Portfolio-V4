import { ChangeDetectionStrategy, Component, computed, inject, resource } from '@angular/core';
import type { IconSvgObject } from '@hugeicons/angular';
import {
  File01Icon,
  GithubIcon,
  LinkSquare02Icon,
  Linkedin01Icon,
  MailAtSign01Icon,
  NewTwitterIcon,
  YoutubeIcon,
} from '@hugeicons-pro/core-stroke-rounded';

import { PortfolioApi } from '../../../../core/services/portfolio-api.service';
import type { PortfolioLink } from '../../../../core/models/portfolio.models';
import { ThemeService } from '../../../../core/services/theme.service';
import { HeroSectionComponent } from '../../components/hero-section/hero-section.component';
import type { HeroLink } from '../../models/home.models';

const fallbackIntro = {
  title: 'Software | AI - Engineer',
  description: 'I build thoughtful software with strong engineering, clean craft, and attention to both details and the bigger picture.',
};

const fallbackCenterLinks: PortfolioLink[] = [
  {
    id: 'fallback-github',
    label: 'GitHub',
    url: 'https://github.com/eralmendral',
    icon_class: 'hugeicons-pro:github',
    sort_order: 10,
    star: true,
    status: 'published',
  },
  {
    id: 'fallback-resume',
    label: 'Resume',
    url: '/assets/cv.pdf',
    icon_class: 'hugeicons-pro:file-star',
    sort_order: 20,
    star: true,
    status: 'published',
  },
];

const linkIconRegistry: Record<string, IconSvgObject> = {
  'hugeicons-pro:file-star': File01Icon,
  'hugeicons-pro:github': GithubIcon,
  'hugeicons-pro:linkedin-01': Linkedin01Icon,
  'hugeicons-pro:mail-at-sign-01': MailAtSign01Icon,
  'hugeicons-pro:new-twitter': NewTwitterIcon,
  'hugeicons-pro:youtube': YoutubeIcon,
};

@Component({
  selector: 'app-home-page',
  imports: [HeroSectionComponent],
  templateUrl: './home-page.component.html',
  styleUrl: './home-page.component.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class HomePageComponent {
  private readonly portfolioApi = inject(PortfolioApi);
  protected readonly theme = inject(ThemeService);

  protected readonly introResource = resource({
    defaultValue: fallbackIntro,
    loader: ({ abortSignal }) => this.portfolioApi.getIntro(abortSignal),
  });
  protected readonly linksResource = resource({
    defaultValue: [] as PortfolioLink[],
    loader: ({ abortSignal }) => this.portfolioApi.listPublishedLinks(abortSignal),
  });
  protected readonly title = computed(() => this.currentIntro().title);
  protected readonly introduction = computed(() => this.currentIntro().description);
  protected readonly centerLinks = computed<HeroLink[]>(() =>
    this.currentCenterLinks().map((link) => ({
      id: link.id,
      label: link.label,
      url: link.url,
      icon: this.resolveLinkIcon(link),
    })),
  );
  protected readonly linksLoadFailed = computed(() => Boolean(this.linksResource.error()));

  private resolveLinkIcon(link: PortfolioLink): IconSvgObject {
    const configuredIcon = link.icon_class ? linkIconRegistry[link.icon_class.toLowerCase()] : undefined;
    if (configuredIcon) {
      return configuredIcon;
    }

    const iconText = `${link.icon_class ?? ''} ${link.label}`.toLowerCase();

    if (iconText.includes('github')) {
      return GithubIcon;
    }
    if (iconText.includes('linkedin')) {
      return Linkedin01Icon;
    }
    if (iconText.includes('youtube')) {
      return YoutubeIcon;
    }
    if (iconText.includes('twitter') || iconText.includes('x.com')) {
      return NewTwitterIcon;
    }
    if (iconText.includes('mail') || iconText.includes('email')) {
      return MailAtSign01Icon;
    }
    if (iconText.includes('resume') || iconText.includes('cv') || iconText.includes('file')) {
      return File01Icon;
    }

    return LinkSquare02Icon;
  }

  private currentIntro(): typeof fallbackIntro {
    try {
      return this.introResource.value();
    } catch {
      return fallbackIntro;
    }
  }

  private currentLinks(): PortfolioLink[] {
    try {
      return this.linksResource.value();
    } catch {
      return [];
    }
  }

  private currentCenterLinks(): PortfolioLink[] {
    const linksByURL = new Map<string, PortfolioLink>();

    for (const link of [
      ...fallbackCenterLinks,
      ...this.currentLinks().filter((link) => link.star && this.isCenterLink(link)),
    ]) {
      linksByURL.set(link.url, link);
    }

    return Array.from(linksByURL.values())
      .sort((left, right) => left.sort_order - right.sort_order || left.label.localeCompare(right.label));
  }

  private isCenterLink(link: PortfolioLink): boolean {
    const text = this.linkSearchText(link);
    return text.includes('github') || text.includes('resume') || text.includes('cv') || text.includes('file-star');
  }

  private linkSearchText(link: PortfolioLink): string {
    return `${link.id} ${link.label} ${link.url} ${link.icon_class ?? ''}`.toLowerCase();
  }
}
