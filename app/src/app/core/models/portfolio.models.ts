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

export interface IntroSummary {
  title: string;
  description: string;
}
