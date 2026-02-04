import { ChangeDetectorRef, Component, OnDestroy, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { MatButtonModule } from '@angular/material/button';
import { MatCardModule } from '@angular/material/card';
import { MatDialog, MatDialogModule } from '@angular/material/dialog';
import { MatIconModule } from '@angular/material/icon';
import { MatTableModule } from '@angular/material/table';
import { MatSnackBar, MatSnackBarModule } from '@angular/material/snack-bar';
import { MatTooltipModule } from '@angular/material/tooltip';
import { MatChipsModule } from '@angular/material/chips';

import { AppState } from '../../core/services/app-state';
import { Variable } from '../../core/models/project.model';
import { VariableService } from '../../core/services/variable.service';
import { DialogVariableComponent } from './dialog-variable/dialog-variable.component';
import { ConfirmDialogComponent } from '../../shared/components/confirm-dialog/confirm-dialog.component';
import { ProjectService } from '../../core/services/project.service';
import { Subscription } from 'rxjs';

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
    MatChipsModule,
    MatSnackBarModule
  ],
  templateUrl: './variables.component.html',
  styleUrls: ['./variables.component.scss']
})
export class VariablesComponent implements OnInit, OnDestroy {
  variables: Variable[] = [];
  displayedColumns: string[] = ['name', 'type', 'value', 'description', 'actions'];
  loading = false;
  private projectSub?: Subscription;

  constructor(
    private variableService: VariableService,
    public appState: AppState,
    private dialog: MatDialog,
    private snackBar: MatSnackBar,
    private projectService: ProjectService,
    private cdr: ChangeDetectorRef
  ) {}

  ngOnInit() {
    this.projectSub = this.projectService.selectedProject$.subscribe(() => {
      this.loadVariables();
    });
  }

  ngOnDestroy() {
    this.projectSub?.unsubscribe();
  }

  loadVariables() {
    const projectId = this.appState.project?.id;
    if (!projectId) {
      this.showMessage('Nenhum projeto selecionado');
      this.loading = false;
      this.cdr.detectChanges();
      return;
    }

    this.loading = true;
    this.cdr.detectChanges();
    this.variableService.listVariables(projectId).subscribe({
      next: (variables) => {
        this.variables = variables || [];
        this.loading = false;
        this.cdr.detectChanges();
      },
      error: () => {
        this.showMessage('Erro ao carregar variaveis');
        this.loading = false;
        this.cdr.detectChanges();
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
        this.showMessage('Variavel criada com sucesso');
        this.loadVariables();
      },
      error: (error) => {
        this.showMessage('Erro ao criar variavel: ' + (error.error?.message || error.message));
      }
    });
  }

  updateVariable(originalName: string, variable: Variable) {
    const projectId = this.appState.project?.id;
    if (!projectId) return;

    this.variableService.updateVariable(projectId, originalName, variable).subscribe({
      next: () => {
        this.showMessage('Variavel atualizada com sucesso');
        this.loadVariables();
      },
      error: () => {
        this.showMessage('Erro ao atualizar variavel');
      }
    });
  }

  deleteVariable(variable: Variable) {
    const dialogRef = this.dialog.open(ConfirmDialogComponent, {
      data: {
        title: 'Confirmar Exclusao',
        message: `Tem certeza que deseja excluir a variavel "${variable.name}"?`,
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
            this.showMessage('Variavel excluida com sucesso');
            this.loadVariables();
          },
          error: () => {
            this.showMessage('Erro ao excluir variavel');
          }
        });
      }
    });
  }

  getTypeLabel(type?: string): string {
    switch (type) {
      case 'string':
        return 'Texto';
      case 'number':
        return 'Numero';
      case 'boolean':
        return 'Booleano';
      case 'date':
        return 'Data';
      default:
        return 'Texto';
    }
  }

  getTypeColor(type?: string): string {
    switch (type) {
      case 'string':
        return 'primary';
      case 'number':
        return 'accent';
      case 'boolean':
        return 'warn';
      case 'date':
        return '';
      default:
        return 'primary';
    }
  }

  formatValue(value: string, type?: string): string {
    return this.variableService.formatVariableValue(value, type || 'string');
  }

  copyToClipboard(variableName: string) {
    const textToCopy = '${' + variableName + '}';
    navigator.clipboard
      .writeText(textToCopy)
      .then(() => {
        this.showMessage(`Referencia da variavel copiada: ${textToCopy}`);
      })
      .catch(() => {
        this.showMessage('Erro ao copiar referencia da variavel');
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
