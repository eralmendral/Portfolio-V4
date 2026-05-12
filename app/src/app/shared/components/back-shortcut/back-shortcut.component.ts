import { Location } from '@angular/common';
import { ChangeDetectionStrategy, Component, inject } from '@angular/core';
import { HugeiconsIconComponent } from '@hugeicons/angular';
import { ArrowLeft02Icon } from '@hugeicons-pro/core-stroke-rounded';

@Component({
  selector: 'app-back-shortcut',
  imports: [HugeiconsIconComponent],
  templateUrl: './back-shortcut.component.html',
  styleUrl: './back-shortcut.component.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class BackShortcutComponent {
  private readonly location = inject(Location);
  protected readonly backIcon = ArrowLeft02Icon;

  protected goBack(): void {
    this.location.back();
  }
}
