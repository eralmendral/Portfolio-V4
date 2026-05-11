import { Routes } from '@angular/router';

import { HomeComponent } from './home.component';
import {
  certificatesPageData,
  projectsPageData,
  SectionPageComponent,
  skillsPageData,
  toolsPageData,
  workHistoryPageData,
} from './section-page.component';

export const routes: Routes = [
  {
    path: '',
    component: HomeComponent,
  },
  {
    path: 'projects',
    component: SectionPageComponent,
    data: projectsPageData,
  },
  {
    path: 'work-history',
    component: SectionPageComponent,
    data: workHistoryPageData,
  },
  {
    path: 'certificates',
    component: SectionPageComponent,
    data: certificatesPageData,
  },
  {
    path: 'skills',
    component: SectionPageComponent,
    data: skillsPageData,
  },
  {
    path: 'tools',
    component: SectionPageComponent,
    data: toolsPageData,
  },
  {
    path: '**',
    redirectTo: '',
  },
];
