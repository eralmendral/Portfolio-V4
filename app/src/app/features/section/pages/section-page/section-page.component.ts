import { ChangeDetectionStrategy, Component, inject, signal } from '@angular/core';
import { ActivatedRoute, RouterLink } from '@angular/router';
import { HugeiconsIconComponent } from '@hugeicons/angular';

import { readPageData } from '../../data/section-page.data';

@Component({
  selector: 'app-section-page',
  imports: [HugeiconsIconComponent, RouterLink],
  template: `
    <main class="SectionPage" aria-labelledby="section-title">
      <a class="BackLink" routerLink="/">Home</a>
      <section class="SectionPanel">
        <span class="SectionIcon" aria-hidden="true">
          <hugeicons-icon
            [icon]="page().icon"
            [size]="42"
            color="currentColor"
            [strokeWidth]="2.2"
          />
        </span>
        <p class="Eyebrow">Detail</p>
        <h1 id="section-title">{{ page().title }}</h1>
        <p>{{ page().summary }}</p>
        @if (page().links?.length) {
          <nav class="SectionLinks" aria-label="Section links">
            @for (link of page().links ?? []; track link.url) {
              <a [href]="link.url" target="_blank" rel="noreferrer">
                {{ link.label }}
              </a>
            }
          </nav>
        }
      </section>
    </main>
  `,
  styles: [`
    :host {
      display: block;
    }

    .SectionPage {
      min-height: 100dvh;
      display: grid;
      align-content: center;
      gap: 24px;
      padding: 96px 32px;
    }

    .BackLink {
      width: fit-content;
      color: var(--app-muted);
      font-weight: 900;
      text-decoration: none;
    }

    .BackLink:focus-visible {
      outline: 3px solid var(--app-toggle-focus);
      outline-offset: 4px;
    }

    .SectionPanel {
      width: min(100%, 820px);
      display: grid;
      gap: 16px;
    }

    .SectionIcon {
      width: 72px;
      height: 72px;
      border-radius: 8px;
      background: var(--app-toggle-track);
      color: var(--app-toggle-thumb-color);
      display: grid;
      place-items: center;
    }

    hugeicons-icon {
      display: inline-grid;
      color: currentColor;
      line-height: 0;
    }

    .Eyebrow {
      margin: 0;
      color: var(--app-muted);
      font-size: 12px;
      font-weight: 900;
      line-height: 1.25;
      text-transform: uppercase;
    }

    h1 {
      margin: 0;
      color: var(--app-heading);
      font-size: 58px;
      font-weight: 900;
      line-height: 1.04;
    }

    p {
      max-width: 700px;
      margin: 0;
      color: var(--app-copy);
      font-size: 19px;
      line-height: 1.55;
    }

    .SectionLinks {
      display: flex;
      flex-wrap: wrap;
      gap: 18px;
    }

    .SectionLinks a {
      color: var(--app-heading);
      font-size: 16px;
      font-weight: 900;
      text-decoration: none;
    }

    .SectionLinks a:hover {
      text-decoration: underline;
      text-underline-offset: 5px;
    }

    .SectionLinks a:focus-visible {
      outline: 3px solid var(--app-toggle-focus);
      outline-offset: 4px;
    }

    @media (max-width: 720px) {
      .SectionPage {
        padding: 92px 20px 56px;
      }

      h1 {
        font-size: 40px;
      }
    }
  `],
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class SectionPageComponent {
  private readonly route = inject(ActivatedRoute);
  protected readonly page = signal(readPageData(this.route.snapshot.data));
}
