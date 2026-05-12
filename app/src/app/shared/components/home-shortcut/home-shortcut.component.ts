import { ChangeDetectionStrategy, Component } from '@angular/core';
import { RouterLink } from '@angular/router';

@Component({
  selector: 'app-home-shortcut',
  imports: [RouterLink],
  templateUrl: './home-shortcut.component.html',
  styleUrl: './home-shortcut.component.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class HomeShortcutComponent {}
