import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core';
import { toSignal } from '@angular/core/rxjs-interop';
import { NavigationEnd, Router, RouterLink, RouterLinkActive } from '@angular/router';
import { HugeiconsIconComponent } from '@hugeicons/angular';
import type { IconSvgObject } from '@hugeicons/angular';
import {
  Award01Icon,
  CodeIcon,
  Folder01Icon,
  Film01Icon,
  GameController01Icon,
  Link01Icon,
  MusicNote01Icon,
  PlayListIcon,
  WorkIcon,
} from '@hugeicons-pro/core-stroke-rounded';
import { filter, map, startWith } from 'rxjs';

interface NavigationLink {
  title: string;
  route: string;
  icon: IconSvgObject;
}

const entertainmentRoutes = ['/etc', '/music', '/games', '/animes'];

@Component({
  selector: 'app-navigation',
  imports: [HugeiconsIconComponent, RouterLink, RouterLinkActive],
  templateUrl: './app-navigation.component.html',
  styleUrl: './app-navigation.component.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class AppNavigationComponent {
  private readonly router = inject(Router);
  private readonly currentUrl = toSignal(
    this.router.events.pipe(
      filter((event): event is NavigationEnd => event instanceof NavigationEnd),
      map(() => this.router.url),
      startWith(this.router.url),
    ),
    { initialValue: this.router.url },
  );

  protected readonly sectionLinks: NavigationLink[] = [
    { title: 'Projects', route: '/projects', icon: Folder01Icon },
    { title: 'Work', route: '/work-history', icon: WorkIcon },
    { title: 'Certs', route: '/certificates', icon: Award01Icon },
    { title: 'Skills', route: '/skills', icon: CodeIcon },
    { title: 'Socials', route: '/social-links', icon: Link01Icon },
  ];
  protected readonly entertainmentToggle: NavigationLink = {
    title: '/etc',
    route: '/etc',
    icon: PlayListIcon,
  };
  protected readonly entertainmentLinks: NavigationLink[] = [
    { title: 'Music', route: '/music', icon: MusicNote01Icon },
    { title: 'Games', route: '/games', icon: GameController01Icon },
    { title: 'Animes', route: '/animes', icon: Film01Icon },
  ];
  protected readonly entertainmentOpen = signal(false);
  protected readonly entertainmentActive = computed(() => {
    const url = this.currentUrl();
    return entertainmentRoutes.some((route) => url === route || url.startsWith(`${route}/`));
  });
  protected readonly showEntertainmentLinks = computed(
    () => this.entertainmentOpen() || this.entertainmentActive(),
  );

  protected toggleEntertainment(): void {
    this.entertainmentOpen.update((open) => !open);
  }
}
