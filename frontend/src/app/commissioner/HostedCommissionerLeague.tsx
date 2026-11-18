import { lazy, Suspense, useEffect, useMemo, useState } from 'react';
import { HQAct } from '../shell/HQAct';
import { HostedRules } from './HostedRules';
import { Review } from './HostedReview';
import { createHostedLeagueRoom, type HostedLeagueRoom, type HostedPort } from './hostedRoom';
import { HostedControlRoom } from './HostedControlRoom';

const NativeLeaguePulse = lazy(() => import('../shell/NativeLeaguePulse'));

type CommissionerTab = 'pulse' | 'rules' | 'control';
export function HostedCommissionerLeague({ league, port }: { league: string; port?: HostedPort }) {
  const room = useMemo(() => createHostedLeagueRoom(league, port), [league, port]);
  const state = room.use();
  const [tab, setTab] = useState<CommissionerTab>('pulse');
  const [pulseTab, setPulseTab] = useState<'now' | 'race'>('now');
  useEffect(() => { void room.refresh(); return room.dispose; }, [room]);
  return <section className="hosted-league" aria-label="Hosted commissioner league">
    <h2>{state.reading?.rules.name || 'Your league'}</h2>
    <p>Commissioner tools · No franchise is selected. Team actions are unavailable.</p>
    <nav aria-label="Commissioner tasks">
      <HQAct selected={tab === 'pulse'} onAction={() => setTab('pulse')}>League Pulse</HQAct>
      <HQAct selected={tab === 'rules'} onAction={() => setTab('rules')}>League rules</HQAct>
      <HQAct selected={tab === 'control'} onAction={() => setTab('control')}>Control Room</HQAct>
      <HQAct disabled={state.pending} onAction={() => room.refresh()}>Refresh league server</HQAct>
    </nav>
    {tab !== 'control' && state.error && <p role="alert">{state.error}</p>}
    {tab !== 'control' && state.stale &&
      <p role="status">Rules are read-only until refreshed from the league server.</p>}
    {tab !== 'control' && state.message && <p role="status">{state.message}</p>}
    {tab === 'pulse' && <>
      <p>League-wide results open on the current league week. Use the week controls to choose another week;
        no team is selected.</p>
      <Suspense fallback={<p>Opening League Pulse…</p>}>
        <NativeLeaguePulse league={league} mine="" port={port} serverTime tab={pulseTab}
          navigate={setPulseTab} back={() => setTab('rules')} leave={() => setTab('rules')} />
      </Suspense>
    </>}
    {tab === 'rules' && state.reading && <HostedRules rules={state.reading.rules} commissioner room={room} />}
    {tab === 'control' && <HostedControlRoom league={league} port={port} heldRoom={room} />}
    {state.review && tab === 'pulse' && <Review room={room} />}
  </section>;
}
