import { Component, Inject, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { MatButtonModule } from '@angular/material/button';
import { MatDialogRef, MAT_DIALOG_DATA, MatDialogModule } from '@angular/material/dialog';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import { MatSelectModule } from '@angular/material/select';
import { VariableService } from '../../../core/services/variable.service';

@Component({
  standalone: true,
  selector: 'app-dialog-variable',
  imports: [
    CommonModule,
    FormsModule,
    MatButtonModule,
    MatDialogModule,
    MatFormFieldModule,
    MatInputModule,
    MatSelectModule
  ],
  templateUrl: './dialog-variable.component.html',
  styleUrls: ['./dialog-variable.component.scss']
})
export class DialogVariableComponent implements OnInit {
  variable = {
    name: '',
    value: '',
    description: '',
    type: 'string'
  };

  variableTypes = [
    { value: 'string', label: 'Texto' },
    { value: 'number', label: 'Numero' },
    { value: 'boolean', label: 'Booleano' },
    { value: 'date', label: 'Data' }
  ];

  constructor(
    public dialogRef: MatDialogRef<DialogVariableComponent>,
    private variableService: VariableService,
    @Inject(MAT_DIALOG_DATA) public data: any
  ) {}

  ngOnInit() {
    if (this.data.variable) {
      this.variable = { ...this.data.variable };
    }
  }

  cancel() {
    this.dialogRef.close();
  }

  save() {
    if (!this.variable.name || !this.variable.value) {
      return;
    }

    const variableToSave = {
      name: this.variable.name,
      value: this.variable.value,
      description: this.variable.description || '',
      type: this.variable.type
    };

    this.dialogRef.close(variableToSave);
  }

  validateValue() {
    const { value, type } = this.variable;
    if (this.variableService.validateVariableValue(value, type)) {
      return true;
    } else {
      this.variable.value = '';
      return false;
    }
  }
}
