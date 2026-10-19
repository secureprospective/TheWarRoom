import { useEffect, useRef, type ReactNode } from 'react';
import { Act } from '../commands/Act';
import { connectScreenDensity } from './screenDensity';
import { DENSITIES } from '../cards/Card';
import './commissioner.css';

type Props = { screen: string; label: string; children: ReactNode; pending?: boolean };

export function DensityScreen({ screen, label, children, pending }: Props) {
  const root = useRef<HTMLElement>(null);
  useEffect(() => connectScreenDensity(screen, root.current!), [screen]);
  return (
    <section ref={root} className="commissioner-screen" aria-label={label}
      aria-busy={pending} data-density="tactical">
      <div className="commissioner-density" role="group" aria-label={`${label} view`}>
        {DENSITIES.map(view => <Act key={view} verb="commissioner.surface" variant="density"
          args={{ screen: `density.${screen}`, action: 'view', value: view }} densityKey={view}
          label={view[0].toUpperCase() + view.slice(1)} />)}
      </div>
      {children}
    </section>
  );
}
