import { isPlatformBrowser } from '@angular/common';
import {
  AfterViewInit, Component, ElementRef,
  EventEmitter, Inject, Input,
  OnChanges, Output, PLATFORM_ID,
  SimpleChanges, ViewChild
} from '@angular/core';
import { ThemeService } from '../../../../services/theme.service';
import { format } from 'sql-formatter'; // instalar: npm i sql-formatter

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

  constructor(
    @Inject(PLATFORM_ID) private platformId: Object,
    private themeService: ThemeService
  ) {
    this.isBrowser = isPlatformBrowser(this.platformId);
  }

  async ngAfterViewInit() {
    if (this.isBrowser) {
      this.monaco = await import(/* @vite-ignore */ 'monaco-editor');

      // 1. Registrar linguagem customizada
      this.monaco.languages.register({ id: 'custom-sql' });

      // 2. Definir tokens SQL
      this.monaco.languages.setMonarchTokensProvider('custom-sql', {
        tokenizer: {
          root: [
            // Keywords
            [/\b(select|insert|update|delete|from|where|join|inner|left|right|full|on|as|group|by|order|limit|offset|having|distinct|create|alter|drop|table|view|index|if|exists|case|when|then|else|end)\b/, 'keyword'],
            // Constants
            [/\b(true|false|null)\b/, 'constant'],
            // Strings
            [/'([^']*)'/, 'string'],
            [/".*?"/, 'string'],
            // Numbers
            [/[0-9]+/, 'number'],
            // Comments
            [/--+.*/, 'comment'],
            [/\/\*[^]*?\*\//, 'comment']
          ]
        }
      });

      // 3. Definir temas customizados
      this.monaco.editor.defineTheme('custom-dark', {
        base: 'vs-dark',
        inherit: true,
        rules: [
          { token: 'keyword', foreground: 'C586C0', fontStyle: 'bold' },
          { token: 'constant', foreground: '569CD6' },
          { token: 'string', foreground: 'CE9178' },
          { token: 'number', foreground: 'B5CEA8' },
          { token: 'comment', foreground: '6A9955', fontStyle: 'italic' }
        ],
        colors: {}
      });

      this.monaco.editor.defineTheme('custom-light', {
        base: 'vs',
        inherit: true,
        rules: [
          { token: 'keyword', foreground: '0000FF', fontStyle: 'bold' },
          { token: 'constant', foreground: '0070C1' },
          { token: 'string', foreground: 'A31515' },
          { token: 'number', foreground: '098658' },
          { token: 'comment', foreground: '008000', fontStyle: 'italic' }
        ],
        colors: {}
      });

      // 4. Criar editor só depois de registrar linguagem e tema
      this.createEditor();

      // 5. Atalho para formatar SQL
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
      theme: this.themeService.isDark() ? 'custom-dark' : 'custom-light',
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

  /** 🔹 Registra ação para formatar SQL (Ctrl+Shift+F) */
  private registerFormatterAction() {
    if (!this.editorInstance || !this.monaco) return;

    this.editorInstance.addAction({
      id: 'format-sql',
      label: 'Formatar SQL',
      keybindings: [this.monaco.KeyMod.CtrlCmd | this.monaco.KeyMod.Shift | this.monaco.KeyCode.KeyF],
      run: (editor) => {
        const code = editor.getValue();
        const formatted = format(code, {
          language: 'postgresql', // pode trocar para mysql, sql etc
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
