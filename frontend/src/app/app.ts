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
import { RouterModule } from '@angular/router';
import { DialogProject } from './pages/project/dialog-project/dialog-project';
import { AppState } from './services/app-state';
import { ProjectService } from './services/project.service';

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
    MatOptionModule
  ],
  templateUrl: './app.html',
  styleUrl: './app.scss'
})

@Injectable({
  providedIn: 'root'
})
export class App {
  protected title = 'frontend';

    projects: any[] = [];
    selectedProjectId?: number;

  constructor(
    private projectservice: ProjectService,
    private appState: AppState,
    private dialog: MatDialog
  ) {}

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
    this.appState.projectID = event.value;  
    console.log(this.appState.projectID);
  }

  openEditDialog(job: any | null) {
    const dialogRef = this.dialog.open(DialogProject, {
      panelClass: 'custom-dialog-container',
      minWidth: '90vw',
      height: '90vh',
      data: { ...job } // passa uma cópia do job
    });


    dialogRef.afterClosed().subscribe((result) => {
      if (result) {
        // Aqui você pode salvar o resultado
        console.log('Configurações atualizadas:', result);
        // Pode chamar um service aqui para salvar no backend
      }
    });
  }

}
