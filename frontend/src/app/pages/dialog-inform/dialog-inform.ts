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
        <div class="dialog-header">
            <h2 mat-dialog-title>{{ data.title || 'Informação' }}</h2>
        </div>
        <mat-dialog-content class="dialog-content">
            <p class="dialog-message">{{ data.message || 'Informação!' }}</p>
        </mat-dialog-content>
        <mat-dialog-actions class="dialog-actions" align="end">
            <button mat-raised-button color="primary" class="ok-btn" (click)="onConfirm()">Ok</button>
        </mat-dialog-actions>
    `,
    styles: [`
        .dialog-header {
            padding: 1.5rem 1.5rem 0 1.5rem;
            border-bottom: 1px solid var(--border-color, #e0e0e0);
            margin-bottom: 1rem;
        }
        .dialog-header h2 {
            margin: 0 0 1rem 0;
            color: var(--text-primary, #333);
            font-size: 1.25rem;
            font-weight: 600;
        }
        .dialog-content {
            padding: 0 1.5rem 1rem 1.5rem;
            min-height: 60px;
        }
        .dialog-message {
            margin: 0;
            color: var(--text-secondary, #666);
            font-size: 0.95rem;
            line-height: 1.5;
        }
        .dialog-actions {
            padding: 1rem 1.5rem 1.5rem 1.5rem;
            border-top: 1px solid var(--border-color, #e0e0e0);
        }
        .ok-btn {
            min-width: 80px;
            height: 36px;
            border-radius: 6px;
            font-weight: 500;
            text-transform: none;
        }
    `]
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
