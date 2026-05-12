import {
  ArchiveIcon,
  BrainCogIcon,
  Certificate02Icon,
  CodeFolderIcon,
  Film01Icon,
  GameController01Icon,
  LinkSquare02Icon,
  MusicNote01Icon,
  WorkHistoryIcon,
} from '@hugeicons-pro/core-stroke-rounded';
import type { IconSvgObject } from '@hugeicons/angular';

import type { SectionPageData, SectionPageLink } from '../models/section-page.models';

export const projectsPageData: SectionPageData = {
  title: 'Projects',
  summary: 'Selected builds that show practical engineering work.',
  icon: CodeFolderIcon,
};

export const archivedProjectsPageData: SectionPageData = {
  title: 'Archived Projects',
  summary: 'Older builds and experiments kept for reference.',
  icon: ArchiveIcon,
};

export const workHistoryPageData: SectionPageData = {
  title: 'Work History',
  summary: 'Professional experience across teams, products, and systems.',
  icon: WorkHistoryIcon,
};

export const certificatesPageData: SectionPageData = {
  title: 'Certificates',
  summary: 'Credentials that support practical engineering work.',
  icon: Certificate02Icon,
};

export const skillsPageData: SectionPageData = {
  title: 'Skills and Tools',
  summary: 'Technologies and tools used to build reliable products.',
  icon: BrainCogIcon,
};

export const musicPageData: SectionPageData = {
  title: 'Music',
  summary: 'Tracks and listening notes.',
  icon: MusicNote01Icon,
};

export const gamesPageData: SectionPageData = {
  title: 'Games',
  summary: 'Games and play notes.',
  icon: GameController01Icon,
};

export const animesPageData: SectionPageData = {
  title: 'Animes',
  summary: 'Anime watchlist and notes.',
  icon: Film01Icon,
};

export const etcPageData: SectionPageData = {
  title: 'Entertainment',
  summary: 'Music, games, and anime links grouped under the short /etc path.',
  icon: LinkSquare02Icon,
};

export const socialLinksPageData: SectionPageData = {
  title: 'Social Links',
  summary: 'Profiles and contact links.',
  icon: LinkSquare02Icon,
  links: [
    {
      label: 'LinkedIn',
      url: 'https://www.linkedin.com/in/eralmendral',
    },
    {
      label: 'GitHub',
      url: 'https://github.com/eralmendral',
    },
  ],
};

export function readPageData(data: Record<string, unknown>): SectionPageData {
  const title = data['title'];
  const summary = data['summary'];
  const icon = data['icon'];

  if (typeof title === 'string' && typeof summary === 'string' && Array.isArray(icon)) {
    return {
      title,
      summary,
      icon: icon as IconSvgObject,
      links: readLinks(data['links']),
    };
  }

  return projectsPageData;
}

function readLinks(value: unknown): SectionPageLink[] | undefined {
  if (!Array.isArray(value)) {
    return undefined;
  }

  const links = value.filter((item): item is SectionPageLink => (
    typeof item === 'object'
    && item !== null
    && typeof (item as Record<string, unknown>)['label'] === 'string'
    && typeof (item as Record<string, unknown>)['url'] === 'string'
  ));

  return links.length ? links : undefined;
}
