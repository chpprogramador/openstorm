import { Component, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { MatButtonModule } from '@angular/material/button';
import { MatCardModule } from '@angular/material/card';
import { MatDialog, MatDialogModule } from '@angular/material/dialog';
import { MatIconModule } from '@angular/material/icon';
import { MatTableModule } from '@angular/material/table';
import { MatSnackBar } from '@angular/material/snack-bar';
import { MatTooltipModule } from '@angular/material/tooltip';
import { MatChipsModule } from '@angular/material/chips';

import { AppState } from '../../core/services/app-state.service'; // Adjusted path
import { Variable } from '../../core/models/variable.model'; // Adjusted path
import { VariableService } from '../../core/services/variable.service'; // Adjusted path
import { DialogVariableComponent } from './dialog-variable/dialog-variable';
import { ConfirmDialogComponent } from '../../shared/components/confirm-dialog/confirm-dialog.component'; // Adjusted path

@Component({
  standalone: true,
  selector: 'app-variables',
  imports: [
    CommonModule,
    FormsModule,
    MatButtonModule,
    MatCardModule,
    MatDialogModule,
    MatIconModule,
    MatTableModule,
    MatTooltipModule,
    MatChipsModule
  ],
  templateUrl: './variables.html',
  styleUrls: ['./variables.scss']
})
export class Variables implements OnInit {
  variables: Variable[] = [];
  displayedColumns: string[] = ['name', 'type', 'value', 'description', 'actions'];
  loading = false;

  constructor(
    private variableService: VariableService,
    public appState: AppState,
    private dialog: MatDialog,
    private snackBar: MatSnackBar
  ) {}

  ngOnInit() {
    this.loadVariables();
  }

  loadVariables() {
    const projectId = this.appState.project?.id;
    if (!projectId) {
      this.showMessage('Nenhum projeto selecionado');
      return;
    }

    this.loading = true;
    this.variableService.listVariables(projectId).subscribe({
      next: (variables) => {
        this.variables = variables || [];
        this.loading = false;
      },
      error: (error) => {
        console.error('Erro ao carregar variáveis:', error);
        this.showMessage('Erro ao carregar variáveis');
        this.loading = false;
      }
    });
  }

  openVariableDialog(variable?: Variable) {
    const dialogRef = this.dialog.open(DialogVariableComponent, {
      width: '500px',
      data: { 
        variable: variable ? { ...variable } : null,
        isEdit: !!variable,
        existingNames: this.variables.map(v => v.name).filter(name => name !== variable?.name)
      }
    });

    dialogRef.afterClosed().subscribe(result => {
      if (result) {
        if (variable) {
          this.updateVariable(variable.name, result);
        } else {
          this.createVariable(result);
        }
      }
    });
  }

  createVariable(variable: Variable) {
    const projectId = this.appState.project?.id;
    if (!projectId) return;


    this.variableService.createVariable(projectId, variable).subscribe({
      next: () => {
        this.showMessage('Variável criada com sucesso');
        this.loadVariables();
      },
      error: (error) => {
        console.error('Erro ao criar variável:', error);
        console.error('Response body:', error.error);
        console.error('Status:', error.status);
        this.showMessage('Erro ao criar variável: ' + (error.error?.message || error.message));
      }
    });
  }

  updateVariable(originalName: string, variable: Variable) {
    const projectId = this.appState.project?.id;
    if (!projectId) return;

    this.variableService.updateVariable(projectId, originalName, variable).subscribe({
      next: () => {
        this.showMessage('Variável atualizada com sucesso');
        this.loadVariables();
      },
      error: (error) => {
        console.error('Erro ao atualizar variável:', error);
        this.showMessage('Erro ao atualizar variável');
      }
    });
  }

  deleteVariable(variable: Variable) {
    const dialogRef = this.dialog.open(ConfirmDialogComponent, {
      data: {
        title: 'Confirmar Exclusão',
        message: `Tem certeza que deseja excluir a variável "${variable.name}"?`,
        confirmText: 'Excluir',
        cancelText: 'Cancelar'
      }
    });

    dialogRef.afterClosed().subscribe(result => {
      if (result) {
        const projectId = this.appState.project?.id;
        if (!projectId) return;

        this.variableService.deleteVariable(projectId, variable.name).subscribe({
          next: () => {
            this.showMessage('Variável excluída com sucesso');
            this.loadVariables();
          },
          error: (error) => {
            console.error('Erro ao excluir variável:', error);
            this.showMessage('Erro ao excluir variável');
          }
        });
      }
    });
  }

  getTypeLabel(type?: string): string {
    switch (type) {
      case 'string': return 'Texto';
      case 'number': return 'Número';
      case 'boolean': return 'Booleano';
      case 'date': return 'Data';
      default: return 'Texto';
    }
  }

  getTypeColor(type?: string): string {
    switch (type) {
      case 'string': return 'primary';
      case 'number': return 'accent';
      case 'boolean': return 'warn';
      case 'date': return '';
      default: return 'primary';
    }
  }

  formatValue(value: string, type?: string): string {
    return this.variableService.formatVariableValue(value, type || 'string');
  }

  copyToClipboard(variableName: string) {
    const textToCopy = `\${${variableName}}`;
    navigator.clipboard.writeText(textToCopy).then(() => {
      this.showMessage(`Referência da variável copiada: ${textToCopy}`);
    }).catch(() => {
      this.showMessage('Erro ao copiar referência da variável');
    });
  }

  private showMessage(message: string) {
    this.snackBar.open(message, 'Fechar', {
      duration: 3000,
      horizontalPosition: 'right',
      verticalPosition: 'top'
    });
  }
}
