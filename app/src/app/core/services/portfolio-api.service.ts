import { DOCUMENT } from '@angular/common';
import { inject, Injectable } from '@angular/core';

import type { AuthSession, LoginRequest } from '../models/auth.models';
import type {
  ContactProfile,
  ContactSubmission,
  ContactSubmissionRequest,
} from '../models/contact.models';
import type { MusicEntry } from '../models/music.models';
import type { IntroSummary, PortfolioLink, ProjectSummary } from '../models/portfolio.models';

interface LinksResponse {
  links: PortfolioLink[];
}

interface ProjectsResponse {
  projects: ProjectSummary[];
}

interface MusicResponse {
  music: MusicEntry[];
}

interface IntroResponse extends IntroSummary {
  id: string;
}

interface LoginResponse {
  token: string;
  token_type: string;
  expires_in: number;
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

  async listPublishedLinks(abortSignal: AbortSignal): Promise<PortfolioLink[]> {
    const payload = await this.getJson('/links?status=published', abortSignal);
    if (!isLinksResponse(payload)) {
      throw new Error('Links response was not valid.');
    }

    return payload.links
      .filter((link) => link.status === 'published')
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

  async getIntro(abortSignal: AbortSignal): Promise<IntroSummary> {
    const payload = await this.getJson('/intro', abortSignal);
    if (!isIntroResponse(payload)) {
      throw new Error('Intro response was not valid.');
    }

    return {
      title: payload.title,
      description: payload.description,
    };
  }

  async login(input: LoginRequest): Promise<AuthSession> {
    const response = await this.fetch(`${this.apiBaseUrl}/auth/login`, {
      method: 'POST',
      headers: {
        Accept: 'application/json',
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(input),
    });

    if (!response.ok) {
      throw new Error(`Login failed with status ${response.status}.`);
    }

    const payload: unknown = await response.json();
    if (!isLoginResponse(payload)) {
      throw new Error('Login response was not valid.');
    }

    return {
      token: payload.token,
      tokenType: payload.token_type,
      expiresIn: payload.expires_in,
    };
  }

  async submitContact(input: ContactSubmissionRequest): Promise<ContactSubmission> {
    const response = await this.fetch(`${this.apiBaseUrl}/contact-submissions`, {
      method: 'POST',
      headers: {
        Accept: 'application/json',
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(input),
    });

    if (!response.ok) {
      throw new Error(`Contact submission failed with status ${response.status}.`);
    }

    const payload: unknown = await response.json();
    if (!isContactSubmission(payload)) {
      throw new Error('Contact response was not valid.');
    }

    return payload;
  }

  async getContactProfile(abortSignal: AbortSignal): Promise<ContactProfile> {
    const payload = await this.getJson('/contact-profile', abortSignal);
    if (!isContactProfile(payload)) {
      throw new Error('Contact profile response was not valid.');
    }

    return payload;
  }

  async listTopMusic(abortSignal: AbortSignal): Promise<MusicEntry[]> {
    const payload = await this.getJson('/music?status=published', abortSignal);
    if (!isMusicResponse(payload)) {
      throw new Error('Music response was not valid.');
    }

    return payload.music
      .filter((entry) => entry.status === 'published')
      .sort((a, b) => a.sort_order - b.sort_order || a.title.localeCompare(b.title))
      .slice(0, 5);
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

function isIntroResponse(payload: unknown): payload is IntroResponse {
  return (
    isRecord(payload)
    && typeof payload['id'] === 'string'
    && typeof payload['title'] === 'string'
    && typeof payload['description'] === 'string'
  );
}

function isLoginResponse(payload: unknown): payload is LoginResponse {
  return (
    isRecord(payload)
    && typeof payload['token'] === 'string'
    && typeof payload['token_type'] === 'string'
    && typeof payload['expires_in'] === 'number'
  );
}

function isContactSubmission(payload: unknown): payload is ContactSubmission {
  return (
    isRecord(payload)
    && typeof payload['id'] === 'string'
    && typeof payload['name'] === 'string'
    && typeof payload['email'] === 'string'
    && (typeof payload['subject'] === 'string' || payload['subject'] === undefined)
    && typeof payload['message'] === 'string'
    && typeof payload['created_at'] === 'string'
  );
}

function isContactProfile(payload: unknown): payload is ContactProfile {
  return (
    isRecord(payload)
    && typeof payload['id'] === 'string'
    && typeof payload['work_email'] === 'string'
    && typeof payload['phone_number'] === 'string'
    && typeof payload['created_at'] === 'string'
    && typeof payload['updated_at'] === 'string'
  );
}

function isMusicResponse(payload: unknown): payload is MusicResponse {
  return isRecord(payload) && Array.isArray(payload['music']) && payload['music'].every(isMusicEntry);
}

function isMusicEntry(payload: unknown): payload is MusicEntry {
  return (
    isRecord(payload)
    && typeof payload['id'] === 'string'
    && typeof payload['title'] === 'string'
    && typeof payload['artist'] === 'string'
    && (typeof payload['album'] === 'string' || payload['album'] === undefined)
    && (typeof payload['spotify_url'] === 'string' || payload['spotify_url'] === undefined)
    && (typeof payload['youtube_url'] === 'string' || payload['youtube_url'] === undefined)
    && typeof payload['mostly_listened_on'] === 'string'
    && (typeof payload['notes'] === 'string' || payload['notes'] === undefined)
    && typeof payload['sort_order'] === 'number'
    && isStatus(payload['status'])
  );
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
