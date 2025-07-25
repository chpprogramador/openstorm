import {
  ChangeDetectorRef,
  Component,
  ElementRef,
  OnDestroy,
  OnInit,
  ViewChild
} from '@angular/core';
import { Subscription } from 'rxjs';
import { LogEntry, LogStatusService } from '../services/log-status.service';

@Component({
  selector: 'app-log-viewer',
  styleUrls: ['./app-log-viewer.component.scss'],
  template: `
    <pre #logBox class="log-container">
{{ logText }}
    </pre>
  `
})
export class LogViewerComponent implements OnInit, OnDestroy {
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
}
