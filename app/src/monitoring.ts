import * as Sentry from '@sentry/angular';

interface MonitoringWindow extends Window {
  __SENTRY_DSN__?: string;
  __SENTRY_ENVIRONMENT__?: string;
  __SENTRY_RELEASE__?: string;
}

export function initMonitoring(): void {
  const dsn = readMonitoringValue('sentry-dsn', '__SENTRY_DSN__');
  if (!dsn) {
    return;
  }

  Sentry.init({
    dsn,
    environment: readMonitoringValue('sentry-environment', '__SENTRY_ENVIRONMENT__') || 'production',
    release: readMonitoringValue('sentry-release', '__SENTRY_RELEASE__') || undefined,
  });
}

function readMonitoringValue(metaName: string, windowKey: keyof MonitoringWindow): string {
  const metaValue = document
    .querySelector<HTMLMetaElement>(`meta[name="${metaName}"]`)
    ?.content
    .trim();
  const windowValue = (window as MonitoringWindow)[windowKey]?.trim();

  return metaValue || windowValue || '';
}
