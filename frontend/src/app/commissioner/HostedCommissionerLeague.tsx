import { lazy, Suspense, useEffect, useMemo, useState } from 'react';
import { HQAct } from '../shell/HQAct';
import { createHostedLeagueRoom, type HostedPort } from './hostedRoom';
import { HostedRules } from './HostedRules';
import { Review } from './HostedReview';
import type { PlayerCandidate } from '../shell/ThinPlayerSearch';
import type { Node } from '../shell/nodes';

const NativeLeaguePulse = lazy(() => import('../shell/NativeLeaguePulse'));

export function HostedCommissionerLeague({ league, port, embedded, view = 'home', pulseTab = 'now',
  navigatePulse, leave = () => {}, onPlayer, onPlayers }: { league: string; port?: HostedPort;
    embedded?: boolean; view?: 'home' | 'control'; pulseTab?: 'now' | 'race';
    navigatePulse?: (tab: 'now' | 'race') => void; leave?: (node?: Node) => void;
    onPlayer?: (player: PlayerCandidate) => void; onPlayers?: (players: PlayerCandidate[]) => void }) {
  const room = useMemo(() => createHostedLeagueRoom(league, port), [league, port]);
  const state = room.use();
  const [localTab, setLocalTab] = useState<'pulse' | 'rules'>('pulse');
  const tab = embedded ? view === 'control' ? 'rules' : 'pulse' : localTab;
  const [localPulseTab, setLocalPulseTab] = useState<'now' | 'race'>('now');
  const activePulseTab = embedded ? pulseTab : localPulseTab;
  useEffect(() => { void room.refresh(); return room.dispose; }, [room]);
  return <section className={`hosted-league${embedded ? ' hosted-embedded' : ''}`}
    aria-label="Hosted league">
    <h2>{state.reading?.rules.name || 'Your league'}</h2>
    <p>Commissioner tools · No franchise assigned. Team actions are unavailable.</p>
    <nav aria-label="Commissioner tools">
      {!embedded && <>
        <HQAct onAction={() => setLocalTab('pulse')}>League Pulse</HQAct>
        <HQAct onAction={() => setLocalTab('rules')}>League rules</HQAct>
      </>}
      <HQAct disabled={state.pending} onAction={() => room.refresh()}>Refresh league server</HQAct>
    </nav>
    {state.error && <p role="alert">{state.error}</p>}
    {state.stale && <p role="status">Rules are read-only until refreshed from the league server.</p>}
    {state.message && <p role="status">{state.message}</p>}
    {tab === 'pulse' && <p>League-wide results open on the current league week. Use the week controls
      to choose another week; no team is selected.</p>}
    {tab === 'pulse' && <Suspense fallback={<p>Opening League Pulse…</p>}>
      <NativeLeaguePulse league={league} mine="" port={port} serverTime embedded={embedded}
        tab={activePulseTab} navigate={embedded ? navigatePulse : setLocalPulseTab}
        back={() => embedded ? leave() : setLocalTab('rules')} leave={leave}
        onPlayer={onPlayer} onPlayers={onPlayers} />
    </Suspense>}
    {tab === 'rules' && state.reading &&
      <HostedRules rules={state.reading.rules} commissioner room={room} />}
    {state.review && tab !== 'rules' && <Review room={room} />}
  </section>;
}
