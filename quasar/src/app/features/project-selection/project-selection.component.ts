import { Component, OnInit, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { Router } from '@angular/router';
import { Observable } from 'rxjs';
import { ProjectService } from '../../core/services/project.service';
import { Project } from '../../core/models/project.model';

interface ProjectFormData {
  id: string;
  projectName: string;
  concurrency: number;
  jobs: string[];
  connections: Project['connections'];
  variables: Project['variables'];
  sourceDatabase: {
    type: string;
    host: string;
    port: number;
    user: string;
    password: string;
    database: string;
  };
  destinationDatabase: {
    type: string;
    host: string;
    port: number;
    user: string;
    password: string;
    database: string;
  };
}

@Component({
  selector: 'app-project-selection',
  standalone: true,
  imports: [CommonModule, FormsModule],
  template: `
    <div class="container" (click)="closeMenus()">
      <div class="header-section">
        <div>
          <p class="kicker">Workspace</p>
          <h1 class="title">Projetos</h1>
          <p class="subtitle">Selecione ou crie um projeto para continuar</p>
        </div>
        <div class="header-actions">
          <button class="btn-secondary" (click)="openImportModal()">
            <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" />
              <polyline points="7 10 12 15 17 10" />
              <line x1="12" y1="15" x2="12" y2="3" />
            </svg>
            Importar
          </button>
          <button class="btn-primary" (click)="openCreateModal()">
            <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <line x1="12" y1="5" x2="12" y2="19" />
              <line x1="5" y1="12" x2="19" y2="12" />
            </svg>
            Novo Projeto
          </button>
        </div>
      </div>

      <ng-container *ngIf="projects$ | async as projects">
      <div class="projects-grid" *ngIf="projects.length > 0">
          <div class="project-card" *ngFor="let project of projects" (click)="selectProject(project)">
          <div class="card-header">
            <div>
              <h3 class="project-title">{{ project.projectName || project.name || 'Sem nome' }}</h3>
              <p class="project-description">{{ formatDbSummary(project) }}</p>
            </div>
            <div class="card-actions" (click)="$event.stopPropagation()">
              <button class="btn-icon" (click)="toggleMenu(project, $event)" title="Ações">
                <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                  <circle cx="12" cy="5" r="1.5" />
                  <circle cx="12" cy="12" r="1.5" />
                  <circle cx="12" cy="19" r="1.5" />
                </svg>
              </button>
              <div class="actions-menu" *ngIf="openMenuId === project.id" (click)="$event.stopPropagation()">
                <button class="menu-item" (click)="exportProject(project)">
                  <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                    <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" />
                    <polyline points="17 8 12 3 7 8" />
                    <line x1="12" y1="3" x2="12" y2="15" />
                  </svg>
                  <span>Exportar</span>
                </button>
                <button class="menu-item" (click)="openDuplicateModal(project)">
                  <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                    <rect x="9" y="9" width="13" height="13" rx="2" ry="2" />
                    <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1" />
                  </svg>
                  <span>Duplicar</span>
                </button>
                <button class="menu-item" (click)="openEditModal(project)">
                  <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                    <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7" />
                    <path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z" />
                  </svg>
                  <span>Editar</span>
                </button>
                <button class="menu-item danger" (click)="openDeleteModal(project)">
                  <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                    <polyline points="3 6 5 6 21 6" />
                    <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2" />
                  </svg>
                  <span>Excluir</span>
                </button>
              </div>
            </div>
          </div>

          <div class="project-footer">
            <div class="project-meta">
              <span class="meta-pill">Origem: {{ project.sourceDatabase?.type || '-' }}</span>
              <span class="meta-pill">Destino: {{ project.destinationDatabase?.type || '-' }}</span>
              <span class="meta-pill">Concorrencia: {{ project.concurrency || 1 }}</span>
              <span class="meta-pill">Jobs: {{ project.jobs?.length || 0 }}</span>
            </div>
            <span class="project-date" *ngIf="formatDate(project.updatedAt) as updatedLabel">{{ updatedLabel }}</span>
          </div>
        </div>
      </div>

      <div class="empty-state" *ngIf="projects.length === 0">
        <svg xmlns="http://www.w3.org/2000/svg" width="64" height="64" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1" stroke-linecap="round" stroke-linejoin="round">
          <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z" />
        </svg>
        <h3>Nenhum projeto criado</h3>
        <p>Crie seu primeiro projeto para comecar</p>
      </div>
      </ng-container>

      <div class="modal" *ngIf="showModal" (click)="closeModal()">
        <div class="modal-content modal-large" (click)="$event.stopPropagation()">
          <div class="modal-header">
            <h2>{{ editingProject ? 'Editar Projeto' : 'Novo Projeto' }}</h2>
            <button class="btn-close" (click)="closeModal()">×</button>
          </div>
          <div class="modal-body">
            <div class="form-group">
              <label>Nome do Projeto</label>
              <input type="text" [(ngModel)]="formData.projectName" placeholder="Digite o nome do projeto" class="input" />
            </div>
            <div class="form-group">
              <label>Concorrencia</label>
              <input type="number" min="1" [(ngModel)]="formData.concurrency" class="input" />
            </div>

            <div class="form-section">
              <h3>Banco de Origem</h3>
              <div class="form-grid">
                <div class="form-group">
                  <label>Tipo</label>
                  <select [(ngModel)]="formData.sourceDatabase.type" class="input">
                    <option value="postgres">Postgres</option>
                    <option value="sqlserver">SQLServer</option>
                    <option value="mysql">MySQL</option>
                    <option value="access">Access</option>
                  </select>
                </div>
                <div class="form-group">
                  <label>Host</label>
                  <input type="text" [(ngModel)]="formData.sourceDatabase.host" class="input" placeholder="localhost" />
                </div>
                <div class="form-group">
                  <label>Porta</label>
                  <input type="number" [(ngModel)]="formData.sourceDatabase.port" class="input" placeholder="5432" />
                </div>
                <div class="form-group">
                  <label>Usuario</label>
                  <input type="text" [(ngModel)]="formData.sourceDatabase.user" class="input" />
                </div>
                <div class="form-group">
                  <label>Senha</label>
                  <input type="password" [(ngModel)]="formData.sourceDatabase.password" class="input" />
                </div>
                <div class="form-group">
                  <label>Database</label>
                  <input type="text" [(ngModel)]="formData.sourceDatabase.database" class="input" />
                </div>
              </div>
            </div>

            <div class="form-section">
              <h3>Banco de Destino</h3>
              <div class="form-grid">
                <div class="form-group">
                  <label>Tipo</label>
                  <select [(ngModel)]="formData.destinationDatabase.type" class="input">
                    <option value="postgres">Postgres</option>
                    <option value="sqlserver">SQLServer</option>
                    <option value="mysql">MySQL</option>
                    <option value="access">Access</option>
                  </select>
                </div>
                <div class="form-group">
                  <label>Host</label>
                  <input type="text" [(ngModel)]="formData.destinationDatabase.host" class="input" placeholder="localhost" />
                </div>
                <div class="form-group">
                  <label>Porta</label>
                  <input type="number" [(ngModel)]="formData.destinationDatabase.port" class="input" placeholder="5432" />
                </div>
                <div class="form-group">
                  <label>Usuario</label>
                  <input type="text" [(ngModel)]="formData.destinationDatabase.user" class="input" />
                </div>
                <div class="form-group">
                  <label>Senha</label>
                  <input type="password" [(ngModel)]="formData.destinationDatabase.password" class="input" />
                </div>
                <div class="form-group">
                  <label>Database</label>
                  <input type="text" [(ngModel)]="formData.destinationDatabase.database" class="input" />
                </div>
              </div>
            </div>
          </div>
          <div class="modal-footer">
            <button class="btn-secondary" (click)="closeModal()">Cancelar</button>
            <button class="btn-primary" (click)="saveProject()" [disabled]="!formData.projectName.trim()">
              {{ editingProject ? 'Salvar' : 'Criar' }}
            </button>
          </div>
        </div>
      </div>

      <div class="modal" *ngIf="showDeleteModal" (click)="closeDeleteModal()">
        <div class="modal-content modal-small" (click)="$event.stopPropagation()">
          <div class="modal-header">
            <h2>Confirmar Exclusao</h2>
            <button class="btn-close" (click)="closeDeleteModal()">×</button>
          </div>
          <div class="modal-body">
            <p>Tem certeza que deseja excluir o projeto <strong>{{ projectToDelete?.name }}</strong>?</p>
            <p class="warning-text">Esta acao nao pode ser desfeita.</p>
          </div>
          <div class="modal-footer">
            <button class="btn-secondary" (click)="closeDeleteModal()">Cancelar</button>
            <button class="btn-danger" (click)="confirmDelete()">Excluir</button>
          </div>
        </div>
      </div>

      <div class="modal" *ngIf="showDuplicateModal" (click)="closeDuplicateModal()">
        <div class="modal-content modal-small" (click)="$event.stopPropagation()">
          <div class="modal-header">
            <h2>Duplicar Projeto</h2>
            <button class="btn-close" (click)="closeDuplicateModal()">×</button>
          </div>
          <div class="modal-body">
            <p>Deseja duplicar o projeto <strong>{{ projectToDuplicate?.projectName || projectToDuplicate?.name }}</strong>?</p>
            <p class="warning-text">Uma copia sera criada automaticamente.</p>
          </div>
          <div class="modal-footer">
            <button class="btn-secondary" (click)="closeDuplicateModal()">Cancelar</button>
            <button class="btn-primary" (click)="confirmDuplicate()">Duplicar</button>
          </div>
        </div>
      </div>

      <div class="modal" *ngIf="showImportModal" (click)="closeImportModal()">
        <div class="modal-content modal-small" (click)="$event.stopPropagation()">
          <div class="modal-header">
            <h2>Importar Projeto</h2>
            <button class="btn-close" (click)="closeImportModal()">x</button>
          </div>
          <div class="modal-body">
            <div class="form-group">
              <label>Arquivo ZIP</label>
              <input type="file" class="input" accept=".zip" (change)="handleImportFile($event)" />
              <small class="helper-text">Aceita arquivo .zip exportado pelo sistema.</small>
            </div>
            <div class="form-group">
              <label>Novo nome (opcional)</label>
              <input type="text" [(ngModel)]="importProjectName" class="input" placeholder="Digite o novo nome" />
            </div>
          </div>
          <div class="modal-footer">
            <button class="btn-secondary" (click)="closeImportModal()">Cancelar</button>
            <button class="btn-primary" (click)="confirmImport()" [disabled]="!importFile">Importar</button>
          </div>
        </div>
      </div>
    </div>
  `,
  styles: [
    `
      .container {
        max-width: 1400px;
        margin: 0 auto;
        padding: 3rem 2rem;
      }

      .header-section {
        display: flex;
        justify-content: space-between;
        align-items: center;
        margin-bottom: 2.5rem;
        gap: 2rem;
      }

      .header-actions {
        display: flex;
        gap: 0.75rem;
        flex-wrap: wrap;
      }

      .kicker {
        text-transform: uppercase;
        letter-spacing: 0.24em;
        color: var(--text-muted);
        font-size: 0.7rem;
        font-weight: 600;
        margin-bottom: 0.5rem;
      }

      .title {
        font-size: 2.4rem;
        font-weight: 700;
        color: var(--text-primary);
        margin: 0 0 0.5rem 0;
      }

      .subtitle {
        color: var(--text-secondary);
        margin: 0;
      }

      .projects-grid {
        display: grid;
        grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
        gap: 1.5rem;
      }

      .project-card {
        background: var(--card-bg);
        border: 1px solid var(--border-color);
        border-radius: 16px;
        padding: 1.5rem;
        cursor: pointer;
        transition: all 0.3s ease;
        display: flex;
        flex-direction: column;
        gap: 1rem;
        box-shadow: var(--shadow-sm);
      }

      .project-card:hover {
        transform: translateY(-4px);
        box-shadow: 0 16px 30px rgba(15, 23, 42, 0.12);
        border-color: rgba(34, 211, 238, 0.4);
      }

      .card-header {
        display: flex;
        justify-content: space-between;
        align-items: flex-start;
        gap: 1rem;
      }

      .project-title {
        font-size: 1.25rem;
        font-weight: 600;
        color: var(--text-primary);
        margin: 0 0 0.25rem 0;
      }

      .project-description {
        color: var(--text-secondary);
        line-height: 1.6;
        margin: 0;
      }

      .card-actions {
        display: flex;
        gap: 0.5rem;
        opacity: 0.7;
        transition: opacity 0.2s;
        position: relative;
      }

      .project-card:hover .card-actions {
        opacity: 1;
      }

      .project-footer {
        display: flex;
        justify-content: space-between;
        align-items: center;
        padding-top: 1rem;
        border-top: 1px solid var(--border-color);
        gap: 1rem;
      }

      .project-meta {
        display: flex;
        flex-wrap: wrap;
        gap: 0.5rem;
      }

      .meta-pill {
        background: var(--hover-bg);
        padding: 0.3rem 0.6rem;
        border-radius: 999px;
        font-size: 0.75rem;
        font-weight: 600;
        color: var(--text-secondary);
      }

      .project-date {
        font-size: 0.875rem;
        color: var(--text-secondary);
      }

      .btn-primary,
      .btn-secondary,
      .btn-danger,
      .btn-icon,
      .btn-close {
        border: none;
        border-radius: 10px;
        font-weight: 600;
        cursor: pointer;
        transition: all 0.2s ease;
        display: inline-flex;
        align-items: center;
        gap: 0.5rem;
      }

      .btn-primary {
        background: linear-gradient(135deg, var(--accent-strong), var(--accent));
        color: white;
        padding: 0.75rem 1.25rem;
        box-shadow: 0 8px 20px rgba(34, 211, 238, 0.25);
      }

      .btn-primary:hover {
        transform: translateY(-1px);
        box-shadow: 0 12px 24px rgba(34, 211, 238, 0.3);
      }

      .btn-secondary {
        background: var(--hover-bg);
        color: var(--text-primary);
        padding: 0.65rem 1.1rem;
      }

      .btn-danger {
        background: rgba(239, 68, 68, 0.15);
        color: var(--error-color);
        padding: 0.65rem 1.1rem;
      }

      .btn-icon {
        background: var(--hover-bg);
        padding: 0.45rem;
        color: var(--text-secondary);
      }

      .btn-icon:hover {
        color: var(--text-primary);
        background: var(--active-bg);
      }

      .btn-icon.btn-danger {
        background: rgba(239, 68, 68, 0.12);
        color: var(--error-color);
      }

      .actions-menu {
        position: absolute;
        right: 0;
        top: 2.6rem;
        background: var(--card-bg);
        border: 1px solid var(--border-color);
        border-radius: 12px;
        padding: 0.4rem;
        min-width: 160px;
        display: flex;
        flex-direction: column;
        gap: 0.15rem;
        box-shadow: 0 18px 30px rgba(15, 23, 42, 0.2);
        z-index: 50;
      }

      .menu-item {
        background: transparent;
        border: none;
        color: var(--text-primary);
        text-align: left;
        padding: 0.55rem 0.75rem;
        border-radius: 10px;
        cursor: pointer;
        font-weight: 600;
        display: flex;
        align-items: center;
        gap: 0.6rem;
      }

      .menu-item:hover {
        background: var(--hover-bg);
      }

      .menu-item.danger {
        color: var(--error-color);
      }

      .menu-item svg {
        opacity: 0.9;
      }

      .btn-close {
        background: transparent;
        font-size: 1.5rem;
        padding: 0.25rem 0.5rem;
        color: var(--text-secondary);
      }

      .modal {
        position: fixed;
        inset: 0;
        background: rgba(15, 23, 42, 0.45);
        display: flex;
        align-items: center;
        justify-content: center;
        z-index: 200;
        padding: 1.5rem;
        backdrop-filter: blur(6px);
      }

      .modal-content {
        background: var(--card-bg);
        border-radius: 16px;
        padding: 1.5rem;
        width: 100%;
        max-width: 640px;
        border: 1px solid var(--border-color);
        box-shadow: 0 20px 40px rgba(15, 23, 42, 0.25);
      }

      .modal-small {
        max-width: 420px;
      }

      .modal-large {
        max-width: 900px;
      }

      .modal-header {
        display: flex;
        align-items: center;
        justify-content: space-between;
        margin-bottom: 1rem;
      }

      .modal-body {
        display: flex;
        flex-direction: column;
        gap: 1rem;
      }

      .modal-footer {
        display: flex;
        justify-content: flex-end;
        gap: 0.75rem;
        margin-top: 1.5rem;
      }

      .form-group {
        display: flex;
        flex-direction: column;
        gap: 0.5rem;
      }

      .input,
      .textarea {
        background: var(--input-bg);
        border: 1px solid var(--border-color);
        border-radius: 10px;
        padding: 0.65rem 0.85rem;
        color: var(--text-primary);
        font-family: var(--font-sans);
      }

      .input:focus,
      .textarea:focus {
        outline: 2px solid rgba(34, 211, 238, 0.3);
        border-color: rgba(34, 211, 238, 0.5);
      }

      .warning-text {
        color: var(--error-color);
        font-weight: 600;
      }

      .helper-text {
        font-size: 0.8rem;
        color: var(--text-secondary);
      }

      .empty-state {
        text-align: center;
        padding: 4rem 2rem;
        color: var(--text-secondary);
      }

      .form-section {
        margin-top: 1.5rem;
        padding-top: 1.5rem;
        border-top: 1px solid var(--border-color);
      }

      .form-section h3 {
        margin: 0 0 1rem 0;
        font-size: 1.1rem;
        color: var(--text-primary);
      }

      .form-grid {
        display: grid;
        grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
        gap: 1rem;
      }

      @media (max-width: 768px) {
        .container {
          padding: 2rem 1rem;
        }

        .header-section {
          flex-direction: column;
          align-items: flex-start;
        }

        .header-actions {
          width: 100%;
        }

        .form-grid {
          grid-template-columns: 1fr;
        }
      }
    `
  ]
})
export class ProjectSelectionComponent implements OnInit {
  private projectService = inject(ProjectService);
  private router = inject(Router);

  projects$: Observable<Project[]> = this.projectService.projects$;
  showModal = false;
  showDeleteModal = false;
  showDuplicateModal = false;
  showImportModal = false;
  editingProject: Project | null = null;
  projectToDelete: Project | null = null;
  projectToDuplicate: Project | null = null;
  openMenuId: string | null = null;
  importFile: File | null = null;
  importProjectName = '';

  formData: ProjectFormData = this.getDefaultFormData();

  ngOnInit(): void {
    this.refreshProjects();
  }

  refreshProjects(): void {
    this.projectService.listProjects().subscribe({
      error: (error: any) => {
        console.error('Erro ao carregar projetos:', error);
      }
    });
  }

  selectProject(project: Project): void {
    this.openMenuId = null;
    this.projectService.selectProject(project);
    this.router.navigate(['/home']);
  }

  toggleMenu(project: Project, event: Event): void {
    event.stopPropagation();
    this.openMenuId = this.openMenuId === project.id ? null : project.id;
  }

  closeMenus(): void {
    this.openMenuId = null;
  }

  openCreateModal(): void {
    this.editingProject = null;
    this.formData = this.getDefaultFormData();
    this.showModal = true;
  }

  openEditModal(project: Project): void {
    this.closeMenus();
    this.editingProject = project;
    this.formData = {
      id: project.id,
      projectName: project.projectName || project.name || '',
      concurrency: project.concurrency || 1,
      jobs: project.jobs || [],
      connections: project.connections || [],
      variables: project.variables || [],
      sourceDatabase: {
        type: project.sourceDatabase?.type || '',
        host: project.sourceDatabase?.host || '',
        port: project.sourceDatabase?.port || 5432,
        user: project.sourceDatabase?.user || '',
        password: project.sourceDatabase?.password || '',
        database: project.sourceDatabase?.database || ''
      },
      destinationDatabase: {
        type: project.destinationDatabase?.type || '',
        host: project.destinationDatabase?.host || '',
        port: project.destinationDatabase?.port || 5432,
        user: project.destinationDatabase?.user || '',
        password: project.destinationDatabase?.password || '',
        database: project.destinationDatabase?.database || ''
      }
    };
    this.showModal = true;
  }

  closeModal(): void {
    this.showModal = false;
    this.editingProject = null;
  }

  saveProject(): void {
    if (!this.formData.projectName.trim()) return;
    const isEdit = !!this.editingProject;
    const editId = this.editingProject?.id;
    const payload = {
      ...this.formData,
      name: this.formData.projectName,
      projectName: this.formData.projectName
    } as any;

    this.closeModal();

    if (isEdit && editId) {
      const result = this.projectService.updateProject(editId, payload);

      if (result && typeof (result as any).subscribe === 'function') {
        (result as Observable<Project>).subscribe({
          next: () => {
            this.refreshProjects();
          },
          error: (error: any) => {
            console.error('Erro ao atualizar projeto:', error);
          }
        });
      } else {
        this.refreshProjects();
      }
    } else {
      const result = this.projectService.createProject(payload);

      if (result && typeof (result as any).subscribe === 'function') {
        (result as Observable<Project>).subscribe({
          next: () => {
            this.refreshProjects();
          },
          error: (error: any) => {
            console.error('Erro ao criar projeto:', error);
          }
        });
      } else {
        this.refreshProjects();
      }
    }
  }

  openDeleteModal(project: Project): void {
    this.closeMenus();
    this.projectToDelete = project;
    this.showDeleteModal = true;
  }

  closeDeleteModal(): void {
    this.showDeleteModal = false;
    this.projectToDelete = null;
  }

  openDuplicateModal(project: Project): void {
    this.closeMenus();
    this.projectToDuplicate = project;
    this.showDuplicateModal = true;
  }

  closeDuplicateModal(): void {
    this.showDuplicateModal = false;
    this.projectToDuplicate = null;
  }

  confirmDelete(): void {
    if (!this.projectToDelete) return;

    const result = this.projectService.deleteProject(this.projectToDelete.id);

    if (result && typeof (result as any).subscribe === 'function') {
      (result as Observable<any>).subscribe({
        next: () => {
          this.refreshProjects();
          this.closeDeleteModal();
        },
        error: (error: any) => {
          console.error('Erro ao excluir projeto:', error);
        }
      });
    } else {
        this.refreshProjects();
      this.closeDeleteModal();
    }
  }

  confirmDuplicate(): void {
    if (!this.projectToDuplicate) return;

    const result = this.projectService.duplicateProject(this.projectToDuplicate.id);

    if (result && typeof (result as any).subscribe === 'function') {
      (result as Observable<Project>).subscribe({
        next: () => {
          this.refreshProjects();
          this.closeDuplicateModal();
        },
        error: (error: any) => {
          console.error('Erro ao duplicar projeto:', error);
        }
      });
    } else {
      this.refreshProjects();
      this.closeDuplicateModal();
    }
  }

  openImportModal(): void {
    this.importFile = null;
    this.importProjectName = '';
    this.showImportModal = true;
  }

  closeImportModal(): void {
    this.showImportModal = false;
    this.importFile = null;
    this.importProjectName = '';
  }

  handleImportFile(event: Event): void {
    const input = event.target as HTMLInputElement;
    this.importFile = input.files && input.files.length > 0 ? input.files[0] : null;
  }

  confirmImport(): void {
    if (!this.importFile) return;
    const projectName = this.importProjectName.trim();
    const result = this.projectService.importProject(this.importFile, projectName ? projectName : undefined);

    if (result && typeof (result as any).subscribe === 'function') {
      (result as Observable<Project | null>).subscribe({
        next: () => {
          this.refreshProjects();
          this.closeImportModal();
        },
        error: (error: any) => {
          console.error('Erro ao importar projeto:', error);
        }
      });
    }
  }

  exportProject(project: Project): void {
    this.closeMenus();
    const result = this.projectService.exportProject(project.id);
    if (!result) return;

    result.subscribe({
      next: (blob) => {
        const filename = `project_${project.id}.zip`;
        const url = window.URL.createObjectURL(blob);
        const anchor = document.createElement('a');
        anchor.href = url;
        anchor.download = filename;
        anchor.click();
        window.URL.revokeObjectURL(url);
      },
      error: (error: any) => {
        console.error('Erro ao exportar projeto:', error);
      }
    });
  }

  formatDate(date: Date): string {
    if (!date) return '';
    const parsed = new Date(date);
    if (Number.isNaN(parsed.getTime())) return '';
    return `Atualizado em ${parsed.toLocaleDateString('pt-BR', {
      day: '2-digit',
      month: '2-digit',
      year: 'numeric'
    })}`;
  }

  private getDefaultFormData(): ProjectFormData {
    return {
      id: '',
      projectName: '',
      concurrency: 1,
      jobs: [],
      connections: [],
      variables: [],
      sourceDatabase: {
        type: '',
        host: '',
        port: 5432,
        user: '',
        password: '',
        database: ''
      },
      destinationDatabase: {
        type: '',
        host: '',
        port: 5432,
        user: '',
        password: '',
        database: ''
      }
    };
  }

  formatDbSummary(project: Project): string {
    const source = project.sourceDatabase;
    const dest = project.destinationDatabase;
    if (!source && !dest) {
      return 'Sem configuracao de banco';
    }
    const sourceLabel = source
      ? `${source.type || 'db'} ${source.host || '-'}:${source.port || '-'}`
      : 'origem -';
    const destLabel = dest
      ? `${dest.type || 'db'} ${dest.host || '-'}:${dest.port || '-'}`
      : 'destino -';
    return `${sourceLabel} → ${destLabel}`;
  }
}
