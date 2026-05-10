import type {
  ApiFieldErrors,
  Article,
  ArticlePayload,
  Certificate,
  CertificatePayload,
  Intro,
  IntroPayload,
  Link,
  LinkListFilter,
  LinkPayload,
  ListFilter,
  Product,
  ProductListFilter,
  ProductPayload,
  ProductSection,
  ProductSectionPayload,
  Project,
  ProjectPayload,
  Skill,
  SkillCategory,
  SkillCategoryPayload,
  SkillListFilter,
  SkillPayload,
  Status,
  StatusListFilter,
  Tool,
  ToolListFilter,
  ToolPayload,
  WorkExperience,
  WorkExperienceListFilter,
  WorkExperiencePayload,
} from './types.js'

const DEFAULT_API_BASE_URL = 'http://localhost:8080'
const ALL_STATUSES: Status[] = ['draft', 'published', 'archived']

export const API_BASE_URL = (
  import.meta.env.PUBLIC_API_BASE_URL ?? DEFAULT_API_BASE_URL
).replace(/\/$/, '')

interface LoginResponse {
  token: string
  token_type: string
  expires_in: number
}

interface ApiErrorPayload {
  error?: string
  fields?: ApiFieldErrors
}

export class ApiError extends Error {
  status: number
  fields?: ApiFieldErrors

  constructor(status: number, message: string, fields?: ApiFieldErrors) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.fields = fields
  }
}

export interface ApiClient {
  login(username: string, password: string): Promise<LoginResponse>
  listProjects(filter: ListFilter): Promise<Project[]>
  createProject(payload: ProjectPayload): Promise<Project>
  updateProject(idOrSlug: string, payload: ProjectPayload): Promise<Project>
  deleteProject(idOrSlug: string): Promise<void>
  uploadProjectMainImage(idOrSlug: string, file: File, altText: string, caption: string): Promise<Project>
  uploadProjectGalleryImage(idOrSlug: string, file: File, altText: string, caption: string): Promise<Project>
  deleteProjectImage(idOrSlug: string, imageID: string): Promise<Project>
  listCertificates(filter: ListFilter): Promise<Certificate[]>
  createCertificate(payload: CertificatePayload): Promise<Certificate>
  updateCertificate(idOrSlug: string, payload: CertificatePayload): Promise<Certificate>
  deleteCertificate(idOrSlug: string): Promise<void>
  uploadCertificateImage(idOrSlug: string, file: File, altText: string, caption: string): Promise<Certificate>
  deleteCertificateImage(idOrSlug: string): Promise<Certificate>
  listArticles(filter: ListFilter): Promise<Article[]>
  createArticle(payload: ArticlePayload): Promise<Article>
  updateArticle(id: string, payload: ArticlePayload): Promise<Article>
  deleteArticle(id: string): Promise<void>
  listWorkExperiences(filter: WorkExperienceListFilter): Promise<WorkExperience[]>
  createWorkExperience(payload: WorkExperiencePayload): Promise<WorkExperience>
  updateWorkExperience(idOrSlug: string, payload: WorkExperiencePayload): Promise<WorkExperience>
  deleteWorkExperience(idOrSlug: string): Promise<void>
  listLinks(filter: LinkListFilter): Promise<Link[]>
  createLink(payload: LinkPayload): Promise<Link>
  updateLink(id: string, payload: LinkPayload): Promise<Link>
  deleteLink(id: string): Promise<void>
  listSkillCategories(filter: StatusListFilter): Promise<SkillCategory[]>
  createSkillCategory(payload: SkillCategoryPayload): Promise<SkillCategory>
  updateSkillCategory(idOrSlug: string, payload: SkillCategoryPayload): Promise<SkillCategory>
  deleteSkillCategory(idOrSlug: string): Promise<void>
  listSkills(filter: SkillListFilter): Promise<Skill[]>
  createSkill(payload: SkillPayload): Promise<Skill>
  updateSkill(id: string, payload: SkillPayload): Promise<Skill>
  deleteSkill(id: string): Promise<void>
  listTools(filter: ToolListFilter): Promise<Tool[]>
  createTool(payload: ToolPayload): Promise<Tool>
  updateTool(id: string, payload: ToolPayload): Promise<Tool>
  deleteTool(id: string): Promise<void>
  listProducts(filter: ProductListFilter): Promise<Product[]>
  createProduct(payload: ProductPayload): Promise<Product>
  updateProduct(idOrSlug: string, payload: ProductPayload): Promise<Product>
  deleteProduct(idOrSlug: string): Promise<void>
  getProductSection(): Promise<ProductSection>
  updateProductSection(payload: ProductSectionPayload): Promise<ProductSection>
  getIntro(): Promise<Intro>
  updateIntro(payload: IntroPayload): Promise<Intro>
  deleteIntro(): Promise<void>
  uploadIntroProfilePicture(file: File, altText: string, caption: string): Promise<Intro>
  deleteIntroProfilePicture(): Promise<Intro>
}

export function createApiClient(getToken: () => string | null): ApiClient {
  const request = async <T>(path: string, init: RequestInit = {}): Promise<T> => {
    const headers = new Headers(init.headers)
    const token = getToken()

    if (token) {
      headers.set('Authorization', `Bearer ${token}`)
    }
    if (init.body && !(init.body instanceof FormData) && !headers.has('Content-Type')) {
      headers.set('Content-Type', 'application/json')
    }
    if (!headers.has('Accept')) {
      headers.set('Accept', 'application/json')
    }

    const response = await fetch(`${API_BASE_URL}${path}`, {
      ...init,
      headers,
    })

    if (response.status === 204) {
      return undefined as T
    }

    const contentType = response.headers.get('Content-Type') ?? ''
    const payload = contentType.includes('application/json')
      ? await response.json() as unknown
      : await response.text()

    if (!response.ok) {
      throw toApiError(response.status, payload)
    }

    return payload as T
  }

  const upload = <T>(path: string, file: File, altText: string, caption: string, field: 'image' | 'images') => {
    const body = new FormData()
    body.append(field, file)
    body.append('alt_text', altText)
    body.append('caption', caption)
    return request<T>(path, {
      method: 'POST',
      body,
    })
  }

  return {
    login(username, password) {
      return request<LoginResponse>('/auth/login', {
        method: 'POST',
        body: JSON.stringify({ username, password }),
      })
    },
    async listProjects(filter) {
      const data = await request<{ projects: Project[] }>(`/projects${contentQueryString(filter)}`)
      return data.projects
    },
    createProject(payload) {
      return request<Project>('/projects', {
        method: 'POST',
        body: JSON.stringify(payload),
      })
    },
    updateProject(idOrSlug, payload) {
      return request<Project>(`/projects/${encodeURIComponent(idOrSlug)}`, {
        method: 'PATCH',
        body: JSON.stringify(payload),
      })
    },
    deleteProject(idOrSlug) {
      return request<void>(`/projects/${encodeURIComponent(idOrSlug)}`, { method: 'DELETE' })
    },
    uploadProjectMainImage(idOrSlug, file, altText, caption) {
      return upload<Project>(`/projects/${encodeURIComponent(idOrSlug)}/images/main`, file, altText, caption, 'image')
    },
    uploadProjectGalleryImage(idOrSlug, file, altText, caption) {
      return upload<Project>(`/projects/${encodeURIComponent(idOrSlug)}/images`, file, altText, caption, 'images')
    },
    deleteProjectImage(idOrSlug, imageID) {
      return request<Project>(`/projects/${encodeURIComponent(idOrSlug)}/images/${encodeURIComponent(imageID)}`, {
        method: 'DELETE',
      })
    },
    async listCertificates(filter) {
      const data = await request<{ certificates: Certificate[] }>(`/certificates${contentQueryString(filter)}`)
      return data.certificates
    },
    createCertificate(payload) {
      return request<Certificate>('/certificates', {
        method: 'POST',
        body: JSON.stringify(payload),
      })
    },
    updateCertificate(idOrSlug, payload) {
      return request<Certificate>(`/certificates/${encodeURIComponent(idOrSlug)}`, {
        method: 'PATCH',
        body: JSON.stringify(payload),
      })
    },
    deleteCertificate(idOrSlug) {
      return request<void>(`/certificates/${encodeURIComponent(idOrSlug)}`, { method: 'DELETE' })
    },
    uploadCertificateImage(idOrSlug, file, altText, caption) {
      return upload<Certificate>(`/certificates/${encodeURIComponent(idOrSlug)}/image`, file, altText, caption, 'image')
    },
    deleteCertificateImage(idOrSlug) {
      return request<Certificate>(`/certificates/${encodeURIComponent(idOrSlug)}/image`, { method: 'DELETE' })
    },
    async listArticles(filter) {
      const data = await request<{ articles: Article[] }>(`/articles${contentQueryString(filter)}`)
      return data.articles
    },
    createArticle(payload) {
      return request<Article>('/articles', {
        method: 'POST',
        body: JSON.stringify(payload),
      })
    },
    updateArticle(id, payload) {
      return request<Article>(`/articles/${encodeURIComponent(id)}`, {
        method: 'PATCH',
        body: JSON.stringify(payload),
      })
    },
    deleteArticle(id) {
      return request<void>(`/articles/${encodeURIComponent(id)}`, { method: 'DELETE' })
    },
    async listWorkExperiences(filter) {
      const data = await request<{ work_experiences: WorkExperience[] }>(`/work-experiences${workExperienceQueryString(filter)}`)
      return data.work_experiences
    },
    createWorkExperience(payload) {
      return request<WorkExperience>('/work-experiences', {
        method: 'POST',
        body: JSON.stringify(payload),
      })
    },
    updateWorkExperience(idOrSlug, payload) {
      return request<WorkExperience>(`/work-experiences/${encodeURIComponent(idOrSlug)}`, {
        method: 'PATCH',
        body: JSON.stringify(payload),
      })
    },
    deleteWorkExperience(idOrSlug) {
      return request<void>(`/work-experiences/${encodeURIComponent(idOrSlug)}`, { method: 'DELETE' })
    },
    async listLinks(filter) {
      if (filter.status === 'all') {
        const batches = await Promise.all(
          ALL_STATUSES.map(status => request<{ links: Link[] }>(`/links${linkQueryString({ ...filter, status })}`)),
        )
        return dedupeByID(batches.flatMap(batch => batch.links), item => item.label)
      }
      const data = await request<{ links: Link[] }>(`/links${linkQueryString(filter)}`)
      return data.links
    },
    createLink(payload) {
      return request<Link>('/links', {
        method: 'POST',
        body: JSON.stringify(payload),
      })
    },
    updateLink(id, payload) {
      return request<Link>(`/links/${encodeURIComponent(id)}`, {
        method: 'PATCH',
        body: JSON.stringify(payload),
      })
    },
    deleteLink(id) {
      return request<void>(`/links/${encodeURIComponent(id)}`, { method: 'DELETE' })
    },
    async listSkillCategories(filter) {
      if (filter.status === 'all') {
        const batches = await Promise.all(
          ALL_STATUSES.map(status => request<{ skill_categories: SkillCategory[] }>(`/skill-categories${statusQueryString({ ...filter, status })}`)),
        )
        return dedupeByID(batches.flatMap(batch => batch.skill_categories), item => item.name)
      }
      const data = await request<{ skill_categories: SkillCategory[] }>(`/skill-categories${statusQueryString(filter)}`)
      return data.skill_categories
    },
    createSkillCategory(payload) {
      return request<SkillCategory>('/skill-categories', {
        method: 'POST',
        body: JSON.stringify(payload),
      })
    },
    updateSkillCategory(idOrSlug, payload) {
      return request<SkillCategory>(`/skill-categories/${encodeURIComponent(idOrSlug)}`, {
        method: 'PATCH',
        body: JSON.stringify(payload),
      })
    },
    deleteSkillCategory(idOrSlug) {
      return request<void>(`/skill-categories/${encodeURIComponent(idOrSlug)}`, { method: 'DELETE' })
    },
    async listSkills(filter) {
      if (filter.status === 'all') {
        const batches = await Promise.all(
          ALL_STATUSES.map(status => request<{ skills: Skill[] }>(`/skills${skillQueryString({ ...filter, status })}`)),
        )
        return dedupeByID(batches.flatMap(batch => batch.skills), item => item.name)
      }
      const data = await request<{ skills: Skill[] }>(`/skills${skillQueryString(filter)}`)
      return data.skills
    },
    createSkill(payload) {
      return request<Skill>('/skills', {
        method: 'POST',
        body: JSON.stringify(payload),
      })
    },
    updateSkill(id, payload) {
      return request<Skill>(`/skills/${encodeURIComponent(id)}`, {
        method: 'PATCH',
        body: JSON.stringify(payload),
      })
    },
    deleteSkill(id) {
      return request<void>(`/skills/${encodeURIComponent(id)}`, { method: 'DELETE' })
    },
    async listTools(filter) {
      if (filter.status === 'all') {
        const batches = await Promise.all(
          ALL_STATUSES.map(status => request<{ tools: Tool[] }>(`/tools${toolQueryString({ ...filter, status })}`)),
        )
        return dedupeByID(batches.flatMap(batch => batch.tools), item => item.name)
      }
      const data = await request<{ tools: Tool[] }>(`/tools${toolQueryString(filter)}`)
      return data.tools
    },
    createTool(payload) {
      return request<Tool>('/tools', {
        method: 'POST',
        body: JSON.stringify(payload),
      })
    },
    updateTool(id, payload) {
      return request<Tool>(`/tools/${encodeURIComponent(id)}`, {
        method: 'PATCH',
        body: JSON.stringify(payload),
      })
    },
    deleteTool(id) {
      return request<void>(`/tools/${encodeURIComponent(id)}`, { method: 'DELETE' })
    },
    async listProducts(filter) {
      const data = await request<{ products: Product[] }>(`/products${productQueryString(filter)}`)
      return data.products
    },
    createProduct(payload) {
      return request<Product>('/products', {
        method: 'POST',
        body: JSON.stringify(payload),
      })
    },
    updateProduct(idOrSlug, payload) {
      return request<Product>(`/products/${encodeURIComponent(idOrSlug)}`, {
        method: 'PATCH',
        body: JSON.stringify(payload),
      })
    },
    deleteProduct(idOrSlug) {
      return request<void>(`/products/${encodeURIComponent(idOrSlug)}`, { method: 'DELETE' })
    },
    getProductSection() {
      return request<ProductSection>('/products/section')
    },
    updateProductSection(payload) {
      return request<ProductSection>('/products/section', {
        method: 'PATCH',
        body: JSON.stringify(payload),
      })
    },
    getIntro() {
      return request<Intro>('/intro')
    },
    updateIntro(payload) {
      return request<Intro>('/intro', {
        method: 'PATCH',
        body: JSON.stringify(payload),
      })
    },
    deleteIntro() {
      return request<void>('/intro', { method: 'DELETE' })
    },
    uploadIntroProfilePicture(file, altText, caption) {
      return upload<Intro>('/intro/profile-picture', file, altText, caption, 'image')
    },
    deleteIntroProfilePicture() {
      return request<Intro>('/intro/profile-picture', { method: 'DELETE' })
    },
  }
}

export function resolveAssetUrl(url?: string): string {
  if (!url) {
    return ''
  }
  if (/^https?:\/\//i.test(url)) {
    return url
  }
  return `${API_BASE_URL}${url.startsWith('/') ? '' : '/'}${url}`
}

function contentQueryString(filter: ListFilter): string {
  const params = new URLSearchParams()
  if (filter.q.trim() !== '') {
    params.set('q', filter.q.trim())
  }
  if (filter.status !== 'all') {
    params.set('status', filter.status)
  }
  if (filter.featured !== 'all') {
    params.set('featured', String(filter.featured === 'featured'))
  }
  return toQuery(params)
}

function linkQueryString(filter: LinkListFilter): string {
  const params = new URLSearchParams()
  if (filter.q.trim() !== '') {
    params.set('q', filter.q.trim())
  }
  if (filter.status !== 'all') {
    params.set('status', filter.status)
  }
  if (filter.star !== 'all') {
    params.set('star', String(filter.star === 'starred'))
  }
  return toQuery(params)
}

function workExperienceQueryString(filter: WorkExperienceListFilter): string {
  const params = new URLSearchParams()
  if (filter.q.trim() !== '') {
    params.set('q', filter.q.trim())
  }
  if (filter.status !== 'all') {
    params.set('status', filter.status)
  }
  if (filter.featured !== 'all') {
    params.set('featured', String(filter.featured === 'featured'))
  }
  if (filter.current !== 'all') {
    params.set('current', String(filter.current === 'current'))
  }
  return toQuery(params)
}

function statusQueryString(filter: StatusListFilter): string {
  const params = new URLSearchParams()
  if (filter.q.trim() !== '') {
    params.set('q', filter.q.trim())
  }
  if (filter.status !== 'all') {
    params.set('status', filter.status)
  }
  return toQuery(params)
}

function skillQueryString(filter: SkillListFilter): string {
  const params = new URLSearchParams()
  if (filter.q.trim() !== '') {
    params.set('q', filter.q.trim())
  }
  if (filter.status !== 'all') {
    params.set('status', filter.status)
  }
  if (filter.featured !== 'all') {
    params.set('featured', String(filter.featured === 'featured'))
  }
  if (filter.category_id.trim() !== '') {
    params.set('category_id', filter.category_id.trim())
  }
  if (filter.category.trim() !== '') {
    params.set('category', filter.category.trim())
  }
  return toQuery(params)
}

function toolQueryString(filter: ToolListFilter): string {
  const params = new URLSearchParams()
  if (filter.q.trim() !== '') {
    params.set('q', filter.q.trim())
  }
  if (filter.status !== 'all') {
    params.set('status', filter.status)
  }
  if (filter.featured !== 'all') {
    params.set('featured', String(filter.featured === 'featured'))
  }
  if (filter.category.trim() !== '') {
    params.set('category', filter.category.trim())
  }
  if (filter.tag.trim() !== '') {
    params.set('tag', filter.tag.trim())
  }
  return toQuery(params)
}

function productQueryString(filter: ProductListFilter): string {
  const params = new URLSearchParams()
  if (filter.q.trim() !== '') {
    params.set('q', filter.q.trim())
  }
  if (filter.status !== 'all') {
    params.set('status', filter.status)
  } else {
    params.set('status', 'all')
  }
  if (filter.featured !== 'all') {
    params.set('featured', String(filter.featured === 'featured'))
  }
  if (filter.category.trim() !== '') {
    params.set('category', filter.category.trim())
  }
  if (filter.tag.trim() !== '') {
    params.set('tag', filter.tag.trim())
  }
  return toQuery(params)
}

function toQuery(params: URLSearchParams): string {
  const query = params.toString()
  return query ? `?${query}` : ''
}

function dedupeByID<T extends { id: string, sort_order: number }>(items: T[], label: (item: T) => string): T[] {
  const seen = new Set<string>()
  return items
    .filter(item => {
      if (seen.has(item.id)) {
        return false
      }
      seen.add(item.id)
      return true
    })
    .sort((a, b) => a.sort_order - b.sort_order || label(a).localeCompare(label(b)))
}

function toApiError(status: number, payload: unknown): ApiError {
  if (isApiErrorPayload(payload)) {
    return new ApiError(status, payload.error || `Request failed with status ${status}`, payload.fields)
  }
  if (typeof payload === 'string' && payload.trim() !== '') {
    return new ApiError(status, payload)
  }
  return new ApiError(status, `Request failed with status ${status}`)
}

function isApiErrorPayload(payload: unknown): payload is ApiErrorPayload {
  return typeof payload === 'object' && payload !== null && ('error' in payload || 'fields' in payload)
}
