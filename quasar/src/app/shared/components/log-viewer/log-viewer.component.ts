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
import { LogEntry, LogStatusService } from '../../../core/services/log-status.service';

@Component({
  selector: 'app-log-viewer',
  standalone: true,
  imports: [CommonModule, MatIconModule, MatButtonModule, MatTooltipModule],
  styleUrls: ['./log-viewer.component.scss'],
  templateUrl: './log-viewer.component.html'
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
      this.logText = logs
        .map(log => `[${new Date(log.timestamp).toLocaleTimeString()}] ${log.message}`)
        .join('\n');

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
