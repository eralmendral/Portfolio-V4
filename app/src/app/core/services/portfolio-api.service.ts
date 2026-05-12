import { DOCUMENT } from '@angular/common';
import { inject, Injectable } from '@angular/core';

import type { AuthSession, LoginRequest } from '../models/auth.models';
import type {
  ContactProfile,
  ContactSubmission,
  ContactSubmissionRequest,
} from '../models/contact.models';
import type { MusicEntry } from '../models/music.models';
import type {
  Certificate,
  CertificateImage,
  IntroSummary,
  PortfolioLink,
  ProjectSummary,
  Skill,
  SkillCategory,
  Tool,
  WorkExperience,
} from '../models/portfolio.models';

interface LinksResponse {
  links: PortfolioLink[];
}

interface ProjectsResponse {
  projects: ProjectSummary[];
}

interface SkillCategoriesResponse {
  skill_categories: SkillCategory[];
}

interface SkillsResponse {
  skills: Skill[];
}

interface ToolsResponse {
  tools: Tool[];
}

interface CertificatesResponse {
  certificates: Certificate[];
}

interface WorkExperiencesResponse {
  work_experiences: WorkExperience[];
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
    const payload = await this.getJson('/projects?status=published&featured=true&archived=false', abortSignal);
    if (!isProjectsResponse(payload)) {
      throw new Error('Projects response was not valid.');
    }

    return payload.projects
      .filter((project) => project.featured && project.status === 'published' && !project.archived)
      .sort((a, b) => a.sort_order - b.sort_order || a.title.localeCompare(b.title));
  }

  async listPublishedProjects(abortSignal: AbortSignal): Promise<ProjectSummary[]> {
    const payload = await this.getJson('/projects?status=published&archived=false', abortSignal);
    if (!isProjectsResponse(payload)) {
      throw new Error('Projects response was not valid.');
    }

    return payload.projects
      .filter((project) => project.status === 'published' && !project.archived)
      .sort((a, b) => a.sort_order - b.sort_order || a.title.localeCompare(b.title));
  }

  async listArchivedProjects(abortSignal: AbortSignal): Promise<ProjectSummary[]> {
    const payload = await this.getJson('/projects?status=published&archived=true', abortSignal);
    if (!isProjectsResponse(payload)) {
      throw new Error('Projects response was not valid.');
    }

    return payload.projects
      .filter((project) => project.status === 'published' && project.archived)
      .sort((a, b) => a.sort_order - b.sort_order || a.title.localeCompare(b.title));
  }

  async listPublishedSkillCategories(abortSignal: AbortSignal): Promise<SkillCategory[]> {
    const payload = await this.getJson('/skill-categories?status=published', abortSignal);
    if (!isSkillCategoriesResponse(payload)) {
      throw new Error('Skill categories response was not valid.');
    }

    return payload.skill_categories
      .filter((category) => category.status === 'published')
      .sort((a, b) => a.sort_order - b.sort_order || a.name.localeCompare(b.name));
  }

  async listPublishedSkills(abortSignal: AbortSignal): Promise<Skill[]> {
    const payload = await this.getJson('/skills?status=published', abortSignal);
    if (!isSkillsResponse(payload)) {
      throw new Error('Skills response was not valid.');
    }

    return payload.skills
      .filter((skill) => skill.status === 'published')
      .sort((a, b) => a.sort_order - b.sort_order || a.name.localeCompare(b.name));
  }

  async listPublishedTools(abortSignal: AbortSignal): Promise<Tool[]> {
    const payload = await this.getJson('/tools?status=published', abortSignal);
    if (!isToolsResponse(payload)) {
      throw new Error('Tools response was not valid.');
    }

    return payload.tools
      .filter((tool) => tool.status === 'published')
      .sort((a, b) => a.sort_order - b.sort_order || a.name.localeCompare(b.name));
  }

  async listPublishedCertificates(abortSignal: AbortSignal): Promise<Certificate[]> {
    const payload = await this.getJson('/certificates?status=published', abortSignal);
    if (!isCertificatesResponse(payload)) {
      throw new Error('Certificates response was not valid.');
    }

    return payload.certificates
      .filter((certificate) => certificate.status === 'published')
      .sort((a, b) => a.sort_order - b.sort_order || a.title.localeCompare(b.title));
  }

  async listPublishedWorkExperiences(abortSignal: AbortSignal): Promise<WorkExperience[]> {
    const payload = await this.getJson('/work-experiences?status=published', abortSignal);
    if (!isWorkExperiencesResponse(payload)) {
      throw new Error('Work experiences response was not valid.');
    }

    return payload.work_experiences
      .filter((experience) => experience.status === 'published')
      .sort((a, b) => (
        a.sort_order - b.sort_order
        || Date.parse(b.started_at) - Date.parse(a.started_at)
        || a.company.localeCompare(b.company)
      ));
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

function isSkillCategoriesResponse(payload: unknown): payload is SkillCategoriesResponse {
  return (
    isRecord(payload)
    && Array.isArray(payload['skill_categories'])
    && payload['skill_categories'].every(isSkillCategory)
  );
}

function isSkillsResponse(payload: unknown): payload is SkillsResponse {
  return isRecord(payload) && Array.isArray(payload['skills']) && payload['skills'].every(isSkill);
}

function isToolsResponse(payload: unknown): payload is ToolsResponse {
  return isRecord(payload) && Array.isArray(payload['tools']) && payload['tools'].every(isTool);
}

function isCertificatesResponse(payload: unknown): payload is CertificatesResponse {
  return (
    isRecord(payload)
    && Array.isArray(payload['certificates'])
    && payload['certificates'].every(isCertificate)
  );
}

function isWorkExperiencesResponse(payload: unknown): payload is WorkExperiencesResponse {
  return (
    isRecord(payload)
    && Array.isArray(payload['work_experiences'])
    && payload['work_experiences'].every(isWorkExperience)
  );
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
    && (typeof value['description'] === 'string' || value['description'] === undefined)
    && (isStringArray(value['tech_stack']) || value['tech_stack'] === undefined)
    && (isStringArray(value['tags']) || value['tags'] === undefined)
    && (isProjectImage(value['main_image']) || value['main_image'] === undefined)
    && (isProjectImages(value['images']) || value['images'] === undefined)
    && (typeof value['github_url'] === 'string' || value['github_url'] === undefined)
    && (typeof value['demo_url'] === 'string' || value['demo_url'] === undefined)
    && typeof value['featured'] === 'boolean'
    && typeof value['archived'] === 'boolean'
    && typeof value['sort_order'] === 'number'
    && isStatus(value['status'])
  );
}

function isProjectImages(value: unknown): value is ProjectSummary['images'] {
  return Array.isArray(value) && value.every(isProjectImage);
}

function isProjectImage(value: unknown): value is ProjectSummary['main_image'] {
  return (
    isRecord(value)
    && typeof value['id'] === 'string'
    && typeof value['url'] === 'string'
    && (typeof value['alt_text'] === 'string' || value['alt_text'] === undefined)
    && (typeof value['caption'] === 'string' || value['caption'] === undefined)
    && typeof value['sort_order'] === 'number'
  );
}

function isSkillCategory(value: unknown): value is SkillCategory {
  return (
    isRecord(value)
    && typeof value['id'] === 'string'
    && typeof value['slug'] === 'string'
    && typeof value['name'] === 'string'
    && (typeof value['description'] === 'string' || value['description'] === undefined)
    && (typeof value['icon_class'] === 'string' || value['icon_class'] === undefined)
    && typeof value['sort_order'] === 'number'
    && isStatus(value['status'])
  );
}

function isSkill(value: unknown): value is Skill {
  return (
    isRecord(value)
    && typeof value['id'] === 'string'
    && typeof value['category_id'] === 'string'
    && typeof value['name'] === 'string'
    && (typeof value['summary'] === 'string' || value['summary'] === undefined)
    && (typeof value['icon_class'] === 'string' || value['icon_class'] === undefined)
    && typeof value['sort_order'] === 'number'
    && typeof value['featured'] === 'boolean'
    && isStatus(value['status'])
  );
}

function isTool(value: unknown): value is Tool {
  return (
    isRecord(value)
    && typeof value['id'] === 'string'
    && typeof value['name'] === 'string'
    && typeof value['category'] === 'string'
    && (typeof value['summary'] === 'string' || value['summary'] === undefined)
    && (typeof value['icon_class'] === 'string' || value['icon_class'] === undefined)
    && (isStringArray(value['tags']) || value['tags'] === undefined)
    && typeof value['sort_order'] === 'number'
    && typeof value['featured'] === 'boolean'
    && isStatus(value['status'])
  );
}

function isCertificate(value: unknown): value is Certificate {
  return (
    isRecord(value)
    && typeof value['id'] === 'string'
    && typeof value['slug'] === 'string'
    && typeof value['title'] === 'string'
    && typeof value['issuer'] === 'string'
    && (typeof value['summary'] === 'string' || value['summary'] === undefined)
    && (typeof value['description'] === 'string' || value['description'] === undefined)
    && (typeof value['credential_url'] === 'string' || value['credential_url'] === undefined)
    && (isCertificateImage(value['image']) || value['image'] === undefined)
    && typeof value['featured'] === 'boolean'
    && typeof value['sort_order'] === 'number'
    && isStatus(value['status'])
    && (typeof value['issued_at'] === 'string' || value['issued_at'] === undefined)
  );
}

function isCertificateImage(value: unknown): value is CertificateImage {
  return (
    isRecord(value)
    && typeof value['id'] === 'string'
    && typeof value['url'] === 'string'
    && (typeof value['alt_text'] === 'string' || value['alt_text'] === undefined)
    && (typeof value['caption'] === 'string' || value['caption'] === undefined)
  );
}

function isWorkExperience(value: unknown): value is WorkExperience {
  return (
    isRecord(value)
    && typeof value['id'] === 'string'
    && typeof value['slug'] === 'string'
    && typeof value['title'] === 'string'
    && typeof value['company'] === 'string'
    && (typeof value['company_url'] === 'string' || value['company_url'] === undefined)
    && (typeof value['employment_type'] === 'string' || value['employment_type'] === undefined)
    && (typeof value['summary'] === 'string' || value['summary'] === undefined)
    && (typeof value['description'] === 'string' || value['description'] === undefined)
    && (isStringArray(value['responsibilities']) || value['responsibilities'] === undefined)
    && typeof value['started_at'] === 'string'
    && (typeof value['ended_at'] === 'string' || value['ended_at'] === undefined)
    && typeof value['current'] === 'boolean'
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
