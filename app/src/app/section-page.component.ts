import { ChangeDetectionStrategy, Component, inject, signal } from '@angular/core';
import { RouterLink, ActivatedRoute } from '@angular/router';
import { HugeiconsIconComponent } from '@hugeicons/angular';
import type { IconSvgObject } from '@hugeicons/angular';
import {
  BrainCogIcon,
  Certificate02Icon,
  CodeFolderIcon,
  ToolsIcon,
  WorkHistoryIcon,
} from '@hugeicons-pro/core-stroke-rounded';

interface SectionPageData {
  title: string;
  summary: string;
  icon: IconSvgObject;
}

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
            [strokeWidth]="1.7"
          />
        </span>
        <p class="Eyebrow">Portfolio detail</p>
        <h1 id="section-title">{{ page().title }}</h1>
        <p>{{ page().summary }}</p>
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

export const projectsPageData: SectionPageData = {
  title: 'Projects',
  summary: 'A deeper archive for selected builds, production systems, experiments, and implementation notes.',
  icon: CodeFolderIcon,
};

export const workHistoryPageData: SectionPageData = {
  title: 'Work history',
  summary: 'A focused timeline of roles, responsibilities, outcomes, and the operating contexts behind the work.',
  icon: WorkHistoryIcon,
};

export const certificatesPageData: SectionPageData = {
  title: 'Certificates',
  summary: 'Credential details, issuing organizations, dates, and related proof links.',
  icon: Certificate02Icon,
};

export const skillsPageData: SectionPageData = {
  title: 'Skills',
  summary: 'A structured view of engineering strengths across backend, frontend, delivery, quality, and AI workflows.',
  icon: BrainCogIcon,
};

export const toolsPageData: SectionPageData = {
  title: 'Tools',
  summary: 'A practical inventory of the platforms, developer tools, and AI systems used to ship work.',
  icon: ToolsIcon,
};

function readPageData(data: Record<string, unknown>): SectionPageData {
  const title = data['title'];
  const summary = data['summary'];
  const icon = data['icon'];

  if (typeof title === 'string' && typeof summary === 'string' && Array.isArray(icon)) {
    return {
      title,
      summary,
      icon: icon as IconSvgObject,
    };
  }

  return projectsPageData;
}
