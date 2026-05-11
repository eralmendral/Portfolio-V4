import { DOCUMENT } from '@angular/common';
import { inject, Injectable } from '@angular/core';

export interface PortfolioLink {
  id: string;
  label: string;
  url: string;
  icon_class?: string;
  sort_order: number;
  star: boolean;
  status: 'draft' | 'published' | 'archived';
}

export interface ProjectSummary {
  id: string;
  slug: string;
  title: string;
  summary?: string;
  tech_stack?: string[];
  featured: boolean;
  sort_order: number;
  status: 'draft' | 'published' | 'archived';
}

interface LinksResponse {
  links: PortfolioLink[];
}

interface ProjectsResponse {
  projects: ProjectSummary[];
}

interface PortfolioWindow extends Window {
  __PORTFOLIO_API_BASE_URL__?: string;
}

@Injectable({
  providedIn: 'root',
})
export class PortfolioApi {
  private readonly document = inject(DOCUMENT);
  private readonly apiBaseUrl = this.resolveApiBaseUrl();

  async listStarredLinks(abortSignal: AbortSignal): Promise<PortfolioLink[]> {
    const payload = await this.getJson('/links?status=published&star=true', abortSignal);
    if (!isLinksResponse(payload)) {
      throw new Error('Links response was not valid.');
    }

    return payload.links
      .filter((link) => link.star && link.status === 'published')
      .sort((a, b) => a.sort_order - b.sort_order || a.label.localeCompare(b.label));
  }

  async listFeaturedProjects(abortSignal: AbortSignal): Promise<ProjectSummary[]> {
    const payload = await this.getJson('/projects?status=published&featured=true', abortSignal);
    if (!isProjectsResponse(payload)) {
      throw new Error('Projects response was not valid.');
    }

    return payload.projects
      .filter((project) => project.featured && project.status === 'published')
      .sort((a, b) => a.sort_order - b.sort_order || a.title.localeCompare(b.title));
  }

  private async getJson(path: string, abortSignal: AbortSignal): Promise<unknown> {
    const response = await this.fetch(`${this.apiBaseUrl}${path}`, {
      headers: {
        Accept: 'application/json',
      },
      signal: abortSignal,
    });

    if (!response.ok) {
      throw new Error(`Request failed with status ${response.status}.`);
    }

    return response.json() as Promise<unknown>;
  }

  private fetch(input: RequestInfo | URL, init?: RequestInit): Promise<Response> {
    const view = this.document.defaultView;
    const fetchRef = globalThis.fetch ?? view?.fetch;

    if (!fetchRef) {
      throw new Error('Fetch is not available.');
    }

    return fetchRef.call(view ?? globalThis, input, init);
  }

  private resolveApiBaseUrl(): string {
    const configuredUrl = this.configuredApiBaseUrl();
    if (configuredUrl) {
      return configuredUrl;
    }

    const location = this.document.defaultView?.location;
    if (!location) {
      return 'http://localhost:8080';
    }

    if (isLocalHost(location.hostname)) {
      return 'http://localhost:8080';
    }

    if (location.hostname.startsWith('admin.')) {
      return `${location.protocol}//api.${location.hostname.slice('admin.'.length)}`;
    }

    return location.origin;
  }

  private configuredApiBaseUrl(): string | null {
    const metaUrl = this.document
      .querySelector<HTMLMetaElement>('meta[name="portfolio-api-base-url"]')
      ?.content
      .trim();
    const windowUrl = (this.document.defaultView as PortfolioWindow | null)
      ?.__PORTFOLIO_API_BASE_URL__
      ?.trim();
    const url = metaUrl || windowUrl;

    return url ? url.replace(/\/$/, '') : null;
  }
}

function isLocalHost(hostname: string): boolean {
  return hostname === 'localhost' || hostname === '127.0.0.1' || hostname === '::1';
}

function isLinksResponse(payload: unknown): payload is LinksResponse {
  if (!isRecord(payload) || !Array.isArray(payload['links'])) {
    return false;
  }

  return payload['links'].every(isPortfolioLink);
}

function isProjectsResponse(payload: unknown): payload is ProjectsResponse {
  if (!isRecord(payload) || !Array.isArray(payload['projects'])) {
    return false;
  }

  return payload['projects'].every(isProjectSummary);
}

function isPortfolioLink(value: unknown): value is PortfolioLink {
  if (!isRecord(value)) {
    return false;
  }

  return (
    typeof value['id'] === 'string'
    && typeof value['label'] === 'string'
    && typeof value['url'] === 'string'
    && (typeof value['icon_class'] === 'string' || value['icon_class'] === undefined)
    && typeof value['sort_order'] === 'number'
    && typeof value['star'] === 'boolean'
    && isStatus(value['status'])
  );
}

function isProjectSummary(value: unknown): value is ProjectSummary {
  if (!isRecord(value)) {
    return false;
  }

  return (
    typeof value['id'] === 'string'
    && typeof value['slug'] === 'string'
    && typeof value['title'] === 'string'
    && (typeof value['summary'] === 'string' || value['summary'] === undefined)
    && (isStringArray(value['tech_stack']) || value['tech_stack'] === undefined)
    && typeof value['featured'] === 'boolean'
    && typeof value['sort_order'] === 'number'
    && isStatus(value['status'])
  );
}

function isStatus(value: unknown): value is 'draft' | 'published' | 'archived' {
  return value === 'draft' || value === 'published' || value === 'archived';
}

function isStringArray(value: unknown): value is string[] {
  return Array.isArray(value) && value.every((item) => typeof item === 'string');
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null;
}
