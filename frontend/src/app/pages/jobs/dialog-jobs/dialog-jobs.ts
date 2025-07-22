import { Component, Inject } from '@angular/core';
// Update the path below to the correct relative path and extension for SqlEditorComponent
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
import { AppState } from '../../../services/app-state';
import { Job, JobService, ValidateJob } from '../../../services/job.service';
import { InformComponent } from '../../dialog-inform/dialog-inform';
import { SqlEditor } from './sql-editor/sql-editor';


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
  templateUrl: './dialog-jobs.html',
  styleUrl: './dialog-jobs.scss'
})
export class DialogJobs {

    form!: FormGroup;
    sqlSelect = '';
    sqlInsert = '';
    selectAtualizado = '';
    insertAtualizado = '';

  constructor(
    public dialogRef: MatDialogRef<DialogJobs>,
    private dialog: MatDialog,
    @Inject(MAT_DIALOG_DATA) public data: Job,
    private fb: FormBuilder,
    private jobService: JobService,
    private appState: AppState
  ) {
    console.log('DialogJobs data:', data.stopOnError);
    this.form = this.fb.group({
      id: [this.data?.id || ''],  
      jobName: [this.data?.jobName || '', []],
      selectSql: [this.data?.selectSql || '', []],
      insertSql: [this.data?.insertSql || '', []],
      recordsPerPage: [this.data?.recordsPerPage || 100, []],
      type: [this.data?.type || 'insert', []],
      stopOnError: [this.data?.stopOnError, []],
      top: [this.data?.top || 0, []],
      left: [this.data?.left || 0, []],
      columns: [this.data?.columns || []]
    });
    this.sqlSelect = this.data?.selectSql || '';
    this.sqlInsert = this.data?.insertSql || '';

  }

  onCancel() {
    this.dialogRef.close();
  }

  onSave() {
    if (this.form.valid) {
      
      if (this.form.get('type')?.value === 'insert') {

        let validateJob: ValidateJob = {
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
            console.error('Validation failed:', error);
            this.openInformDialog('Erro na validação: ' + error.error.message, false);
          }
        });

      } else {
        this.dialogRef.close(this.form.value);
      }
    }
  }

  //open InformComponent Dialog
  openInformDialog(message: string, success: boolean) {
    const dialogRefInf = this.dialog.open(InformComponent, {
      panelClass: 'custom-dialog-container',
      data: {
        title: 'Informação',
        message: message
      }
    });

    dialogRefInf.afterClosed().subscribe((result: any) => {
      if (success) {
        dialogRefInf.close(this.form.value);
        this.dialogRef.close(this.form.value);
      }      
      
    });
  }


  onSelectAtualizado(novoSql: string) {
    this.selectAtualizado = novoSql;
    this.data.selectSql = novoSql;
    this.form.patchValue({ selectSql: novoSql });
  }

  onInsertAtualizado(novoSql: string) {
    this.insertAtualizado = novoSql;
    this.data.insertSql = novoSql;  
    this.form.patchValue({ insertSql: novoSql });
  }



}
