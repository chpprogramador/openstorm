// src/app/core/services/project.service.ts
import { Injectable, PLATFORM_ID, inject } from '@angular/core';
import { isPlatformBrowser } from '@angular/common';
import { HttpClient } from '@angular/common/http';
import { BehaviorSubject, Observable, of } from 'rxjs';
import { tap, catchError } from 'rxjs/operators';
import { Project } from '../models/project.model';
import { environment } from '../../../environments/environment';

@Injectable({
    providedIn: 'root'
})
export class ProjectService {
    private platformId = inject(PLATFORM_ID);
    private http = inject(HttpClient);
    private isBrowser: boolean;
    private apiUrl = `${environment.apiUrl}/projects`;

    private selectedProjectSubject = new BehaviorSubject<Project | null>(null);
    public selectedProject$ = this.selectedProjectSubject.asObservable();

    private projectsSubject = new BehaviorSubject<Project[]>([]);
    public projects$ = this.projectsSubject.asObservable();

    // Flag para determinar se deve usar API ou localStorage
    private useApi = true; // TRUE - Usa API

    constructor() {
        this.isBrowser = isPlatformBrowser(this.platformId);
        this.loadProjects();
        this.loadSelectedProject();
    }

    private loadSelectedProject(): void {
        if (!this.isBrowser) {
            return;
        }

        const stored = localStorage.getItem('quasar_selected_project');
        if (stored) {
            try {
                const parsed = JSON.parse(stored);
                this.selectedProjectSubject.next(parsed);
            } catch {
                localStorage.removeItem('quasar_selected_project');
            }
        }
    }

    private loadProjects(): void {
        // if (!this.isBrowser) {
        //     console.log('⚠️ Não está no browser, pulando carregamento');
        //     return;
        // }

        console.log('🔧 useApi:', this.useApi);

        if (this.useApi) {
            console.log('📡 Buscando projetos da API:', this.apiUrl);
            this.listProjects().subscribe({
                next: (projects) => {
                    console.log('✅ Projetos recebidos da API:', projects);
                    this.projectsSubject.next(projects);
                },
                error: (error) => {
                    console.error('❌ Erro na API:', error);
                    this.loadFromLocalStorage();
                }
            });
        } else {
            console.log('💾 Carregando do localStorage');
            this.loadFromLocalStorage();
        }
    }

    private loadFromLocalStorage(): void {
        if (!this.isBrowser) {
            return;
        }

        const stored = localStorage.getItem('quasar_projects');
        if (stored) {
            const projects = JSON.parse(stored);
            this.projectsSubject.next(projects);
        } else {
            // Projetos de exemplo
            const defaultProjects: Project[] = [
                {
                    id: '1',
                    name: 'Projeto Alpha',
                    description: 'Projeto de demonstração',
                    createdAt: new Date(),
                    updatedAt: new Date()
                }
            ];
            this.projectsSubject.next(defaultProjects);
            this.saveToLocalStorage(defaultProjects);
        }
    }

    private saveToLocalStorage(projects: Project[]): void {
        if (!this.isBrowser) {
            return;
        }
        localStorage.setItem('quasar_projects', JSON.stringify(projects));
    }

    /**
     * Lista todos os projetos (API)
     */
    listProjects(): Observable<Project[]> {
        console.log('🚀 Fazendo requisição GET para:', this.apiUrl);
        return this.http.get<any[]>(this.apiUrl, {
            headers: { 'Content-Type': 'application/json' }
        }).pipe(
            tap(projects => {
                console.log('📥 Resposta da API (raw):', projects);
                // Mapeia projectName para name e adiciona campos faltantes
                const mappedProjects = projects.map(p => ({
                    id: p.id,
                    name: p.projectName || p.name,
                    description: p.description || '',
                    createdAt: p.createdAt ? new Date(p.createdAt) : new Date(),
                    updatedAt: p.updatedAt ? new Date(p.updatedAt) : new Date(),
                    // Mantém campos do legado
                    projectName: p.projectName,
                    jobs: p.jobs || [],
                    connections: p.connections || [],
                    sourceDatabase: p.sourceDatabase,
                    destinationDatabase: p.destinationDatabase,
                    concurrency: p.concurrency,
                    variables: p.variables,
                    visualElements: p.visualElements
                }));
                console.log('🗺️ Projetos mapeados:', mappedProjects);
                this.projectsSubject.next(mappedProjects);
            }),
            catchError((error) => {
                console.warn('⚠️ Falha ao carregar projetos da API, usando localStorage:', error);
                this.loadFromLocalStorage();
                return of(this.projectsSubject.value);
            })
        );
    }

    /**
     * Busca um projeto por ID (API)
     */
    getProject(id: string): Observable<Project> {
        return this.http.get<any>(`${this.apiUrl}/${id}`).pipe(
            tap(p => {
                // Mapeia projectName para name
                const mappedProject = {
                    id: p.id,
                    name: p.projectName || p.name,
                    description: p.description || '',
                    createdAt: p.createdAt ? new Date(p.createdAt) : new Date(),
                    updatedAt: p.updatedAt ? new Date(p.updatedAt) : new Date(),
                    projectName: p.projectName,
                    jobs: p.jobs || [],
                    connections: p.connections || [],
                    sourceDatabase: p.sourceDatabase,
                    destinationDatabase: p.destinationDatabase,
                    concurrency: p.concurrency,
                    variables: p.variables,
                    visualElements: p.visualElements
                };
                return mappedProject;
            })
        );
    }

    /**
     * Retorna os projetos locais
     */
    getProjects(): Project[] {
        return this.projectsSubject.value;
    }

    /**
     * Retorna o projeto selecionado
     */
    getSelectedProject(): Project | null {
        return this.selectedProjectSubject.value;
    }

    /**
     * Seleciona um projeto
     */
    selectProject(project: Project): void {
        this.selectedProjectSubject.next(project);
        if (this.isBrowser) {
            localStorage.setItem('quasar_selected_project', JSON.stringify(project));
        }
    }

    /**
     * Cria um novo projeto
     */
    createProject(project: Omit<Project, 'id' | 'createdAt' | 'updatedAt'>): Observable<Project> | Project {
        const newProject: any = {
            projectName: project.name, // Backend espera projectName
            description: project.description,
            jobs: project.jobs || [],
            connections: project.connections || [],
            sourceDatabase: project.sourceDatabase,
            destinationDatabase: project.destinationDatabase,
            concurrency: project.concurrency || 10,
            variables: project.variables || null,
            visualElements: project.visualElements || null
        };

        if (this.useApi) {
            return this.http.post<any>(this.apiUrl, newProject).pipe(
                tap(createdProject => {
                    // Mapeia resposta do backend
                    const mapped = {
                        id: createdProject.id,
                        name: createdProject.projectName || createdProject.name,
                        description: createdProject.description || '',
                        createdAt: new Date(),
                        updatedAt: new Date(),
                        projectName: createdProject.projectName,
                        jobs: createdProject.jobs || [],
                        connections: createdProject.connections || [],
                        sourceDatabase: createdProject.sourceDatabase,
                        destinationDatabase: createdProject.destinationDatabase,
                        concurrency: createdProject.concurrency,
                        variables: createdProject.variables,
                        visualElements: createdProject.visualElements
                    };
                    const projects = [...this.projectsSubject.value, mapped];
                    this.projectsSubject.next(projects);
                })
            );
        } else {
            const localProject: Project = {
                ...project,
                id: Date.now().toString(),
                createdAt: new Date(),
                updatedAt: new Date()
            };
            const projects = [...this.projectsSubject.value, localProject];
            this.projectsSubject.next(projects);
            this.saveToLocalStorage(projects);
            return localProject;
        }
    }

    /**
     * Atualiza um projeto existente
     */
    updateProject(id: string, updates: Partial<Project>): Observable<Project> | void {
        if (this.useApi) {
            const currentProject = this.projectsSubject.value.find(p => p.id === id);
            const resolvedName =
                updates.name ||
                updates.projectName ||
                currentProject?.projectName ||
                currentProject?.name ||
                '';
            const updatedProject = {
                id,
                ...currentProject,
                ...updates,
                name: resolvedName,
                projectName: resolvedName,
                description: updates.description ?? currentProject?.description ?? '',
                updatedAt: new Date()
            };

            return this.http.put<any>(`${this.apiUrl}/${id}`, updatedProject).pipe(
                tap(project => {
                    // Mapeia resposta
                    const mapped = {
                        id: project.id,
                        name: project.projectName || project.name,
                        description: project.description || '',
                        createdAt: project.createdAt ? new Date(project.createdAt) : new Date(),
                        updatedAt: new Date(),
                        projectName: project.projectName,
                        jobs: project.jobs || [],
                        connections: project.connections || [],
                        sourceDatabase: project.sourceDatabase,
                        destinationDatabase: project.destinationDatabase,
                        concurrency: project.concurrency,
                        variables: project.variables,
                        visualElements: project.visualElements
                    };

                    const projects = this.projectsSubject.value.map(p =>
                        p.id === id ? mapped : p
                    );
                    this.projectsSubject.next(projects);

                    const selected = this.selectedProjectSubject.value;
                    if (selected && selected.id === id) {
                        this.selectedProjectSubject.next(mapped);
                    }
                })
            );
        } else {
            const projects = this.projectsSubject.value.map(p =>
                p.id === id ? { ...p, ...updates, updatedAt: new Date() } : p
            );
            this.projectsSubject.next(projects);
            this.saveToLocalStorage(projects);

            const selected = this.selectedProjectSubject.value;
            if (selected && selected.id === id) {
                this.selectedProjectSubject.next({ ...selected, ...updates, updatedAt: new Date() });
            }
        }
    }

    /**
     * Exclui um projeto pelo ID
     */
    deleteProject(id: string): Observable<any> | void {
        if (this.useApi) {
            return this.http.delete(`${this.apiUrl}/${id}`).pipe(
                tap(() => {
                    const projects = this.projectsSubject.value.filter(p => p.id !== id);
                    this.projectsSubject.next(projects);

                    const selected = this.selectedProjectSubject.value;
                    if (selected && selected.id === id) {
                        this.clearSelection();
                    }
                })
            );
        } else {
            const projects = this.projectsSubject.value.filter(p => p.id !== id);
            this.projectsSubject.next(projects);
            this.saveToLocalStorage(projects);

            const selected = this.selectedProjectSubject.value;
            if (selected && selected.id === id) {
                this.clearSelection();
            }
        }
    }

    /**
     * Duplica um projeto pelo ID
     */
    duplicateProject(id: string, projectName?: string): Observable<Project> | Project | void {
        if (this.useApi) {
            const body = projectName ? { projectName } : {};
            return this.http.post<any>(`${this.apiUrl}/${id}/duplicate`, body).pipe(
                tap(createdProject => {
                    const mapped = {
                        id: createdProject.id,
                        name: createdProject.projectName || createdProject.name,
                        description: createdProject.description || '',
                        createdAt: createdProject.createdAt ? new Date(createdProject.createdAt) : new Date(),
                        updatedAt: createdProject.updatedAt ? new Date(createdProject.updatedAt) : new Date(),
                        projectName: createdProject.projectName,
                        jobs: createdProject.jobs || [],
                        connections: createdProject.connections || [],
                        sourceDatabase: createdProject.sourceDatabase,
                        destinationDatabase: createdProject.destinationDatabase,
                        concurrency: createdProject.concurrency,
                        variables: createdProject.variables,
                        visualElements: createdProject.visualElements
                    };
                    const projects = [...this.projectsSubject.value, mapped];
                    this.projectsSubject.next(projects);
                })
            );
        }

        const current = this.projectsSubject.value.find(p => p.id === id);
        if (!current) {
            return;
        }
        const copyName = projectName || `${current.projectName || current.name || 'Projeto'} (Cópia)`;
        const localProject: Project = {
            ...current,
            id: Date.now().toString(),
            name: copyName,
            projectName: copyName,
            createdAt: new Date(),
            updatedAt: new Date()
        };
        const projects = [...this.projectsSubject.value, localProject];
        this.projectsSubject.next(projects);
        this.saveToLocalStorage(projects);
        return localProject;
    }

    /**
     * Exporta um projeto (ZIP)
     */
    exportProject(id: string): Observable<Blob> | null {
        if (!this.useApi) {
            return null;
        }
        return this.http.get(`${this.apiUrl}/${id}/export`, {
            responseType: 'blob'
        });
    }

    /**
     * Importa um projeto via ZIP
     */
    importProject(file: File, projectName?: string): Observable<Project | null> | null {
        if (!this.useApi) {
            return null;
        }

        const form = new FormData();
        form.append('file', file);
        if (projectName) {
            form.append('projectName', projectName);
        }

        return this.http.post<any>(`${this.apiUrl}/import`, form).pipe(
            tap(importedProject => {
                if (!importedProject || !importedProject.id) {
                    return;
                }
                const mapped = {
                    id: importedProject.id,
                    name: importedProject.projectName || importedProject.name,
                    description: importedProject.description || '',
                    createdAt: importedProject.createdAt ? new Date(importedProject.createdAt) : new Date(),
                    updatedAt: importedProject.updatedAt ? new Date(importedProject.updatedAt) : new Date(),
                    projectName: importedProject.projectName,
                    jobs: importedProject.jobs || [],
                    connections: importedProject.connections || [],
                    sourceDatabase: importedProject.sourceDatabase,
                    destinationDatabase: importedProject.destinationDatabase,
                    concurrency: importedProject.concurrency,
                    variables: importedProject.variables,
                    visualElements: importedProject.visualElements
                } as Project;

                const exists = this.projectsSubject.value.some(p => p.id === mapped.id);
                if (!exists) {
                    this.projectsSubject.next([...this.projectsSubject.value, mapped]);
                }
            })
        );
    }

    /**
     * Executa um projeto
     */
    runProject(id: string): Observable<any> {
        return this.http.post(`${this.apiUrl}/${id}/run`, {});
    }

    /**
     * Fecha um projeto
     */
    closeProject(id: string): Observable<any> {
        return this.http.post(`${this.apiUrl}/${id}/close`, {});
    }

    /**
     * Interrompe a pipeline em execucao
     */
    stopProject(id: string): Observable<any> {
        return this.http.post(`${this.apiUrl}/${id}/stop`, {});
    }

    /**
     * Limpa a seleção do projeto
     */
    clearSelection(): void {
        this.selectedProjectSubject.next(null);
        if (this.isBrowser) {
            localStorage.removeItem('quasar_selected_project');
        }
    }
}
