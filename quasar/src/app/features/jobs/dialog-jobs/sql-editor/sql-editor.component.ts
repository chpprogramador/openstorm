import { isPlatformBrowser } from '@angular/common';
import {
  AfterViewInit,
  Component,
  ElementRef,
  EventEmitter,
  Inject,
  Input,
  OnChanges,
  Output,
  PLATFORM_ID,
  SimpleChanges,
  ViewChild
} from '@angular/core';
import { ThemeService } from '../../../../core/services/theme.service';
import { format } from 'sql-formatter';

@Component({
  selector: 'app-sql-editor',
  standalone: true,
  templateUrl: './sql-editor.component.html',
  styleUrls: ['./sql-editor.component.scss']
})
export class SqlEditor implements AfterViewInit, OnChanges {
  @ViewChild('editorContainer', { static: true }) editorContainer!: ElementRef<HTMLDivElement>;
  @Input() initialSql = '';
  @Output() sqlChanged = new EventEmitter<string>();

  isBrowser: boolean;
  private editorInstance?: import('monaco-editor').editor.IStandaloneCodeEditor;
  private monaco: typeof import('monaco-editor') | null = null;

  constructor(
    @Inject(PLATFORM_ID) private platformId: Object,
    private themeService: ThemeService
  ) {
    this.isBrowser = isPlatformBrowser(this.platformId);
  }

  async ngAfterViewInit() {
    if (this.isBrowser) {
      this.monaco = await import(/* @vite-ignore */ 'monaco-editor');

      this.monaco.languages.register({ id: 'custom-sql' });

      this.monaco.languages.setMonarchTokensProvider('custom-sql', {
        tokenizer: {
          root: [
            [/\b(select|insert|update|delete|from|where|join|inner|left|right|full|on|as|group|by|order|limit|offset|having|distinct|create|alter|drop|table|view|index|if|exists|case|when|then|else|end)\b/, 'keyword'],
            [/\b(true|false|null)\b/, 'constant'],
            [/'([^']*)'/, 'string'],
            [/".*?"/, 'string'],
            [/[0-9]+/, 'number'],
            [/--+.*/, 'comment'],
            [/\/\*[^]*?\*\//, 'comment']
          ]
        }
      });

      this.monaco.editor.defineTheme('custom-dark', {
        base: 'vs-dark',
        inherit: true,
        rules: [
          { token: 'keyword', foreground: '22d3ee', fontStyle: 'bold' },
          { token: 'constant', foreground: '38bdf8' },
          { token: 'string', foreground: 'f8b4b4' },
          { token: 'number', foreground: 'fbbf24' },
          { token: 'comment', foreground: '7dd3fc', fontStyle: 'italic' }
        ],
        colors: {}
      });

      this.monaco.editor.defineTheme('custom-light', {
        base: 'vs',
        inherit: true,
        rules: [
          { token: 'keyword', foreground: '0ea5e9', fontStyle: 'bold' },
          { token: 'constant', foreground: '0284c7' },
          { token: 'string', foreground: 'dc2626' },
          { token: 'number', foreground: 'ea580c' },
          { token: 'comment', foreground: '64748b', fontStyle: 'italic' }
        ],
        colors: {}
      });

      this.createEditor();
      this.registerFormatterAction();
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
      language: 'custom-sql',
      theme: this.themeService.isDarkMode() ? 'custom-dark' : 'custom-light',
      fontSize: 14,
      fontFamily: 'JetBrains Mono, monospace',
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

  private registerFormatterAction() {
    if (!this.editorInstance || !this.monaco) return;

    this.editorInstance.addAction({
      id: 'format-sql',
      label: 'Formatar SQL',
      keybindings: [this.monaco.KeyMod.CtrlCmd | this.monaco.KeyMod.Shift | this.monaco.KeyCode.KeyF],
      run: (editor) => {
        const code = editor.getValue();
        const formatted = format(code, {
          language: 'postgresql',
          keywordCase: 'upper',
          indentStyle: 'standard'
        });
        editor.setValue(formatted);
      }
    });
  }

  public getSql(): string {
    return this.editorInstance?.getValue() ?? '';
  }
}
