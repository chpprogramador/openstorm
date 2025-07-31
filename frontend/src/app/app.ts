import { CommonModule, NgForOf } from '@angular/common';
import { HttpClientModule } from '@angular/common/http';
import { Component, Injectable } from '@angular/core';
import { MatButtonModule } from '@angular/material/button';
import { MatOptionModule } from '@angular/material/core';
import { MatDialog } from '@angular/material/dialog';
import { MatIconModule } from '@angular/material/icon';
import { MatListModule } from '@angular/material/list';
import { MatSelectModule } from '@angular/material/select';
import { MatSidenavModule } from '@angular/material/sidenav';
import { MatToolbarModule } from '@angular/material/toolbar';
import { MatTooltipModule } from '@angular/material/tooltip';
import { RouterModule } from '@angular/router';
import { ConfirmDialogComponent } from './pages/dialog-confirm/dialog-confirm';
import { DialogProject } from './pages/project/dialog-project/dialog-project';
import { AppState } from './services/app-state';
import { Project, ProjectService } from './services/project.service';
import { ThemeService } from './services/theme.service';

@Component({
  selector: 'app-root',
  standalone: true, 
  imports: [
    RouterModule,   
    MatToolbarModule,
    MatSidenavModule,
    MatButtonModule,
    MatIconModule,
    MatListModule,
    MatSelectModule,
    HttpClientModule,
    CommonModule,
    NgForOf, 
    MatOptionModule,
    MatTooltipModule
  ],
  templateUrl: './app.html',
  styleUrl: './app.scss'
})

@Injectable({
  providedIn: 'root'
})
export class App {
[x: string]: any;
  protected title = 'frontend';

    projects: any[] = [];
    selectedProject?: Project | null;

  constructor(
    private projectservice: ProjectService,
    private appState: AppState,
    private dialog: MatDialog,
    public themeService: ThemeService
  ) {
    // Inicializa o tema
    this.themeService.setTheme(this.themeService.getCurrentTheme());
  }

  toggleTheme(): void {
    this.themeService.toggleTheme();
  }

  ngOnInit() {
    this.projectservice.listProjects().subscribe({
      next: (projects) => {
        this.projects = projects;
        console.log('Projetos listados com sucesso:', this.projects);
      }
      , error: (error) => {
        console.error('Erro ao listar projetos:', error);
      }
    });
  }

   onSelectChange(event: any) {
    this.appState.project = event.value;  
  }

  openEditDialog(project: Project | null) {
    const dialogRef = this.dialog.open(DialogProject, {
      panelClass: 'custom-dialog-container',
      minWidth: '60vw',
      data: project 
    });


    dialogRef.afterClosed().subscribe((result) => {
      if (result) {
        console.log('Projeto salvo:', result);
        if (result.id) {
          this.projectservice.updateProject(result).subscribe({
            next: (updatedProject) => {
              const index = this.projects.findIndex(p => p.id === updatedProject.id);
              if (index !== -1) {
                this.projects[index] = updatedProject;
              }
              this.selectedProject = updatedProject;
            },
            error: (error) => {
              console.error('Erro ao atualizar projeto:', error);
            }
          });
        } else {
          this.projectservice.createProject(result).subscribe({
            next: (newProject) => {
              this.projects.push(newProject);
              this.appState.project = newProject; 
              this.selectedProject = newProject;
            },
            error: (error) => {
              console.error('Erro ao criar projeto:', error);
            }
          });
        }
      }
    });
  }

  removeProject(project: Project) {

    const dialogRef = this.dialog.open(ConfirmDialogComponent, {
      minWidth: '30vw',
      minHeight: '20vh',
      data: {
        title: 'Remover Projeto',
        message: 'Tem certeza que deseja remover este projeto?'
      }
    });

    dialogRef.afterClosed().subscribe(confirmed => {
      if (confirmed) {
        
        this.projectservice.deleteProject(project.id).subscribe({
          next: () => { 
            this.projects = this.projects.filter(p => p.id !== project.id);
            if (this.appState.project?.id === project.id) {
              this.appState.project = null; 
            }
            this.selectedProject = null;
          }
        });

      }
    });
    
  }

}
