// src/app/features/variables/variables.component.ts
import { Component } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';

interface Variable {
  id: string;
  key: string;
  value: string;
  description: string;
  type: 'string' | 'number' | 'boolean' | 'secret';
  createdAt: Date;
}

@Component({
  selector: 'app-variables',
  standalone: true,
  imports: [CommonModule, FormsModule],
  template: `
    <div class="container">
      <div class="page-header">
        <div>
          <h1 class="page-title">Variáveis</h1>
          <p class="page-subtitle">Gerencie as variáveis do projeto</p>
        </div>
        <button class="btn-primary" (click)="openModal()">
          <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <line x1="12" y1="5" x2="12" y2="19"/>
            <line x1="5" y1="12" x2="19" y2="12"/>
          </svg>
          Nova Variável
        </button>
      </div>

      <div class="variables-table" *ngIf="variables.length > 0">
        <div class="table-header">
          <div class="col-key">Chave</div>
          <div class="col-value">Valor</div>
          <div class="col-type">Tipo</div>
          <div class="col-description">Descrição</div>
          <div class="col-actions">Ações</div>
        </div>

        <div class="table-row" *ngFor="let variable of variables">
          <div class="col-key">
            <code>{{ variable.key }}</code>
          </div>
          <div class="col-value">
            <span *ngIf="variable.type !== 'secret'">{{ variable.value }}</span>
            <span *ngIf="variable.type === 'secret'" class="secret-value">••••••••</span>
          </div>
          <div class="col-type">
            <span class="type-badge" [attr.data-type]="variable.type">
              {{ variable.type }}
            </span>
          </div>
          <div class="col-description">{{ variable.description || '-' }}</div>
          <div class="col-actions">
            <button class="btn-icon" (click)="editVariable(variable)" title="Editar">
              <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/>
                <path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/>
              </svg>
            </button>
            <button class="btn-icon btn-danger" (click)="deleteVariable(variable)" title="Excluir">
              <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <polyline points="3 6 5 6 21 6"/>
                <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/>
              </svg>
            </button>
          </div>
        </div>
      </div>

      <div class="empty-state" *ngIf="variables.length === 0">
        <svg xmlns="http://www.w3.org/2000/svg" width="64" height="64" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1" stroke-linecap="round" stroke-linejoin="round">
          <path d="M12 2v20M17 5H9.5a3.5 3.5 0 0 0 0 7h5a3.5 3.5 0 0 1 0 7H6"/>
        </svg>
        <h3>Nenhuma variável criada</h3>
        <p>Adicione variáveis para usar no seu projeto</p>
      </div>

      <!-- Modal -->
      <div class="modal" *ngIf="showModal" (click)="closeModal()">
        <div class="modal-content" (click)="$event.stopPropagation()">
          <div class="modal-header">
            <h2>{{ editingVariable ? 'Editar Variável' : 'Nova Variável' }}</h2>
            <button class="btn-close" (click)="closeModal()">×</button>
          </div>
          <div class="modal-body">
            <div class="form-group">
              <label>Chave</label>
              <input type="text" [(ngModel)]="formData.key" placeholder="ex: API_KEY" class="input">
            </div>
            <div class="form-group">
              <label>Valor</label>
              <input [type]="formData.type === 'secret' ? 'password' : 'text'" [(ngModel)]="formData.value" placeholder="Valor da variável" class="input">
            </div>
            <div class="form-group">
              <label>Tipo</label>
              <select [(ngModel)]="formData.type" class="select">
                <option value="string">String</option>
                <option value="number">Number</option>
                <option value="boolean">Boolean</option>
                <option value="secret">Secret</option>
              </select>
            </div>
            <div class="form-group">
              <label>Descrição (opcional)</label>
              <textarea [(ngModel)]="formData.description" placeholder="Descrição da variável" class="textarea" rows="3"></textarea>
            </div>
          </div>
          <div class="modal-footer">
            <button class="btn-secondary" (click)="closeModal()">Cancelar</button>
            <button class="btn-primary" (click)="saveVariable()" [disabled]="!formData.key.trim() || !formData.value.trim()">
              {{ editingVariable ? 'Salvar' : 'Criar' }}
            </button>
          </div>
        </div>
      </div>
    </div>
  `,
  styles: [`
    .container {
      max-width: 1400px;
      margin: 0 auto;
      padding: 3rem 2rem;
    }

    .page-header {
      display: flex;
      justify-content: space-between;
      align-items: flex-start;
      margin-bottom: 2.5rem;
    }

    .page-title {
      font-size: 2rem;
      font-weight: 700;
      color: var(--text-primary);
      margin: 0 0 0.5rem 0;
    }

    .page-subtitle {
      color: var(--text-secondary);
      margin: 0;
    }

    .variables-table {
      background: var(--card-bg);
      border: 1px solid var(--border-color);
      border-radius: 12px;
      overflow: hidden;
    }

    .table-header,
    .table-row {
      display: grid;
      grid-template-columns: 200px 1fr 120px 200px 100px;
      gap: 1rem;
      padding: 1rem 1.5rem;
      align-items: center;
    }

    .table-header {
      background: var(--hover-bg);
      font-weight: 600;
      font-size: 0.875rem;
      color: var(--text-secondary);
      text-transform: uppercase;
      letter-spacing: 0.5px;
    }

    .table-row {
      border-top: 1px solid var(--border-color);
      transition: background 0.2s;
    }

    .table-row:hover {
      background: var(--hover-bg);
    }

    .col-key code {
      background: var(--hover-bg);
      padding: 0.25rem 0.5rem;
      border-radius: 4px;
      font-size: 0.875rem;
      color: #667eea;
      font-family: 'Courier New', monospace;
    }

    .col-value {
      color: var(--text-primary);
      word-break: break-all;
    }

    .secret-value {
      color: var(--text-secondary);
      letter-spacing: 2px;
    }

    .type-badge {
      display: inline-block;
      padding: 0.25rem 0.75rem;
      border-radius: 6px;
      font-size: 0.75rem;
      font-weight: 600;
      text-transform: uppercase;
    }

    .type-badge[data-type="string"] {
      background: rgba(59, 130, 246, 0.1);
      color: #3b82f6;
    }

    .type-badge[data-type="number"] {
      background: rgba(139, 92, 246, 0.1);
      color: #8b5cf6;
    }

    .type-badge[data-type="boolean"] {
      background: rgba(34, 197, 94, 0.1);
      color: #22c55e;
    }

    .type-badge[data-type="secret"] {
      background: rgba(239, 68, 68, 0.1);
      color: #ef4444;
    }

    .col-description {
      color: var(--text-secondary);
      font-size: 0.9rem;
    }

    .col-actions {
      display: flex;
      gap: 0.5rem;
      justify-content: flex-end;
    }

    .empty-state {
      text-align: center;
      padding: 4rem 2rem;
      color: var(--text-secondary);
    }

    .empty-state svg {
      margin-bottom: 1.5rem;
      opacity: 0.3;
    }

    .empty-state h3 {
      font-size: 1.5rem;
      color: var(--text-primary);
      margin-bottom: 0.5rem;
    }

    .btn-primary {
      background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
      color: white;
      border: none;
      padding: 0.75rem 1.5rem;
      border-radius: 8px;
      font-weight: 600;
      cursor: pointer;
      display: flex;
      align-items: center;
      gap: 0.5rem;
      transition: all 0.2s;
    }

    .btn-primary:hover:not(:disabled) {
      transform: translateY(-2px);
      box-shadow: 0 4px 12px rgba(102, 126, 234, 0.4);
    }

    .btn-primary:disabled {
      opacity: 0.5;
      cursor: not-allowed;
    }

    .btn-secondary {
      background: var(--card-bg);
      color: var(--text-primary);
      border: 1px solid var(--border-color);
      padding: 0.75rem 1.5rem;
      border-radius: 8px;
      font-weight: 600;
      cursor: pointer;
      transition: all 0.2s;
    }

    .btn-secondary:hover {
      background: var(--hover-bg);
    }

    .btn-icon {
      background: transparent;
      border: none;
      padding: 0.5rem;
      cursor: pointer;
      border-radius: 6px;
      color: var(--text-secondary);
      display: flex;
      align-items: center;
      justify-content: center;
      transition: all 0.2s;
    }

    .btn-icon:hover {
      background: var(--hover-bg);
      color: var(--text-primary);
    }

    .btn-icon.btn-danger:hover {
      background: rgba(239, 68, 68, 0.1);
      color: #ef4444;
    }

    .modal {
      position: fixed;
      top: 0;
      left: 0;
      right: 0;
      bottom: 0;
      background: rgba(0, 0, 0, 0.6);
      display: flex;
      align-items: center;
      justify-content: center;
      z-index: 1000;
      backdrop-filter: blur(4px);
    }

    .modal-content {
      background: var(--card-bg);
      border-radius: 16px;
      width: 90%;
      max-width: 500px;
      box-shadow: 0 20px 60px rgba(0, 0, 0, 0.3);
      animation: modalSlideIn 0.3s ease;
    }

    @keyframes modalSlideIn {
      from {
        opacity: 0;
        transform: translateY(-20px);
      }
      to {
        opacity: 1;
        transform: translateY(0);
      }
    }

    .modal-header {
      display: flex;
      justify-content: space-between;
      align-items: center;
      padding: 1.5rem;
      border-bottom: 1px solid var(--border-color);
    }

    .modal-header h2 {
      margin: 0;
      font-size: 1.5rem;
      color: var(--text-primary);
    }

    .btn-close {
      background: transparent;
      border: none;
      font-size: 2rem;
      cursor: pointer;
      color: var(--text-secondary);
      line-height: 1;
      padding: 0;
      width: 32px;
      height: 32px;
      display: flex;
      align-items: center;
      justify-content: center;
      border-radius: 6px;
      transition: all 0.2s;
    }

    .btn-close:hover {
      background: var(--hover-bg);
      color: var(--text-primary);
    }

    .modal-body {
      padding: 1.5rem;
    }

    .modal-footer {
      display: flex;
      justify-content: flex-end;
      gap: 1rem;
      padding: 1.5rem;
      border-top: 1px solid var(--border-color);
    }

    .form-group {
      margin-bottom: 1.5rem;
    }

    .form-group:last-child {
      margin-bottom: 0;
    }

    .form-group label {
      display: block;
      margin-bottom: 0.5rem;
      font-weight: 600;
      color: var(--text-primary);
    }

    .input,
    .select,
    .textarea {
      width: 100%;
      padding: 0.75rem;
      border: 1px solid var(--border-color);
      border-radius: 8px;
      background: var(--input-bg);
      color: var(--text-primary);
      font-size: 1rem;
      font-family: inherit;
      transition: all 0.2s;
    }

    .input:focus,
    .select:focus,
    .textarea:focus {
      outline: none;
      border-color: #667eea;
      box-shadow: 0 0 0 3px rgba(102, 126, 234, 0.1);
    }

    .textarea {
      resize: vertical;
    }

    @media (max-width: 1024px) {
      .table-header,
      .table-row {
        grid-template-columns: 150px 1fr 100px 150px 80px;
        font-size: 0.875rem;
      }
    }

    @media (max-width: 768px) {
      .container {
        padding: 2rem 1rem;
      }

      .page-header {
        flex-direction: column;
        gap: 1rem;
      }

      .variables-table {
        overflow-x: auto;
      }

      .table-header,
      .table-row {
        min-width: 700px;
      }
    }
  `]
})
export class VariablesComponent {
  variables: Variable[] = [
    {
      id: '1',
      key: 'API_KEY',
      value: 'sk_test_123456789',
      description: 'Chave de API para produção',
      type: 'secret',
      createdAt: new Date()
    },
    {
      id: '2',
      key: 'DATABASE_URL',
      value: 'postgresql://localhost:5432/mydb',
      description: 'URL de conexão com o banco',
      type: 'string',
      createdAt: new Date()
    },
    {
      id: '3',
      key: 'MAX_CONNECTIONS',
      value: '100',
      description: 'Número máximo de conexões',
      type: 'number',
      createdAt: new Date()
    }
  ];

  showModal = false;
  editingVariable: Variable | null = null;

  formData = {
    key: '',
    value: '',
    type: 'string' as 'string' | 'number' | 'boolean' | 'secret',
    description: ''
  };

  openModal(): void {
    this.editingVariable = null;
    this.formData = {
      key: '',
      value: '',
      type: 'string',
      description: ''
    };
    this.showModal = true;
  }

  editVariable(variable: Variable): void {
    this.editingVariable = variable;
    this.formData = {
      key: variable.key,
      value: variable.value,
      type: variable.type,
      description: variable.description
    };
    this.showModal = true;
  }

  closeModal(): void {
    this.showModal = false;
    this.editingVariable = null;
  }

  saveVariable(): void {
    if (!this.formData.key.trim() || !this.formData.value.trim()) return;

    if (this.editingVariable) {
      const index = this.variables.findIndex(v => v.id === this.editingVariable!.id);
      if (index !== -1) {
        this.variables[index] = {
          ...this.editingVariable,
          ...this.formData
        };
      }
    } else {
      const newVariable: Variable = {
        id: Date.now().toString(),
        ...this.formData,
        createdAt: new Date()
      };
      this.variables.push(newVariable);
    }

    this.closeModal();
  }

  deleteVariable(variable: Variable): void {
    if (confirm(`Deseja realmente excluir a variável ${variable.key}?`)) {
      this.variables = this.variables.filter(v => v.id !== variable.id);
    }
  }
}