import { Component, Inject } from '@angular/core';
// Update the path below to the correct relative path and extension for SqlEditorComponent
import { CommonModule } from '@angular/common';
import { FormBuilder, FormGroup, ReactiveFormsModule } from '@angular/forms';
import { MatButtonModule } from '@angular/material/button';
import { MAT_DIALOG_DATA, MatDialogModule, MatDialogRef } from '@angular/material/dialog';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatGridListModule } from '@angular/material/grid-list';
import { MatInputModule } from '@angular/material/input';
import { Job } from '../../../services/job.service';
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
    MatGridListModule
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
    @Inject(MAT_DIALOG_DATA) public data: Job,
    private fb: FormBuilder
  ) {
    this.form = this.fb.group({
      id: [this.data?.id || ''],  
      jobName: [this.data?.jobName || '', []],
      selectSql: [this.data?.selectSql || '', []],
      insertSql: [this.data?.insertSql || '', []],
      recordsPerPage: [this.data?.recordsPerPage || 100, []],
      top: [this.data?.top || 0, []],
      left: [this.data?.left || 0, []]
    });
    this.sqlSelect = this.data?.selectSql || '';
    this.sqlInsert = this.data?.insertSql || '';

  }

  onCancel() {
    this.dialogRef.close();
  }

  onSave() {
    if (this.form.valid) {
      this.dialogRef.close(this.form.value);
    }
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
