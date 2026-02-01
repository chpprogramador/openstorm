import {
    ChangeDetectorRef,
    Component,
    ElementRef,
    Input,
    OnDestroy,
    OnInit,
    ViewChild
} from '@angular/core';
import { CommonModule } from '@angular/common';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatTooltipModule } from '@angular/material/tooltip';
import { Subscription } from 'rxjs';
import { LogEntry, LogStatusService } from '../services/log-status.service';

@Component({
  selector: 'app-log-viewer',
  standalone: true,
  imports: [CommonModule, MatIconModule, MatButtonModule, MatTooltipModule],
  styleUrls: ['./app-log-viewer.component.scss'],
  template: `
    <div class="log-viewer-container" [class.embedded]="embedded">
      <div class="log-header" *ngIf="showHeader">
        <mat-icon>terminal</mat-icon>
        <span>Logs em Tempo Real</span>
        <button mat-icon-button (click)="clearLogs()" matTooltip="Limpar logs">
          <mat-icon>clear</mat-icon>
        </button>
      </div>
      <pre #logBox class="log-container">
{{ logText }}
      </pre>
    </div>
  `
})
export class LogViewerComponent implements OnInit, OnDestroy {
  @Input() showHeader = true;
  @Input() embedded = false;
  logText = '';
  private sub!: Subscription;

  @ViewChild('logBox') logBoxRef!: ElementRef<HTMLPreElement>;

  constructor(
    private logService: LogStatusService,
    private cdRef: ChangeDetectorRef
  ) {}

  ngOnInit() {
    this.sub = this.logService.listen().subscribe((logs: LogEntry[]) => {
      console.log('Logs recebidos:', logs);
      this.logText = logs
        .map(log => `[${new Date(log.timestamp).toLocaleTimeString()}] ${log.message}`)
        .join('\n');

      // Espera o Angular renderizar antes de rolar
      this.cdRef.detectChanges();

      setTimeout(() => this.scrollToBottom(), 0);
    });
  }

  ngOnDestroy() {
    this.sub.unsubscribe();
  }

  private scrollToBottom() {
    const el = this.logBoxRef?.nativeElement;
    if (el) {
      el.scrollTop = el.scrollHeight;
    }
  }

  clearLogs() {
    this.logText = '';
    this.logService.clearLogs();
  }
}
