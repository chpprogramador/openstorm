import { isPlatformBrowser } from '@angular/common';
import {
  AfterViewInit, Component, ElementRef,
  EventEmitter,
  Inject,
  Input,
  OnChanges,
  Output,
  PLATFORM_ID,
  SimpleChanges, ViewChild
} from '@angular/core';
import { ThemeService } from '../../../../services/theme.service';

@Component({
  selector: 'app-sql-editor',
  standalone: true,
  templateUrl: './sql-editor.html',
  styleUrls: ['./sql-editor.scss']
})
export class SqlEditor implements AfterViewInit, OnChanges {
  @ViewChild('editorContainer', { static: true }) editorContainer!: ElementRef<HTMLDivElement>;
  @Input() initialSql = '';
  @Output() sqlChanged = new EventEmitter<string>();

  

  isBrowser: boolean;
  private editorInstance?: import('monaco-editor').editor.IStandaloneCodeEditor;

  private monaco: typeof import('monaco-editor') | null = null;

  constructor(@
    Inject(PLATFORM_ID) private platformId: Object,
    private themeService: ThemeService
  ) {
    this.isBrowser = isPlatformBrowser(this.platformId);
  }

  async ngAfterViewInit() {
    if (this.isBrowser) {
      this.monaco = await import(/* @vite-ignore */ 'monaco-editor');
      this.createEditor();
    }
  }

  ngOnChanges(changes: SimpleChanges): void {
    if (changes['initialSql'] && this.editorInstance) {
      this.editorInstance.setValue(this.initialSql || '');
    }
  }

  private createEditor() {
    if (!this.editorContainer || !this.monaco) return;

    

    this.editorInstance = this.monaco.editor.create(this.editorContainer.nativeElement, {
      value: this.initialSql || '',
      language: 'sql',
      theme: this.themeService.isDark() ? 'vs-dark' : 'vs-light',
      fontSize: 14,
      fontFamily: 'Fira Code, monospace',
      minimap: { enabled: false },
      lineNumbers: 'on',
      wordWrap: 'on',
      automaticLayout: true,
      scrollbar: {
        verticalScrollbarSize: 10,
        horizontalScrollbarSize: 8
      }
    });

    this.editorInstance.onDidChangeModelContent(() => {
      const value = this.editorInstance?.getValue();
      this.sqlChanged.emit(value ?? '');
    });
  }

  public getSql(): string {
    return this.editorInstance?.getValue() ?? '';
  }
}
