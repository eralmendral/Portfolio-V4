import { DOCUMENT } from '@angular/common';
import {
  ChangeDetectionStrategy,
  Component,
  HostListener,
  computed,
  inject,
  signal,
} from '@angular/core';
import { Router } from '@angular/router';

import { AuthService } from '../../../../core/services/auth.service';
import {
  LABORATORY_CONNECTIONS,
  PERSONAL_LABORATORIES,
} from '../../data/personal-laboratory.data';
import type { Laboratory } from '../../models/personal-laboratory.models';

@Component({
  selector: 'app-admin-page',
  templateUrl: './admin-page.component.html',
  styleUrl: './admin-page.component.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class AdminPageComponent {
  private readonly auth = inject(AuthService);
  private readonly router = inject(Router);
  private readonly document = inject(DOCUMENT);
  private bootTimer = this.document.defaultView?.setTimeout(() => this.booting.set(false), 1900);
  private returnFocus: HTMLElement | null = null;

  protected readonly labs = PERSONAL_LABORATORIES;
  protected readonly connections = LABORATORY_CONNECTIONS.map(([sourceId, targetId], index) => ({
    id: `${sourceId}-${targetId}`,
    index,
    sourceId,
    targetId,
    source: PERSONAL_LABORATORIES.find((lab) => lab.id === sourceId)!,
    target: PERSONAL_LABORATORIES.find((lab) => lab.id === targetId)!,
  }));
  protected readonly booting = signal(true);
  protected readonly motionPaused = signal(false);
  protected readonly hoveredLabId = signal<string | null>(null);
  protected readonly selectedLab = signal<Laboratory | null>(null);
  protected readonly hoveredLab = computed(() =>
    this.labs.find((lab) => lab.id === this.hoveredLabId()) ?? null,
  );
  protected readonly parallax = signal({ x: '0px', y: '0px' });

  protected openLab(lab: Laboratory, trigger?: EventTarget | null): void {
    this.returnFocus = trigger instanceof HTMLElement ? trigger : null;
    this.selectedLab.set(lab);
    this.document.defaultView?.setTimeout(() => {
      this.document.querySelector<HTMLElement>('[data-panel-close]')?.focus();
    });
  }

  protected closeLab(): void {
    this.selectedLab.set(null);
    this.document.defaultView?.setTimeout(() => this.returnFocus?.focus());
  }

  protected setHoveredLab(lab: Laboratory | null): void {
    this.hoveredLabId.set(lab?.id ?? null);
  }

  protected isConnectionActive(sourceId: string, targetId: string): boolean {
    const activeId = this.hoveredLabId();
    return activeId === null || sourceId === activeId || targetId === activeId;
  }

  protected updateParallax(event: PointerEvent): void {
    if (this.motionPaused() || event.pointerType === 'touch') return;

    const target = event.currentTarget as HTMLElement;
    const bounds = target.getBoundingClientRect();
    const x = ((event.clientX - bounds.left) / bounds.width - 0.5) * 12;
    const y = ((event.clientY - bounds.top) / bounds.height - 0.5) * 12;
    this.parallax.set({ x: `${x.toFixed(2)}px`, y: `${y.toFixed(2)}px` });
  }

  protected resetParallax(): void {
    this.parallax.set({ x: '0px', y: '0px' });
  }

  protected toggleMotion(): void {
    this.motionPaused.update((paused) => !paused);
  }

  @HostListener('document:keydown', ['$event'])
  protected handleKeydown(event: KeyboardEvent): void {
    if (!this.selectedLab()) return;

    if (event.key === 'Escape') {
      event.preventDefault();
      this.closeLab();
      return;
    }

    if (event.key !== 'Tab') return;

    const panel = this.document.querySelector<HTMLElement>('.LabPanel');
    const focusable = panel?.querySelectorAll<HTMLElement>(
      'button:not([disabled]), a[href], [tabindex]:not([tabindex="-1"])',
    );
    if (!focusable?.length) return;

    const first = focusable[0];
    const last = focusable[focusable.length - 1];
    if (event.shiftKey && this.document.activeElement === first) {
      event.preventDefault();
      last.focus();
    } else if (!event.shiftKey && this.document.activeElement === last) {
      event.preventDefault();
      first.focus();
    }
  }

  ngOnDestroy(): void {
    if (this.bootTimer) this.document.defaultView?.clearTimeout(this.bootTimer);
  }

  protected async signOut(): Promise<void> {
    this.auth.clearToken();
    await this.router.navigateByUrl('/');
  }
}
