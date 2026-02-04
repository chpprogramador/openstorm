import { CdkContextMenuTrigger, CdkMenuModule } from '@angular/cdk/menu';
import { CommonModule, isPlatformBrowser } from '@angular/common';
import {
  AfterViewInit,
  Component,
  ElementRef,
  HostListener,
  Inject,
  Input,
  OnChanges,
  OnDestroy,
  PLATFORM_ID,
  QueryList,
  SimpleChanges,
  ViewChild,
  ViewChildren
} from '@angular/core';
import { FormsModule } from '@angular/forms';
import { MatButtonModule } from '@angular/material/button';
import { MatDialog } from '@angular/material/dialog';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatIconModule } from '@angular/material/icon';
import { MatInputModule } from '@angular/material/input';
import { MatOptionModule } from '@angular/material/core';
import { MatProgressBarModule } from '@angular/material/progress-bar';
import { MatSelectModule } from '@angular/material/select';
import { MatSnackBar, MatSnackBarModule } from '@angular/material/snack-bar';
import { MatTooltipModule } from '@angular/material/tooltip';
import { of } from 'rxjs';
import { delay } from 'rxjs/operators';
import { v4 as uuidv4 } from 'uuid';
import { LogViewerComponent } from '../../../shared/components/log-viewer/log-viewer.component';
import { JobExtended } from '../../../core/services/job-state.service';
import { Job, JobService } from '../../../core/services/job.service';
import { Project } from '../../../core/models/project.model';
import { ProjectService } from '../../../core/services/project.service';
import { VisualElement } from '../../../core/models/visual-element.model';
import { VisualElementService } from '../../../core/services/visual-element.service';
import { ConfirmDialogComponent } from '../../../shared/components/confirm-dialog/confirm-dialog.component';
import { DialogJobs } from '../dialog-jobs/dialog-jobs.component';

@Component({
  selector: 'app-diagram',
  standalone: true,
  imports: [
    MatIconModule,
    MatButtonModule,
    MatFormFieldModule,
    MatInputModule,
    MatOptionModule,
    MatSelectModule,
    CdkContextMenuTrigger,
    CdkMenuModule,
    MatTooltipModule,
    MatProgressBarModule,
    MatSnackBarModule,
    CommonModule,
    FormsModule,
    LogViewerComponent
  ],
  templateUrl: './diagram.component.html',
  styleUrls: ['./diagram.component.scss']
})
export class Diagram implements AfterViewInit, OnChanges, OnDestroy {
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

  isPanning = false;
  viewOffsetX = 0;
  viewOffsetY = 0;
  panStartX = 0;
  panStartY = 0;
  panOriginX = 0;
  panOriginY = 0;

  zoom = 1;
  minZoom = 0.3;
  maxZoom = 1.6;
  zoomStep = 0.05;

  selectedJob: JobExtended | null = null;
  isLoading = false;
  isSaving = false;
  isBrowser: boolean;
  instance: any;
  gridX = 350;
  gridY = 100;
  showLogs = false;
  visualElements: VisualElement[] = [];
  selectedVisualElement: VisualElement | null = null;
  private draggingElementId: string | null = null;
  private dragStartX = 0;
  private dragStartY = 0;
  private dragOriginX = 0;
  private dragOriginY = 0;
  private dragOriginX2 = 0;
  private dragOriginY2 = 0;
  private activeMoveHandler: ((e: MouseEvent) => void) | null = null;
  private activeUpHandler: (() => void) | null = null;
  private elementSaveTimers = new Map<string, ReturnType<typeof setTimeout>>();
  private lastDragWasElement = false;
  private resizingElementId: string | null = null;
  private resizeHandle: 'nw' | 'ne' | 'sw' | 'se' | null = null;
  private resizeStartX = 0;
  private resizeStartY = 0;
  private resizeOriginX = 0;
  private resizeOriginY = 0;
  private resizeOriginW = 0;
  private resizeOriginH = 0;
  private lastProjectId: string | null = null;
  private rebuildTimer: ReturnType<typeof setTimeout> | null = null;
  private wheelHandler: ((event: WheelEvent) => void) | null = null;
  private blurHandler: (() => void) | null = null;
  private visibilityHandler: (() => void) | null = null;
  private suppressConnectionEvents = false;
  private repaintTimer: ReturnType<typeof setTimeout> | null = null;
  fontOptions: string[] = [
    'Space Grotesk, sans-serif',
    'IBM Plex Sans, sans-serif',
    'Inter, sans-serif',
    'Roboto, sans-serif',
    'Arial, sans-serif',
    'Helvetica, Arial, sans-serif',
    'Verdana, sans-serif',
    'Tahoma, sans-serif',
    'Trebuchet MS, sans-serif',
    'Georgia, serif',
    'Times New Roman, serif',
    'Garamond, serif',
    'JetBrains Mono, monospace',
    'Courier New, monospace',
    'Consolas, monospace',
    'Monaco, monospace'
  ];

  @HostListener('window:keydown', ['$event'])
  onGlobalKeyDown(event: KeyboardEvent) {
    if (event.key !== 'Delete' && event.key !== 'Backspace') {
      return;
    }

    if (this.isEditableTarget(event.target)) {
      return;
    }

    if (!this.selectedVisualElement) {
      return;
    }

    event.preventDefault();
    event.stopPropagation();
    this.confirmDeleteVisualElement(this.selectedVisualElement);
  }

  constructor(
    @Inject(PLATFORM_ID) private platformId: any,
    private jobService: JobService,
    private projectService: ProjectService,
    private dialog: MatDialog,
    private visualElementService: VisualElementService,
    private snackBar: MatSnackBar
  ) {
    this.isBrowser = isPlatformBrowser(this.platformId);
  }

  async ngAfterViewInit() {
    if (!this.isBrowser) return;

    this.isLoading = true;
    await this.initJsPlumbOnce();

    if (this.project) {
      this.project.connections = this.project.connections || [];
      this.project.jobs = this.project.jobs || [];
    }

    setTimeout(() => {
      this.jobs.forEach((job) => this.addJobToJsPlumb(job));
      this.addExistingConnections();
    }, 1000);

    const container = this.scrollContainer.nativeElement;
    const storedZoom = localStorage.getItem('diagramZoom');
    if (storedZoom) {
      this.zoom = +storedZoom;
      this.instance.setZoom(this.zoom);
    }

    this.wheelHandler = (event: WheelEvent) => {
      if (event.ctrlKey) {
        event.preventDefault();

        const rect = container.getBoundingClientRect();
        const mouseX = event.clientX - rect.left;
        const mouseY = event.clientY - rect.top;
        const prevZoom = this.zoom;

        this.zoom += event.deltaY < 0 ? this.zoomStep : -this.zoomStep;
        this.zoom = Math.min(Math.max(this.zoom, this.minZoom), this.maxZoom);

        if (this.instance) {
          this.viewOffsetX = mouseX - ((mouseX - this.viewOffsetX) * (this.zoom / prevZoom));
          this.viewOffsetY = mouseY - ((mouseY - this.viewOffsetY) * (this.zoom / prevZoom));
          this.instance.setZoom(this.zoom);
          this.instance.repaintEverything();
          localStorage.setItem('diagramZoom', this.zoom.toString());
          localStorage.setItem('diagramOffset', JSON.stringify({ x: this.viewOffsetX, y: this.viewOffsetY }));
        }
      }
    };
    container.addEventListener('wheel', this.wheelHandler, { passive: false });

    this.blurHandler = () => this.flushVisualElementInteraction('blur');
    this.visibilityHandler = () => {
      if (document.hidden) {
        this.flushVisualElementInteraction('visibilitychange');
      }
    };
    window.addEventListener('blur', this.blurHandler);
    document.addEventListener('visibilitychange', this.visibilityHandler);

    const storedOffset = localStorage.getItem('diagramOffset');
    if (storedOffset) {
      try {
        const parsed = JSON.parse(storedOffset);
        this.viewOffsetX = typeof parsed.x === 'number' ? parsed.x : 0;
        this.viewOffsetY = typeof parsed.y === 'number' ? parsed.y : 0;
      } catch {
        this.viewOffsetX = 0;
        this.viewOffsetY = 0;
      }
    }

    const projectId = this.project?.id;
    this.lastProjectId = projectId ?? null;
    if (projectId) {
      this.visualElementService.list(projectId).subscribe({
        next: (elements) => {
          const list = Array.isArray(elements) ? elements : [];
          list.forEach(el => this.normalizeElement(el));
          this.visualElements = list;
        },
        error: (error) => {
          console.error('Erro ao listar elementos visuais:', error);
        }
      });
    }
  }

  ngOnChanges(changes: SimpleChanges) {
    if (!this.isBrowser || !this.instance) return;

    const projectChanged = !!changes['project'] && this.project?.id !== this.lastProjectId;
    const jobsChanged = !!changes['jobs'] && !changes['jobs'].firstChange;

    if (projectChanged) {
      this.lastProjectId = this.project?.id ?? null;
      this.visualElements = [];
      const projectId = this.project?.id;
      if (projectId) {
        this.visualElementService.list(projectId).subscribe({
          next: (elements) => {
            const list = Array.isArray(elements) ? elements : [];
            list.forEach(el => this.normalizeElement(el));
            this.visualElements = list;
          },
          error: (error) => {
            console.error('Erro ao listar elementos visuais:', error);
          }
        });
      }
    }

    if (projectChanged) {
      this.scheduleRebuild('project-change');
      return;
    }

    if (jobsChanged) {
      const prevJobs = (changes['jobs']?.previousValue as JobExtended[] | undefined) ?? [];
      const currJobs = this.jobs ?? [];
      const idsChanged =
        prevJobs.length !== currJobs.length ||
        prevJobs.some((job, idx) => job?.id !== currJobs[idx]?.id);

      if (idsChanged) {
        this.scheduleRebuild('jobs-structure-change');
      } else {
        this.scheduleRepaint('jobs-update');
      }
    }
  }

  ngOnDestroy() {
    if (this.rebuildTimer) {
      clearTimeout(this.rebuildTimer);
      this.rebuildTimer = null;
    }
    if (this.repaintTimer) {
      clearTimeout(this.repaintTimer);
      this.repaintTimer = null;
    }

    const container = this.scrollContainer?.nativeElement;
    if (container && this.wheelHandler) {
      container.removeEventListener('wheel', this.wheelHandler);
    }
    if (this.blurHandler) {
      window.removeEventListener('blur', this.blurHandler);
    }
    if (this.visibilityHandler) {
      document.removeEventListener('visibilitychange', this.visibilityHandler);
    }
  }

  private scheduleRebuild(reason: string) {
    if (this.rebuildTimer) {
      clearTimeout(this.rebuildTimer);
    }
    this.isLoading = true;
    this.rebuildTimer = setTimeout(() => {
      this.rebuildTimer = null;
      this.rebuildPlumb(reason);
    }, 100);
  }

  private scheduleRepaint(reason: string) {
    if (!this.instance) return;
    if (this.repaintTimer) {
      clearTimeout(this.repaintTimer);
    }
    this.repaintTimer = setTimeout(() => {
      this.repaintTimer = null;
      this.instance.repaintEverything();
    }, 50);
  }

  private rebuildPlumb(reason: string) {
    if (!this.instance) return;

    // Clear existing endpoints and connections to avoid cross-project artifacts.
    this.suppressConnectionEvents = true;
    try {
      this.instance.deleteEveryConnection();
      this.instance.deleteEveryEndpoint();
    } catch {
      // no-op, jsPlumb might not be fully ready yet
    }

    // Re-attach jobs and connections for the current project.
    this.jobs.forEach((job) => this.addJobToJsPlumb(job));
    this.addExistingConnections();
    this.instance.repaintEverything();
    requestAnimationFrame(() => this.instance?.repaintEverything());
    this.suppressConnectionEvents = false;
  }

  private jsPlumbInitialized = false;

  async initJsPlumbOnce(): Promise<void> {
    if (this.jsPlumbInitialized) return;
    this.jsPlumbInitialized = true;

    const jsPlumbModule = await import('jsplumb');
    const jsPlumb = jsPlumbModule.jsPlumb;

    this.instance = jsPlumb.getInstance();
    this.instance.setContainer('diagramContainer');

    this.instance.bind('beforeDrop', (info: any) => {
      if (info.sourceId === info.targetId) return false;
      const existing = this.instance.getConnections({
        source: info.sourceId,
        target: info.targetId
      });
      return existing.length === 0;
    });

    this.instance.bind('connection', (info: any) => {
      if (!this.isLoading && !this.suppressConnectionEvents && this.project) {
        this.project.connections = this.project.connections || [];
        this.project.connections.push({
          source: info.sourceId,
          target: info.targetId
        });
        this.saveProject();
      }
    });

    this.instance.bind('connectionDetached', (info: any) => {
      if (this.suppressConnectionEvents || this.isLoading) return;
      if (!this.project || !this.project.connections) return;
      const index = this.project.connections.findIndex(
        (conn) => conn.source === info.sourceId && conn.target === info.targetId
      );
      if (index >= 0) {
        this.project.connections.splice(index, 1);
        this.saveProject();
      }
    });
  }

  saveProject() {
    this.isSaving = true;
    this.syncProjectVisualElements();
    const result = this.projectService.updateProject(this.project!.id, this.project!);
    if (result && typeof (result as any).subscribe === 'function') {
      result.subscribe({
        next: () => {
          this.isSaved();
        },
        error: (error: unknown) => {
          this.notifyPersistError('Erro ao salvar o projeto. Verifique sua conexao.', error);
          this.isSaved();
        }
      });
    } else {
      this.isSaved();
    }
  }

  addJobToJsPlumb(job: Job) {
    const id = job.id;
    if (!this.instance) return;

    this.instance.makeSource(id, {
      filter: '.handle',
      anchor: 'Right',
      connector: ['Flowchart', { stub: 30, gap: 8, cornerRadius: 8, alwaysRespectStubs: true }],
      endpoint: 'Dot',
      connectorOverlays: [['Arrow', { width: 10, length: 10, location: 1 }]],
      maxConnections: -1
    });

    this.instance.makeTarget(id, {
      anchor: 'Left',
      allowLoopback: false,
      endpoint: 'Blank'
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

        const movedJob = this.jobs.find(j => j.id === el.id);
        if (movedJob) {
          movedJob.left = x;
          movedJob.top = y;

          this.jobService.updateJob(this.project?.id || '', movedJob.id, movedJob).subscribe({
            next: () => {
              this.isSaved();
            },
            error: () => {
              this.isSaved();
            }
          });
        }
      }
    });
  }

  addExistingConnections() {
    if (!this.instance || !this.project?.connections) return;

    this.project.connections.forEach((conn) => {
      this.instance.connect({
        source: conn.source,
        target: conn.target,
        anchors: ['Right', 'Left'],
        connector: ['Flowchart', { stub: 30, gap: 8, cornerRadius: 8, alwaysRespectStubs: true }],
        overlays: [['Arrow', { width: 10, length: 10, location: 1 }]]
      });
    });
    this.isLoading = false;
  }

  addNewJob(): void {
    this.addNewJobAt(10, 10);
  }

  addNewJobAt(x: number, y: number): void {
    this.isSaving = true;

    const newJob: Job = {
      id: uuidv4(),
      jobName: 'Novo Job',
      selectSql: '',
      insertSql: '',
      posInsertSql: '',
      columns: [],
      recordsPerPage: 1000,
      type: 'insert',
      stopOnError: true,
      top: y,
      left: x
    };

    this.jobs.push(newJob);

    if (this.project) {
      this.project.jobs = this.jobs.map(job => `jobs/${job.id}.json`);
    }

    this.scheduleJobPlumbAttach(newJob.id);

    this.jobService.addJob(this.project?.id || '', newJob).subscribe({
      next: () => {
        this.saveProject();
      }
    });
  }

  onRightClick(event: MouseEvent, job: Job) {
    event.preventDefault();
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
            this.jobs = this.jobs.filter(j => j.id !== job.id);
            if (this.project?.jobs) {
              const index = this.project.jobs.indexOf(`jobs/${job.id}.json`);
              if (index >= 0) {
                this.project.jobs.splice(index, 1);
              }
            }
            this.instance.removeAllEndpoints(job.id);
            this.instance.remove(job.id);
            this.saveProject();
            this.isSaved();
          },
          error: () => {
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
      if (result) {
        if (result.id) {
          this.jobService.updateJob(this.project?.id || '', result.id, result).subscribe({
            next: (updatedJob) => {
              const index = this.jobs.findIndex(j => j.id === updatedJob.id);
              if (index !== -1) {
                this.jobs[index] = { ...this.jobs[index], ...updatedJob };
              }
              this.selectedJob = this.jobs[index] ?? updatedJob;
              this.saveProject();
              if (this.instance) {
                setTimeout(() => this.instance.repaintEverything(), 0);
              }
            },
            error: () => {
              this.notifyPersistError('Erro ao atualizar job');
            }
          });
        }
      }
    });
  }

  addVisualElement(type: VisualElement['type']) {
    this.addVisualElementAt(type, 120, 120);
  }

  addVisualElementAt(type: VisualElement['type'], x: number, y: number) {
    if (!this.project?.id) {
      this.notifyPersistError('Projeto ainda nao esta carregado. Nao foi possivel criar o elemento visual.');
      return;
    }
    this.isSaving = true;
    const base: VisualElement = {
      type,
      x,
      y,
      width: 200,
      height: 120,
      fillColor: 'rgba(15, 23, 42, 0.65)',
      borderColor: 'rgba(148, 163, 184, 0.6)',
      borderWidth: 2,
      text: type === 'text' ? 'Texto' : '',
      textColor: '#e2e8f0',
      fontSize: 12,
      fontFamily: 'Space Grotesk',
      textAlign: 'center',
      cornerRadius: 8
    };

    if (type === 'circle') {
      base.width = 140;
      base.height = 140;
      base.cornerRadius = 9999;
    }

    if (type === 'line') {
      base.x2 = x + 200;
      base.y2 = y;
      base.borderColor = '#94a3b8';
      base.borderWidth = 2;
    }

    const payload = this.toPayload(base);
    this.visualElementService.create(this.project.id, payload).subscribe({
      next: (created) => {
        const normalized = this.normalizeElement(created);
        this.visualElements = [...this.visualElements, normalized];
        this.selectedVisualElement = normalized;
        this.isSaved();
      },
      error: (error) => {
        this.notifyPersistError('Erro ao criar elemento visual. Verifique a conexao e tente novamente.', error);
        this.isSaved();
      }
    });
  }

  private notifyPersistError(message: string, error?: unknown) {
    if (error) {
      console.error(message, error);
    } else {
      console.error(message);
    }
    this.snackBar.open(message, 'Fechar', {
      duration: 7000,
      panelClass: ['error-snackbar']
    });
  }

  private persistVisualElement(element: VisualElement, context: string) {
    const elementId = this.getElementId(element);
    if (!elementId || !this.project?.id) {
      this.notifyPersistError(`Falha ao salvar elemento (${context}). Projeto ou ID ausente.`);
      return;
    }
    this.isSaving = true;
    const payload = this.toPayload(element);
    this.visualElementService.update(this.project.id, elementId, payload).subscribe({
      next: () => {
        this.isSaved();
      },
      error: (error) => {
        this.notifyPersistError(`Erro ao salvar elemento (${context}).`, error);
        this.isSaved();
      }
    });
  }

  private syncProjectVisualElements() {
    if (!this.project) return;
    this.project.visualElements = this.visualElements.map(element => {
      const id = this.getElementId(element);
      const payload = this.toPayload(element);
      return {
        ...payload,
        id
      };
    });
  }

  private flushVisualElementInteraction(reason: string) {
    if (this.draggingElementId) {
      const updated = this.visualElements.find(e => this.getElementId(e) === this.draggingElementId);
      if (updated) {
        this.normalizeElement(updated);
        this.persistVisualElement(updated, `arraste interrompido (${reason})`);
      }
    }
    if (this.resizingElementId) {
      const updated = this.visualElements.find(e => this.getElementId(e) === this.resizingElementId);
      if (updated) {
        this.normalizeElement(updated);
        this.persistVisualElement(updated, `redimensionamento interrompido (${reason})`);
      }
    }
    if (this.activeMoveHandler) {
      window.removeEventListener('mousemove', this.activeMoveHandler);
      this.activeMoveHandler = null;
    }
    if (this.activeUpHandler) {
      window.removeEventListener('mouseup', this.activeUpHandler);
      this.activeUpHandler = null;
    }
    this.draggingElementId = null;
    this.resizingElementId = null;
    this.resizeHandle = null;
    this.lastDragWasElement = false;
  }

  private scheduleJobPlumbAttach(jobId: string, attempt = 0) {
    if (!this.instance) return;
    const el = document.getElementById(jobId);
    if (el) {
      this.addJobToJsPlumb({ id: jobId } as Job);
      return;
    }
    if (attempt >= 10) return;
    setTimeout(() => this.scheduleJobPlumbAttach(jobId, attempt + 1), 50);
  }

  onToolbarDragStart(event: DragEvent, payload: { kind: 'job' } | { kind: 'visual'; type: VisualElement['type'] }) {
    if (!event.dataTransfer) return;
    event.dataTransfer.setData('application/json', JSON.stringify(payload));
    event.dataTransfer.effectAllowed = 'copy';
  }

  onDiagramDragOver(event: DragEvent) {
    event.preventDefault();
    if (event.dataTransfer) {
      event.dataTransfer.dropEffect = 'copy';
    }
  }

  onDiagramDrop(event: DragEvent) {
    event.preventDefault();
    if (!event.dataTransfer || !this.scrollContainer) return;
    if (!this.project?.id) {
      this.notifyPersistError('Projeto ainda nao esta carregado. Nao foi possivel criar o item.');
      return;
    }
    const raw = event.dataTransfer.getData('application/json');
    if (!raw) return;

    let payload: { kind: 'job' } | { kind: 'visual'; type: VisualElement['type'] };
    try {
      payload = JSON.parse(raw);
    } catch {
      return;
    }

    const rect = this.scrollContainer.nativeElement.getBoundingClientRect();
    const scale = this.zoom || 1;
    const x = (event.clientX - rect.left - this.viewOffsetX) / scale;
    const y = (event.clientY - rect.top - this.viewOffsetY) / scale;

    if (payload.kind === 'job') {
      this.addNewJobAt(Math.round(x), Math.round(y));
      return;
    }
    this.addVisualElementAt(payload.type, Math.round(x), Math.round(y));
  }

  onElementMouseDown(event: MouseEvent, element: VisualElement) {
    event.stopPropagation();
    event.preventDefault();
    this.selectedVisualElement = element;
    this.lastDragWasElement = true;
    this.draggingElementId = this.getElementId(element) || null;
    this.dragStartX = event.clientX;
    this.dragStartY = event.clientY;
    this.dragOriginX = element.x;
    this.dragOriginY = element.y;
    this.dragOriginX2 = element.x2 || 0;
    this.dragOriginY2 = element.y2 || 0;

    const onMove = (moveEvent: MouseEvent) => {
      if (!this.draggingElementId) return;
      const dx = moveEvent.clientX - this.dragStartX;
      const dy = moveEvent.clientY - this.dragStartY;
      const target = this.visualElements.find(e => e.id === this.draggingElementId);
      if (!target) return;
      const scale = this.zoom || 1;
      target.x = this.dragOriginX + dx / scale;
      target.y = this.dragOriginY + dy / scale;
      if (target.type === 'line' && target.x2 !== undefined && target.y2 !== undefined) {
        target.x2 = this.dragOriginX2 + dx / scale;
        target.y2 = this.dragOriginY2 + dy / scale;
      }
      this.normalizeElement(target);
    };

    const onUp = () => {
      if (this.draggingElementId) {
        const updated = this.visualElements.find(e => e.id === this.draggingElementId);
        if (updated) {
          this.normalizeElement(updated);
          this.persistVisualElement(updated, 'arraste');
        }
      }
      this.draggingElementId = null;
      this.lastDragWasElement = false;
      window.removeEventListener('mousemove', onMove);
      window.removeEventListener('mouseup', onUp);
      this.activeMoveHandler = null;
      this.activeUpHandler = null;
    };

    this.activeMoveHandler = onMove;
    this.activeUpHandler = onUp;
    window.addEventListener('mousemove', onMove);
    window.addEventListener('mouseup', onUp);
  }

  onResizeMouseDown(event: MouseEvent, element: VisualElement, handle: 'nw' | 'ne' | 'sw' | 'se') {
    event.stopPropagation();
    event.preventDefault();
    this.selectedVisualElement = element;
    this.lastDragWasElement = true;
    this.resizingElementId = this.getElementId(element) || null;
    this.resizeHandle = handle;
    this.resizeStartX = event.clientX;
    this.resizeStartY = event.clientY;
    this.resizeOriginX = element.x;
    this.resizeOriginY = element.y;
    this.resizeOriginW = this.getElementWidth(element);
    this.resizeOriginH = this.getElementHeight(element);

    const onMove = (moveEvent: MouseEvent) => {
      if (!this.resizingElementId || !this.resizeHandle) return;
      const target = this.visualElements.find(e => this.getElementId(e) === this.resizingElementId);
      if (!target) return;
      const scale = this.zoom || 1;
      const dx = (moveEvent.clientX - this.resizeStartX) / scale;
      const dy = (moveEvent.clientY - this.resizeStartY) / scale;

      let newX = this.resizeOriginX;
      let newY = this.resizeOriginY;
      let newW = this.resizeOriginW;
      let newH = this.resizeOriginH;

      if (this.resizeHandle === 'nw') {
        newX = this.resizeOriginX + dx;
        newY = this.resizeOriginY + dy;
        newW = this.resizeOriginW - dx;
        newH = this.resizeOriginH - dy;
      } else if (this.resizeHandle === 'ne') {
        newY = this.resizeOriginY + dy;
        newW = this.resizeOriginW + dx;
        newH = this.resizeOriginH - dy;
      } else if (this.resizeHandle === 'sw') {
        newX = this.resizeOriginX + dx;
        newW = this.resizeOriginW - dx;
        newH = this.resizeOriginH + dy;
      } else if (this.resizeHandle === 'se') {
        newW = this.resizeOriginW + dx;
        newH = this.resizeOriginH + dy;
      }

      const minSize = 40;
      newW = Math.max(minSize, newW);
      newH = Math.max(minSize, newH);

      if (target.type === 'circle') {
        const size = Math.max(newW, newH);
        newW = size;
        newH = size;
      }

      target.x = newX;
      target.y = newY;
      target.width = newW;
      target.height = newH;
      this.normalizeElement(target);
    };

    const onUp = () => {
      if (this.resizingElementId) {
        const updated = this.visualElements.find(e => this.getElementId(e) === this.resizingElementId);
        if (updated) {
          this.normalizeElement(updated);
          this.persistVisualElement(updated, 'redimensionamento');
        }
      }
      this.resizingElementId = null;
      this.resizeHandle = null;
      this.lastDragWasElement = false;
      window.removeEventListener('mousemove', onMove);
      window.removeEventListener('mouseup', onUp);
      this.activeMoveHandler = null;
      this.activeUpHandler = null;
    };

    this.activeMoveHandler = onMove;
    this.activeUpHandler = onUp;
    window.addEventListener('mousemove', onMove);
    window.addEventListener('mouseup', onUp);
  }

  onLineHandleMouseDown(event: MouseEvent, element: VisualElement, handle: 'start' | 'end') {
    event.stopPropagation();
    event.preventDefault();
    this.selectedVisualElement = element;
    this.lastDragWasElement = true;
    this.draggingElementId = this.getElementId(element) || null;
    this.dragStartX = event.clientX;
    this.dragStartY = event.clientY;
    this.dragOriginX = element.x;
    this.dragOriginY = element.y;
    this.dragOriginX2 = element.x2 || 0;
    this.dragOriginY2 = element.y2 || 0;

    const onMove = (moveEvent: MouseEvent) => {
      if (!this.draggingElementId) return;
      const target = this.visualElements.find(e => this.getElementId(e) === this.draggingElementId);
      if (!target) return;
      const scale = this.zoom || 1;
      const dx = (moveEvent.clientX - this.dragStartX) / scale;
      const dy = (moveEvent.clientY - this.dragStartY) / scale;

      if (handle === 'start') {
        target.x = this.dragOriginX + dx;
        target.y = this.dragOriginY + dy;
      } else {
        target.x2 = this.dragOriginX2 + dx;
        target.y2 = this.dragOriginY2 + dy;
      }
      this.normalizeElement(target);
    };

    const onUp = () => {
      if (this.draggingElementId) {
        const updated = this.visualElements.find(e => this.getElementId(e) === this.draggingElementId);
        if (updated) {
          this.normalizeElement(updated);
          this.persistVisualElement(updated, 'ajuste de linha');
        }
      }
      this.draggingElementId = null;
      this.lastDragWasElement = false;
      window.removeEventListener('mousemove', onMove);
      window.removeEventListener('mouseup', onUp);
      this.activeMoveHandler = null;
      this.activeUpHandler = null;
    };

    this.activeMoveHandler = onMove;
    this.activeUpHandler = onUp;
    window.addEventListener('mousemove', onMove);
    window.addEventListener('mouseup', onUp);
  }

  selectElement(element: VisualElement, event: MouseEvent) {
    event.stopPropagation();
    this.selectedVisualElement = element;
  }

  clearSelection() {
    this.selectedVisualElement = null;
  }

  onDiagramBackgroundClick(event: MouseEvent) {
    if (this.lastDragWasElement) return;
    const target = event.target as HTMLElement | null;
    if (!target) return;
    if (target.closest('.visual-panel, .visual-element, .box, .mat-mdc-form-field, .visual-lines')) {
      return;
    }
    this.clearSelection();
  }

  onElementChange(element: VisualElement) {
    this.normalizeElement(element);
    this.scheduleElementSave(element);
  }

  scheduleElementSave(element: VisualElement) {
    const elementId = this.getElementId(element);
    if (!elementId || !this.project?.id) return;
    const existing = this.elementSaveTimers.get(elementId);
    if (existing) {
      clearTimeout(existing);
    }
    const timer = setTimeout(() => {
      this.persistVisualElement(element, 'edicao');
    }, 300);
    this.elementSaveTimers.set(elementId, timer);
  }

  normalizeElement(element: VisualElement): VisualElement {
    if (!element.id && element.elementId) {
      element.id = element.elementId;
    }
    element.x = this.toNumber(element.x, 0);
    element.y = this.toNumber(element.y, 0);
    if (element.width !== undefined) element.width = this.toNumber(element.width, 160);
    if (element.height !== undefined) element.height = this.toNumber(element.height, 100);
    if (element.x2 !== undefined) element.x2 = this.toNumber(element.x2, element.x + 200);
    if (element.y2 !== undefined) element.y2 = this.toNumber(element.y2, element.y);
    if (element.borderWidth !== undefined) element.borderWidth = this.toNumber(element.borderWidth, 1);
    if (element.fontSize !== undefined) element.fontSize = this.toNumber(element.fontSize, 12);
    if (element.cornerRadius !== undefined) element.cornerRadius = this.toNumber(element.cornerRadius, 0);

    if (element.type === 'circle') {
      const size = element.width || element.height || 120;
      element.width = size;
      element.height = size;
      element.cornerRadius = 9999;
    }

    if (element.type === 'line') {
      if (element.x2 === undefined) element.x2 = element.x + 200;
      if (element.y2 === undefined) element.y2 = element.y;
    }
    return element;
  }

  private toNumber(value: unknown, fallback: number): number {
    const parsed = typeof value === 'string' ? Number(value) : (value as number);
    return Number.isFinite(parsed) ? parsed : fallback;
  }

  private confirmDeleteVisualElement(element: VisualElement) {
    const dialogRef = this.dialog.open(ConfirmDialogComponent, {
      minWidth: '30vw',
      minHeight: '20vh',
      data: {
        title: 'Remover elemento visual',
        message: 'Tem certeza que deseja remover este elemento visual?'
      }
    });

    dialogRef.afterClosed().subscribe(confirmed => {
      if (confirmed) {
        this.deleteVisualElement(element);
      }
    });
  }

  deleteVisualElement(element: VisualElement) {
    const elementId = this.getElementId(element);
    if (!this.project?.id || !elementId) return;
    this.isSaving = true;
    this.visualElementService.remove(this.project.id, elementId).subscribe({
      next: () => {
        this.visualElements = this.visualElements.filter(e => this.getElementId(e) !== elementId);
        if (this.selectedVisualElement && this.getElementId(this.selectedVisualElement) === elementId) {
          this.selectedVisualElement = null;
        }
        this.isSaved();
      },
      error: () => {
        this.isSaved();
      }
    });
  }

  getElementId(element: VisualElement): string | undefined {
    return element.id || element.elementId;
  }

  private toPayload(element: VisualElement): VisualElement {
    const base = {
      type: element.type,
      x: this.toInt(element.x, 0),
      y: this.toInt(element.y, 0)
    };

    if (element.type === 'line') {
      return {
        ...base,
        x2: this.toInt(element.x2 ?? element.x + 200, base.x + 200),
        y2: this.toInt(element.y2 ?? element.y, base.y),
        borderColor: element.borderColor ?? '#94a3b8',
        borderWidth: this.toNonNegativeInt(element.borderWidth ?? 2, 2)
      };
    }

    const width = this.toNonNegativeInt(element.width ?? 160, 160);
    const height = this.toNonNegativeInt(element.height ?? 100, 100);

    return {
      ...base,
      width,
      height,
      fillColor: element.fillColor ?? 'transparent',
      borderColor: element.borderColor ?? 'transparent',
      borderWidth: this.toNonNegativeInt(element.borderWidth ?? 1, 1),
      text: element.text ?? '',
      textColor: element.textColor ?? '#e2e8f0',
      fontSize: this.toNonNegativeInt(element.fontSize ?? 12, 12),
      fontFamily: element.fontFamily ?? 'Space Grotesk',
      textAlign: element.textAlign ?? 'center',
      cornerRadius: this.toNonNegativeInt(element.cornerRadius ?? 0, 0)
    };
  }

  private toNonNegativeInt(value: unknown, fallback: number): number {
    const parsed = typeof value === 'string' ? Number(value) : (value as number);
    if (!Number.isFinite(parsed)) return fallback;
    return Math.max(0, Math.round(parsed));
  }

  private toInt(value: unknown, fallback: number): number {
    const parsed = typeof value === 'string' ? Number(value) : (value as number);
    if (!Number.isFinite(parsed)) return fallback;
    return Math.round(parsed);
  }

  private isEditableTarget(target: EventTarget | null): boolean {
    if (!target || !(target instanceof Element)) return false;
    return !!target.closest('input, textarea, select, [contenteditable="true"]');
  }

  getElementWidth(element: VisualElement): number {
    if (element.type === 'circle') {
      return element.width || element.height || 120;
    }
    return element.width || 160;
  }

  getElementHeight(element: VisualElement): number {
    if (element.type === 'circle') {
      return element.height || element.width || 120;
    }
    return element.height || 100;
  }

  getElementBorderRadius(element: VisualElement): string {
    return element.type === 'circle' ? '50%' : `${element.cornerRadius || 0}px`;
  }

  getElementJustifyContent(element: VisualElement): string {
    const { horizontal } = this.parseTextAlign(element.textAlign);
    if (horizontal === 'left') return 'flex-start';
    if (horizontal === 'right') return 'flex-end';
    return 'center';
  }

  getElementAlignItems(element: VisualElement): string {
    const { vertical } = this.parseTextAlign(element.textAlign);
    if (vertical === 'top') return 'flex-start';
    if (vertical === 'bottom') return 'flex-end';
    return 'center';
  }

  getElementTextAlign(element: VisualElement): string {
    const { horizontal } = this.parseTextAlign(element.textAlign);
    return horizontal;
  }

  private parseTextAlign(value?: string): { vertical: 'top' | 'center' | 'bottom'; horizontal: 'left' | 'center' | 'right' } {
    const align = this.normalizeTextAlign(value);
    if (align === 'center') return { vertical: 'center', horizontal: 'center' };
    const [verticalRaw, horizontalRaw] = align.split('-');
    const vertical = (verticalRaw as 'top' | 'center' | 'bottom') || 'center';
    const horizontal = (horizontalRaw as 'left' | 'center' | 'right') || 'center';
    return { vertical, horizontal };
  }

  private normalizeTextAlign(value?: string): string {
    if (!value) return 'center';
    if (value === 'left') return 'center-left';
    if (value === 'right') return 'center-right';
    return value;
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
      next: () => {
        this.isSaved();
      },
      error: () => {
        this.isSaved();
      }
    });
  }

  stopProject() {
    if (!this.project?.id) {
      this.notifyPersistError('Projeto ainda nao esta carregado. Nao foi possivel parar a pipeline.');
      return;
    }
    this.isSaving = true;
    this.projectService.stopProject(this.project.id).subscribe({
      next: () => {
        this.isRunning = false;
        this.isSaved();
      },
      error: (error) => {
        if (error?.status === 404) {
          this.notifyPersistError('Nenhuma pipeline ativa para este projeto.');
        } else {
          this.notifyPersistError('Erro ao parar a pipeline. Tente novamente.', error);
        }
        this.isSaved();
      }
    });
  }

  showHideLogs() {
    this.showLogs = !this.showLogs;
  }

  onMouseDown(e: MouseEvent): void {
    const target = e.target as HTMLElement | null;
    if (target && target.closest('.visual-element, .resize-handle, .box, .visual-panel, .context-menu, .mat-mdc-form-field')) {
      return;
    }
    this.isPanning = true;
    const container = this.scrollContainer.nativeElement;
    this.panStartX = e.clientX;
    this.panStartY = e.clientY;
    this.panOriginX = this.viewOffsetX;
    this.panOriginY = this.viewOffsetY;
    container.style.cursor = 'grabbing';
  }

  onMouseMove(e: MouseEvent): void {
    if (!this.isPanning) return;
    e.preventDefault();
    const dx = e.clientX - this.panStartX;
    const dy = e.clientY - this.panStartY;
    this.viewOffsetX = this.panOriginX + dx;
    this.viewOffsetY = this.panOriginY + dy;
  }

  onMouseUp(): void {
    if (this.isPanning) {
      localStorage.setItem('diagramOffset', JSON.stringify({ x: this.viewOffsetX, y: this.viewOffsetY }));
    }
    this.isPanning = false;
    this.scrollContainer.nativeElement.style.cursor = 'grab';
  }

  removeZoom() {
    this.zoom = 1;
    if (this.instance) {
      this.instance.setZoom(this.zoom);
      this.instance.repaintEverything();
      localStorage.setItem('diagramZoom', this.zoom.toString());
    }
  }

  getDiagramTransform(): string {
    return `translate(${this.viewOffsetX}px, ${this.viewOffsetY}px) scale(${this.zoom})`;
  }

  tempoTotal(dataIso: string, dataIso2: string): string {
    const agora = new Date();
    const data = new Date(dataIso);
    const data2 = new Date(dataIso2);

    if (isNaN(data.getTime())) return '';

    let diffMs: number;
    if (isNaN(data2.getTime())) {
      diffMs = agora.getTime() - data.getTime();
    } else {
      diffMs = data2.getTime() - data.getTime();
    }

    const totalSeconds = Math.floor(diffMs / 1000);
    const hours = Math.floor(totalSeconds / 3600);
    const minutes = Math.floor((totalSeconds % 3600) / 60);
    const seconds = totalSeconds % 60;

    return [
      hours.toString().padStart(2, '0'),
      minutes.toString().padStart(2, '0'),
      seconds.toString().padStart(2, '0')
    ].join(':');
  }
}
