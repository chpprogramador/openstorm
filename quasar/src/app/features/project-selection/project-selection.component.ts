// src/app/features/project-selection/project-selection.component.ts
import { CommonModule } from '@angular/common';
import { ChangeDetectorRef, Component, OnInit, inject } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { Router } from '@angular/router';
import { Observable } from 'rxjs';
import { Project } from '../../core/models/project.model';
import { ProjectService } from '../../core/services/project.service';


@Component({
  selector: 'app-project-selection',
  standalone: true,
  imports: [CommonModule, FormsModule],
  template: `
    <div class="container">
      <div class="header-section">
        <h1 class="title">Meus Projetos</h1>
        <button class="btn-primary" (click)="openCreateModal()">
          <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <line x1="12" y1="5" x2="12" y2="19"/>
            <line x1="5" y1="12" x2="19" y2="12"/>
          </svg>
          Novo Projeto
        </button>
      </div>

      <div class="projects-grid" *ngIf="projects.length > 0">
        <div class="project-card" *ngFor="let project of projects" (click)="selectProject(project)">
          <div class="card-header">
            <h3 class="project-title">{{ project.name }}</h3>
            <div class="card-actions" (click)="$event.stopPropagation()">
              <button class="btn-icon" (click)="openEditModal(project)" title="Editar">
                <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                  <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/>
                  <path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/>
                </svg>
              </button>
              <button class="btn-icon btn-danger" (click)="openDeleteModal(project)" title="Excluir">
                <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                  <polyline points="3 6 5 6 21 6"/>
                  <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/>
                </svg>
              </button>
            </div>
          </div>
          <p class="project-description">{{ project.projectName || 'Sem descrição' }}</p>
          <div class="project-footer">
            <span class="project-date">Atualizado em {{ formatDate(project.updatedAt) }}</span>
          </div>
        </div>
      </div>

      <div class="empty-state" *ngIf="projects.length === 0">
        <svg xmlns="http://www.w3.org/2000/svg" width="64" height="64" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1" stroke-linecap="round" stroke-linejoin="round">
          <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/>
        </svg>
        <h3>Nenhum projeto criado</h3>
        <p>Crie seu primeiro projeto para começar</p>
      </div>

      <!-- Modal Create/Edit -->
      <div class="modal" *ngIf="showModal" (click)="closeModal()">
        <div class="modal-content" (click)="$event.stopPropagation()">
          <div class="modal-header">
            <h2>{{ editingProject ? 'Editar Projeto' : 'Novo Projeto' }}</h2>
            <button class="btn-close" (click)="closeModal()">×</button>
          </div>
          <div class="modal-body">
            <div class="form-group">
              <label>Nome do Projeto</label>
              <input type="text" [(ngModel)]="formData.name" placeholder="Digite o nome do projeto" class="input">
            </div>
            <div class="form-group">
              <label>Descrição</label>
              <textarea [(ngModel)]="formData.description" placeholder="Digite uma descrição (opcional)" class="textarea" rows="4"></textarea>
            </div>
          </div>
          <div class="modal-footer">
            <button class="btn-secondary" (click)="closeModal()">Cancelar</button>
            <button class="btn-primary" (click)="saveProject()" [disabled]="!formData.name.trim()">
              {{ editingProject ? 'Salvar' : 'Criar' }}
            </button>
          </div>
        </div>
      </div>

      <!-- Modal Delete -->
      <div class="modal" *ngIf="showDeleteModal" (click)="closeDeleteModal()">
        <div class="modal-content modal-small" (click)="$event.stopPropagation()">
          <div class="modal-header">
            <h2>Confirmar Exclusão</h2>
            <button class="btn-close" (click)="closeDeleteModal()">×</button>
          </div>
          <div class="modal-body">
            <p>Tem certeza que deseja excluir o projeto <strong>{{ projectToDelete?.name }}</strong>?</p>
            <p class="warning-text">Esta ação não pode ser desfeita.</p>
          </div>
          <div class="modal-footer">
            <button class="btn-secondary" (click)="closeDeleteModal()">Cancelar</button>
            <button class="btn-danger" (click)="confirmDelete()">Excluir</button>
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

    .header-section {
      display: flex;
      justify-content: space-between;
      align-items: center;
      margin-bottom: 2.5rem;
    }

    .title {
      font-size: 2rem;
      font-weight: 700;
      color: var(--text-primary);
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
      border-radius: 12px;
      padding: 1.5rem;
      cursor: pointer;
      transition: all 0.3s ease;
      display: flex;
      flex-direction: column;
      gap: 1rem;
    }

    .project-card:hover {
      transform: translateY(-4px);
      box-shadow: 0 8px 24px rgba(102, 126, 234, 0.15);
      border-color: #667eea;
    }

    .card-header {
      display: flex;
      justify-content: space-between;
      align-items: flex-start;
    }

    .project-title {
      font-size: 1.25rem;
      font-weight: 600;
      color: var(--text-primary);
      margin: 0;
      flex: 1;
    }

    .card-actions {
      display: flex;
      gap: 0.5rem;
      opacity: 0.7;
      transition: opacity 0.2s;
    }

    .project-card:hover .card-actions {
      opacity: 1;
    }

    .project-description {
      color: var(--text-secondary);
      line-height: 1.6;
      margin: 0;
      flex: 1;
    }

    .project-footer {
      display: flex;
      justify-content: space-between;
      align-items: center;
      padding-top: 1rem;
      border-top: 1px solid var(--border-color);
    }

    .project-date {
      font-size: 0.875rem;
      color: var(--text-secondary);
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

    .btn-danger {
      background: #ef4444;
      color: white;
      border: none;
      padding: 0.75rem 1.5rem;
      border-radius: 8px;
      font-weight: 600;
      cursor: pointer;
      transition: all 0.2s;
    }

    .btn-danger:hover {
      background: #dc2626;
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

    .modal-small {
      max-width: 400px;
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
    .textarea:focus {
      outline: none;
      border-color: #667eea;
      box-shadow: 0 0 0 3px rgba(102, 126, 234, 0.1);
    }

    .textarea {
      resize: vertical;
      min-height: 100px;
    }

    .warning-text {
      color: #ef4444;
      font-size: 0.875rem;
      margin-top: 0.5rem;
    }

    @media (max-width: 768px) {
      .container {
        padding: 2rem 1rem;
      }

      .header-section {
        flex-direction: column;
        align-items: flex-start;
        gap: 1rem;
      }

      .projects-grid {
        grid-template-columns: 1fr;
      }

      .modal-content {
        width: 95%;
        margin: 1rem;
      }
    }
  `]
})
export class ProjectSelectionComponent implements OnInit {

  private projectService = inject(ProjectService);
  private router = inject(Router);
  private cdr = inject(ChangeDetectorRef);

  projects: Project[] = [];
  showModal = false;
  showDeleteModal = false;
  editingProject: Project | null = null;
  projectToDelete: Project | null = null;

  formData = {
    name: '',
    description: ''
  };

  ngOnInit(): void {
    console.log('Initializing ProjectSelectionComponent');
    this.loadProjects();
  }

  loadProjects(): void {
    console.log('🔄 Subscribing to project updates...');
    this.projectService.projects$.subscribe({
      next: (projects) => {
        console.log('🌐 Projects updated from Observable:', projects);
        this.projects = projects;
        this.cdr.detectChanges();
      },
      error: (error: any) => {
        console.error('❌ Error loading projects:', error);
      }
    });
  }

  selectProject(project: Project): void {
    this.projectService.selectProject(project);
    this.router.navigate(['/home']);
  }

  openCreateModal(): void {
    this.editingProject = null;
    this.formData = {
      name: '',
      description: ''
    };
    this.showModal = true;
  }

  openEditModal(project: Project): void {
    this.editingProject = project;
    this.formData = {
      name: project.name,
      description: project.description
    };
    this.showModal = true;
  }

  closeModal(): void {
    this.showModal = false;
    this.editingProject = null;
  }

  saveProject(): void {
    if (!this.formData.name.trim()) return;

    if (this.editingProject) {
      // Atualizar projeto
      const result = this.projectService.updateProject(this.editingProject.id, this.formData);

      if (result && typeof (result as any).subscribe === 'function') {
        // Modo API
        (result as Observable<Project>).subscribe({
          next: () => {
            this.loadProjects();
            this.closeModal();
          },
          error: (error: any) => {
            console.error('Erro ao atualizar projeto:', error);
          }
        });
      } else {
        // Modo localStorage
        this.loadProjects();
        this.closeModal();
      }
    } else {
      // Criar novo projeto
      const result = this.projectService.createProject(this.formData);

      if (result && typeof (result as any).subscribe === 'function') {
        // Modo API
        (result as Observable<Project>).subscribe({
          next: () => {
            this.loadProjects();
            this.closeModal();
          },
          error: (error: any) => {
            console.error('Erro ao criar projeto:', error);
          }
        });
      } else {
        // Modo localStorage
        this.loadProjects();
        this.closeModal();
      }
    }
  }

  openDeleteModal(project: Project): void {
    this.projectToDelete = project;
    this.showDeleteModal = true;
  }

  closeDeleteModal(): void {
    this.showDeleteModal = false;
    this.projectToDelete = null;
  }

  confirmDelete(): void {
    if (!this.projectToDelete) return;

    const result = this.projectService.deleteProject(this.projectToDelete.id);

    if (result && typeof (result as any).subscribe === 'function') {
      // Modo API
      (result as Observable<any>).subscribe({
        next: () => {
          this.loadProjects();
          this.closeDeleteModal();
        },
        error: (error: any) => {
          console.error('Erro ao excluir projeto:', error);
        }
      });
    } else {
      // Modo localStorage
      this.loadProjects();
      this.closeDeleteModal();
    }
  }

  formatDate(date: Date): string {
    return new Date(date).toLocaleDateString('pt-BR', {
      day: '2-digit',
      month: '2-digit',
      year: 'numeric'
    });
  }
}
