import { bootstrapApplication } from '@angular/platform-browser';
import { initMonitoring } from './monitoring';
import { appConfig } from './app/app.config';
import { App } from './app/app';

initMonitoring();

bootstrapApplication(App, appConfig)
  .catch((err) => console.error(err));
