import { lazy, Suspense, useEffect, useMemo, useState } from 'react';
import { HQAct } from '../shell/HQAct';
import type { PlayerCandidate } from '../shell/ThinPlayerSearch';
import type { Node } from '../shell/nodes';
import type { PulseTab } from '../shell/NativeLeaguePulse';
import { HostedRules } from './HostedRules';
import { FailureAlert } from '../data/FailureAlert';
import { Review } from './HostedReview';
import { createHostedLeagueRoom, type HostedLeagueRoom, type HostedPort } from './hostedRoom';
import { HostedControlRoom } from './HostedControlRoom';

const NativeLeaguePulse = lazy(() => import('../shell/NativeLeaguePulse'));

type CommissionerTab = 'pulse' | 'rules' | 'control';
export function HostedCommissionerLeague({ league, port, embedded, view = 'home', pulseTab = 'now',
  navigatePulse, leave = () => {}, onPlayer, onPlayers }: { league: string; port?: HostedPort;
    embedded?: boolean; view?: 'home' | 'control'; pulseTab?: PulseTab;
    navigatePulse?: (tab: PulseTab) => void; leave?: (node?: Node) => void;
    onPlayer?: (player: PlayerCandidate) => void; onPlayers?: (players: PlayerCandidate[]) => void }) {
  const room = useMemo(() => createHostedLeagueRoom(league, port), [league, port]);
  const state = room.use();
  const [localTab, setLocalTab] = useState<CommissionerTab>('pulse');
  const tab: CommissionerTab = embedded ? view === 'control' ? 'control' : 'pulse' : localTab;
  const [localPulseTab, setLocalPulseTab] = useState<'now' | 'race'>('now');
  const activePulseTab = embedded ? pulseTab : localPulseTab;
  const changePulseTab = (next: PulseTab) => {
    if (embedded) navigatePulse?.(next);
    else if (next === 'now' || next === 'race') setLocalPulseTab(next);
  };
  useEffect(() => { void room.refresh(); return room.dispose; }, [room]);
  return <section className={`hosted-league${embedded ? ' hosted-embedded' : ''}`}
    aria-label="Hosted commissioner league">
    <h2>{state.reading?.rules.name || 'Your league'}</h2>
    <p>Commissioner tools · League-wide view; no team is selected.</p>
    <nav aria-label="Commissioner tools">
      {!embedded && <>
        <HQAct selected={tab === 'pulse'} onAction={() => setLocalTab('pulse')}>League Pulse</HQAct>
        <HQAct selected={tab === 'rules'} onAction={() => setLocalTab('rules')}>League rules</HQAct>
        <HQAct selected={tab === 'control'} onAction={() => setLocalTab('control')}>Control Room</HQAct>
      </>}
      <HQAct disabled={state.pending} onAction={() => room.refresh()}>Refresh league server</HQAct>
    </nav>
    {tab !== 'control' && state.error && <FailureAlert
      message={state.error}
      league={league}
      screen="Commissioner"
      route="control/league"
      port={port}
    />}
    {tab !== 'control' && state.stale &&
      <p role="status">Rules are read-only until refreshed from the league server.</p>}
    {tab !== 'control' && state.message && <p role="status">{state.message}</p>}
    {tab === 'pulse' && <>
      <p>League-wide results open on the current league week. Use the week controls to choose another week;
        no team is selected.</p>
      <Suspense fallback={<p>Opening League Pulse…</p>}>
        <NativeLeaguePulse league={league} mine="" port={port} serverTime embedded={embedded}
          tab={activePulseTab} navigate={changePulseTab}
          back={() => embedded ? leave() : setLocalTab('rules')} leave={leave}
          onPlayer={onPlayer} onPlayers={onPlayers} />
      </Suspense>
    </>}
    {tab === 'rules' && state.reading && <HostedRules rules={state.reading.rules} commissioner room={room} />}
    {tab === 'control' && <HostedControlRoom league={league} port={port} heldRoom={room} />}
    {state.review && tab === 'pulse' && <Review room={room} />}
  </section>;
}
