import { CdkContextMenuTrigger, CdkMenuModule } from '@angular/cdk/menu';
import { isPlatformBrowser } from '@angular/common';
import {
  AfterViewInit,
  Component,
  ElementRef,
  Inject,
  Input,
  PLATFORM_ID,
  QueryList,
  ViewChild,
  ViewChildren,
} from '@angular/core';
import { MatButtonModule } from '@angular/material/button';
import { MatDialog } from '@angular/material/dialog';
import { MatIconModule } from '@angular/material/icon';
import { MatTooltipModule } from '@angular/material/tooltip';
import { of } from 'rxjs';
import { delay } from 'rxjs/operators';
import { v4 as uuidv4 } from 'uuid';
import { Job, JobService } from '../../../services/job.service';
import { Project, ProjectService } from '../../../services/project.service';
import { ConfirmDialogComponent } from '../../dialog-confirm/dialog-confirm';

@Component({
  selector: 'app-diagram',
  standalone: true,
  imports: [
    MatIconModule,
    MatButtonModule,
    CdkContextMenuTrigger,
    CdkMenuModule,
    MatTooltipModule
  ],
  templateUrl: './diagram.html',
  styleUrls: ['./diagram.scss'],
})
export class Diagram implements AfterViewInit {
  @Input() jobs: Job[] = [];
  @Input() project: Project | null = null;
  @ViewChildren('jobEl') jobElements!: QueryList<ElementRef>;
  @ViewChild('diagramContainer') containerRef!: ElementRef;

  selectedJob: Job | null = null;
  isLoading = false;
  isSaving = false;
  isBrowser: boolean;
  instance: any;
  gridX = 350;
  gridY = 100;

  constructor(
    @Inject(PLATFORM_ID) private platformId: any,
    private jobService: JobService,
    private projectService: ProjectService,
    private dialog: MatDialog
  ) {
    this.isBrowser = isPlatformBrowser(this.platformId);
  }

  async ngAfterViewInit() {
    if (!this.isBrowser) return;

    this.isLoading = true;

    await this.initJsPlumbOnce();

    // Espera 50ms para garantir que o DOM esteja atualizado
    setTimeout(() => {
      this.jobs.forEach((job) => this.addJobToJsPlumb(job));
      this.addExistingConnections();
    }, 50);

    this.jobElements.changes.subscribe(() => {
      // Adiciona apenas o último job novo
      const newJob = this.jobs[this.jobs.length - 1];
      if (newJob) this.addJobToJsPlumb(newJob);
    });
  }

  private jsPlumbInitialized = false;

  async initJsPlumbOnce(): Promise<void> {
    if (this.jsPlumbInitialized) return;
    this.jsPlumbInitialized = true;

    const jsPlumbModule = await import('jsplumb');
    const jsPlumb = jsPlumbModule.jsPlumb;

    this.instance = jsPlumb.getInstance();
    this.instance.setContainer('diagramContainer');

    // Evita conexões duplicadas ou loopback
    this.instance.bind('beforeDrop', (info: any) => {
      if (info.sourceId === info.targetId) return false;
      const existing = this.instance.getConnections({
        source: info.sourceId,
        target: info.targetId,
      });
      return existing.length === 0;
    });

    // Escutando o evento de nova conexão
      this.instance.bind('connection', (info: any) => {
        
        if (!this.isLoading) {
          this.project?.connections.push({
            source: info.sourceId, 
            target: info.targetId
          });
          this.saveProject();
        }

      });

      // Quando uma conexão é removida
    this.instance.bind('connectionDetached', (info: any) => {
      const index = this.project?.connections.findIndex(
        (conn) =>
          conn.source === info.sourceId && conn.target === info.targetId
      );
      if (index !== undefined && index >= 0) {
        this.project?.connections.splice(index, 1);
        this.saveProject();        
      }
    });
  }

  saveProject() {
    this.isSaving = true;
    this.projectService.updateProject(this.project!).subscribe({
      next: (updatedProject) => {
        console.log('Projeto atualizado com sucesso:', updatedProject);
        this.isSaved();
      },
      error: (error) => {
        console.error('Erro ao atualizar projeto:', error);
        this.isSaved();
      }
    });
  }

  addJobToJsPlumb(job: Job) {
    const id = job.id;
    if (!this.instance) return;

    this.instance.makeSource(id, {
      filter: '.handle',
      anchor: 'Continuous',
      connector: ['Flowchart', { stub: 10, gap: 5 }],
      endpoint: 'Dot',
      connectorOverlays: [['Arrow', { width: 10, length: 10, location: 1 }]],
      maxConnections: -1,
    });

    this.instance.makeTarget(id, {
      anchor: 'Continuous',
      allowLoopback: false,
      endpoint: 'Blank',
    });

    const gridX = 100;
    const gridY = 100;

    const snapToGrid = (x: number, y: number): [number, number] => {
      return [Math.round(x / gridX) * gridX, Math.round(y / gridY) * gridY];
    };

    this.instance.draggable(id, {
      stop: (params: any) => {

        this.isSaving = true;
        const el = params.el;
        const [x, y] = snapToGrid(
          parseInt(el.style.left || '0', 10),
          parseInt(el.style.top || '0', 10)
        );
        el.style.left = `${x}px`;
        el.style.top = `${y}px`;
        this.instance.revalidate(el);

        // Atualiza a posição do job e salva
        const movedJob = this.jobs.find(j => j.id === el.id);
        if (movedJob) {
          movedJob.left = x;
          movedJob.top = y;

          // Atualiza o job individualmente (se tiver esse endpoint)
          this.jobService.updateJob(this.project?.id || '', movedJob.id, movedJob).subscribe({
            next: (updatedJob) => {
              console.log('Job atualizado com sucesso:', updatedJob);
              this.isSaved();
            } ,
            error: (error) => {
              console.error('Erro ao atualizar job:', error); 
              this.isSaved();
            }
          });
        }
      },
    });
  }

  addExistingConnections() {
    if (!this.instance || !this.project?.connections) return;

    this.project.connections.forEach((conn) => {
      this.instance.connect({
        source: conn.source,
        target: conn.target,
        anchor: 'Continuous',
        connector: ['Flowchart', { stub: 10, gap: 5 }],
        overlays: [['Arrow', { width: 10, length: 10, location: 1 }]],
      });
    });
    this.isLoading = false;
  }

  addNewJob(): void {

    this.isSaving = true;

    const newJob: Job = {
      id: uuidv4(),
      jobName: 'Novo Job',
      selectSql: '',
      insertSql: '',
      recordsPerPage: 1000,
      concurrency: 1,
      top: -100,
      left: 350,
    };

    this.jobs.push(newJob);

    if (this.project) {
      this.project.jobs = this.jobs.map(job => `jobs/${job.id}.json`);
    }

    this.jobService.addJob(this.project?.id || '', newJob).subscribe({
      next: (job) => {
        this.projectService.updateProject(this.project!).subscribe({
          next: (updatedProject) => {
            console.log('Projeto atualizado com novo job:', updatedProject);
            this.isSaved();
          },
          error: (error) => {
            console.error('Erro ao atualizar projeto com novo job:', error);
            this.isSaved();
          }
        });
      }
    });
  }

  onRightClick(event: MouseEvent, job: Job) {
    event.preventDefault(); // impedir menu nativo do navegador
    this.selectedJob = job;
  }

  removeJob(job: Job | null) {

    if (!job || !this.project) return;

    const dialogRef = this.dialog.open(ConfirmDialogComponent, {
      minWidth: '30vw',
      minHeight: '20vh',
      data: {
        title: 'Remover Job',
        message: 'Tem certeza que deseja remover este Job?'
      }
    });

    dialogRef.afterClosed().subscribe(confirmed => {
      if (confirmed) {
        this.isSaving = true;
        this.jobService.deleteJob(this.project?.id || '', job.id).subscribe({
          next: () => {
            console.log('Job removido com sucesso:', job.id);
            this.jobs = this.jobs.filter(j => j.id !== job.id);
            this.project?.jobs.splice(this.project.jobs.indexOf(`jobs/${job.id}.json`), 1);
            this.instance.removeAllEndpoints(job.id);
            this.instance.remove(job.id);
            this.saveProject();
            this.isSaved();
          }, 
          error: (error) => {
            console.error('Erro ao remover job:', error);
            this.isSaved();
          }
        });

      }
    });
    
  }

  isSaved() {
    of(null).pipe(delay(500)).subscribe(() => {
      this.isSaving = false;
    });
  }

}
