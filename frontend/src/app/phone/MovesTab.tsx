import { useEffect, useMemo, useState } from 'react';
import { hostedWorkflow } from '../commissioner/api';
import { Review } from '../commissioner/HostedReview';
import { parseHostedRoster } from '../commissioner/hostedReadings';
import { parseHostedTrades, parseHostedWaivers } from '../commissioner/hostedMarkets';
import { createHostedLeagueRoom, type HostedPort } from '../commissioner/hostedRoom';
import { reviewHostedOwnerTrade } from '../commissioner/HostedWorkflows';
import type { NativeRequest } from '../commands/native';
import { HQAct } from '../shell/HQAct';
import type { LeagueChoice } from '../shell/leaguePicker';
import { isEndedTrade } from '../shell/warroomPlansReads';
import './MovesTab.css';

export type MovesSegment = 'players' | 'waivers' | 'trades';
type Offer = ReturnType<typeof parseHostedTrades>['offers'][number];
type Part<T> = { value?: T; error?: string };
export type MovesReading = {
  pool: Part<{ id: string; name: string; position: string; waiver?: string }[]>;
  waivers: Part<{ place?: number; total: number; source: string; method: string;
    claims: { id: string; player: string; name: string; words: string }[] }>;
  trades: Part<{ revision: number; notes: string[]; offers: Offer[] }>;
};
const POSITIONS = ['ALL', 'QB', 'RB', 'WR', 'TE', 'K', 'DEF'];
const NOT_YET = 'Not available yet';
const message = (cause: unknown) => cause instanceof Error ? cause.message : String(cause);
async function part<T>(read: () => Promise<T>): Promise<Part<T>> {
  try { return { value: await read() }; } catch (cause) { return { error: message(cause) }; }
}

/** readMoves takes the three existing owner reads; each fails on its own with the server's words. */
export async function readMoves(league: string, mine: string, port: HostedPort): Promise<MovesReading> {
  const read = (command: string, args: Record<string, unknown>) =>
    port({ command, mode: 'read', arguments: { league, ...args } } as NativeRequest).then(r => r.data);
  const [roster, waivers, trades] = await Promise.all([
    part(async () => parseHostedRoster(await read('native.assets.read', {}))),
    part(async () => parseHostedWaivers(await read('waivers.read',
      { franchise: mine, at: new Date().toISOString() }))),
    part(async () => parseHostedTrades(await read('trades.read', { viewer: mine }))),
  ]);
  const names = new Map<string, string>();
  for (const p of [...roster.value?.players ?? [], ...roster.value?.market ?? []]) names.set(p.id, p.name);
  for (const p of waivers.value?.players ?? []) names.set(p.player, p.name);
  const open = (status: string) => status === 'On waivers' || status === 'Closed; awaiting processing';
  const onWaivers = new Map((waivers.value?.players ?? []).filter(p => open(p.status))
    .map(p => [p.player, p.status]));
  const pool = roster.value && { value: [
    ...roster.value.market.map(p => ({ id: p.id, name: p.name, position: p.position,
      waiver: onWaivers.get(p.id) })),
    ...[...onWaivers].filter(([id]) => !roster.value!.market.some(p => p.id === id))
      .map(([id, status]) => ({ id, name: names.get(id) ?? id, position: '', waiver: status })),
  ] };
  const w = waivers.value;
  return {
    pool: pool ?? { error: roster.error },
    waivers: w ? { value: { ...(w.order.includes(mine) ? { place: w.order.indexOf(mine) + 1 } : {}),
      total: w.order.length, source: w.words, method: w.method,
      claims: w.claims.map(c => ({ ...c, name: names.get(c.player) ?? c.player })) } }
      : { error: waivers.error },
    trades: trades.value ? { value: { revision: trades.value.revision, notes: trades.value.notes,
      offers: trades.value.offers.filter(o => !isEndedTrade(o.status)) } } : { error: trades.error },
  };
}

const unavailableLine = (words?: string) =>
  <p className="mv4-empty" role="alert">Unavailable: {words || 'no reading'}</p>;

export function MovesTab({ league, port = hostedWorkflow, read = readMoves }: {
  league?: LeagueChoice; port?: HostedPort; read?: typeof readMoves }) {
  const mine = league?.franchise ?? '';
  const [segment, setSegment] = useState<MovesSegment>('players');
  const [reading, setReading] = useState<MovesReading>();
  const [failure, setFailure] = useState('');
  const [search, setSearch] = useState('');
  const [position, setPosition] = useState('ALL');
  const [revision, setRevision] = useState(0);
  useEffect(() => {
    if (!league || !mine) return;
    let active = true;
    setFailure('');
    void read(league.id, mine, port).then(r => { if (active) setReading(r); })
      .catch(cause => { if (active) setFailure(message(cause)); });
    return () => { active = false; };
  }, [league, mine, port, read, revision]);
  if (!league || !mine) return <section className="mv4" aria-label="Moves">
    <p className="mv4-empty">Moves opens once you hold a franchise in a league.</p></section>;
  if (failure) return <section className="mv4" aria-label="Moves">{unavailableLine(failure)}</section>;
  if (!reading) return <section className="mv4" aria-label="Moves">
    <p role="status">Reading moves…</p></section>;
  const claims = reading.waivers.value?.claims.length;
  const incoming = reading.trades.value?.offers.filter(o => o.status === 'offered'
    && o.proposer !== mine && !o.accepted.includes(mine)).length;
  const segments: [MovesSegment, string, number | undefined][] = [
    ['players', 'Players', undefined], ['waivers', 'Waivers', claims], ['trades', 'Trades', incoming]];
  return <section className="mv4" aria-label="Moves">
    <div className="mv4-seg" role="tablist">
      {segments.map(([id, label, count]) => <HQAct key={id} role="tab" selected={segment === id}
        className={segment === id ? 'on' : ''} onAction={() => setSegment(id)}>
        {label}{count ? <i>{count}</i> : null}</HQAct>)}
    </div>
    {segment === 'players' && <Players reading={reading.pool} search={search} setSearch={setSearch}
      position={position} setPosition={setPosition} />}
    {segment === 'waivers' && <Waivers reading={reading.waivers} />}
    {segment === 'trades' && <Trades reading={reading.trades} league={league.id} mine={mine} port={port}
      onChanged={() => setRevision(n => n + 1)} />}
  </section>;
}

function Players({ reading, search, setSearch, position, setPosition }: {
  reading: MovesReading['pool']; search: string; setSearch: (v: string) => void;
  position: string; setPosition: (v: string) => void }) {
  const shown = useMemo(() => (reading.value ?? []).filter(p =>
    (position === 'ALL' || p.position === position)
    && p.name.toLowerCase().includes(search.trim().toLowerCase())), [reading, search, position]);
  if (!reading.value) return unavailableLine(reading.error);
  return <>
    <div className="mv4-search"><input type="search" aria-label="Search players" placeholder="Search players"
      value={search} onChange={event => setSearch(event.target.value)} /></div>
    <div className="mv4-chips">{POSITIONS.map(p => <HQAct key={p} className={position === p ? 'on' : ''}
      selected={position === p} onAction={() => setPosition(p)}>{p}</HQAct>)}</div>
    {!reading.value.length && <p className="mv4-empty">No free agents in the pool.</p>}
    {reading.value.length > 0 && !shown.length && <p className="mv4-empty">No player matches.</p>}
    <div className="mv4-list">{shown.map(p => <div className="mv4-fa" key={p.id}>
      <span className="mv4-ps">{p.position || '—'}</span>
      <span className="mv4-who"><b>{p.name}</b>{p.waiver && <em className="wv">{p.waiver}</em>}</span>
    </div>)}</div>
  </>;
}

function Waivers({ reading }: { reading: MovesReading['waivers'] }) {
  if (!reading.value) return unavailableLine(reading.error);
  const w = reading.value;
  return <>
    <p className="mv4-note">Waiver order: {w.place ? `${w.place} of ${w.total}` : `not listed of ${w.total}`}
      {w.source ? ` · ${w.source}` : ''}</p>
    {!w.claims.length && <p className="mv4-empty">No claims. Your claims stay private until waivers run.</p>}
    <div className="mv4-list">{w.claims.map((c, i) => <div className="mv4-wc" key={c.id}>
      <b>{i + 1}</b><span><b>{c.name}</b><em>{c.words}</em></span>
      <em className="mv4-later">Reorder and cancel: {NOT_YET}</em>
    </div>)}</div>
  </>;
}

function Trades({ reading, league, mine, port, onChanged }: { reading: MovesReading['trades'];
  league: string; mine: string; port: HostedPort; onChanged: () => void }) {
  const room = useMemo(() => createHostedLeagueRoom(league, port), [league, port]);
  const state = room.use();
  const [error, setError] = useState('');
  useEffect(() => { void room.refresh(); return room.dispose; }, [room]);
  if (!reading.value) return unavailableLine(reading.error);
  const t = reading.value;
  const act = (offer: Offer, action: 'trade.accept' | 'trade.decline' | 'trade.withdraw') => {
    setError('');
    void reviewHostedOwnerTrade(room, league, mine, offer.id, t.revision, action)
      .catch(cause => setError(message(cause)));
  };
  return <>
    {t.notes.map(note => <p className="mv4-note" key={note}>{note}</p>)}
    {!t.offers.length && <p className="mv4-empty">No trade offers.</p>}
    {t.offers.map(offer => {
      const names = new Map(offer.teams);
      const other = offer.parties.filter(id => id !== mine).map(id => names.get(id) ?? id).join(' · ');
      const open = offer.status === 'offered' && offer.parties.includes(mine);
      const incoming = open && offer.proposer !== mine;
      const need = incoming && !offer.accepted.includes(mine);
      return <article className={`mv4-to${need ? ' need' : ''}`} key={offer.id}>
        <h4><span>{incoming ? 'From' : 'To'} {other || offer.id}</span><em>{offer.expires}</em></h4>
        <p className="mv4-words">{offer.words}</p>
        <ul>{offer.details.map((d, i) => <li key={i}>{d}</li>)}</ul>
        <p className="mv4-note">Hold reason: {NOT_YET}</p>
        <div className="mv4-acts">
          {need && <HQAct className="yes" disabled={!room.canChange()}
            onAction={() => act(offer, 'trade.accept')}>Accept</HQAct>}
          {incoming && <HQAct disabled={!room.canChange()}
            onAction={() => act(offer, 'trade.decline')}>Decline</HQAct>}
          {open && !incoming && <HQAct disabled={!room.canChange()}
            onAction={() => act(offer, 'trade.withdraw')}>Withdraw</HQAct>}
        </div>
      </article>;
    })}
    {error && <p role="alert">{error}</p>}
    {state.error && <p role="alert">{state.error}</p>}
    {state.message && <p role="status">{state.message}</p>}
    {state.review && <Review room={room} onConfirmed={onChanged} />}
  </>;
}

export default MovesTab;
