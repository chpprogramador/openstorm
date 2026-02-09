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
  ViewChildren,
  ChangeDetectorRef
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
import { Subscription, of } from 'rxjs';
import { delay } from 'rxjs/operators';
import { v4 as uuidv4 } from 'uuid';
import { LogViewerComponent } from '../../../shared/components/log-viewer/log-viewer.component';
import { JobExtended, jobs_ } from '../../../core/services/job-state.service';
import { Job, JobService } from '../../../core/services/job.service';
import { CountsProgress } from '../../../core/services/counts-status.service';
import { Project } from '../../../core/models/project.model';
import { ProjectService } from '../../../core/services/project.service';
import { VisualElement } from '../../../core/models/visual-element.model';
import { VisualElementService } from '../../../core/services/visual-element.service';
import { WorkersUsage } from '../../../core/services/workers-status.service';
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
  @Input() countsProgress: CountsProgress | null = null;
  @Input() workersUsage: WorkersUsage | null = null;
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
  isSpacePressed = false;
  isSelecting = false;
  selectionStartX = 0;
  selectionStartY = 0;
  selectionRect = { x: 0, y: 0, width: 0, height: 0 };
  selectionMoved = false;
  suppressBackgroundClick = false;
  jobBoxWidth = 300;
  jobBoxHeight = 100;

  zoom = 1;
  minZoom = 0.3;
  maxZoom = 1.6;
  zoomStep = 0.05;

  selectedJob: JobExtended | null = null;
  selectedVisualElements: VisualElement[] = [];
  selectedJobs: JobExtended[] = [];
  panelElement: VisualElement | null = null;
  private undoStack: Array<{
    jobs: Array<{ id: string; before: { left: number; top: number }; after: { left: number; top: number } }>;
    visuals: Array<{
      id: string;
      before: { x: number; y: number; x2?: number; y2?: number; width?: number; height?: number };
      after: { x: number; y: number; x2?: number; y2?: number; width?: number; height?: number };
    }>;
  }> = [];
  private maxUndoEntries = 5;
  private isApplyingUndo = false;
  isLoading = false;
  isSaving = false;
  isBrowser: boolean;
  instance: any;
  gridX = 100;
  gridY = 60;
  showLogs = false;
  visualElements: VisualElement[] = [];
  selectedVisualElement: VisualElement | null = null;
  snapEnabled = true;
  lastSavedAt: Date | null = null;
  showShortcuts = true;
  private draggingElementId: string | null = null;
  private dragStartX = 0;
  private dragStartY = 0;
  private dragOriginX = 0;
  private dragOriginY = 0;
  private dragOriginX2 = 0;
  private dragOriginY2 = 0;
  private snapToGrid(x: number, y: number): [number, number] {
    if (!this.snapEnabled) return [x, y];
    return [Math.round(x / this.gridX) * this.gridX, Math.round(y / this.gridY) * this.gridY];
  }
  private activeMoveHandler: ((e: MouseEvent) => void) | null = null;
  private activeUpHandler: (() => void) | null = null;
  private groupDragOrigins = new Map<string, { x: number; y: number; x2?: number; y2?: number }>();
  private groupJobOrigins = new Map<string, { left: number; top: number }>();
  private isGroupDragging = false;
  private groupRepaintRaf: number | null = null;
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
  private rebuildNonce = 0;
  private activeRebuildNonce = 0;
  private pendingRebuild = false;
  private pendingRebuildKey: string | null = null;
  private lastRebuildKey: string | null = null;
  private blurHandler: (() => void) | null = null;
  private visibilityHandler: (() => void) | null = null;
  private suppressConnectionEvents = false;
  private repaintTimer: ReturnType<typeof setTimeout> | null = null;
  private jobElementsSub: Subscription | null = null;
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
    const isEditable = this.isEditableTarget(event.target);
    const isModifier = event.ctrlKey || event.metaKey;
    if (isModifier && !event.shiftKey && (event.key === 'z' || event.key === 'Z')) {
      if (!isEditable) {
        const handled = this.undoLast();
        if (handled) {
          event.preventDefault();
          event.stopPropagation();
        }
      }
      return;
    }
    if (isModifier && (event.key === 'd' || event.key === 'D')) {
      if (!isEditable) {
        const handled = this.duplicateSelection();
        if (handled) {
          event.preventDefault();
          event.stopPropagation();
        }
      }
      return;
    }
    if (event.code === 'Space') {
      if (!this.isSpacePressed) {
        this.isSpacePressed = true;
      }
      if (!this.isEditableTarget(event.target)) {
        event.preventDefault();
      }
      return;
    }
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

  @HostListener('window:keyup', ['$event'])
  onGlobalKeyUp(event: KeyboardEvent) {
    if (event.code === 'Space') {
      this.isSpacePressed = false;
      if (this.isPanning) {
        this.isPanning = false;
      }
      const container = this.scrollContainer?.nativeElement;
      if (container) {
        container.style.cursor = 'default';
      }
    }
  }

  constructor(
    @Inject(PLATFORM_ID) private platformId: any,
    private jobService: JobService,
    private projectService: ProjectService,
    private dialog: MatDialog,
    private visualElementService: VisualElementService,
    private snackBar: MatSnackBar,
    private cdr: ChangeDetectorRef
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
      this.scheduleRebuild('after-view-init');
    }, 0);

    const container = this.scrollContainer.nativeElement;
    const storedZoom = localStorage.getItem('diagramZoom');
    if (storedZoom) {
      this.zoom = +storedZoom;
      this.instance.setZoom(this.zoom);
    }
    const storedSnap = localStorage.getItem('diagramSnap');
    if (storedSnap) {
      this.snapEnabled = storedSnap === 'true';
    }

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

    // Rebuild once job elements are actually rendered in the DOM.
    this.jobElementsSub = this.jobElements.changes.subscribe(() => {
      if (this.pendingRebuild) {
        this.scheduleRebuild('job-elements-changed');
      }
    });
  }

  ngOnChanges(changes: SimpleChanges) {
    if (!this.isBrowser || !this.instance) return;

    const projectChanged = !!changes['project'] && this.project?.id !== this.lastProjectId;
    const jobsChange = changes['jobs'];
    const jobsChanged = !!jobsChange && !jobsChange.firstChange;

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

    if (jobsChange && jobsChange.firstChange) {
      if ((this.jobs?.length ?? 0) > 0) {
        this.scheduleRebuild('jobs-first-load');
      }
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
    if (this.jobElementsSub) {
      this.jobElementsSub.unsubscribe();
      this.jobElementsSub = null;
    }

    if (this.blurHandler) {
      window.removeEventListener('blur', this.blurHandler);
    }
    if (this.visibilityHandler) {
      document.removeEventListener('visibilitychange', this.visibilityHandler);
    }
  }

  onWheel(event: WheelEvent) {
    if (this.isEditableTarget(event.target)) {
      return;
    }

    const container = this.scrollContainer?.nativeElement as HTMLElement | undefined;
    if (!container) return;

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
      this.cdr.detectChanges();
      return;
    }

    event.preventDefault();
    let deltaX = event.deltaX;
    let deltaY = event.deltaY;
    if (event.shiftKey && !deltaX) {
      deltaX = deltaY;
      deltaY = 0;
    }
    this.viewOffsetX -= deltaX;
    this.viewOffsetY -= deltaY;
    localStorage.setItem('diagramOffset', JSON.stringify({ x: this.viewOffsetX, y: this.viewOffsetY }));
    this.cdr.detectChanges();
  }

  private scheduleRebuild(reason: string, force = false) {
    const key = this.computeStructureKey();
    if (!force) {
      if (this.pendingRebuild && this.pendingRebuildKey === key) {
        if (this.rebuildTimer) {
          return;
        }
      }
      if (!this.pendingRebuild && this.lastRebuildKey === key) {
        return;
      }
    }
    this.pendingRebuild = true;
    this.pendingRebuildKey = key;
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

  private computeStructureKey(): string {
    const jobIds = (this.jobs ?? [])
      .map(job => job.id)
      .filter(Boolean)
      .sort()
      .join(',');
    const conns = (this.project?.connections ?? [])
      .map(conn => `${conn.source}->${conn.target}`)
      .sort()
      .join(',');
    const projectId = this.project?.id ?? '';
    return `${projectId}|jobs:${jobIds}|conns:${conns}`;
  }

  private rebuildPlumb(reason: string) {
    if (!this.instance) return;
    const rebuildNonce = ++this.rebuildNonce;
    this.activeRebuildNonce = rebuildNonce;
    const expectedKey = this.pendingRebuildKey ?? this.computeStructureKey();

    if (expectedKey === this.lastRebuildKey) {
      this.pendingRebuild = false;
      this.pendingRebuildKey = null;
      return;
    }

    const rendered = this.jobElements?.length ?? 0;
    if (rendered < this.jobs.length) {
      // Aguarde o DOM renderizar os jobs e deixe o watcher disparar o rebuild.
      this.pendingRebuild = true;
      this.pendingRebuildKey = expectedKey;
      return;
    }

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
    this.addExistingConnections(rebuildNonce, () => {
      if (this.activeRebuildNonce !== rebuildNonce) return;
      this.instance.repaintEverything();
      requestAnimationFrame(() => this.instance?.repaintEverything());
      this.suppressConnectionEvents = false;
      this.lastRebuildKey = expectedKey;
      this.pendingRebuild = false;
      this.pendingRebuildKey = null;
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
        const conns = this.project.connections || [];
        const exists = conns.some(
          (conn) => conn.source === info.sourceId && conn.target === info.targetId
        );
        if (exists) {
          return;
        }
        conns.push({
          source: info.sourceId,
          target: info.targetId
        });
        this.project.connections = conns;
        this.saveProject();
      }
    });

    this.instance.bind('connectionDetached', (info: any) => {
      if (this.suppressConnectionEvents || this.isLoading) return;
      if (!this.project || !this.project.connections) return;
      const original = this.project.connections.length;
      this.project.connections = this.project.connections.filter(
        (conn) => !(conn.source === info.sourceId && conn.target === info.targetId)
      );
      if (this.project.connections.length !== original) {
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

    this.instance.draggable(id, {
      stop: (params: any) => {
        this.isSaving = true;
        const el = params.el;
        const [x, y] = this.snapToGrid(
          parseInt(el.style.left || '0', 10),
          parseInt(el.style.top || '0', 10)
        );
        el.style.left = `${x}px`;
        el.style.top = `${y}px`;
        this.instance.revalidate(el);

        const movedJob = this.jobs.find(j => j.id === el.id);
        if (movedJob) {
          const before = this.getJobState(movedJob);
          movedJob.left = x;
          movedJob.top = y;
          const after = this.getJobState(movedJob);
          const globalIndex = jobs_.findIndex(j => j.id === movedJob.id);
          if (globalIndex !== -1) {
            jobs_[globalIndex] = { ...jobs_[globalIndex], left: movedJob.left, top: movedJob.top };
          }
          if (this.hasJobChanged(before, after)) {
            this.pushUndoEntry({ jobs: [{ id: movedJob.id, before, after }], visuals: [] });
          }

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

  addExistingConnections(rebuildNonce: number, onDone?: () => void) {
    if (!this.instance || !this.project?.connections) {
      if (onDone) onDone();
      return;
    }

    const isStale = () => this.activeRebuildNonce !== rebuildNonce;
    const seen = new Set<string>();
    const conns = (this.project.connections || []).filter((conn) => {
      const key = `${conn.source}->${conn.target}`;
      if (seen.has(key)) return false;
      seen.add(key);
      return true;
    });
    const chunkSize = 25;
    let index = 0;

    const connectChunk = () => {
      if (isStale()) return;
      if (!this.instance) return;
      const batch = conns.slice(index, index + chunkSize);
      if (batch.length === 0) {
        if (!isStale()) {
          this.isLoading = false;
          if (onDone) onDone();
        }
        return;
      }

      this.instance.batch(() => {
        batch.forEach((conn) => {
          this.instance.connect({
            source: conn.source,
            target: conn.target,
            anchors: ['Right', 'Left'],
            connector: ['Flowchart', { stub: 30, gap: 8, cornerRadius: 8, alwaysRespectStubs: true }],
            overlays: [['Arrow', { width: 10, length: 10, location: 1 }]]
          });
        });
      });

      index += chunkSize;
      requestAnimationFrame(connectChunk);
    };

    connectChunk();
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
            const remaining = jobs_.filter(j => j.id !== job.id);
            jobs_.length = 0;
            jobs_.push(...remaining);
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
      this.lastSavedAt = new Date();
    });
  }

  resumeJob(job: JobExtended | null) {
    if (!job || !this.project?.id) return;
    this.isSaving = true;
    this.jobService.resumeJob(this.project.id, job.id).subscribe({
      next: () => {
        this.isSaved();
      },
      error: (error) => {
        this.notifyPersistError('Erro ao retomar o job.', error);
        this.isSaved();
      }
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
              const globalIndex = jobs_.findIndex(j => j.id === updatedJob.id);
              if (globalIndex !== -1) {
                jobs_[globalIndex] = { ...jobs_[globalIndex], ...updatedJob };
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
    const [sx, sy] = this.snapToGrid(x, y);
    this.isSaving = true;
    const base: VisualElement = {
      type,
      x: sx,
      y: sy,
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
      const [sx2, sy2] = this.snapToGrid(sx + 200, sy);
      base.x2 = sx2;
      base.y2 = sy2;
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
    if (this.isSpacePressed) {
      this.startPanning(event);
      return;
    }
    if (event.ctrlKey) {
      this.toggleElementSelection(element, true);
      return;
    }
    const alreadySelected = this.isElementSelected(element);
    if (!alreadySelected) {
      this.setSelection([element]);
    }
    const totalSelected = this.selectedVisualElements.length + this.selectedJobs.length;
    if (totalSelected > 1) {
      this.startGroupDrag(event);
      return;
    }
    this.lastDragWasElement = true;
    this.draggingElementId = this.getElementId(element) || null;
    this.dragStartX = event.clientX;
    this.dragStartY = event.clientY;
    this.dragOriginX = element.x;
    this.dragOriginY = element.y;
    this.dragOriginX2 = element.x2 || 0;
    this.dragOriginY2 = element.y2 || 0;
    this.groupDragOrigins.clear();

    const onMove = (moveEvent: MouseEvent) => {
      if (!this.draggingElementId) return;
      const dx = moveEvent.clientX - this.dragStartX;
      const dy = moveEvent.clientY - this.dragStartY;
      const scale = this.zoom || 1;
      const target = this.visualElements.find(e => e.id === this.draggingElementId);
      if (!target) return;
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
          if (updated.type === 'line') {
            const [sx, sy] = this.snapToGrid(updated.x, updated.y);
            updated.x = sx;
            updated.y = sy;
            if (updated.x2 !== undefined && updated.y2 !== undefined) {
              const [sx2, sy2] = this.snapToGrid(updated.x2, updated.y2);
              updated.x2 = sx2;
              updated.y2 = sy2;
            }
          } else {
            const [sx, sy] = this.snapToGrid(updated.x, updated.y);
            updated.x = sx;
            updated.y = sy;
          }
          this.normalizeElement(updated);
          const before = this.getVisualStateWithOverrides(updated, {
            x: this.dragOriginX,
            y: this.dragOriginY,
            x2: updated.type === 'line' ? this.dragOriginX2 : undefined,
            y2: updated.type === 'line' ? this.dragOriginY2 : undefined
          });
          const after = this.getVisualState(updated);
          const elementId = this.getElementId(updated);
          if (elementId && this.hasVisualChanged(before, after)) {
            this.pushUndoEntry({ jobs: [], visuals: [{ id: elementId, before, after }] });
          }
          this.persistVisualElement(updated, 'arraste');
        }
      }
      this.draggingElementId = null;
      this.lastDragWasElement = false;
      window.removeEventListener('mousemove', onMove);
      window.removeEventListener('mouseup', onUp);
      this.activeMoveHandler = null;
      this.activeUpHandler = null;
      this.groupDragOrigins.clear();
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
          const [sx, sy] = this.snapToGrid(updated.x, updated.y);
          updated.x = sx;
          updated.y = sy;
          if (updated.type !== 'line') {
            const snappedW = Math.round(this.getElementWidth(updated) / this.gridX) * this.gridX;
            const snappedH = Math.round(this.getElementHeight(updated) / this.gridY) * this.gridY;
            if (updated.type === 'circle') {
              const size = Math.max(snappedW || this.gridX, snappedH || this.gridY);
              updated.width = size;
              updated.height = size;
            } else {
              updated.width = Math.max(40, snappedW || this.gridX);
              updated.height = Math.max(40, snappedH || this.gridY);
            }
          }
          this.normalizeElement(updated);
          const elementId = this.getElementId(updated);
          const before = this.getVisualStateWithOverrides(updated, {
            x: this.resizeOriginX,
            y: this.resizeOriginY,
            width: this.resizeOriginW,
            height: this.resizeOriginH
          });
          const after = this.getVisualState(updated);
          if (elementId && this.hasVisualChanged(before, after)) {
            this.pushUndoEntry({ jobs: [], visuals: [{ id: elementId, before, after }] });
          }
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
    if (this.isSpacePressed) {
      this.startPanning(event);
      return;
    }
    const alreadySelected = this.isElementSelected(element);
    if (!alreadySelected) {
      this.setSelection([element]);
    }
    const totalSelected = this.selectedVisualElements.length + this.selectedJobs.length;
    if (totalSelected > 1) {
      this.startGroupDrag(event);
      return;
    }
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
          const [sx, sy] = this.snapToGrid(updated.x, updated.y);
          updated.x = sx;
          updated.y = sy;
          if (updated.x2 !== undefined && updated.y2 !== undefined) {
            const [sx2, sy2] = this.snapToGrid(updated.x2, updated.y2);
            updated.x2 = sx2;
            updated.y2 = sy2;
          }
          this.normalizeElement(updated);
          const elementId = this.getElementId(updated);
          const before = this.getVisualStateWithOverrides(updated, {
            x: this.dragOriginX,
            y: this.dragOriginY,
            x2: this.dragOriginX2,
            y2: this.dragOriginY2
          });
          const after = this.getVisualState(updated);
          if (elementId && this.hasVisualChanged(before, after)) {
            this.pushUndoEntry({ jobs: [], visuals: [{ id: elementId, before, after }] });
          }
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
    if (event.ctrlKey) {
      this.toggleElementSelection(element, true);
      return;
    }
    this.setSelection([element]);
  }

  openElementPanel(element: VisualElement, event: MouseEvent) {
    event.stopPropagation();
    this.setSelection([element]);
    this.panelElement = element;
  }

  onJobMouseDown(event: MouseEvent, job: JobExtended) {
    if (event.button !== 0) return;
    event.stopPropagation();
    if (this.isSpacePressed) {
      this.startPanning(event);
      return;
    }
    if (event.ctrlKey) {
      this.toggleJobSelection(job, true);
      return;
    }
    const alreadySelected = this.isJobSelected(job);
    if (!alreadySelected) {
      this.setJobSelection([job]);
    }
    const totalSelected = this.selectedVisualElements.length + this.selectedJobs.length;
    if (totalSelected > 1) {
      event.preventDefault();
      this.startGroupDrag(event);
    }
  }

  clearSelection() {
    this.selectedVisualElement = null;
    this.selectedVisualElements = [];
    this.selectedJobs = [];
    this.panelElement = null;
  }

  private setSelection(elements: VisualElement[], preserveJobs = false) {
    this.selectedVisualElements = elements;
    if (!preserveJobs) {
      this.selectedJobs = [];
    }
    this.updateSelectedVisualElement();
  }

  private setJobSelection(jobs: JobExtended[], preserveVisuals = false) {
    this.selectedJobs = jobs;
    if (!preserveVisuals) {
      this.selectedVisualElements = [];
    }
    this.updateSelectedVisualElement();
  }

  private toggleElementSelection(element: VisualElement, preserveJobs = false) {
    const id = this.getElementId(element);
    if (!id) return;
    const exists = this.selectedVisualElements.some((el) => this.getElementId(el) === id);
    if (exists) {
      this.selectedVisualElements = this.selectedVisualElements.filter((el) => this.getElementId(el) !== id);
    } else {
      this.selectedVisualElements = [...this.selectedVisualElements, element];
    }
    if (!preserveJobs) {
      this.selectedJobs = [];
    }
    this.updateSelectedVisualElement();
  }

  private toggleJobSelection(job: JobExtended, preserveVisuals = false) {
    const exists = this.selectedJobs.some((j) => j.id === job.id);
    if (exists) {
      this.selectedJobs = this.selectedJobs.filter((j) => j.id !== job.id);
    } else {
      this.selectedJobs = [...this.selectedJobs, job];
    }
    if (!preserveVisuals) {
      this.selectedVisualElements = [];
    }
    this.updateSelectedVisualElement();
  }

  private updateSelectedVisualElement() {
    this.panelElement = null;
    if (this.selectedVisualElements.length === 1 && this.selectedJobs.length === 0) {
      this.selectedVisualElement = this.selectedVisualElements[0];
      return;
    }
    this.selectedVisualElement = null;
  }

  isElementSelected(element: VisualElement): boolean {
    const id = this.getElementId(element);
    return !!this.selectedVisualElements.find((el) => this.getElementId(el) === id);
  }

  isJobSelected(job: JobExtended): boolean {
    return !!this.selectedJobs.find((j) => j.id === job.id);
  }

  private pushUndoEntry(entry: {
    jobs: Array<{ id: string; before: { left: number; top: number }; after: { left: number; top: number } }>;
    visuals: Array<{
      id: string;
      before: { x: number; y: number; x2?: number; y2?: number; width?: number; height?: number };
      after: { x: number; y: number; x2?: number; y2?: number; width?: number; height?: number };
    }>;
  }) {
    if (this.isApplyingUndo) return;
    if (!entry.jobs.length && !entry.visuals.length) return;
    this.undoStack.push(entry);
    if (this.undoStack.length > this.maxUndoEntries) {
      this.undoStack.shift();
    }
  }

  private getJobState(job: JobExtended): { left: number; top: number } {
    return { left: job.left ?? 0, top: job.top ?? 0 };
  }

  private getVisualState(element: VisualElement): { x: number; y: number; x2?: number; y2?: number; width?: number; height?: number } {
    return {
      x: element.x,
      y: element.y,
      x2: element.x2,
      y2: element.y2,
      width: element.width,
      height: element.height
    };
  }

  private getVisualStateWithOverrides(
    element: VisualElement,
    overrides: Partial<{ x: number; y: number; x2?: number; y2?: number; width?: number; height?: number }>
  ) {
    return { ...this.getVisualState(element), ...overrides };
  }

  private hasJobChanged(before: { left: number; top: number }, after: { left: number; top: number }): boolean {
    return before.left !== after.left || before.top !== after.top;
  }

  private hasVisualChanged(
    before: { x: number; y: number; x2?: number; y2?: number; width?: number; height?: number },
    after: { x: number; y: number; x2?: number; y2?: number; width?: number; height?: number }
  ): boolean {
    return (
      before.x !== after.x ||
      before.y !== after.y ||
      before.x2 !== after.x2 ||
      before.y2 !== after.y2 ||
      before.width !== after.width ||
      before.height !== after.height
    );
  }

  private undoLast(): boolean {
    const entry = this.undoStack.pop();
    if (!entry) return false;
    this.isApplyingUndo = true;
    this.isSaving = true;

    entry.visuals.forEach((change) => {
      const element = this.visualElements.find((el) => this.getElementId(el) === change.id);
      if (!element) return;
      element.x = change.before.x;
      element.y = change.before.y;
      if (change.before.x2 !== undefined) element.x2 = change.before.x2;
      if (change.before.y2 !== undefined) element.y2 = change.before.y2;
      if (change.before.width !== undefined) element.width = change.before.width;
      if (change.before.height !== undefined) element.height = change.before.height;
      this.normalizeElement(element);
      this.persistVisualElement(element, 'undo');
    });

    entry.jobs.forEach((change) => {
      const job = this.jobs.find((j) => j.id === change.id);
      if (!job) return;
      job.left = change.before.left;
      job.top = change.before.top;
      const globalIndex = jobs_.findIndex(j => j.id === job.id);
      if (globalIndex !== -1) {
        jobs_[globalIndex] = { ...jobs_[globalIndex], left: job.left, top: job.top };
      }
      this.instance?.revalidate(job.id);
      this.jobService.updateJob(this.project?.id || '', job.id, job).subscribe({
        next: () => {
          this.isSaved();
        },
        error: () => {
          this.isSaved();
        }
      });
    });

    this.instance?.repaintEverything();
    this.cdr.detectChanges();
    this.isApplyingUndo = false;
    return true;
  }

  private getDuplicateOffsets(): { x: number; y: number } {
    if (this.snapEnabled) {
      return { x: this.gridX, y: this.gridY };
    }
    return { x: 24, y: 24 };
  }

  private duplicateSelection(): boolean {
    if (!this.project?.id) {
      this.notifyPersistError('Projeto ainda nao esta carregado. Nao foi possivel duplicar.');
      return false;
    }
    if (this.selectedVisualElements.length === 1 && this.selectedJobs.length === 0) {
      this.duplicateVisualElement(this.selectedVisualElements[0]);
      return true;
    }
    if (this.selectedJobs.length === 1 && this.selectedVisualElements.length === 0) {
      this.duplicateJob(this.selectedJobs[0]);
      return true;
    }
    return false;
  }

  private duplicateVisualElement(element: VisualElement) {
    if (!this.project?.id) return;
    const offset = this.getDuplicateOffsets();
    const cloned: VisualElement = {
      ...element,
      id: undefined,
      elementId: undefined,
      x: element.x + offset.x,
      y: element.y + offset.y,
      x2: element.x2 !== undefined ? element.x2 + offset.x : undefined,
      y2: element.y2 !== undefined ? element.y2 + offset.y : undefined
    };
    this.isSaving = true;
    const payload = this.toPayload(cloned);
    this.visualElementService.create(this.project.id, payload).subscribe({
      next: (created) => {
        const normalized = this.normalizeElement(created);
        this.visualElements = [...this.visualElements, normalized];
        this.setSelection([normalized]);
        this.panelElement = null;
        this.isSaved();
      },
      error: (error) => {
        this.notifyPersistError('Erro ao duplicar elemento visual.', error);
        this.isSaved();
      }
    });
  }

  private duplicateJob(job: JobExtended) {
    if (!this.project?.id) return;
    const offset = this.getDuplicateOffsets();
    const newJob: Job = {
      id: uuidv4(),
      jobName: job.jobName,
      selectSql: job.selectSql,
      insertSql: job.insertSql,
      posInsertSql: job.posInsertSql,
      columns: Array.isArray(job.columns) ? [...job.columns] : [],
      recordsPerPage: job.recordsPerPage,
      type: job.type,
      stopOnError: job.stopOnError,
      top: (job.top ?? 0) + offset.y,
      left: (job.left ?? 0) + offset.x
    };

    this.isSaving = true;
    this.jobs.push(newJob as JobExtended);
    jobs_.push(newJob as JobExtended);
    if (this.project) {
      this.project.jobs = this.jobs.map(j => `jobs/${j.id}.json`);
    }
    this.scheduleJobPlumbAttach(newJob.id);
    this.setJobSelection([newJob as JobExtended]);
    this.selectedJob = newJob as JobExtended;

    this.jobService.addJob(this.project.id, newJob).subscribe({
      next: () => {
        this.saveProject();
      },
      error: (error) => {
        this.notifyPersistError('Erro ao duplicar job.', error);
        this.isSaved();
      }
    });
  }

  private toDiagramPoint(event: MouseEvent): { x: number; y: number } {
    const rect = this.scrollContainer.nativeElement.getBoundingClientRect();
    const scale = this.zoom || 1;
    const x = (event.clientX - rect.left - this.viewOffsetX) / scale;
    const y = (event.clientY - rect.top - this.viewOffsetY) / scale;
    return { x, y };
  }

  private getElementBounds(element: VisualElement): { left: number; top: number; right: number; bottom: number } {
    if (element.type === 'line') {
      const x1 = element.x;
      const y1 = element.y;
      const x2 = element.x2 ?? element.x;
      const y2 = element.y2 ?? element.y;
      return {
        left: Math.min(x1, x2),
        top: Math.min(y1, y2),
        right: Math.max(x1, x2),
        bottom: Math.max(y1, y2)
      };
    }
    const width = this.getElementWidth(element);
    const height = this.getElementHeight(element);
    return {
      left: element.x,
      top: element.y,
      right: element.x + width,
      bottom: element.y + height
    };
  }

  private isElementInSelection(element: VisualElement, rect: { x: number; y: number; width: number; height: number }): boolean {
    const bounds = this.getElementBounds(element);
    const rectLeft = rect.x;
    const rectTop = rect.y;
    const rectRight = rect.x + rect.width;
    const rectBottom = rect.y + rect.height;
    return !(bounds.right < rectLeft || bounds.left > rectRight || bounds.bottom < rectTop || bounds.top > rectBottom);
  }

  private getJobBounds(job: JobExtended): { left: number; top: number; right: number; bottom: number } {
    const left = job.left ?? 0;
    const top = job.top ?? 0;
    return {
      left,
      top,
      right: left + this.jobBoxWidth,
      bottom: top + this.jobBoxHeight
    };
  }

  private isJobInSelection(job: JobExtended, rect: { x: number; y: number; width: number; height: number }): boolean {
    const bounds = this.getJobBounds(job);
    const rectLeft = rect.x;
    const rectTop = rect.y;
    const rectRight = rect.x + rect.width;
    const rectBottom = rect.y + rect.height;
    return !(bounds.right < rectLeft || bounds.left > rectRight || bounds.bottom < rectTop || bounds.top > rectBottom);
  }

  onDiagramBackgroundClick(event: MouseEvent) {
    if (this.lastDragWasElement) return;
    if (this.suppressBackgroundClick) {
      this.suppressBackgroundClick = false;
      return;
    }
    const target = event.target as HTMLElement | null;
    if (!target) return;
    if (target.closest('.visual-panel, .visual-element, .box, .mat-mdc-form-field, .visual-lines')) {
      return;
    }
    if (!this.isSelecting) {
      this.clearSelection();
    }
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
        if (this.panelElement && this.getElementId(this.panelElement) === elementId) {
          this.panelElement = null;
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

  toggleSnap() {
    this.snapEnabled = !this.snapEnabled;
    if (this.isBrowser) {
      localStorage.setItem('diagramSnap', this.snapEnabled.toString());
    }
  }

  fitToScreen() {
    const container = this.scrollContainer?.nativeElement;
    if (!container) return;

    const bounds = this.getContentBounds();
    if (!bounds) {
      this.removeZoom();
      this.viewOffsetX = 0;
      this.viewOffsetY = 0;
      this.cdr.detectChanges();
      return;
    }

    const padding = 80;
    const width = Math.max(1, bounds.maxX - bounds.minX);
    const height = Math.max(1, bounds.maxY - bounds.minY);
    const availableW = Math.max(1, container.clientWidth - padding * 2);
    const availableH = Math.max(1, container.clientHeight - padding * 2);
    const nextZoom = Math.min(this.maxZoom, Math.max(this.minZoom, Math.min(availableW / width, availableH / height)));

    const centerX = bounds.minX + width / 2;
    const centerY = bounds.minY + height / 2;
    this.zoom = nextZoom;
    if (this.instance) {
      this.instance.setZoom(this.zoom);
    }
    this.viewOffsetX = container.clientWidth / 2 - this.zoom * centerX;
    this.viewOffsetY = container.clientHeight / 2 - this.zoom * centerY;
    if (this.isBrowser) {
      localStorage.setItem('diagramZoom', this.zoom.toString());
      localStorage.setItem('diagramOffset', JSON.stringify({ x: this.viewOffsetX, y: this.viewOffsetY }));
    }
    this.cdr.detectChanges();
  }

  private getContentBounds(): { minX: number; minY: number; maxX: number; maxY: number } | null {
    const items: Array<{ minX: number; minY: number; maxX: number; maxY: number }> = [];

    this.jobs.forEach((job) => {
      const left = job.left ?? 0;
      const top = job.top ?? 0;
      items.push({
        minX: left,
        minY: top,
        maxX: left + this.jobBoxWidth,
        maxY: top + this.jobBoxHeight
      });
    });

    this.visualElements.forEach((el) => {
      const bounds = this.getElementBounds(el);
      items.push({
        minX: bounds.left,
        minY: bounds.top,
        maxX: bounds.right,
        maxY: bounds.bottom
      });
    });

    if (!items.length) return null;

    return items.reduce(
      (acc, curr) => ({
        minX: Math.min(acc.minX, curr.minX),
        minY: Math.min(acc.minY, curr.minY),
        maxX: Math.max(acc.maxX, curr.maxX),
        maxY: Math.max(acc.maxY, curr.maxY)
      }),
      { minX: items[0].minX, minY: items[0].minY, maxX: items[0].maxX, maxY: items[0].maxY }
    );
  }

  formatSavedAt(): string {
    if (!this.lastSavedAt) return '';
    return this.lastSavedAt.toLocaleTimeString('pt-BR', { hour: '2-digit', minute: '2-digit' });
  }

  getStatusCounts() {
    const jobs = this.jobs ?? [];
    const running = jobs.filter(j => j.status === 'running').length;
    const done = jobs.filter(j => j.status === 'done').length;
    const error = jobs.filter(j => j.status === 'error').length;
    const pending = jobs.filter(j => !j.status || j.status === 'pending').length;
    const total = jobs.length;
    return { running, done, error, pending, total };
  }

  getPercent(value?: number | null, total?: number | null): number {
    if (value == null || total == null || total <= 0) return 0;
    const percent = (value / total) * 100;
    if (!Number.isFinite(percent)) return 0;
    return Math.min(100, Math.max(0, percent));
  }

  isEmptyProgress(value?: number | null, total?: number | null): boolean {
    return (value ?? 0) <= 0 && (total ?? 0) <= 0;
  }

  onMouseDown(e: MouseEvent): void {
    const target = e.target as HTMLElement | null;
    if (target && (target.closest('.visual-element, .resize-handle, .line-handle, .box, .visual-panel, .context-menu, .mat-mdc-form-field') || target.tagName === 'line')) {
      return;
    }
    if (this.isSpacePressed) {
      this.startPanning(e);
      return;
    }
    this.isSelecting = true;
    this.selectionMoved = false;
    const point = this.toDiagramPoint(e);
    this.selectionStartX = point.x;
    this.selectionStartY = point.y;
    this.selectionRect = { x: point.x, y: point.y, width: 0, height: 0 };
  }

  onMouseMove(e: MouseEvent): void {
    if (this.isPanning) {
      e.preventDefault();
      const dx = e.clientX - this.panStartX;
      const dy = e.clientY - this.panStartY;
      this.viewOffsetX = this.panOriginX + dx;
      this.viewOffsetY = this.panOriginY + dy;
      return;
    }
    if (!this.isSelecting) return;
    const point = this.toDiagramPoint(e);
    const x1 = this.selectionStartX;
    const y1 = this.selectionStartY;
    const x2 = point.x;
    const y2 = point.y;
    const minX = Math.min(x1, x2);
    const minY = Math.min(y1, y2);
    const maxX = Math.max(x1, x2);
    const maxY = Math.max(y1, y2);
    this.selectionRect = { x: minX, y: minY, width: maxX - minX, height: maxY - minY };
    this.selectionMoved = this.selectionRect.width > 4 || this.selectionRect.height > 4;
  }

  onMouseUp(): void {
    if (this.isPanning) {
      localStorage.setItem('diagramOffset', JSON.stringify({ x: this.viewOffsetX, y: this.viewOffsetY }));
    }
    this.isPanning = false;
    this.scrollContainer.nativeElement.style.cursor = this.isSpacePressed ? 'grab' : 'default';
    if (!this.isSelecting) return;
    this.isSelecting = false;
    if (!this.selectionMoved) {
      this.selectionRect = { x: 0, y: 0, width: 0, height: 0 };
      this.clearSelection();
      return;
    }
    const selected = this.visualElements.filter((el) => this.isElementInSelection(el, this.selectionRect));
    const selectedJobs = this.jobs.filter((job) => this.isJobInSelection(job, this.selectionRect));
    this.setSelection(selected, true);
    this.setJobSelection(selectedJobs, true);
    this.suppressBackgroundClick = true;
    this.selectionRect = { x: 0, y: 0, width: 0, height: 0 };
  }

  private startPanning(e: MouseEvent) {
    this.isPanning = true;
    const container = this.scrollContainer.nativeElement;
    this.panStartX = e.clientX;
    this.panStartY = e.clientY;
    this.panOriginX = this.viewOffsetX;
    this.panOriginY = this.viewOffsetY;
    container.style.cursor = 'grabbing';
  }

  private startGroupDrag(event: MouseEvent) {
    this.isGroupDragging = true;
    this.lastDragWasElement = true;
    this.dragStartX = event.clientX;
    this.dragStartY = event.clientY;
    this.groupDragOrigins.clear();
    this.groupJobOrigins.clear();

    this.selectedVisualElements.forEach((el) => {
      const id = this.getElementId(el);
      if (!id) return;
      this.groupDragOrigins.set(id, {
        x: el.x,
        y: el.y,
        x2: el.x2,
        y2: el.y2
      });
    });

    this.selectedJobs.forEach((job) => {
      this.groupJobOrigins.set(job.id, {
        left: job.left ?? 0,
        top: job.top ?? 0
      });
    });

    const onMove = (moveEvent: MouseEvent) => {
      if (!this.isGroupDragging) return;
      const dx = moveEvent.clientX - this.dragStartX;
      const dy = moveEvent.clientY - this.dragStartY;
      const scale = this.zoom || 1;

      this.selectedVisualElements.forEach((el) => {
        const id = this.getElementId(el);
        if (!id) return;
        const origin = this.groupDragOrigins.get(id);
        if (!origin) return;
        el.x = origin.x + dx / scale;
        el.y = origin.y + dy / scale;
        if (el.type === 'line' && el.x2 !== undefined && el.y2 !== undefined) {
          el.x2 = (origin.x2 || 0) + dx / scale;
          el.y2 = (origin.y2 || 0) + dy / scale;
        }
        this.normalizeElement(el);
      });

      this.selectedJobs.forEach((job) => {
        const origin = this.groupJobOrigins.get(job.id);
        if (!origin) return;
        job.left = origin.left + dx / scale;
        job.top = origin.top + dy / scale;
      });

      if (this.instance && this.groupRepaintRaf === null) {
        this.groupRepaintRaf = requestAnimationFrame(() => {
          this.groupRepaintRaf = null;
          this.instance.repaintEverything();
        });
      }
    };

    const onUp = () => {
      if (this.isGroupDragging) {
        this.selectedVisualElements.forEach((el) => {
          if (el.type === 'line') {
            const [sx, sy] = this.snapToGrid(el.x, el.y);
            el.x = sx;
            el.y = sy;
            if (el.x2 !== undefined && el.y2 !== undefined) {
              const [sx2, sy2] = this.snapToGrid(el.x2, el.y2);
              el.x2 = sx2;
              el.y2 = sy2;
            }
          } else {
            const [sx, sy] = this.snapToGrid(el.x, el.y);
            el.x = sx;
            el.y = sy;
          }
          this.normalizeElement(el);
          this.persistVisualElement(el, 'arraste');
        });

        this.selectedJobs.forEach((job) => {
          const [sx, sy] = this.snapToGrid(job.left ?? 0, job.top ?? 0);
          job.left = sx;
          job.top = sy;
          const globalIndex = jobs_.findIndex(j => j.id === job.id);
          if (globalIndex !== -1) {
            jobs_[globalIndex] = { ...jobs_[globalIndex], left: job.left, top: job.top };
          }
          this.instance?.revalidate(job.id);
          this.jobService.updateJob(this.project?.id || '', job.id, job).subscribe({
            next: () => {
              this.isSaved();
            },
            error: () => {
              this.isSaved();
            }
          });
        });

        const visualChanges: Array<{
          id: string;
          before: { x: number; y: number; x2?: number; y2?: number; width?: number; height?: number };
          after: { x: number; y: number; x2?: number; y2?: number; width?: number; height?: number };
        }> = [];
        const jobChanges: Array<{ id: string; before: { left: number; top: number }; after: { left: number; top: number } }> = [];

        this.selectedVisualElements.forEach((el) => {
          const id = this.getElementId(el);
          if (!id) return;
          const origin = this.groupDragOrigins.get(id);
          if (!origin) return;
          const before = this.getVisualStateWithOverrides(el, {
            x: origin.x,
            y: origin.y,
            x2: origin.x2,
            y2: origin.y2
          });
          const after = this.getVisualState(el);
          if (this.hasVisualChanged(before, after)) {
            visualChanges.push({ id, before, after });
          }
        });

        this.selectedJobs.forEach((job) => {
          const origin = this.groupJobOrigins.get(job.id);
          if (!origin) return;
          const before = { left: origin.left, top: origin.top };
          const after = this.getJobState(job);
          if (this.hasJobChanged(before, after)) {
            jobChanges.push({ id: job.id, before, after });
          }
        });

        if (visualChanges.length || jobChanges.length) {
          this.pushUndoEntry({ jobs: jobChanges, visuals: visualChanges });
        }

        if (this.instance) {
          this.instance.repaintEverything();
        }
      }

      this.isGroupDragging = false;
      this.groupDragOrigins.clear();
      this.groupJobOrigins.clear();
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
