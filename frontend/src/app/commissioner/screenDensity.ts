import { connectDensity } from '../shell/density';
import { createShellState } from '../shell/state';
import { surfaceActions } from './surfaceActions';
import { DENSITIES, type Density } from '../cards/Card';

export function connectScreenDensity(screen: string, root: HTMLElement) {
  const source = createShellState();
  const key = `twr.density.commissioner.${screen}`;
  try {
    const saved = localStorage.getItem(key);
    if (DENSITIES.includes(saved as Density)) source.write({ density: saved as Density });
  } catch { /* Storage may be unavailable; Tactical remains usable. */ }
  const stopDensity = connectDensity(source, root);
  const stopAction = surfaceActions.register(`density.${screen}`, args => {
    if (args.action !== 'view' || !DENSITIES.includes(args.value as Density)) return;
    source.write({ density: args.value as Density });
    try { localStorage.setItem(key, args.value); } catch { /* Keep the current view without memory. */ }
  });
  return () => { stopAction(); stopDensity(); };
}
