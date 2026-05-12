import { ChangeDetectionStrategy, Component, inject } from '@angular/core';
import { NavigationEnd, Router } from '@angular/router';
import { HugeiconsIconComponent } from '@hugeicons/angular';
import { ArrowLeft02Icon } from '@hugeicons-pro/core-stroke-rounded';
import { filter } from 'rxjs';

@Component({
  selector: 'app-back-shortcut',
  imports: [HugeiconsIconComponent],
  templateUrl: './back-shortcut.component.html',
  styleUrl: './back-shortcut.component.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class BackShortcutComponent {
  private readonly router = inject(Router);
  private readonly previousUrls: string[] = [];
  private currentUrl = this.router.url;
  private isNavigatingBack = false;
  protected readonly backIcon = ArrowLeft02Icon;

  constructor() {
    this.router.events
      .pipe(filter((event): event is NavigationEnd => event instanceof NavigationEnd))
      .subscribe((event) => {
        const nextUrl = event.urlAfterRedirects;

        if (this.isNavigatingBack) {
          this.currentUrl = nextUrl;
          this.isNavigatingBack = false;
          return;
        }

        if (nextUrl !== this.currentUrl) {
          this.previousUrls.push(this.currentUrl);
          this.currentUrl = nextUrl;
        }
      });
  }

  protected goBack(): void {
    const previousUrl = this.previousUrls.pop();

    if (previousUrl) {
      this.isNavigatingBack = true;
      void this.router.navigateByUrl(previousUrl);
      return;
    }

    if (this.currentUrl !== '/') {
      void this.router.navigateByUrl('/');
    }
  }
}
