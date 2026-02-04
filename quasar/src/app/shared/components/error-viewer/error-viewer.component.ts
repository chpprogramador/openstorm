import { CommonModule } from '@angular/common';
import { Component, Input, OnInit } from '@angular/core';
import { MatBadgeModule } from '@angular/material/badge';
import { MatButtonModule } from '@angular/material/button';
import { MatCardModule } from '@angular/material/card';
import { MatChipsModule } from '@angular/material/chips';
import { MatDividerModule } from '@angular/material/divider';
import { MatExpansionModule } from '@angular/material/expansion';
import { MatIconModule } from '@angular/material/icon';
import { MatListModule } from '@angular/material/list';
import { MatTooltipModule } from '@angular/material/tooltip';
import { ErrorSummary } from '../../../core/services/error.service';

@Component({
  selector: 'app-error-viewer',
  standalone: true,
  imports: [
    CommonModule,
    MatCardModule,
    MatExpansionModule,
    MatIconModule,
    MatChipsModule,
    MatButtonModule,
    MatTooltipModule,
    MatDividerModule,
    MatListModule,
    MatBadgeModule
  ],
  templateUrl: './error-viewer.component.html',
  styleUrls: ['./error-viewer.component.scss']
})
export class ErrorViewerComponent implements OnInit {
  @Input() pipelineId: string = '';
  @Input() errorSummary: ErrorSummary | null = null;

  get jobErrors() {
    return Array.isArray(this.errorSummary?.error_jobs) ? this.errorSummary!.error_jobs : [];
  }

  get batchErrors() {
    return Array.isArray(this.errorSummary?.error_batches) ? this.errorSummary!.error_batches : [];
  }

  ngOnInit() {}

  getErrorTypes() {
    if (!this.errorSummary?.error_types) return [];
    return Object.entries(this.errorSummary.error_types).map(([type, count]) => ({
      type,
      count
    }));
  }

  getErrorTypeClass(errorType: string | undefined): string {
    const classes: { [key: string]: string } = {
      duplicate_key_error: 'error-chip-duplicate',
      foreign_key_error: 'error-chip-foreign-key',
      connection_error: 'error-chip-connection',
      sql_syntax_error: 'error-chip-syntax',
      permission_error: 'error-chip-permission',
      table_not_found: 'error-chip-table',
      unknown_error: 'error-chip-unknown'
    };
    if (!errorType) return 'error-chip-unknown';
    return classes[errorType] || 'error-chip-unknown';
  }

  getErrorTypeIcon(errorType: string | undefined): string {
    const icons: { [key: string]: string } = {
      duplicate_key_error: 'content_copy',
      foreign_key_error: 'link_off',
      connection_error: 'wifi_off',
      sql_syntax_error: 'code_off',
      permission_error: 'lock',
      table_not_found: 'table_chart',
      unknown_error: 'help'
    };
    if (!errorType) return 'help';
    return icons[errorType] || 'help';
  }

  getErrorTypeLabel(errorType: string | undefined): string {
    const labels: { [key: string]: string } = {
      duplicate_key_error: 'Chave Duplicada',
      foreign_key_error: 'Chave Estrangeira',
      connection_error: 'Conexao',
      sql_syntax_error: 'Sintaxe SQL',
      permission_error: 'Permissao',
      table_not_found: 'Tabela Nao Encontrada',
      unknown_error: 'Erro Desconhecido'
    };
    if (!errorType) return 'Erro Desconhecido';
    return labels[errorType] || 'Erro Desconhecido';
  }

  getErrorSeverity(errorType: string | undefined): string {
    const severities: { [key: string]: string } = {
      duplicate_key_error: 'warn',
      foreign_key_error: 'warn',
      connection_error: 'warn',
      sql_syntax_error: 'warn',
      permission_error: 'warn',
      table_not_found: 'warn',
      unknown_error: 'warn'
    };
    if (!errorType) return 'warn';
    return severities[errorType] || 'warn';
  }

  formatDateTime(dateString: string): string {
    if (!dateString) return '';
    const date = new Date(dateString);
    if (isNaN(date.getTime())) return '';
    return date.toLocaleString('pt-BR');
  }

  formatTimeSafe(dateString?: string): string {
    if (!dateString) return '';
    const date = new Date(dateString);
    if (isNaN(date.getTime())) return '';
    return date.toLocaleTimeString('pt-BR');
  }
}
