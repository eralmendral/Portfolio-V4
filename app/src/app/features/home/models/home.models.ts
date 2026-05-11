import type { IconSvgObject } from '@hugeicons/angular';

export interface HeroLink {
  id: string;
  label: string;
  url: string;
  icon: IconSvgObject;
}

export interface OverviewCard {
  title: string;
  summary: string;
  route?: string;
  url?: string;
  icon: IconSvgObject;
}
