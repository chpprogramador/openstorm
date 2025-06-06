import { ComponentFixture, TestBed } from '@angular/core/testing';

import { DialogJobs } from './dialog-jobs';

describe('DialogJobs', () => {
  let component: DialogJobs;
  let fixture: ComponentFixture<DialogJobs>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [DialogJobs]
    })
    .compileComponents();

    fixture = TestBed.createComponent(DialogJobs);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
