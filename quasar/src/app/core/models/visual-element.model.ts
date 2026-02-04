export type VisualElementType = 'rect' | 'circle' | 'line' | 'text';

export interface VisualElement {
  id?: string;
  elementId?: string;
  type: VisualElementType;
  x: number;
  y: number;
  width?: number;
  height?: number;
  x2?: number;
  y2?: number;
  radius?: number;
  fillColor?: string;
  borderColor?: string;
  borderWidth?: number;
  text?: string;
  textColor?: string;
  fontSize?: number;
  fontFamily?: string;
  textAlign?:
    | 'center'
    | 'top-center'
    | 'bottom-center'
    | 'left'
    | 'center-left'
    | 'right'
    | 'center-right'
    | 'top-left'
    | 'top-right'
    | 'bottom-left'
    | 'bottom-right';
  cornerRadius?: number;
}
