import { isPlatformBrowser } from '@angular/common';
import {
  AfterViewInit,
  Component,
  Inject,
  PLATFORM_ID,
} from '@angular/core';

@Component({
  selector: 'app-diagram',
  standalone: true,
  templateUrl: './diagram.html',
  styleUrls: ['./diagram.scss'],
})
export class Diagram implements AfterViewInit {
  isBrowser: boolean;
  instance: any;

  constructor(@Inject(PLATFORM_ID) private platformId: any) {
    this.isBrowser = isPlatformBrowser(this.platformId);
  }

  async ngAfterViewInit(): Promise<void> {
    if (!this.isBrowser) return;

    const jsPlumbModule = await import('jsplumb');
    const jsPlumb = jsPlumbModule.jsPlumb;
    this.instance = jsPlumb.getInstance();

    this.instance.setContainer('diagramContainer');

    const boxes = ['box1', 'box2'];
    boxes.forEach((id) => {
      this.instance.makeSource(id, {
        filter: '.handle', // Ponto clicável (veja HTML)
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

      this.instance.draggable(id);
    });
  }
}
