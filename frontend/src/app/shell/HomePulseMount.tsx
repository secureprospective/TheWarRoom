import { lazy, Suspense } from 'react';
import type { Snapshot } from '../data/contract';
import type { ClockProps } from '../clock/labels';
import { commands } from '../commands/registry';
import { LeagueChoices, type LeagueChoice } from './leaguePicker';

const SeasonalCard = lazy(() => import('./SeasonalCard'));
const AlertTray = lazy(() => import('./AlertTray'));
const Now = lazy(() => import('../now/Now'));

export default function HomePulseMount({
  snapshot,
  reading,
  leagues = [],
  selectedLeague = '',
  onLeague,
}: ClockProps & {
  snapshot?: Snapshot;
  leagues?: LeagueChoice[];
  selectedLeague?: string;
  onLeague?: (id: string) => void;
}) {
  const { node, workspace } = commands.use();
  let screen;
  if (node === 'home' && workspace.home === 'seasonal-card') {
    screen = <SeasonalCard reading={reading} />;
  } else if (node !== 'home') {
    screen = <Now />;
  } else if (snapshot) {
    screen = <AlertTray snapshot={snapshot} reading={reading} />;
  } else {
    screen = <p role="status">Snapshot loading…</p>;
  }
  return (
    <>
      {node === 'home' && onLeague && (
        <LeagueChoices leagues={leagues} selectedId={selectedLeague} onSelect={onLeague} />
      )}
      <Suspense fallback={<p role="status">Opening workspace…</p>}>{screen}</Suspense>
    </>
  );
}
