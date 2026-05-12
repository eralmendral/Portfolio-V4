import { ChangeDetectionStrategy, Component, HostListener, computed, inject, resource, signal } from '@angular/core';
import { ActivatedRoute, RouterLink } from '@angular/router';
import { HugeiconsIconComponent } from '@hugeicons/angular';
import {
  ArchiveIcon,
  ArrowLeft02Icon,
  ArrowRight02Icon,
  ArrowUpDownIcon,
  Cancel01Icon,
} from '@hugeicons-pro/core-stroke-rounded';

import type {
  Certificate,
  ProjectImage,
  ProjectSummary,
  Skill,
  SkillCategory,
  Tool,
  WorkExperience,
} from '../../../../core/models/portfolio.models';
import { PortfolioApi } from '../../../../core/services/portfolio-api.service';
import type { SectionPageLink } from '../../models/section-page.models';
import { readPageData } from '../../data/section-page.data';

interface SkillsPageContent {
  categories: SkillCategory[];
  skills: Skill[];
  tools: Tool[];
}

@Component({
  selector: 'app-section-page',
  imports: [HugeiconsIconComponent, RouterLink],
  template: `
    <main
      class="SectionPage"
      [class.SectionPageTop]="isTopAlignedPage"
      [class.SectionPageSocial]="isSocialLinksPage"
      [class.SectionPageScrollable]="isScrollablePage"
      aria-labelledby="section-title"
    >
      <section class="SectionPanel">
        <h1 id="section-title">{{ page().title }}</h1>
        <p>{{ page().summary }}</p>
        @if (sectionLinks().length) {
          <nav class="SectionLinks" aria-label="Section links">
            <ul>
              @for (link of sectionLinks(); track link.url) {
                <li>
                  <a [href]="link.url" target="_blank" rel="noreferrer">
                    {{ link.label }}
                  </a>
                </li>
              }
            </ul>
          </nav>
        } @else if (socialLinksLoadFailed()) {
          <p class="SectionEmpty">No data for links.</p>
        }
      </section>

      @if (isProjectsPage) {
        <section class="SectionContent" aria-label="Projects">
          @if (projectsLoadFailed() || !projectsResource.value().length) {
            <p class="SectionEmpty">No data for projects.</p>
          } @else {
            <div class="ProjectGrid">
              @for (project of projectsResource.value(); track project.id) {
                <article class="SectionCard ProjectCard">
                  @if (project.main_image; as image) {
                    <button
                      class="ProjectImageButton ProjectImageButtonMain"
                      type="button"
                      [attr.aria-label]="projectImageButtonLabel(project, image)"
                      (click)="openProjectLightbox(project, image, $event)"
                    >
                      <img [src]="image.url" [alt]="image.alt_text || project.title" loading="lazy">
                    </button>
                  }
                  @if (project.images?.length) {
                    <div class="ProjectGallery" aria-label="Additional project screenshots">
                      @for (image of project.images; track image.id) {
                        <button
                          class="ProjectImageButton ProjectThumbnailButton"
                          type="button"
                          [attr.aria-label]="projectImageButtonLabel(project, image)"
                          (click)="openProjectLightbox(project, image, $event)"
                        >
                          <img [src]="image.url" [alt]="image.alt_text || project.title" loading="lazy">
                        </button>
                      }
                    </div>
                  }
                  <div>
                    <h2>{{ project.title }}</h2>
                    @if (project.summary) {
                      <p>{{ project.summary }}</p>
                    }
                    @if (project.tech_stack?.length) {
                      <ul class="PillList">
                        @for (tool of project.tech_stack; track tool) {
                          <li>{{ tool }}</li>
                        }
                      </ul>
                    }
                    <div class="CardLinks">
                      @if (project.demo_url) {
                        <a [href]="project.demo_url" target="_blank" rel="noreferrer">Demo</a>
                      }
                      @if (project.github_url) {
                        <a [href]="project.github_url" target="_blank" rel="noreferrer">GitHub</a>
                      }
                    </div>
                  </div>
                </article>
              }
            </div>
          }
          @if (!isArchivedProjectsPage) {
            <nav class="ProjectArchiveNav" aria-label="Project archive">
              <a class="ProjectArchiveLink" routerLink="/projects/archived">
                <hugeicons-icon
                  [icon]="archiveIcon"
                  [size]="16"
                  color="currentColor"
                  [strokeWidth]="2.2"
                  aria-hidden="true"
                />
                <span>Archived Projects</span>
              </a>
            </nav>
          }
        </section>
      }

      @if (isSkillsPage) {
        <section class="SectionContent SkillContent" aria-label="Skills and tools">
          @if (skillsLoadFailed() || (!skillsResource.value().categories.length && !skillsResource.value().tools.length)) {
            <p class="SectionEmpty">No data for skills and tools.</p>
          } @else {
            <label class="SkillSearch">
              <span class="ScreenReaderOnly">Search skills and tools</span>
              <input
                type="search"
                autocomplete="off"
                placeholder="Search skills and tools"
                [value]="skillSearchQuery()"
                (input)="updateSkillSearch($event)"
              >
            </label>
            @if (filteredSkillCategories().length) {
              <h2 class="ContentHeading">Skills</h2>
              <div class="SkillGrid">
                @for (category of filteredSkillCategories(); track category.id) {
                  <article
                    class="SectionCard SkillCard"
                    [class.SkillCard--currentFocus]="isFocusSkillCategory(category)"
                  >
                    <h2>{{ category.name }}</h2>
                    @if (category.description) {
                      <p>{{ category.description }}</p>
                    }
                    <ul class="PillList SkillPillList">
                      @for (skill of filteredSkillsForCategory(category); track skill.id) {
                        <li>{{ skill.name }}</li>
                      }
                    </ul>
                  </article>
                }
              </div>
            }
            @if (filteredTools().length) {
              <h2 class="ContentHeading">Tools</h2>
              <div class="ToolGrid">
                @for (tool of filteredTools(); track tool.id) {
                  <article class="SectionCard ToolCard">
                    <div>
                      <h3>{{ tool.name }}</h3>
                      <span>{{ tool.category }}</span>
                    </div>
                    @if (tool.summary) {
                      <p>{{ tool.summary }}</p>
                    }
                  </article>
                }
              </div>
            }
            @if (!filteredSkillCategories().length && !filteredTools().length) {
              <p class="SectionEmpty">No matching skills or tools.</p>
            }
          }
        </section>
      }

      @if (isCertificatesPage) {
        <section class="SectionContent" aria-label="Certificates">
          @if (certificatesLoadFailed() || !certificatesResource.value().length) {
            <p class="SectionEmpty">No data for certificates.</p>
          } @else {
            <div class="CertificateGrid">
              @for (certificate of certificatesResource.value(); track certificate.id) {
                <article class="SectionCard CertificateCard">
                  @if (certificate.image; as image) {
                    <img [src]="image.url" [alt]="image.alt_text || certificate.title" loading="lazy">
                  }
                  <div>
                    <h2>{{ certificate.title }}</h2>
                    @if (certificate.issuer) {
                      <p>{{ certificate.issuer }}</p>
                    }
                    @if (certificate.credential_url) {
                      <a [href]="certificate.credential_url" target="_blank" rel="noreferrer">
                        View credential
                      </a>
                    }
                  </div>
                </article>
              }
            </div>
          }
        </section>
      }

      @if (isWorkHistoryPage) {
        <section class="SectionContent" aria-label="Work history">
          @if (workExperiencesLoadFailed() || !workExperiencesResource.value().length) {
            <p class="SectionEmpty">No data for work history.</p>
          } @else {
            <div class="WorkToolbar">
              <button
                class="SortButton"
                type="button"
                [attr.aria-label]="workSortLabel()"
                [title]="workSortLabel()"
                (click)="toggleWorkSort()"
              >
                <hugeicons-icon
                  [icon]="sortIcon"
                  [size]="18"
                  color="currentColor"
                  [strokeWidth]="2"
                  aria-hidden="true"
                />
              </button>
            </div>
            <div class="WorkList">
              @for (experience of sortedWorkExperiences(); track experience.id) {
                <article class="SectionCard WorkCard">
                  <div class="WorkHeader">
                    <div>
                      <h2>{{ experience.title }}</h2>
                      <p>
                        @if (experience.company_url) {
                          <a [href]="experience.company_url" target="_blank" rel="noreferrer">{{ experience.company }}</a>
                        } @else {
                          {{ experience.company }}
                        }
                      </p>
                    </div>
                    <span>{{ workDateRange(experience) }}</span>
                  </div>
                  @if (experience.employment_type) {
                    <strong>{{ experience.employment_type }}</strong>
                  }
                  @if (experience.description || experience.summary) {
                    <p>{{ experience.description || experience.summary }}</p>
                  }
                  @if (experience.responsibilities; as responsibilities) {
                    <ul>
                      @for (responsibility of responsibilities; track responsibility) {
                        <li>{{ responsibility }}</li>
                      }
                    </ul>
                  }
                </article>
              }
            </div>
          }
        </section>
      }
    </main>

    @if (activeProjectImage(); as activeImage) {
      <div
        class="ProjectLightbox"
        role="dialog"
        aria-modal="true"
        [attr.aria-label]="activeProjectTitle() + ' screenshots'"
        (click)="closeProjectLightbox()"
      >
        <section class="ProjectLightboxPanel" (click)="$event.stopPropagation()">
          <div class="ProjectLightboxTop">
            <div>
              <p class="ProjectLightboxEyebrow">Project image</p>
              <h2>{{ activeProjectTitle() }}</h2>
            </div>
            <button
              class="ProjectLightboxIconButton"
              type="button"
              aria-label="Close image viewer"
              (click)="closeProjectLightbox()"
            >
              <hugeicons-icon
                [icon]="closeIcon"
                [size]="22"
                color="currentColor"
                [strokeWidth]="2.2"
                aria-hidden="true"
              />
            </button>
          </div>

          <div class="ProjectLightboxImageWrap">
            @if (hasMultipleProjectImages()) {
              <button
                class="ProjectLightboxNav ProjectLightboxNavPrev"
                type="button"
                aria-label="Previous image"
                (click)="showPreviousProjectImage()"
              >
                <hugeicons-icon
                  [icon]="previousIcon"
                  [size]="24"
                  color="currentColor"
                  [strokeWidth]="2.2"
                  aria-hidden="true"
                />
              </button>
            }

            <img
              [src]="activeImage.url"
              [alt]="activeImage.alt_text || activeProjectTitle()"
            >

            @if (hasMultipleProjectImages()) {
              <button
                class="ProjectLightboxNav ProjectLightboxNavNext"
                type="button"
                aria-label="Next image"
                (click)="showNextProjectImage()"
              >
                <hugeicons-icon
                  [icon]="nextIcon"
                  [size]="24"
                  color="currentColor"
                  [strokeWidth]="2.2"
                  aria-hidden="true"
                />
              </button>
            }
          </div>

          <footer class="ProjectLightboxFooter">
            <p>{{ activeProjectImageCaption() }}</p>
            @if (hasMultipleProjectImages()) {
              <span>{{ activeProjectImageIndex() + 1 }} / {{ activeProjectImages().length }}</span>
            }
          </footer>
        </section>
      </div>
    }
  `,
  styles: [`
    :host {
      display: block;
    }

    .SectionPage {
      min-height: 100dvh;
      display: grid;
      align-content: center;
      justify-items: center;
      gap: 24px;
      padding: 96px 32px;
    }

    .SectionPageTop {
      align-content: start;
      justify-items: center;
      padding-block-start: 118px;
    }

    .SectionPageScrollable {
      height: 100dvh;
      min-height: 0;
      overflow-y: auto;
      overscroll-behavior: contain;
      padding-block-start: 0;
      padding-block-end: 118px;
      scrollbar-gutter: stable;
    }

    .SectionPageScrollable .SectionPanel {
      position: sticky;
      inset-block-start: 0;
      z-index: 4;
      margin-block-start: 118px;
      padding-block: 10px 16px;
      isolation: isolate;
      background: #ffffff;
    }

    .SectionPageScrollable .SectionPanel::before {
      content: '';
      position: absolute;
      inset-block: 0;
      inset-inline: -100vw;
      z-index: -1;
      background: #ffffff;
    }

    .SectionPageScrollable .SectionPanel h1 {
      font-size: 40px;
    }

    :host-context([data-theme='dark']) .SectionPageScrollable .SectionPanel,
    :host-context([data-theme='dark']) .SectionPageScrollable .SectionPanel::before {
      background: #17191e;
    }

    .SectionPanel {
      width: min(100%, 820px);
      display: grid;
      justify-items: center;
      gap: 18px;
      text-align: center;
    }

    .SectionPageTop .SectionPanel {
      width: min(100%, 960px);
      justify-items: start;
      gap: 20px;
      text-align: start;
    }

    h1 {
      margin: 0;
      color: var(--app-heading);
      font-size: 44px;
      font-weight: 900;
      line-height: 1.08;
    }

    p {
      max-width: 620px;
      margin: 0;
      color: var(--app-copy);
      font-size: 12px;
      line-height: 1.5;
    }

    .SectionLinks {
      width: 100%;
    }

    .SectionLinks ul {
      display: flex;
      flex-wrap: wrap;
      justify-content: center;
      gap: 18px 32px;
      margin: 4px 0 0;
      padding-inline-start: 1.25rem;
    }

    .SectionLinks li {
      color: var(--app-heading);
    }

    .SectionPageSocial .SectionLinks ul {
      display: grid;
      gap: 18px;
      justify-items: start;
      justify-content: start;
      margin-block-start: 6px;
    }

    .SectionLinks a {
      color: var(--app-heading);
      font-size: 15px;
      font-weight: 750;
      line-height: 1.45;
      text-decoration: none;
    }

    .SectionPageSocial .SectionLinks a {
      font-size: 16px;
      font-weight: 400;
    }

    .SectionLinks a:hover {
      text-decoration: underline;
      text-underline-offset: 5px;
    }

    .SectionLinks a:focus-visible {
      outline: 3px solid var(--app-toggle-focus);
      outline-offset: 4px;
    }

    .SectionEmpty {
      color: var(--app-muted);
      font-size: 15px;
      line-height: 1.5;
    }

    .SectionContent {
      width: min(100%, 960px);
    }

    .SkillContent {
      width: min(100%, 1040px);
    }

    .SkillSearch {
      display: block;
      margin-block-end: 18px;
    }

    .SkillSearch input {
      width: min(100%, 360px);
      min-height: 38px;
      border: 1px solid color-mix(in srgb, var(--app-heading) 14%, transparent);
      border-radius: 8px;
      background: transparent;
      color: var(--app-heading);
      font: inherit;
      font-size: 13px;
      padding: 8px 11px;
    }

    .SkillSearch input::placeholder {
      color: var(--app-muted);
    }

    .SkillSearch input:focus,
    .SkillSearch input:focus-visible,
    .SkillSearch input:active {
      border-color: color-mix(in srgb, var(--app-heading) 28%, var(--app-panel-border));
      outline: 0;
    }

    .ContentHeading {
      margin: 0 0 12px;
      color: var(--app-heading);
      font-size: 15px;
      font-weight: 850;
      line-height: 1.2;
    }

    .ScreenReaderOnly {
      position: absolute;
      width: 1px;
      height: 1px;
      padding: 0;
      margin: -1px;
      overflow: hidden;
      clip: rect(0, 0, 0, 0);
      white-space: nowrap;
      border: 0;
    }

    .ProjectGrid,
    .SkillGrid,
    .CertificateGrid,
    .ToolGrid,
    .WorkList {
      display: grid;
      gap: 16px;
    }

    .ProjectGrid,
    .SkillGrid,
    .CertificateGrid {
      grid-template-columns: repeat(auto-fit, minmax(260px, 1fr));
    }

    .ProjectArchiveNav {
      margin-block-start: 28px;
    }

    .ProjectArchiveLink {
      display: inline-flex;
      align-items: center;
      gap: 7px;
      color: var(--app-heading);
      font-size: 14px;
      font-weight: 850;
      text-decoration: none;
    }

    .ProjectArchiveLink:hover {
      text-decoration: underline;
      text-underline-offset: 5px;
    }

    .SkillGrid {
      grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
      align-items: start;
      gap: 28px 24px;
      margin-block: 28px 40px;
    }

    .ToolGrid {
      grid-template-columns: repeat(auto-fit, minmax(210px, 1fr));
      align-items: start;
    }

    .SectionCard {
      display: grid;
      gap: 12px;
      border: 1px solid color-mix(in srgb, var(--app-heading) 14%, transparent);
      border-radius: 8px;
      padding: 18px;
      background: color-mix(in srgb, var(--app-background) 88%, var(--app-heading) 4%);
    }

    .SectionCard h2,
    .SectionCard h3,
    .SectionCard p,
    .SectionCard ul {
      margin: 0;
    }

    .SectionCard h2 {
      color: var(--app-heading);
      font-size: 18px;
      font-weight: 850;
      line-height: 1.25;
    }

    .SectionCard h3 {
      color: var(--app-heading);
      font-size: 15px;
      font-weight: 850;
      line-height: 1.2;
    }

    .SectionCard p,
    .SectionCard li,
    .SectionCard span,
    .SectionCard strong,
    .SectionCard a {
      color: var(--app-copy);
      font-size: 14px;
      line-height: 1.5;
    }

    .SectionCard a {
      color: var(--app-heading);
      font-weight: 750;
      text-decoration-thickness: 1px;
      text-underline-offset: 4px;
    }

    .PillList {
      display: flex;
      flex-wrap: wrap;
      gap: 8px;
      padding: 0;
      list-style: none;
    }

    .PillList li {
      border: 1px solid color-mix(in srgb, var(--app-heading) 14%, transparent);
      border-radius: 999px;
      padding: 5px 10px;
      color: var(--app-heading);
      font-weight: 700;
    }

    .SkillCard {
      gap: 10px;
      padding: 16px;
    }

    .SkillCard--currentFocus {
      position: relative;
      z-index: 1;
      border-color: #17191e;
      background: #17191e;
      box-shadow: rgba(0, 0, 0, 0.35) 0 5px 15px;
      transform: scale(1.1);
    }

    .SkillCard--currentFocus h2,
    .SkillCard--currentFocus p,
    .SkillCard--currentFocus .SkillPillList li {
      color: #ffffff;
    }

    .SkillCard--currentFocus .SkillPillList li {
      border-color: rgba(255, 255, 255, 0.24);
      background: rgba(255, 255, 255, 0.08);
    }

    :host-context([data-theme='dark']) .SkillCard--currentFocus {
      border-color: #f7f4ee;
      background: #f7f4ee;
    }

    :host-context([data-theme='dark']) .SkillCard--currentFocus h2,
    :host-context([data-theme='dark']) .SkillCard--currentFocus p,
    :host-context([data-theme='dark']) .SkillCard--currentFocus .SkillPillList li {
      color: #17191e;
    }

    :host-context([data-theme='dark']) .SkillCard--currentFocus .SkillPillList li {
      border-color: rgba(23, 25, 30, 0.22);
      background: rgba(23, 25, 30, 0.06);
    }

    .SkillCard h2 {
      font-size: 16px;
    }

    .SkillCard p {
      font-size: 13px;
      line-height: 1.4;
    }

    .SkillPillList {
      gap: 6px;
    }

    .SkillPillList li {
      border-radius: 6px;
      padding: 4px 7px;
      font-size: 12px;
      font-weight: 750;
      line-height: 1.25;
      overflow-wrap: anywhere;
    }

    .ToolCard {
      gap: 8px;
      padding: 14px;
    }

    .ToolCard > div {
      display: flex;
      align-items: baseline;
      justify-content: space-between;
      gap: 12px;
    }

    .ToolCard span {
      color: var(--app-muted);
      font-size: 11px;
      font-weight: 750;
      line-height: 1.25;
      text-align: end;
      text-transform: uppercase;
    }

    .ToolCard p {
      font-size: 13px;
      line-height: 1.4;
    }

    .CertificateCard {
      grid-template-columns: 96px 1fr;
      align-items: center;
    }

    .CertificateCard img {
      width: 96px;
      aspect-ratio: 4 / 3;
      object-fit: cover;
      border-radius: 6px;
      background: color-mix(in srgb, var(--app-heading) 8%, transparent);
    }

    .ProjectCard {
      align-content: start;
      overflow: hidden;
      padding: 0;
    }

    .ProjectCard > div {
      display: grid;
      gap: 12px;
      padding: 0 18px 18px;
    }

    .ProjectImageButton {
      border: 1px solid color-mix(in srgb, var(--app-heading) 12%, transparent);
      background: color-mix(in srgb, var(--app-background) 88%, var(--app-heading) 7%);
      color: inherit;
      display: block;
      padding: 0;
      cursor: zoom-in;
    }

    .ProjectImageButton:focus-visible,
    .ProjectLightboxIconButton:focus-visible,
    .ProjectLightboxNav:focus-visible {
      outline: 3px solid var(--app-toggle-focus);
      outline-offset: 3px;
    }

    .ProjectImageButton img {
      display: block;
      filter: grayscale(0.72) saturate(0.58) contrast(1.02) brightness(1.02);
      transition:
        filter 180ms ease,
        opacity 160ms ease,
        transform 180ms ease;
    }

    .ProjectImageButton:hover img,
    .ProjectImageButton:focus-visible img {
      filter: grayscale(0) saturate(1) contrast(1) brightness(1);
      opacity: 0.9;
      transform: scale(1.015);
    }

    .ProjectImageButtonMain {
      overflow: hidden;
    }

    .ProjectImageButtonMain img {
      width: 100%;
      aspect-ratio: 16 / 9;
      object-fit: cover;
      background: color-mix(in srgb, var(--app-heading) 8%, transparent);
    }

    .ProjectGallery {
      display: grid;
      grid-template-columns: repeat(4, minmax(0, 1fr));
      gap: 4px;
      padding: 4px 4px 0;
    }

    .ProjectThumbnailButton {
      overflow: hidden;
      border-radius: 4px;
    }

    .ProjectThumbnailButton img {
      width: 100%;
      aspect-ratio: 4 / 3;
      object-fit: cover;
      background: color-mix(in srgb, var(--app-heading) 8%, transparent);
    }

    .ProjectLightbox {
      position: fixed;
      inset: 0;
      z-index: 80;
      display: grid;
      place-items: center;
      padding: 24px;
      background: rgba(10, 10, 10, 0.74);
    }

    .ProjectLightboxPanel {
      width: min(100%, 1040px);
      max-height: calc(100dvh - 48px);
      border: 1px solid color-mix(in srgb, #ffffff 20%, transparent);
      border-radius: 8px;
      background: var(--app-page-bg);
      color: var(--app-heading);
      display: grid;
      grid-template-rows: auto minmax(0, 1fr) auto;
      overflow: hidden;
      box-shadow: 0 28px 80px rgba(0, 0, 0, 0.28);
    }

    .ProjectLightboxTop,
    .ProjectLightboxFooter {
      display: flex;
      align-items: center;
      justify-content: space-between;
      gap: 18px;
      padding: 16px 18px;
    }

    .ProjectLightboxTop h2,
    .ProjectLightboxTop p,
    .ProjectLightboxFooter p {
      margin: 0;
    }

    .ProjectLightboxTop h2 {
      color: var(--app-heading);
      font-size: 18px;
      font-weight: 850;
      line-height: 1.2;
    }

    .ProjectLightboxEyebrow {
      color: var(--app-muted);
      font-size: 11px;
      font-weight: 900;
      line-height: 1.2;
      text-transform: uppercase;
    }

    .ProjectLightboxIconButton,
    .ProjectLightboxNav {
      border: 0;
      background: transparent;
      color: var(--app-heading);
      display: grid;
      place-items: center;
      padding: 0;
      cursor: pointer;
    }

    .ProjectLightboxIconButton {
      width: 38px;
      height: 38px;
      flex: 0 0 auto;
    }

    .ProjectLightboxImageWrap {
      position: relative;
      min-height: 0;
      display: grid;
      place-items: center;
      background: color-mix(in srgb, var(--app-heading) 5%, transparent);
      overflow: hidden;
    }

    .ProjectLightboxImageWrap img {
      width: 100%;
      height: 100%;
      max-height: calc(100dvh - 190px);
      object-fit: contain;
      display: block;
    }

    .ProjectLightboxNav {
      position: absolute;
      inset-block-start: 50%;
      z-index: 1;
      width: 44px;
      height: 44px;
      border-radius: 999px;
      background: color-mix(in srgb, var(--app-page-bg) 88%, transparent);
      box-shadow: 0 14px 34px rgba(0, 0, 0, 0.18);
      transform: translateY(-50%);
    }

    .ProjectLightboxNavPrev {
      inset-inline-start: 14px;
    }

    .ProjectLightboxNavNext {
      inset-inline-end: 14px;
    }

    .ProjectLightboxFooter {
      color: var(--app-copy);
    }

    .ProjectLightboxFooter p {
      max-width: none;
      font-size: 13px;
      line-height: 1.45;
    }

    .ProjectLightboxFooter span {
      flex: 0 0 auto;
      color: var(--app-muted);
      font-size: 12px;
      font-weight: 850;
    }

    .CardLinks {
      display: flex;
      flex-wrap: wrap;
      gap: 12px;
    }

    .WorkToolbar {
      display: flex;
      justify-content: flex-end;
      margin-block-end: 18px;
    }

    .SortButton {
      width: 34px;
      height: 34px;
      border: 0;
      background: transparent;
      color: var(--app-muted);
      display: grid;
      place-items: center;
      padding: 0;
      cursor: pointer;
      transition:
        color 160ms ease,
        transform 160ms ease;
    }

    .SortButton:hover {
      color: var(--app-heading);
      transform: translateY(-1px);
    }

    .SortButton:focus-visible {
      outline: 3px solid var(--app-toggle-focus);
      outline-offset: 3px;
    }

    hugeicons-icon {
      display: inline-grid;
      color: currentColor;
      line-height: 0;
    }

    .WorkCard {
      position: relative;
      gap: 18px;
      margin-inline-start: 28px;
      padding: 22px;
    }

    .WorkCard::before {
      content: '';
      position: absolute;
      inset-block-start: 34px;
      inset-inline-start: -28px;
      width: 9px;
      height: 9px;
      border-radius: 50%;
      background: var(--app-heading);
      box-shadow: 0 0 0 5px var(--app-page-bg);
    }

    .WorkCard::after {
      content: '';
      position: absolute;
      inset-block-start: 47px;
      inset-block-end: -31px;
      inset-inline-start: -24px;
      width: 1px;
      background: color-mix(in srgb, var(--app-heading) 22%, transparent);
    }

    .WorkCard:last-child::after {
      display: none;
    }

    .WorkCard ul {
      display: grid;
      gap: 12px;
      margin-block-start: 6px;
      padding-inline-start: 1.2rem;
    }

    .WorkHeader {
      display: flex;
      justify-content: space-between;
      gap: 28px;
    }

    .WorkHeader > div {
      display: grid;
      gap: 6px;
    }

    .WorkHeader > span {
      flex: 0 0 auto;
      text-align: end;
    }

    @media (max-width: 720px) {
      .SectionPage {
        align-content: start;
        gap: 20px;
        padding: 92px 16px 118px;
      }

      .SectionPageTop {
        padding-block-start: 96px;
      }

      .SectionPageScrollable {
        height: 100dvh;
        min-height: 0;
        overflow-y: auto;
        scrollbar-gutter: stable;
        padding-block-start: 0;
      }

      .SectionPageScrollable .SectionPanel {
        inset-block-start: 0;
        margin-block-start: 96px;
      }

      .SectionPageScrollable .SectionPanel h1 {
        font-size: 31px;
      }

      .SectionPanel,
      .SectionPageTop .SectionPanel,
      .SectionContent {
        width: 100%;
      }

      h1 {
        font-size: 34px;
        line-height: 1.12;
      }

      p {
        font-size: 12px;
        line-height: 1.5;
      }

      .SectionPageTop .SectionPanel {
        gap: 16px;
      }

      .ProjectGrid,
      .SkillGrid,
      .ToolGrid,
      .CertificateGrid {
        grid-template-columns: minmax(0, 1fr);
      }

      .ToolCard > div {
        align-items: start;
      }

      .SectionCard {
        min-width: 0;
        padding: 16px;
      }

      .CertificateCard {
        grid-template-columns: 1fr;
      }

      .CertificateCard img {
        width: min(100%, 220px);
      }

      .WorkHeader {
        display: grid;
        gap: 10px;
      }

      .WorkHeader > span {
        text-align: start;
      }

      .ProjectLightbox {
        align-items: stretch;
        padding: 0;
      }

      .ProjectLightboxPanel {
        width: 100%;
        max-height: 100dvh;
        min-height: 100dvh;
        border: 0;
        border-radius: 0;
      }

      .ProjectLightboxTop,
      .ProjectLightboxFooter {
        padding-inline: 16px;
      }

      .ProjectLightboxImageWrap img {
        max-height: calc(100dvh - 190px);
      }

      .ProjectLightboxNav {
        inset-block-start: auto;
        inset-block-end: 14px;
        transform: none;
      }

      .WorkCard {
        margin-inline-start: 22px;
        padding: 18px;
      }

      .WorkCard::before {
        inset-block-start: 28px;
        inset-inline-start: -23px;
      }

      .WorkCard::after {
        inset-block-start: 41px;
        inset-block-end: -25px;
        inset-inline-start: -19px;
      }

      .SectionLinks ul {
        gap: 14px 26px;
      }

      .SectionPageSocial .SectionLinks ul {
        gap: 16px;
      }
    }

    @media (max-width: 380px) {
      .SectionPage {
        padding-inline: 14px;
      }

      h1 {
        font-size: 30px;
      }

      p {
        font-size: 12px;
      }
    }
  `],
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class SectionPageComponent {
  private readonly route = inject(ActivatedRoute);
  private readonly portfolioApi = inject(PortfolioApi);
  private readonly currentPath = this.route.snapshot.routeConfig?.path;
  private readonly workSortDirection = signal<'asc' | 'desc'>('desc');
  private lightboxReturnTarget: HTMLElement | null = null;
  protected readonly skillSearchQuery = signal('');
  protected readonly activeProject = signal<ProjectSummary | null>(null);
  protected readonly activeProjectImageIndex = signal(0);
  protected readonly sortIcon = ArrowUpDownIcon;
  protected readonly archiveIcon = ArchiveIcon;
  protected readonly closeIcon = Cancel01Icon;
  protected readonly previousIcon = ArrowLeft02Icon;
  protected readonly nextIcon = ArrowRight02Icon;
  protected readonly isSocialLinksPage = this.route.snapshot.routeConfig?.path === 'social-links';
  protected readonly isSkillsPage = this.currentPath === 'skills';
  protected readonly isCertificatesPage = this.currentPath === 'certificates';
  protected readonly isWorkHistoryPage = this.currentPath === 'work-history';
  protected readonly isArchivedProjectsPage = this.currentPath === 'projects/archived';
  protected readonly isProjectsPage = this.currentPath === 'projects' || this.isArchivedProjectsPage;
  protected readonly isTopAlignedPage = [
    'projects',
    'projects/archived',
    'work-history',
    'certificates',
    'skills',
    'social-links',
  ].includes(this.currentPath ?? '');
  protected readonly isScrollablePage = (
    this.isProjectsPage
    || this.isWorkHistoryPage
    || this.isCertificatesPage
    || this.isSkillsPage
  );

  protected readonly page = signal(readPageData(this.route.snapshot.data));
  protected readonly socialLinksResource = resource({
    defaultValue: [] as SectionPageLink[],
    loader: ({ abortSignal }) => {
      if (!this.isSocialLinksPage) {
        return Promise.resolve([]);
      }

      return this.portfolioApi.listPublishedLinks(abortSignal)
        .then((links) => links.map((link) => ({
          label: link.label,
          url: link.url,
        })));
    },
  });
  protected readonly projectsResource = resource({
    defaultValue: [] as ProjectSummary[],
    params: () => ({ archived: this.isArchivedProjectsPage }),
    loader: ({ abortSignal, params }) => {
      if (!this.isProjectsPage) {
        return Promise.resolve([]);
      }

      return params.archived
        ? this.portfolioApi.listArchivedProjects(abortSignal)
        : this.portfolioApi.listPublishedProjects(abortSignal);
    },
  });
  protected readonly skillsResource = resource({
    defaultValue: { categories: [], skills: [], tools: [] } as SkillsPageContent,
    loader: async ({ abortSignal }) => {
      if (!this.isSkillsPage) {
        return { categories: [], skills: [], tools: [] };
      }

      const [categories, skills, tools] = await Promise.all([
        this.portfolioApi.listPublishedSkillCategories(abortSignal),
        this.portfolioApi.listPublishedSkills(abortSignal),
        this.portfolioApi.listPublishedTools(abortSignal),
      ]);

      return { categories, skills, tools };
    },
  });
  protected readonly certificatesResource = resource({
    defaultValue: [] as Certificate[],
    loader: ({ abortSignal }) => {
      if (!this.isCertificatesPage) {
        return Promise.resolve([]);
      }

      return this.portfolioApi.listPublishedCertificates(abortSignal);
    },
  });
  protected readonly workExperiencesResource = resource({
    defaultValue: [] as WorkExperience[],
    loader: ({ abortSignal }) => {
      if (!this.isWorkHistoryPage) {
        return Promise.resolve([]);
      }

      return this.portfolioApi.listPublishedWorkExperiences(abortSignal);
    },
  });
  protected readonly sectionLinks = computed<SectionPageLink[]>(() => {
    if (!this.isSocialLinksPage) {
      return this.page().links ?? [];
    }

    const linksByUrl = new Map<string, SectionPageLink>();

    for (const link of [
      ...(this.page().links ?? []),
      ...this.socialLinksResource.value(),
    ]) {
      linksByUrl.set(link.url, link);
    }

    return Array.from(linksByUrl.values());
  });
  protected readonly socialLinksLoadFailed = computed(() => (
    this.isSocialLinksPage && Boolean(this.socialLinksResource.error())
  ));
  protected readonly projectsLoadFailed = computed(() => (
    this.isProjectsPage && Boolean(this.projectsResource.error())
  ));
  protected readonly skillsLoadFailed = computed(() => (
    this.isSkillsPage && Boolean(this.skillsResource.error())
  ));
  protected readonly certificatesLoadFailed = computed(() => (
    this.isCertificatesPage && Boolean(this.certificatesResource.error())
  ));
  protected readonly workExperiencesLoadFailed = computed(() => (
    this.isWorkHistoryPage && Boolean(this.workExperiencesResource.error())
  ));
  protected readonly filteredSkillCategories = computed(() => {
    const query = this.normalizedSkillSearch();
    const categories = this.skillsResource.value().categories;

    if (!query) {
      return categories;
    }

    return categories.filter((category) => (
      this.categoryMatchesSkillSearch(category, query)
      || this.skillsForCategory(category.id).some((skill) => this.skillMatchesSearch(skill, query))
    ));
  });
  protected readonly filteredTools = computed(() => {
    const query = this.normalizedSkillSearch();
    const tools = this.skillsResource.value().tools;

    if (!query) {
      return tools;
    }

    return tools.filter((tool) => this.toolMatchesSearch(tool, query));
  });
  protected readonly sortedWorkExperiences = computed(() => (
    [...this.workExperiencesResource.value()].sort((left, right) => {
      const leftTime = Date.parse(left.started_at);
      const rightTime = Date.parse(right.started_at);
      const direction = this.workSortDirection() === 'asc' ? 1 : -1;

      return direction * (leftTime - rightTime);
    })
  ));
  protected readonly activeProjectImages = computed(() => {
    const project = this.activeProject();

    return project ? this.projectImages(project) : [];
  });
  protected readonly activeProjectImage = computed(() => (
    this.activeProjectImages()[this.activeProjectImageIndex()]
  ));

  protected workSortLabel(): string {
    return this.workSortDirection() === 'desc'
      ? 'Sort work history oldest first'
      : 'Sort work history newest first';
  }

  protected toggleWorkSort(): void {
    this.workSortDirection.update((direction) => (direction === 'desc' ? 'asc' : 'desc'));
  }

  protected projectImages(project: ProjectSummary): ProjectImage[] {
    const images = [
      project.main_image,
      ...(project.images ?? []),
    ].filter((image): image is ProjectImage => Boolean(image));
    const seen = new Set<string>();

    return images
      .filter((image) => {
        const key = image.id || image.url;

        if (seen.has(key)) {
          return false;
        }

        seen.add(key);
        return true;
      })
      .sort((left, right) => left.sort_order - right.sort_order);
  }

  protected projectImageButtonLabel(project: ProjectSummary, image: ProjectImage): string {
    return `View ${image.caption || image.alt_text || project.title} screenshot`;
  }

  protected openProjectLightbox(project: ProjectSummary, image: ProjectImage, event: Event): void {
    const images = this.projectImages(project);
    const imageIndex = Math.max(0, images.findIndex((candidate) => (
      candidate.id === image.id || candidate.url === image.url
    )));

    this.lightboxReturnTarget = event.currentTarget instanceof HTMLElement ? event.currentTarget : null;
    this.activeProject.set(project);
    this.activeProjectImageIndex.set(imageIndex);
  }

  protected closeProjectLightbox(): void {
    this.activeProject.set(null);
    this.activeProjectImageIndex.set(0);

    window.setTimeout(() => {
      this.lightboxReturnTarget?.focus();
      this.lightboxReturnTarget = null;
    });
  }

  protected hasMultipleProjectImages(): boolean {
    return this.activeProjectImages().length > 1;
  }

  protected activeProjectTitle(): string {
    return this.activeProject()?.title ?? 'Project';
  }

  protected activeProjectImageCaption(): string {
    const image = this.activeProjectImage();

    return image?.caption || image?.alt_text || this.activeProjectTitle();
  }

  protected showPreviousProjectImage(): void {
    const images = this.activeProjectImages();

    if (images.length < 2) {
      return;
    }

    this.activeProjectImageIndex.update((index) => (index + images.length - 1) % images.length);
  }

  protected showNextProjectImage(): void {
    const images = this.activeProjectImages();

    if (images.length < 2) {
      return;
    }

    this.activeProjectImageIndex.update((index) => (index + 1) % images.length);
  }

  @HostListener('document:keydown', ['$event'])
  protected handleProjectLightboxKeydown(event: KeyboardEvent): void {
    if (!this.activeProject()) {
      return;
    }

    if (event.key === 'Escape') {
      event.preventDefault();
      this.closeProjectLightbox();
      return;
    }

    if (event.key === 'ArrowLeft') {
      event.preventDefault();
      this.showPreviousProjectImage();
      return;
    }

    if (event.key === 'ArrowRight') {
      event.preventDefault();
      this.showNextProjectImage();
    }
  }

  protected updateSkillSearch(event: Event): void {
    const input = event.target instanceof HTMLInputElement ? event.target.value : '';
    this.skillSearchQuery.set(input);
  }

  protected filteredSkillsForCategory(category: SkillCategory): Skill[] {
    const query = this.normalizedSkillSearch();
    const skills = this.skillsForCategory(category.id);

    if (!query || this.categoryMatchesSkillSearch(category, query)) {
      return skills;
    }

    return skills.filter((skill) => this.skillMatchesSearch(skill, query));
  }

  protected skillsForCategory(categoryID: string): Skill[] {
    return this.skillsResource.value().skills.filter((skill) => skill.category_id === categoryID);
  }

  protected isFocusSkillCategory(category: SkillCategory): boolean {
    const slug = category.slug.toLowerCase();
    const name = category.name.toLowerCase();

    return slug === 'current-focus'
      || slug === 'ai-assisted-engineering'
      || name === 'current focus'
      || name === 'ai-assisted engineering';
  }

  private normalizedSkillSearch(): string {
    return this.skillSearchQuery().trim().toLowerCase();
  }

  private categoryMatchesSkillSearch(category: SkillCategory, query: string): boolean {
    return `${category.name} ${category.description ?? ''}`.toLowerCase().includes(query);
  }

  private skillMatchesSearch(skill: Skill, query: string): boolean {
    return `${skill.name} ${skill.summary ?? ''} ${skill.icon_class ?? ''}`.toLowerCase().includes(query);
  }

  private toolMatchesSearch(tool: Tool, query: string): boolean {
    return `${tool.name} ${tool.category} ${tool.summary ?? ''} ${(tool.tags ?? []).join(' ')}`.toLowerCase().includes(query);
  }

  protected workDateRange(experience: WorkExperience): string {
    const start = this.formatMonthYear(experience.started_at);
    const end = experience.current ? 'Present' : this.formatMonthYear(experience.ended_at);

    return end ? `${start} - ${end}` : start;
  }

  private formatMonthYear(value: string | undefined): string {
    if (!value) {
      return '';
    }

    const formatter = new Intl.DateTimeFormat('en', {
      month: 'short',
      year: 'numeric',
      timeZone: 'UTC',
    });

    return formatter.format(new Date(value));
  }
}
