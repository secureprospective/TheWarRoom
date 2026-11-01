import { lazy, Suspense, useEffect, useState } from 'react';
import { appLeagueFranchise, RailLeaguePicker, type LeagueChoice } from './leaguePicker';
import { localReadOnlyWorkflow, localWorkflow } from './localWorkflow';
import { Act } from '../commands/Act';
import type { Node } from './nodes';

const NativeFranchiseHQ = lazy(() => import('./NativeFranchiseHQ'));
const WarRoomNative = lazy(() => import('./WarRoomNative'));

export default function AppLeagueMount({
  league,
  view,
  leave,
}: {
  league: LeagueChoice;
  view: 'hq' | 'war';
  leave: (node?: Node) => void;
}) {
  const [mine, setMine] = useState<string>();
  const [error, setError] = useState<string>();
  useEffect(() => {
    let active = true;
    setMine(undefined);
    setError(undefined);
    appLeagueFranchise(league.id).then(
      (franchise) => {
        if (active) setMine(franchise);
      },
      (cause) => {
        if (active) setError(cause instanceof Error ? cause.message : String(cause));
      },
    );
    return () => {
      active = false;
    };
  }, [league.id]);
  if (error)
    return (
      <main className="twr-app">
        <RailLeaguePicker />
        <p role="alert">Could not open {league.name}: {error}</p>
        <Act variant="button" verb="nav.open" args={{ node: 'home' }}>
          Back to Home
        </Act>
      </main>
    );
  if (mine === undefined) return <div className="twr-app"><RailLeaguePicker />
    <p role="status">Opening {league.name}…</p></div>;
  const port = mine ? localWorkflow : localReadOnlyWorkflow;
  if (view === 'hq')
    return (
      <Suspense fallback={<div className="twr-app"><RailLeaguePicker />
        <p role="status">Opening {league.name}…</p></div>}>
        <NativeFranchiseHQ
          key={`${league.id}:${mine}`}
          league={league.id}
          mine={mine}
          name={league.name}
          port={port}
          leave={leave}
        />
      </Suspense>
    );
  return (
    <Suspense fallback={<div className="twr-app"><RailLeaguePicker />
        <p role="status">Opening {league.name}…</p></div>}>
      <WarRoomNative league={league.id} mine={mine} port={port} leave={leave} />
    </Suspense>
  );
}
