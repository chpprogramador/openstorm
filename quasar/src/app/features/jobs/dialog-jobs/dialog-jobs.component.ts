import { Component, Inject, ViewEncapsulation } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormBuilder, FormGroup, ReactiveFormsModule } from '@angular/forms';
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
import { InformDialogComponent } from '../../../shared/components/inform-dialog/inform-dialog.component';
import { SqlEditor } from './sql-editor/sql-editor.component';

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
  private localJob: Job;

  constructor(
    public dialogRef: MatDialogRef<DialogJobs>,
    private dialog: MatDialog,
    @Inject(MAT_DIALOG_DATA) public data: Job,
    private fb: FormBuilder,
    private jobService: JobService,
    private appState: AppState
  ) {
    this.localJob = structuredClone(this.data);
    this.form = this.fb.group({
      id: [this.localJob?.id || ''],
      jobName: [this.localJob?.jobName || '', []],
      selectSql: [this.localJob?.selectSql || '', []],
      insertSql: [this.localJob?.insertSql || '', []],
      posInsertSql: [this.localJob?.posInsertSql || '', []],
      recordsPerPage: [this.localJob?.recordsPerPage || 100, []],
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
    if (this.form.valid) {
      if (this.form.get('type')?.value === 'insert') {
        const validateJob: ValidateJob = {
          selectSQL: this.form.value.selectSql,
          insertSQL: this.form.value.insertSql,
          limit: this.form.value.recordsPerPage,
          projectId: this.appState.project?.id || ''
        };

        this.jobService.validate(validateJob).subscribe({
          next: (response) => {
            this.data.columns = response.columns || [];
            this.form.patchValue({ columns: response.columns });
            this.openInformDialog('Querys validadas com sucesso!', true);
          },
          error: (error) => {
            this.openInformDialog('Erro na validacao: ' + error.error.message, false);
          }
        });
      } else {
        this.dialogRef.close(this.form.value);
      }
    }
  }

  openInformDialog(message: string, success: boolean) {
    const dialogRefInf = this.dialog.open(InformDialogComponent, {
      panelClass: 'custom-dialog-container',
      data: {
        title: 'Informacao',
        message
      }
    });

    dialogRefInf.afterClosed().subscribe(() => {
      if (success) {
        dialogRefInf.close(this.form.value);
        this.dialogRef.close(this.form.value);
      }
    });
  }

  onSelectAtualizado(novoSql: string) {
    this.selectAtualizado = novoSql;
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
}
