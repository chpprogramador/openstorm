// src/app/core/models/project.model.ts
import { VisualElement } from './visual-element.model';

export interface DatabaseConfig {
    type: string;
    host: string;
    port: number;
    user: string;
    password: string;
    database: string;
}

export interface JobConnection {
    source: string;
    target: string;
}

export interface Variable {
    name: string;
    value: string;
    description: string;
    type?: 'string' | 'number' | 'boolean' | 'date' | 'secret';
}

export interface Project {
    id: string;
    name: string;
    description: string;
    createdAt: Date;
    updatedAt: Date;

    // Propriedades do projeto legado
    projectName?: string;
    jobs?: string[];
    connections?: JobConnection[];
    sourceDatabase?: DatabaseConfig;
    destinationDatabase?: DatabaseConfig;
    concurrency?: number;
    variables?: Variable[];
    visualElements?: VisualElement[];
}
