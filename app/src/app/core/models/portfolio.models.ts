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
  description?: string;
  tech_stack?: string[];
  tags?: string[];
  main_image?: ProjectImage;
  images?: ProjectImage[];
  github_url?: string;
  demo_url?: string;
  featured: boolean;
  archived: boolean;
  sort_order: number;
  status: 'draft' | 'published' | 'archived';
}

export interface ProjectImage {
  id: string;
  url: string;
  alt_text?: string;
  caption?: string;
  sort_order: number;
}

export interface IntroSummary {
  title: string;
  description: string;
}

export interface SkillCategory {
  id: string;
  slug: string;
  name: string;
  description?: string;
  icon_class?: string;
  sort_order: number;
  status: 'draft' | 'published' | 'archived';
}

export interface Skill {
  id: string;
  category_id: string;
  name: string;
  summary?: string;
  icon_class?: string;
  sort_order: number;
  featured: boolean;
  status: 'draft' | 'published' | 'archived';
}

export interface Tool {
  id: string;
  name: string;
  category: string;
  summary?: string;
  icon_class?: string;
  tags?: string[];
  sort_order: number;
  featured: boolean;
  status: 'draft' | 'published' | 'archived';
}

export interface CertificateImage {
  id: string;
  url: string;
  alt_text?: string;
  caption?: string;
}

export interface Certificate {
  id: string;
  slug: string;
  title: string;
  issuer: string;
  summary?: string;
  description?: string;
  credential_url?: string;
  image?: CertificateImage;
  featured: boolean;
  sort_order: number;
  status: 'draft' | 'published' | 'archived';
  issued_at?: string;
}

export interface WorkExperience {
  id: string;
  slug: string;
  title: string;
  company: string;
  company_url?: string;
  employment_type?: string;
  summary?: string;
  description?: string;
  responsibilities?: string[];
  started_at: string;
  ended_at?: string;
  current: boolean;
  sort_order: number;
  status: 'draft' | 'published' | 'archived';
}
