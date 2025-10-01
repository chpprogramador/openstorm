import { ComponentFixture, TestBed } from '@angular/core/testing';

import { SqlEditor } from './sql-editor';

describe('SqlEditor', () => {
  let component: SqlEditor;
  let fixture: ComponentFixture<SqlEditor>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [SqlEditor]
    })
    .compileComponents();

    fixture = TestBed.createComponent(SqlEditor);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
