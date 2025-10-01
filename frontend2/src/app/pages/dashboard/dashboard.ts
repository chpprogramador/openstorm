import { BreakpointObserver, Breakpoints } from '@angular/cdk/layout';
import { Component } from '@angular/core';
import { MatCardModule } from '@angular/material/card';
import { MatChipsModule } from '@angular/material/chips';
import { MatGridListModule } from '@angular/material/grid-list';
import { MatIconModule } from '@angular/material/icon';
import { MatProgressBarModule } from '@angular/material/progress-bar';

@Component({
  standalone: true,
  selector: 'app-dashboard',
  imports: [MatCardModule, MatChipsModule, MatProgressBarModule, MatGridListModule, MatIconModule],
  templateUrl: './dashboard.html',
  styleUrls: ['./dashboard.scss']
})
export class Dashboard {

  cols: number = 4;

  constructor(private breakpointObserver: BreakpointObserver) {
    // Inicialização do componente
  }

  ngOnInit() {
    this.breakpointObserver.observe([
    Breakpoints.XSmall,
    Breakpoints.Small,
    Breakpoints.Medium
  ]).subscribe(result => {
    if (result.breakpoints[Breakpoints.XSmall]) this.cols = 1;
    else if (result.breakpoints[Breakpoints.Small]) this.cols = 2;
    else if (result.breakpoints[Breakpoints.Medium]) this.cols = 3;
    else this.cols = 4;
  });
  }

}
