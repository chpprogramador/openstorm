import { CommonModule } from '@angular/common';
import { Component, Inject, ViewEncapsulation } from '@angular/core';
import { FormBuilder, FormGroup, ReactiveFormsModule, Validators } from '@angular/forms';
import { MatButtonModule } from '@angular/material/button';
import { MAT_DIALOG_DATA, MatDialogModule, MatDialogRef } from '@angular/material/dialog';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatGridListModule } from '@angular/material/grid-list';
import { MatInputModule } from '@angular/material/input';
import { MatSelectModule } from '@angular/material/select';
import { Project } from '../../../services/project.service';

@Component({
  standalone: true,
  selector: 'app-dialog-project',
  imports: [
    MatDialogModule,
    CommonModule,
    MatFormFieldModule,
    MatInputModule,
    MatButtonModule,
    ReactiveFormsModule,
    MatGridListModule,
    MatSelectModule
  ],
  templateUrl: './dialog-project.html',
  styleUrl: './dialog-project.scss',
  encapsulation: ViewEncapsulation.None
})
export class DialogProject {
  form!: FormGroup;

  constructor(
    public dialogRef: MatDialogRef<DialogProject>,
    @Inject(MAT_DIALOG_DATA) public data: Project,
    private fb: FormBuilder
  ) {
    
  }

  ngOnInit() {
    this.form = this.fb.group({
      id: [this.data?.id || ''],
      jobs: [this.data?.jobs || []],
      connections: [this.data?.connections || []],
      variables: [this.data?.variables || []],
      projectName: [this.data?.projectName || '', [Validators.required]],
      concurrency: [this.data?.concurrency || 1, [Validators.required, Validators.min(1)]],
      sourceDatabase: this.fb.group({
        type: [this.data?.sourceDatabase?.type || '', [Validators.required]],
        host: [this.data?.sourceDatabase?.host || '', [Validators.required]],
        port: [this.data?.sourceDatabase?.port || '', [Validators.required]],
        user: [this.data?.sourceDatabase?.user || '', [Validators.required]],
        password: [this.data?.sourceDatabase?.password || '', [Validators.required]],
        database: [this.data?.sourceDatabase?.database || '', [Validators.required]],
      }),
      destinationDatabase: this.fb.group({
        type: [this.data?.destinationDatabase?.type || '', [Validators.required]],
        host: [this.data?.destinationDatabase?.host || '', [Validators.required]],
        port: [this.data?.destinationDatabase?.port || '', [Validators.required]],
        user: [this.data?.destinationDatabase?.user || '', [Validators.required]],
        password: [this.data?.destinationDatabase?.password || '', [Validators.required]],
        database: [this.data?.destinationDatabase?.database || '', [Validators.required]],
      }),
    });
  }

  onSave() {
    if (this.form.valid) {
      this.dialogRef.close(this.form.value);
    }
  }

  onCancel() {
    this.dialogRef.close();
  }
}
