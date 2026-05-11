import { Routes } from '@angular/router';

import { authGuard } from './core/guards/auth.guard';
import {
  animesPageData,
  certificatesPageData,
  etcPageData,
  gamesPageData,
  musicPageData,
  projectsPageData,
  skillsPageData,
  socialLinksPageData,
  workHistoryPageData,
} from './features/section/data/section-page.data';

export const routes: Routes = [
  {
    path: '',
    loadComponent: () => import('./features/home/pages/home-page/home-page.component')
      .then((m) => m.HomePageComponent),
  },
  {
    path: 'login',
    loadComponent: () => import('./features/auth/pages/login-page/login-page.component')
      .then((m) => m.LoginPageComponent),
  },
  {
    path: 'admin',
    canActivate: [authGuard],
    loadComponent: () => import('./features/admin/pages/admin-page/admin-page.component')
      .then((m) => m.AdminPageComponent),
  },
  {
    path: 'projects',
    loadComponent: () => import('./features/section/pages/section-page/section-page.component')
      .then((m) => m.SectionPageComponent),
    data: projectsPageData,
  },
  {
    path: 'work-history',
    loadComponent: () => import('./features/section/pages/section-page/section-page.component')
      .then((m) => m.SectionPageComponent),
    data: workHistoryPageData,
  },
  {
    path: 'certificates',
    loadComponent: () => import('./features/section/pages/section-page/section-page.component')
      .then((m) => m.SectionPageComponent),
    data: certificatesPageData,
  },
  {
    path: 'skills',
    loadComponent: () => import('./features/section/pages/section-page/section-page.component')
      .then((m) => m.SectionPageComponent),
    data: skillsPageData,
  },
  {
    path: 'music',
    loadComponent: () => import('./features/section/pages/section-page/section-page.component')
      .then((m) => m.SectionPageComponent),
    data: musicPageData,
  },
  {
    path: 'games',
    loadComponent: () => import('./features/section/pages/section-page/section-page.component')
      .then((m) => m.SectionPageComponent),
    data: gamesPageData,
  },
  {
    path: 'animes',
    loadComponent: () => import('./features/section/pages/section-page/section-page.component')
      .then((m) => m.SectionPageComponent),
    data: animesPageData,
  },
  {
    path: 'etc',
    loadComponent: () => import('./features/section/pages/section-page/section-page.component')
      .then((m) => m.SectionPageComponent),
    data: etcPageData,
  },
  {
    path: 'social-links',
    loadComponent: () => import('./features/section/pages/section-page/section-page.component')
      .then((m) => m.SectionPageComponent),
    data: socialLinksPageData,
  },
  {
    path: '**',
    redirectTo: '',
  },
];
