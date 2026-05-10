export type Status = 'draft' | 'published' | 'archived'

export type FeaturedFilter = 'all' | 'featured' | 'standard'

export type StarFilter = 'all' | 'starred' | 'standard'

export type CurrentFilter = 'all' | 'current' | 'past'

export type ResourceTab =
  | 'projects'
  | 'certificates'
  | 'articles'
  | 'work'
  | 'links'
  | 'skill-categories'
  | 'skills'
  | 'tools'
  | 'products'
  | 'intro'

export interface ListFilter {
  q: string
  status: 'all' | Status
  featured: FeaturedFilter
}

export interface LinkListFilter {
  q: string
  status: 'all' | Status
  star: StarFilter
}

export interface WorkExperienceListFilter {
  q: string
  status: 'all' | Status
  featured: FeaturedFilter
  current: CurrentFilter
}

export interface StatusListFilter {
  q: string
  status: 'all' | Status
}

export interface SkillListFilter {
  q: string
  status: 'all' | Status
  featured: FeaturedFilter
  category_id: string
  category: string
}

export interface ToolListFilter {
  q: string
  status: 'all' | Status
  featured: FeaturedFilter
  category: string
  tag: string
}

export interface ProductListFilter {
  q: string
  status: 'all' | Status
  featured: FeaturedFilter
  category: string
  tag: string
}

export interface ApiFieldErrors {
  [field: string]: string
}

export interface CommonImage {
  id: string
  url: string
  path?: string
  alt_text?: string
  caption?: string
  content_type?: string
  size_bytes?: number
  width?: number
  height?: number
  sort_order?: number
  uploaded_at: string
}

export type ProjectImage = CommonImage
export type CertificateImage = CommonImage
export type IntroImage = CommonImage

export interface Project {
  id: string
  slug: string
  title: string
  summary?: string
  description?: string
  body?: string
  tech_stack?: string[]
  tags?: string[]
  main_image?: ProjectImage
  images?: ProjectImage[]
  github_url?: string
  demo_url?: string
  featured: boolean
  sort_order: number
  status: Status
  created_at: string
  updated_at: string
  published_at?: string
}

export interface ProjectPayload {
  slug: string
  title: string
  summary: string
  description: string
  body: string
  tech_stack: string[]
  tags: string[]
  github_url: string
  demo_url: string
  featured: boolean
  sort_order: number
  status: Status
}

export interface Certificate {
  id: string
  slug: string
  title: string
  issuer: string
  summary?: string
  description?: string
  credential_url?: string
  image?: CertificateImage
  featured: boolean
  sort_order: number
  status: Status
  issued_at?: string
  expires_at?: string
  created_at: string
  updated_at: string
}

export interface CertificatePayload {
  slug: string
  title: string
  issuer: string
  summary: string
  description: string
  credential_url: string
  featured: boolean
  sort_order: number
  status: Status
  issued_at: string | null
  expires_at: string | null
}

export interface Article {
  id: string
  title: string
  url: string
  source: string
  summary?: string
  cover_image_url?: string
  featured: boolean
  sort_order: number
  status: Status
  published_at?: string
  created_at: string
  updated_at: string
}

export interface ArticlePayload {
  title: string
  url: string
  source: string
  summary: string
  cover_image_url: string
  featured: boolean
  sort_order: number
  status: Status
  published_at: string | null
}

export interface Link {
  id: string
  label: string
  url: string
  icon_class?: string
  sort_order: number
  star: boolean
  status: Status
  created_at: string
  updated_at: string
}

export interface LinkPayload {
  label: string
  url: string
  icon_class: string
  sort_order: number
  star: boolean
  status: Status
}

export interface WorkExperience {
  id: string
  slug: string
  title: string
  company: string
  company_url?: string
  company_logo_url?: string
  employment_type?: string
  location?: string
  location_type?: string
  summary?: string
  description?: string
  highlights?: string[]
  responsibilities?: string[]
  tech_stack?: string[]
  skills?: string[]
  started_at: string
  ended_at?: string
  current: boolean
  featured: boolean
  sort_order: number
  status: Status
  published_at?: string
  created_at: string
  updated_at: string
}

export interface WorkExperiencePayload {
  slug: string
  title: string
  company: string
  company_url: string
  company_logo_url: string
  employment_type: string
  location: string
  location_type: string
  summary: string
  description: string
  highlights: string[]
  responsibilities: string[]
  tech_stack: string[]
  skills: string[]
  started_at: string | null
  ended_at: string | null
  current: boolean
  featured: boolean
  sort_order: number
  status: Status
  published_at: string | null
}

export interface SkillCategory {
  id: string
  slug: string
  name: string
  description?: string
  icon_class?: string
  sort_order: number
  status: Status
  created_at: string
  updated_at: string
}

export interface SkillCategoryPayload {
  slug: string
  name: string
  description: string
  icon_class: string
  sort_order: number
  status: Status
}

export interface Skill {
  id: string
  category_id: string
  name: string
  summary?: string
  icon_class?: string
  sort_order: number
  featured: boolean
  status: Status
  created_at: string
  updated_at: string
}

export interface SkillPayload {
  category_id: string
  name: string
  summary: string
  icon_class: string
  sort_order: number
  featured: boolean
  status: Status
}

export interface Tool {
  id: string
  name: string
  category: string
  summary?: string
  icon_class?: string
  tags?: string[]
  sort_order: number
  featured: boolean
  status: Status
  created_at: string
  updated_at: string
}

export interface ToolPayload {
  name: string
  category: string
  summary: string
  icon_class: string
  tags: string[]
  sort_order: number
  featured: boolean
  status: Status
}

export interface Product {
  id: string
  slug: string
  title: string
  summary?: string
  description?: string
  cover_image_url?: string
  price_label?: string
  cta_label?: string
  cta_url: string
  category?: string
  tags?: string[]
  featured: boolean
  sort_order: number
  status: Status
  published_at?: string
  created_at: string
  updated_at: string
}

export interface ProductPayload {
  slug: string
  title: string
  summary: string
  description: string
  cover_image_url: string
  price_label: string
  cta_label: string
  cta_url: string
  category: string
  tags: string[]
  featured: boolean
  sort_order: number
  status: Status
  published_at: string | null
}

export interface ProductSection {
  enabled: boolean
  title: string
  description: string
  updated_at: string
}

export interface ProductSectionPayload {
  enabled: boolean
  title: string
  description: string
}

export interface Intro {
  id: string
  title: string
  description: string
  profile_picture?: IntroImage
  created_at: string
  updated_at: string
}

export interface IntroPayload {
  title: string
  description: string
}
