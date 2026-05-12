import { ChangeDetectionStrategy, Component, computed, inject, resource, signal } from '@angular/core';
import { NonNullableFormBuilder, ReactiveFormsModule, Validators } from '@angular/forms';
import { HugeiconsIconComponent } from '@hugeicons/angular';
import {
  Cancel01Icon,
  Chat01Icon,
  MailSend01Icon,
} from '@hugeicons-pro/core-stroke-rounded';

import type { ContactProfile } from '../../../core/models/contact.models';
import { PortfolioApi } from '../../../core/services/portfolio-api.service';

@Component({
  selector: 'app-contact-widget',
  imports: [HugeiconsIconComponent, ReactiveFormsModule],
  templateUrl: './contact-widget.component.html',
  styleUrl: './contact-widget.component.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class ContactWidgetComponent {
  private readonly api = inject(PortfolioApi);
  private readonly formBuilder = inject(NonNullableFormBuilder);

  protected readonly contactIcon = Chat01Icon;
  protected readonly closeIcon = Cancel01Icon;
  protected readonly sendIcon = MailSend01Icon;
  protected readonly isOpen = signal(false);
  protected readonly isSubmitting = signal(false);
  protected readonly submitError = signal<string | null>(null);
  protected readonly submitSuccess = signal(false);
  protected readonly profileResource = resource({
    loader: ({ abortSignal }) => this.api.getContactProfile(abortSignal),
  });
  protected readonly contactProfile = computed<ContactProfile | null>(() => {
    try {
      return this.profileResource.value() ?? null;
    } catch {
      return null;
    }
  });
  protected readonly form = this.formBuilder.group({
    name: ['', [Validators.required, Validators.maxLength(120)]],
    email: ['', [Validators.required, Validators.email, Validators.maxLength(254)]],
    subject: ['', [Validators.maxLength(160)]],
    message: ['', [Validators.required, Validators.minLength(10), Validators.maxLength(4000)]],
  });

  protected togglePanel(): void {
    this.isOpen.update((open) => !open);
    this.submitError.set(null);
  }

  protected closePanel(): void {
    this.isOpen.set(false);
    this.submitError.set(null);
  }

  protected async submit(): Promise<void> {
    if (this.form.invalid) {
      this.form.markAllAsTouched();
      return;
    }

    this.isSubmitting.set(true);
    this.submitError.set(null);
    this.submitSuccess.set(false);

    try {
      const value = this.form.getRawValue();
      await this.api.submitContact({
        name: value.name.trim(),
        email: value.email.trim(),
        subject: value.subject.trim(),
        message: value.message.trim(),
      });
      this.form.reset();
      this.submitSuccess.set(true);
    } catch {
      this.submitError.set('Message could not be sent right now.');
    } finally {
      this.isSubmitting.set(false);
    }
  }

  protected phoneHref(phoneNumber: string): string {
    return `tel:${phoneNumber.replace(/[^\d+]/g, '')}`;
  }
}
