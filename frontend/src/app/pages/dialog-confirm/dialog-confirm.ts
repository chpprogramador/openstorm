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
        <h2 mat-dialog-title>{{ data.title || 'Confirmar' }}</h2>
        <mat-dialog-content>{{ data.message || 'Tem certeza que deseja continuar?' }}</mat-dialog-content>
        <mat-dialog-actions align="end">
        <button mat-button (click)="onCancel()">Cancelar</button>
        <button mat-raised-button color="warn" (click)="onConfirm()">Remover</button>
        </mat-dialog-actions>
    `,
})
export class ConfirmDialogComponent {
  constructor(
    private dialogRef: MatDialogRef<ConfirmDialogComponent>,
    @Inject(MAT_DIALOG_DATA) public data: any
  ) {}

  onConfirm() {
    this.dialogRef.close(true);
  }

  onCancel() {
    this.dialogRef.close(false);
  }
}
