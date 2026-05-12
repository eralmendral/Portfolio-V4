import { ChangeDetectionStrategy, Component } from '@angular/core';
import { RouterLink } from '@angular/router';
import { HugeiconsIconComponent } from '@hugeicons/angular';
import { Home03Icon } from '@hugeicons-pro/core-stroke-rounded';

@Component({
  selector: 'app-home-shortcut',
  imports: [HugeiconsIconComponent, RouterLink],
  templateUrl: './home-shortcut.component.html',
  styleUrl: './home-shortcut.component.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class HomeShortcutComponent {
  protected readonly homeIcon = Home03Icon;
}
