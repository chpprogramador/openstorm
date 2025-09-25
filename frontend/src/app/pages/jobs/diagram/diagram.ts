import { CdkContextMenuTrigger, CdkMenuModule } from '@angular/cdk/menu';
import { CommonModule, isPlatformBrowser } from '@angular/common';
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
import { MatProgressBarModule } from '@angular/material/progress-bar';
import { MatTooltipModule } from '@angular/material/tooltip';
import { of } from 'rxjs';
import { delay } from 'rxjs/operators';
import { v4 as uuidv4 } from 'uuid';
import { LogViewerComponent } from "../../../components/app-log-viewer.component";
import { JobExtended } from '../../../services/job-state.service';
import { Job, JobService } from '../../../services/job.service';
import { Project, ProjectService } from '../../../services/project.service';
import { ConfirmDialogComponent } from '../../dialog-confirm/dialog-confirm';
import { DialogJobs } from '../dialog-jobs/dialog-jobs';

@Component({
  selector: 'app-diagram',
  standalone: true,
  imports: [
    MatIconModule,
    MatButtonModule,
    CdkContextMenuTrigger,
    CdkMenuModule,
    MatTooltipModule,
    MatProgressBarModule,
    CommonModule,
    LogViewerComponent
],
  templateUrl: './diagram.html',
  styleUrls: ['./diagram.scss'],
})
export class Diagram implements AfterViewInit {

  @Input() jobs: JobExtended[] = [];
  @Input() project: Project | null = null;
  @Input() isRunning = false;
  @ViewChildren('jobEl') jobElements!: QueryList<ElementRef>;
  @ViewChild('diagramContainer') containerRef!: ElementRef;
  @ViewChild('scrollContainer', { static: true }) scrollContainer!: ElementRef;

  isDragging = false;
  startX = 0;
  startY = 0;
  scrollLeft = 0;
  scrollTop = 0;

  zoom = 1;
  minZoom = 0.3;
  maxZoom = 1.6;
  zoomStep = 0.1;

  selectedJob: JobExtended | null = null;
  isLoading = false;
  isSaving = false;
  isBrowser: boolean;
  instance: any;
  gridX = 350;
  gridY = 100;
  showLogs = false;

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

  setTimeout(() => {
    this.jobs.forEach((job) => this.addJobToJsPlumb(job));
    this.addExistingConnections();
  }, 50);

  const container = this.scrollContainer.nativeElement;
  const content = container.querySelector('.diagram-content') as HTMLElement;

  const storedZoom = localStorage.getItem('diagramZoom');
  if (storedZoom) {
    this.zoom = +storedZoom;
    this.instance.setZoom(this.zoom);
  }

  container.addEventListener(
  'wheel',
  (event: WheelEvent) => {
    if (event.ctrlKey) {
      event.preventDefault();

      this.zoom += event.deltaY < 0 ? this.zoomStep : -this.zoomStep;
      this.zoom = Math.min(Math.max(this.zoom, this.minZoom), this.maxZoom);

      if (this.instance) {
        this.instance.setZoom(this.zoom);
        this.instance.repaintEverything();  
        localStorage.setItem('diagramZoom', this.zoom.toString());
      }
    }
  },
  { passive: false }
);
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
    const gridY = 60;

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
    columns: [],
    recordsPerPage: 1000,
    type: 'insert',
    stopOnError: true,
    top: 10,
    left: 10,
  };

  this.jobs.push(newJob);

  if (this.project) {
    this.project.jobs = this.jobs.map(job => `jobs/${job.id}.json`);
  }

  // aguarda Angular renderizar o novo job no DOM
  setTimeout(() => {
    this.addJobToJsPlumb(newJob);
  }, 0);

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
    of(null).pipe(delay(200)).subscribe(() => {
      this.isSaving = false;
    });
  }

  openEditDialog(job: Job | null) {
    const dialogRef = this.dialog.open(DialogJobs, {
      panelClass: 'custom-dialog-container',
      minWidth: '90vw',
      minHeight: '90vh',
      data: job
    });


    dialogRef.afterClosed().subscribe((result) => {
      console.log('Dialog closed with result:', result);
      if (result) {
        console.log('Job salvo:', result);
        if (result.id) {

          this.jobService.updateJob(this.project?.id || '', result.id, result).subscribe({
            next: (updatedJob) => {
              const index = this.jobs.findIndex(j => j.id === updatedJob.id);
              if (index !== -1) {
                this.jobs[index] = updatedJob;
              }
              this.selectedJob = updatedJob;
              this.saveProject();
            },
            error: (error) => {
              console.error('Erro ao atualizar job:', error);
            }
          });

        }
      }
    });
  }

  runProject() {
    if (!this.project) return;

    this.isSaving = true;
    this.isRunning = true;

    this.jobs.forEach(job => {
      job.total = 0;
      job.processed = 0;
      job.progress = 0;
      job.status = 'pending';
      job.startedAt = '';
      job.endedAt = '';
      job.error = '';
    });


    this.projectService.runProject(this.project.id).subscribe({
      next: (response) => {
        console.log('Projeto executado com sucesso:', response);
        this.isSaved();
      },
      error: (error) => {
        console.error('Erro ao executar projeto:', error);
        this.isSaved();
      }
    });
  }

  stopProject() {
    this.isRunning = true;
  }

  showHideLogs() {
    this.showLogs = !this.showLogs;
  }

  

  onMouseDown(e: MouseEvent): void {
    this.isDragging = true;
    const container = this.scrollContainer.nativeElement;
    this.startX = e.pageX - container.offsetLeft;
    this.startY = e.pageY - container.offsetTop;
    this.scrollLeft = container.scrollLeft;
    this.scrollTop = container.scrollTop;
    container.style.cursor = 'grabbing';
  }

  onMouseMove(e: MouseEvent): void {
    if (!this.isDragging) return;

    e.preventDefault();
    const container = this.scrollContainer.nativeElement;
    const x = e.pageX - container.offsetLeft;
    const y = e.pageY - container.offsetTop;
    const walkX = x - this.startX;
    const walkY = y - this.startY;

    container.scrollLeft = this.scrollLeft - walkX;
    container.scrollTop = this.scrollTop - walkY;
  }

  onMouseUp(): void {
    this.isDragging = false;
    this.scrollContainer.nativeElement.style.cursor = 'grab';
  }

  removeZoom(){
    this.zoom = 1;
    if (this.instance) {
      this.instance.setZoom(this.zoom);
      this.instance.repaintEverything(); 
      localStorage.setItem('diagramZoom', this.zoom.toString());
    }
  }


  tempoTotal(dataIso: string): string {
  const agora = new Date();
  const data = new Date(dataIso);
  const diffMs = agora.getTime() - data.getTime(); // diferença em milissegundos

  const segundos = Math.floor(diffMs / 1000);
  const minutos = Math.floor(segundos / 60);
  const horas = Math.floor(minutos / 60);
  const dias = Math.floor(horas / 24);

  if (dias > 0) return `há ${dias} dia${dias > 1 ? 's' : ''}`;
  if (horas > 0) return `há ${horas} hora${horas > 1 ? 's' : ''}`;
  if (minutos > 0) return `há ${minutos} minuto${minutos > 1 ? 's' : ''}`;
  return `há ${segundos} segundo${segundos > 1 ? 's' : ''}`;
}

}
