import { ComponentFixture, TestBed } from '@angular/core/testing';

import { DialogProject } from './dialog-project';

describe('DialogProject', () => {
  let component: DialogProject;
  let fixture: ComponentFixture<DialogProject>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [DialogProject]
    })
    .compileComponents();

    fixture = TestBed.createComponent(DialogProject);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
