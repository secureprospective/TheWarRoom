import { lazy, Suspense, useEffect, useMemo, useState } from 'react';
import { HQAct } from '../shell/HQAct';
import { createHostedLeagueRoom, type HostedPort } from './hostedRoom';
import { HostedRules } from './HostedRules';
import { Review } from './HostedReview';

const NativeLeaguePulse = lazy(() => import('../shell/NativeLeaguePulse'));

export function HostedCommissionerLeague({ league, port }: { league: string; port?: HostedPort }) {
  const room = useMemo(() => createHostedLeagueRoom(league, port), [league, port]);
  const state = room.use();
  const [tab, setTab] = useState<'pulse' | 'rules'>('pulse');
  const [pulseTab, setPulseTab] = useState<'now' | 'race'>('now');
  useEffect(() => { void room.refresh(); return room.dispose; }, [room]);
  return <section className="hosted-league" aria-label="Hosted league">
    <h2>{state.reading?.rules.name || 'Your league'}</h2>
    <p>Commissioner tools · No franchise assigned. Team actions are unavailable.</p>
    <nav aria-label="League tasks">
      <HQAct onAction={() => setTab('pulse')}>League Pulse</HQAct>
      <HQAct onAction={() => setTab('rules')}>League rules</HQAct>
      <HQAct disabled={state.pending} onAction={() => room.refresh()}>Refresh league server</HQAct>
    </nav>
    {state.error && <p role="alert">{state.error}</p>}
    {state.stale && <p role="status">Rules are read-only until refreshed from the league server.</p>}
    {state.message && <p role="status">{state.message}</p>}
    {tab === 'pulse' && <p>League-wide results start with the first scheduled week. Use the week controls
      to choose another week; no team is selected.</p>}
    {tab === 'pulse' && <Suspense fallback={<p>Opening League Pulse…</p>}>
      <NativeLeaguePulse league={league} mine="" port={port} serverTime tab={pulseTab}
        navigate={setPulseTab} back={() => setTab('rules')} leave={() => setTab('rules')} />
    </Suspense>}
    {tab === 'rules' && state.reading &&
      <HostedRules rules={state.reading.rules} commissioner room={room} />}
    {state.review && tab !== 'rules' && <Review room={room} />}
  </section>;
}
