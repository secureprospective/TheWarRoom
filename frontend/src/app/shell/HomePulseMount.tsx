import { lazy, Suspense } from 'react';
import type { Snapshot } from '../data/contract';
import type { ClockProps } from '../clock/labels';
import { commands } from '../commands/registry';

const SeasonalCard = lazy(() => import('./SeasonalCard'));
const AlertTray = lazy(() => import('./AlertTray'));
const PulseNow = lazy(() => import('./PulseNow'));

export default function HomePulseMount({ snapshot, reading }: ClockProps & { snapshot?: Snapshot }) {
  const { node, workspace } = commands.use();
  let screen;
  if (node === 'home' && workspace.home === 'seasonal-card') {
    screen = <SeasonalCard reading={reading} />;
  } else if (snapshot) {
    screen = node === 'home' ? <AlertTray snapshot={snapshot} /> : <PulseNow snapshot={snapshot} />;
  } else {
    screen = <p role="status">Snapshot loading…</p>;
  }
  return <Suspense fallback={<p role="status">Opening workspace…</p>}>{screen}</Suspense>;
}
