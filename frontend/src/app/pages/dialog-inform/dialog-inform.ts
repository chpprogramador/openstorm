import { Component, Inject } from '@angular/core';
import { MatButtonModule } from '@angular/material/button';
import { MAT_DIALOG_DATA, MatDialogModule, MatDialogRef } from '@angular/material/dialog';

@Component({
    standalone: true,
    selector: 'app-confirm-dialog',
    imports: [
        MatDialogModule,
        MatButtonModule
    ],
    template: `
        <h2 mat-dialog-title>{{ data.title || 'Informação' }}</h2>
        <mat-dialog-content>{{ data.message || 'Informação!' }}</mat-dialog-content>
        <mat-dialog-actions align="end">
        <button mat-raised-button color="warn" (click)="onConfirm()">Ok</button>
        </mat-dialog-actions>
    `,
})
export class InformComponent {
  constructor(
    private dialogRef: MatDialogRef<InformComponent>,
    @Inject(MAT_DIALOG_DATA) public data: any
  ) {}

  onConfirm() {
    this.dialogRef.close(true);
  }
  
}
