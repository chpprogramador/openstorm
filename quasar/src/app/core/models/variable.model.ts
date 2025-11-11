export interface Variable {
  name: string;
  value: string;
  description: string;
  type?: 'string' | 'number' | 'boolean' | 'date';
}
