import { INavData } from '@coreui/angular';

export const navItems: INavData[] = [
  {
    name: 'Dashboard',
    url: '/dashboard',
    iconComponent: { name: 'cil-speedometer' },
  },
  {
    name: 'Variáveis',
    url: '/variables',
    iconComponent: { name: 'cil-code' },
  },
  {
    name: 'Jobs',
    url: '/jobs',
    iconComponent: { name: 'cil-list' },
  },
  {
    name: 'Histórico',
    url: '/history',
    iconComponent: { name: 'cil-layers' },
  },
  {
    name: 'Erros',
    url: '/errors',
    iconComponent: { name: 'cil-tags' },
  }  
];
