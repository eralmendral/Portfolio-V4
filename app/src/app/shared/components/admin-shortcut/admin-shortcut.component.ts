import { ChangeDetectionStrategy, Component } from '@angular/core';
import { RouterLink } from '@angular/router';
import { HugeiconsIconComponent } from '@hugeicons/angular';
import { UserIcon } from '@hugeicons-pro/core-stroke-rounded';

@Component({
  selector: 'app-admin-shortcut',
  imports: [HugeiconsIconComponent, RouterLink],
  templateUrl: './admin-shortcut.component.html',
  styleUrl: './admin-shortcut.component.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class AdminShortcutComponent {
  protected readonly adminIcon = UserIcon;
}
