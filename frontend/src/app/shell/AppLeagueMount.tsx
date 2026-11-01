import { lazy, Suspense, useEffect, useState } from 'react';
import { createPortal } from 'react-dom';
import { readAppLeaguePeople, type LeagueChoice } from './leaguePicker';
import { localWorkflow } from './localWorkflow';
import { browserStorage, FRANCHISE_KEY } from './franchise';
import type { People } from '../commissioner/people';
import { commands } from '../commands/registry';
import type { HQTab } from './HQShell';
import type { Node } from './nodes';

let loadedHQ: typeof import('./NativeFranchiseHQ').default | undefined;
let loadedWarRoom: typeof import('./WarRoomNative').default | undefined;
function loadHQ() {
  return import('./NativeFranchiseHQ').then(module => {
    loadedHQ = module.default;
    return module;
  });
}
function loadWarRoom() {
  return import('./WarRoomNative').then(module => {
    loadedWarRoom = module.default;
    return module;
  });
}
const NativeFranchiseHQ = lazy(loadHQ);
const WarRoomNative = lazy(loadWarRoom);

export default function AppLeagueMount({ league, view, leave, footer, hqTab, tabs }: {
  league: LeagueChoice;
  view: 'hq' | 'war';
  leave: (node?: Node) => void;
  footer?: HTMLElement | null;
  hqTab?: HQTab; tabs?: HTMLElement | null;
}) {
  const [people, setPeople] = useState<People>();
  const [mine, setMine] = useState('');
  const [error, setError] = useState<string>();
  const key = `${FRANCHISE_KEY}:${league.id}`;
  useEffect(() => {
    let active = true;
    void (view === 'hq' ? loadHQ() : loadWarRoom()).catch(cause => {
      if (active) setError(`Could not load the workspace: ${String(cause)}`);
    });
    return () => { active = false; };
  }, [view]);
  useEffect(() => {
    let active = true;
    setPeople(undefined);
    setMine('');
    setError(undefined);
    readAppLeaguePeople(league.id).then(
      (reading) => {
        if (!active) return;
        const saved = browserStorage()?.getItem(key);
        const selected = reading.seats.some(s => s.franchise === saved) ? saved! : reading.yourFranchise;
        setMine(reading.seats.some(s => s.franchise === selected) ? selected : '');
        setPeople(reading);
      },
      (cause) => {
        if (active) setError(cause instanceof Error ? cause.message : String(cause));
      },
    );
    return () => { active = false; };
  }, [league.id, key]);
  if (error) return <p role="alert">Could not open {league.name}: {error}</p>;
  if (!people) return <p role="status">Opening {league.name}…</p>;
  const chooser = <label>View a franchise
    <select value={mine} onChange={event => {
      const id = event.target.value;
      if (!people.seats.some(s => s.franchise === id)) return;
      try {
        browserStorage()?.setItem(key, id);
        setMine(id);
      } catch (cause) {
        setError(`Franchise selection failed: ${String(cause)}`);
      }
    }}>
      <option value="">Choose a franchise</option>
      {people.seats.map(s => <option key={s.franchise} value={s.franchise}>{s.name}</option>)}
    </select>
  </label>;
  const HQ = loadedHQ ?? NativeFranchiseHQ;
  const WarRoom = loadedWarRoom ?? WarRoomNative;
  return <>
    {footer ? createPortal(chooser, footer) : chooser}
    {!mine ? <p role="status">Choose a franchise to view its roster, contracts and War Room.</p> :
      <Suspense fallback={<p role="status">Opening {league.name}…</p>}>
        {view === 'hq' ? <HQ embedded key={`${league.id}:${mine}`}
          league={league.id} mine={mine} name={league.name} port={localWorkflow} leave={leave}
          tab={hqTab} navigate={tab => commands.dispatch('nav.open', { node: 'hq', workspace: tab })} /> :
          <WarRoom embedded key={`${league.id}:${mine}`} league={league.id} mine={mine}
            port={localWorkflow} leave={leave} tabs={tabs} />}
      </Suspense>}
  </>;
}
