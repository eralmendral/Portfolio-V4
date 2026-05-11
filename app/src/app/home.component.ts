import { NgOptimizedImage } from '@angular/common';
import { ChangeDetectionStrategy, Component, computed, inject, resource, signal } from '@angular/core';
import { RouterLink } from '@angular/router';
import { HugeiconsIconComponent } from '@hugeicons/angular';
import type { IconSvgObject } from '@hugeicons/angular';
import {
  ArrowDownDoubleIcon,
  BrainCogIcon,
  Certificate02Icon,
  CodeFolderIcon,
  FileStarIcon,
  GithubIcon,
  LinkSquare02Icon,
  Linkedin01Icon,
  MailAtSign01Icon,
  NewTwitterIcon,
  Rocket02Icon,
  ToolsIcon,
  WorkHistoryIcon,
  YoutubeIcon,
} from '@hugeicons-pro/core-stroke-rounded';

import { PortfolioApi, type PortfolioLink, type ProjectSummary } from './portfolio-api';
import { ThemeService } from './theme.service';

interface HeroLink {
  id: string;
  label: string;
  url: string;
  icon: IconSvgObject;
}

interface ProjectCard {
  id: string;
  title: string;
  summary?: string;
  techStack: string[];
}

interface OverviewCard {
  title: string;
  summary: string;
  route: string;
  icon: IconSvgObject;
}

@Component({
  selector: 'app-home',
  imports: [HugeiconsIconComponent, NgOptimizedImage, RouterLink],
  templateUrl: './home.component.html',
  styleUrl: './home.component.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class HomeComponent {
  private readonly portfolioApi = inject(PortfolioApi);
  protected readonly theme = inject(ThemeService);

  protected readonly title = signal('Eric Almendral');
  protected readonly introduction = signal(
    'Software engineer building pragmatic APIs, polished web tools, and AI-assisted delivery systems.',
  );
  protected readonly starredLinksResource = resource({
    defaultValue: [] as PortfolioLink[],
    loader: ({ abortSignal }) => this.portfolioApi.listStarredLinks(abortSignal),
  });
  protected readonly featuredProjectsResource = resource({
    defaultValue: [] as ProjectSummary[],
    loader: ({ abortSignal }) => this.portfolioApi.listFeaturedProjects(abortSignal),
  });
  protected readonly starredLinks = computed<HeroLink[]>(() =>
    this.starredLinksResource.value().map((link) => ({
      id: link.id,
      label: link.label,
      url: link.url,
      icon: this.resolveLinkIcon(link),
    })),
  );
  protected readonly featuredProjects = computed<ProjectCard[]>(() =>
    this.featuredProjectsResource.value().slice(0, 3).map((project) => ({
      id: project.id,
      title: project.title,
      summary: project.summary,
      techStack: project.tech_stack?.slice(0, 3) ?? [],
    })),
  );
  protected readonly hasStarredLinks = computed(() => this.starredLinks().length > 0);
  protected readonly hasFeaturedProjects = computed(() => this.featuredProjects().length > 0);
  protected readonly linksLoadFailed = computed(() => Boolean(this.starredLinksResource.error()));
  protected readonly projectsLoadFailed = computed(() => Boolean(this.featuredProjectsResource.error()));
  protected readonly overviewCards: OverviewCard[] = [
    {
      title: 'Work history',
      summary: 'Roles, responsibilities, and shipped outcomes across product, platform, and delivery work.',
      route: '/work-history',
      icon: WorkHistoryIcon,
    },
    {
      title: 'Certificates',
      summary: 'Credentials and learning milestones that back up the engineering practice.',
      route: '/certificates',
      icon: Certificate02Icon,
    },
    {
      title: 'Skills',
      summary: 'Backend, frontend, delivery, and AI workflow strengths organized for quick scanning.',
      route: '/skills',
      icon: BrainCogIcon,
    },
    {
      title: 'Tools',
      summary: 'The practical software, platforms, and AI assistants used to build and ship work.',
      route: '/tools',
      icon: ToolsIcon,
    },
  ];
  protected readonly projectIcon = CodeFolderIcon;
  protected readonly scrollIcon = ArrowDownDoubleIcon;
  protected readonly projectCtaIcon = Rocket02Icon;

  private resolveLinkIcon(link: PortfolioLink): IconSvgObject {
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
      return FileStarIcon;
    }

    return LinkSquare02Icon;
  }
}
