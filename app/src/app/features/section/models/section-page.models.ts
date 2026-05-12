import type { IconSvgObject } from '@hugeicons/angular';

export interface SectionPageLink {
  label: string;
  url: string;
}

export interface SectionPageData {
  title: string;
  summary: string;
  icon: IconSvgObject;
  links?: SectionPageLink[];
}
