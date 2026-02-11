import { Component, Inject, ViewEncapsulation } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormBuilder, FormGroup, ReactiveFormsModule, Validators } from '@angular/forms';
import { MatButtonModule } from '@angular/material/button';
import { MatOptionModule } from '@angular/material/core';
import { MAT_DIALOG_DATA, MatDialog, MatDialogModule, MatDialogRef } from '@angular/material/dialog';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatGridListModule } from '@angular/material/grid-list';
import { MatInputModule } from '@angular/material/input';
import { MatSelectModule } from '@angular/material/select';
import { MatSlideToggleModule } from '@angular/material/slide-toggle';
import { AppState } from '../../../core/services/app-state';
import { Job, JobService, ValidateJob } from '../../../core/services/job.service';
import { ConfirmDialogComponent } from '../../../shared/components/confirm-dialog/confirm-dialog.component';
import { InformDialogComponent } from '../../../shared/components/inform-dialog/inform-dialog.component';
import { SqlEditor } from './sql-editor/sql-editor.component';

interface DialogJobsData {
  job: Job;
  jobs?: Job[];
}

@Component({
  standalone: true,
  selector: 'app-dialog-jobs',
  imports: [
    SqlEditor,
    MatDialogModule,
    CommonModule,
    MatFormFieldModule,
    MatInputModule,
    MatButtonModule,
    ReactiveFormsModule,
    MatGridListModule,
    MatSelectModule,
    MatOptionModule,
    MatSlideToggleModule
  ],
  templateUrl: './dialog-jobs.component.html',
  styleUrl: './dialog-jobs.component.scss',
  encapsulation: ViewEncapsulation.None
})
export class DialogJobs {
  form!: FormGroup;
  sqlSelect = '';
  sqlInsert = '';
  sqlPosInsert = '';
  selectAtualizado = '';
  insertAtualizado = '';
  posInsertAtualizado = '';
  activeEditor: 'insert' | 'pos-insert' | 'select' = 'select';
  columnsValidationError = '';
  private localJob: Job;
  private allJobsInPipeline: Job[] = [];

  constructor(
    public dialogRef: MatDialogRef<DialogJobs>,
    private dialog: MatDialog,
    @Inject(MAT_DIALOG_DATA) public data: Job | DialogJobsData,
    private fb: FormBuilder,
    private jobService: JobService,
    private appState: AppState
  ) {
    const inputData = this.isDialogData(this.data) ? this.data : { job: this.data };
    this.localJob = structuredClone(inputData.job);
    this.allJobsInPipeline = Array.isArray(inputData.jobs) ? inputData.jobs : [];
    this.form = this.fb.group({
      id: [this.localJob?.id || ''],
      jobName: [this.localJob?.jobName || '', [Validators.required]],
      selectSql: [this.localJob?.selectSql || '', [Validators.required]],
      insertSql: [this.localJob?.insertSql || '', []],
      posInsertSql: [this.localJob?.posInsertSql || '', []],
      recordsPerPage: [this.localJob?.recordsPerPage || 1000, []],
      type: [this.localJob?.type || 'insert', []],
      stopOnError: [this.localJob?.stopOnError, []],
      top: [this.localJob?.top || 0, []],
      left: [this.localJob?.left || 0, []],
      columns: [this.localJob?.columns || []]
    });
    this.sqlSelect = this.localJob?.selectSql || '';
    this.sqlInsert = this.localJob?.insertSql || '';
    this.sqlPosInsert = this.localJob?.posInsertSql || '';
  }

  onCancel() {
    this.dialogRef.close();
  }

  onSave() {
    this.columnsValidationError = '';
    this.form.markAllAsTouched();
    if (!this.form.valid) {
      return;
    }

    const formValue = this.form.getRawValue();
    const payload: Job = {
      id: formValue.id,
      jobName: String(formValue.jobName ?? ''),
      selectSql: String(formValue.selectSql ?? ''),
      insertSql: String(formValue.insertSql ?? ''),
      posInsertSql: String(formValue.posInsertSql ?? ''),
      columns: Array.isArray(formValue.columns) ? formValue.columns : [],
      recordsPerPage: this.isMemorySelectType() ? 1000 : Number(formValue.recordsPerPage ?? 1000),
      type: String(formValue.type ?? 'insert'),
      stopOnError: !!formValue.stopOnError,
      top: Number(formValue.top ?? 0),
      left: Number(formValue.left ?? 0)
    };

    if (this.shouldValidateColumnsFromSelect(payload.type)) {
      this.validateAndBuildColumns(payload);
      return;
    }

    this.dialogRef.close(payload);
  }

  hasDuplicateJobName(): boolean {
    const currentName = String(this.form.get('jobName')?.value ?? '').trim().toLowerCase();
    if (!currentName) return false;
    const currentId = String(this.form.get('id')?.value ?? '');
    return this.allJobsInPipeline.some(job => job.id !== currentId && job.jobName.trim().toLowerCase() === currentName);
  }

  isMemorySelectType(): boolean {
    return this.form.get('type')?.value === 'memory-select';
  }

  private shouldValidateColumnsFromSelect(type: string): boolean {
    return type === 'insert' || type === 'memory-select';
  }

  private validateAndBuildColumns(payload: Job) {
    const validateJob: ValidateJob = {
      selectSQL: payload.selectSql,
      limit: payload.recordsPerPage,
      projectId: this.appState.project?.id || '',
      type: payload.type
    };
    if (payload.type === 'insert') {
      validateJob.insertSQL = payload.insertSql;
    }
    if (payload.type === 'memory-select') {
      validateJob.validationMode = 'select-only';
    }

    this.jobService.validate(validateJob).subscribe({
      next: (response) => {
        const columns = Array.isArray(response.columns)
          ? response.columns.map(col => String(col).trim()).filter(col => col.length > 0)
          : [];
        payload.columns = columns;
        this.form.patchValue({ columns }, { emitEvent: false });

        if (payload.type === 'memory-select' && columns.length === 0) {
          this.columnsValidationError = 'columns e obrigatorio';
          return;
        }

        if (payload.type === 'insert') {
          this.openInformDialog('Querys validadas com sucesso!', true, payload);
          return;
        }

        this.dialogRef.close(payload);
      },
      error: (error) => {
        const errorMessage = this.resolveValidationErrorMessage(error);
        this.openValidationErrorConfirmDialog(errorMessage, payload);
      }
    });
  }

  private isDialogData(value: Job | DialogJobsData): value is DialogJobsData {
    return !!(value as DialogJobsData)?.job;
  }

  private resolveValidationErrorMessage(error: any): string {
    const backendMessage = error?.error?.message || error?.message;
    return backendMessage ? String(backendMessage) : 'Erro desconhecido na validacao.';
  }

  private openValidationErrorConfirmDialog(validationErrorMessage: string, payload: Job) {
    const dialogRef = this.dialog.open(ConfirmDialogComponent, {
      panelClass: 'custom-dialog-container',
      minWidth: '36vw',
      data: {
        title: 'Erro de validacao',
        message: `Erro na validacao: ${validationErrorMessage}\n\nGostaria de salvar assim mesmo?`,
        confirmLabel: 'Sim',
        cancelLabel: 'Nao'
      }
    });

    dialogRef.afterClosed().subscribe(confirmed => {
      if (confirmed) {
        this.dialogRef.close(payload);
      }
    });
  }

  openInformDialog(message: string, success: boolean, payload?: Job) {
    const dialogRefInf = this.dialog.open(InformDialogComponent, {
      panelClass: 'custom-dialog-container',
      data: {
        title: 'Informacao',
        message
      }
    });

    dialogRefInf.afterClosed().subscribe(() => {
      if (success) {
        dialogRefInf.close(payload ?? this.form.getRawValue());
        this.dialogRef.close(payload ?? this.form.getRawValue());
      }
    });
  }

  onSelectAtualizado(novoSql: string) {
    this.selectAtualizado = novoSql;
    this.columnsValidationError = '';
    this.form.patchValue({ selectSql: novoSql });
  }

  onInsertAtualizado(novoSql: string) {
    this.insertAtualizado = novoSql;
    this.form.patchValue({ insertSql: novoSql });
  }

  onPosInsertAtualizado(novoSql: string) {
    this.posInsertAtualizado = novoSql;
    this.form.patchValue({ posInsertSql: novoSql });
  }

  setActiveEditor(editor: 'insert' | 'pos-insert' | 'select') {
    this.activeEditor = editor;
  }

  isInsertType(): boolean {
    return this.form.get('type')?.value === 'insert';
  }
}
