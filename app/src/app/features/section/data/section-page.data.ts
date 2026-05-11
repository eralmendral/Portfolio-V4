import {
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
  summary: 'A deeper archive for selected builds, production systems, experiments, and implementation notes.',
  icon: CodeFolderIcon,
};

export const workHistoryPageData: SectionPageData = {
  title: 'Work XP',
  summary: 'A focused timeline of roles, responsibilities, outcomes, and the operating contexts behind the work.',
  icon: WorkHistoryIcon,
};

export const certificatesPageData: SectionPageData = {
  title: 'Certificates',
  summary: 'Credential details, issuing organizations, dates, and related proof links.',
  icon: Certificate02Icon,
};

export const skillsPageData: SectionPageData = {
  title: 'Skills',
  summary: 'A structured view of engineering strengths, practical tools, delivery, quality, and AI workflows.',
  icon: BrainCogIcon,
};

export const musicPageData: SectionPageData = {
  title: 'Music',
  summary: 'Recently played tracks, albums, and listening notes from the entertainment side of the portfolio.',
  icon: MusicNote01Icon,
};

export const gamesPageData: SectionPageData = {
  title: 'Games',
  summary: 'Games in rotation, platforms, and play notes from the entertainment side of the portfolio.',
  icon: GameController01Icon,
};

export const animesPageData: SectionPageData = {
  title: 'Animes',
  summary: 'Anime watchlist, favorites, and viewing notes from the entertainment side of the portfolio.',
  icon: Film01Icon,
};

export const etcPageData: SectionPageData = {
  title: 'Entertainment',
  summary: 'Music, games, and anime links grouped under the short /etc path.',
  icon: LinkSquare02Icon,
};

export const socialLinksPageData: SectionPageData = {
  title: 'Social Links',
  summary: 'LinkedIn and other professional profiles collected in one place.',
  icon: LinkSquare02Icon,
  links: [
    {
      label: 'LinkedIn',
      url: 'https://www.linkedin.com/in/eralmendral',
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
