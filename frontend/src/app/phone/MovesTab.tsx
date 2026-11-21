import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { hostedWorkflow } from '../commissioner/api';
import { Review } from '../commissioner/HostedReview';
import { parseHostedRoster } from '../commissioner/hostedReadings';
import { parseHostedTrades, parseHostedWaivers } from '../commissioner/hostedMarkets';
import { createHostedLeagueRoom, type HostedPort } from '../commissioner/hostedRoom';
import { reviewHostedClaimChange, reviewHostedOwnerTrade } from '../commissioner/HostedWorkflows';
import type { NativeRequest } from '../commands/native';
import { HQAct } from '../shell/HQAct';
import type { LeagueChoice } from '../shell/leaguePicker';
import { isEndedTrade } from '../shell/warroomPlansReads';
import { readNativeHQ } from '../commissioner/hqReading';
import type { WarRoomContext } from '../shell/warroomReads';
import { dealFromWarRoomOffer, type WarRoomDeal } from '../shell/warroomTradeReads';
import { AddSheet, ClaimSheet, PlayerSheet, TradeSheet, type MovesPlayer,
  type MovesRoster } from './MovesSheets';
import { onHostedTopics } from '../data/hostedReads';
import { errorCodeFor, errorMessage } from '../data/errorCodes';
import { FailureAlert } from '../data/FailureAlert';
import type { Game } from '../now/liveContract';
import { useHostedGames } from '../shell/hostedGames';
import { dayTime, gameWords } from './teamModel';
import './MovesTab.css';

type Subscribe = (load: () => void) => () => void;
/** topicsFor reloads when a hosted topic of this league changes (same rule as LiveTab). */
const topicsFor = (league: string): Subscribe => load =>
  onHostedTopics(topics => { if (topics.some(topic => topic.league === league)) load(); });

export type MovesSegment = 'players' | 'waivers' | 'trades';
type Offer = ReturnType<typeof parseHostedTrades>['offers'][number];
type Part<T> = { value?: T; error?: string; errorCode?: string };
export type MovesReading = {
  pool: Part<MovesPlayer[]>;
  roster?: MovesRoster;
  /** openBench is this seat's open bench slots from native.assets.read, when the read holds it. */
  openBench?: number;
  waivers: Part<{ revision: number; place?: number; total: number; source: string; method: string;
    /** runAt is the earliest close of an open waiver player: when the waivers run. */
    runAt?: string;
    claims: { id: string; player: string; name: string; words: string }[] }>;
  trades: Part<{ revision: number; notes: string[]; offers: Offer[] }>;
};
const POSITIONS = ['ALL', 'QB', 'RB', 'WR', 'TE', 'K', 'DEF'];
const message = (cause: unknown) => errorMessage(cause);
async function part<T>(read: () => Promise<T>): Promise<Part<T>> {
  try { return { value: await read() }; }
  catch (cause) { return { error: message(cause), errorCode: errorCodeFor(cause) }; }
}

/** readMoves takes the three existing owner reads; each fails on its own with the server's words. */
export async function readMoves(league: string, mine: string, port: HostedPort): Promise<MovesReading> {
  const read = (command: string, args: Record<string, unknown>) =>
    port({ command, mode: 'read', arguments: { league, ...args } } as NativeRequest).then(r => r.data);
  const [roster, waivers, trades] = await Promise.all([
    part(async () => {
      const response = await port({ command: 'native.assets.read', mode: 'read', arguments: { league } });
      return { ...parseHostedRoster(response.data), writable: response.writable,
        football: response.football };
    }),
    part(async () => parseHostedWaivers(await read('waivers.read',
      { franchise: mine, at: new Date().toISOString() }))),
    part(async () => parseHostedTrades(await read('trades.read', { viewer: mine }))),
  ]);
  const names = new Map<string, string>();
  for (const p of [...roster.value?.players ?? [], ...roster.value?.market ?? []]) names.set(p.id, p.name);
  for (const p of waivers.value?.players ?? []) names.set(p.player, p.name);
  const open = (status: string) => status === 'On waivers' || status === 'Closed; awaiting processing';
  const onWaivers = new Map((waivers.value?.players ?? []).filter(p => open(p.status))
    .map(p => [p.player, { waiver: p.status, window: p.window }]));
  const pool = roster.value && { value: [
    ...roster.value.market.map(p => ({ id: p.id, name: p.name, position: p.position,
      ...(p.team ? { team: p.team } : {}), ...(p.average ? { average: p.average } : {}),
      ...onWaivers.get(p.id) })),
    ...[...onWaivers].filter(([id]) => !roster.value!.market.some(p => p.id === id))
      .map(([id, held]) => ({ id, name: names.get(id) ?? id, position: '', ...held })),
  ] };
  const r = roster.value;
  const w = waivers.value;
  const closes = (w?.players ?? []).filter(p => open(p.status) && p.closesAt)
    .map(p => p.closesAt!).sort((a, b) => Date.parse(a) - Date.parse(b));
  return {
    pool: pool ?? { error: roster.error },
    ...(r?.openBenchSlots?.[mine] !== undefined ? { openBench: r.openBenchSlots[mine] } : {}),
    ...(r ? { roster: { revision: r.revision, writable: r.writable, football: r.football,
      mine: r.players.filter(p => p.franchise === mine).map(p => ({ id: p.id, name: p.name })) } } : {}),
    waivers: w ? { value: { revision: w.revision,
      ...(w.order.includes(mine) ? { place: w.order.indexOf(mine) + 1 } : {}),
      total: w.order.length, source: w.words, method: w.method,
      ...(closes[0] ? { runAt: closes[0] } : {}),
      claims: w.claims.map(c => ({ ...c, name: names.get(c.player) ?? c.player })) } }
      : { error: waivers.error, errorCode: waivers.errorCode },
    trades: trades.value ? { value: { revision: trades.value.revision, notes: trades.value.notes,
      offers: trades.value.offers.filter(o => !isEndedTrade(o.status)) } }
      : { error: trades.error, errorCode: trades.errorCode },
  };
}

/** benchWords is the mock's line above the pool: open bench spots, or that adding means a drop. */
export const benchWords = (open: number) => open === 0 ? 'Bench full: adding means dropping someone'
  : `${open} open bench spot${open === 1 ? '' : 's'}`;
const ordinal = (n: number) => {
  const tail = n % 100 >= 11 && n % 100 <= 13 ? 'th' : ['th', 'st', 'nd', 'rd'][n % 10] ?? 'th';
  return `${n}${tail}`;
};

const unavailableLine = (words?: string, league?: string, port?: HostedPort, code?: string) =>
  <FailureAlert
    message={words || 'no reading'}
    prefix="Unavailable: "
    code={code}
    className="mv4-empty"
    league={league}
    screen="Phone · Moves"
    route="phone/moves"
    port={port}
  />;

/** readMovesContext is the War Room context the move sheets reuse: the hosted HQ reading for this seat. */
export async function readMovesContext(league: string, mine: string,
  port: HostedPort): Promise<WarRoomContext> {
  const reading = await readNativeHQ(league, mine, port);
  const week = reading.week ?? 0;
  return { snapshot: reading.snapshot, mine, week, native: { league, mine, port, owner: true,
    leave: () => undefined, week, reading } };
}

type Sheet = { kind: 'player' | 'add' | 'claim'; player: MovesPlayer }
  | { kind: 'trade'; deal?: WarRoomDeal };

export function MovesTab({ league, port = hostedWorkflow, read = readMoves,
  readContext = readMovesContext, subscribe }: { league?: LeagueChoice; port?: HostedPort;
  read?: typeof readMoves; readContext?: typeof readMovesContext; subscribe?: Subscribe }) {
  if (!league || !league.franchise) return <section className="mv4" aria-label="Moves">
    <p className="mv4-empty">Moves opens once you hold a franchise in a league.</p></section>;
  return <MovesBody key={league.id} league={league} port={port} read={read} readContext={readContext}
    subscribe={subscribe ?? topicsFor(league.id)} />;
}

function MovesBody({ league, port, read, readContext, subscribe }: { league: LeagueChoice;
  port: HostedPort; read: typeof readMoves; readContext: typeof readMovesContext; subscribe: Subscribe }) {
  const mine = league.franchise ?? '';
  const [segment, setSegment] = useState<MovesSegment>('players');
  const [reading, setReading] = useState<MovesReading>();
  const [failure, setFailure] = useState('');
  const [failureCode, setFailureCode] = useState<string>();
  const [search, setSearch] = useState('');
  const [position, setPosition] = useState('ALL');
  const [revision, setRevision] = useState(0);
  const [sheet, setSheet] = useState<Sheet>();
  const [context, setContext] = useState<WarRoomContext>();
  const [contextError, setContextError] = useState('');
  const [contextErrorCode, setContextErrorCode] = useState<string>();
  const [done, setDone] = useState('');
  useEffect(() => {
    if (!league || !mine) return;
    let active = true;
    setFailure('');
    setFailureCode(undefined);
    void read(league.id, mine, port).then(r => { if (active) setReading(r); })
      .catch(cause => {
        if (active) { setFailure(message(cause)); setFailureCode(errorCodeFor(cause)); }
      });
    return () => { active = false; };
  }, [league, mine, port, read, revision]);
  useEffect(() => {
    if (!league || !mine || !sheet || context) return;
    let active = true;
    setContextError('');
    setContextErrorCode(undefined);
    void readContext(league.id, mine, port).then(c => { if (active) setContext(c); })
      .catch(cause => {
        if (active) { setContextError(message(cause)); setContextErrorCode(errorCodeFor(cause)); }
      });
    return () => { active = false; };
  }, [league, mine, port, readContext, sheet, context]);
  const changes = useRef(subscribe); changes.current = subscribe;
  const onChange = useCallback<Subscribe>(load => changes.current(load), []);
  const live = useHostedGames(league.id, port, onChange);
  const games = live.view?.games;
  if (!league || !mine) return <section className="mv4" aria-label="Moves">
    <p className="mv4-empty">Moves opens once you hold a franchise in a league.</p></section>;
  if (failure) return <section className="mv4" aria-label="Moves">
    {unavailableLine(failure, league.id, port, failureCode)}</section>;
  if (!reading) return <section className="mv4" aria-label="Moves">
    <p role="status">Reading moves…</p></section>;
  const claims = reading.waivers.value?.claims.length;
  const incoming = reading.trades.value?.offers.filter(o => o.status === 'offered'
    && o.proposer !== mine && !o.accepted.includes(mine)).length;
  const segments: [MovesSegment, string, number | undefined, string][] = [
    ['players', 'Players', undefined, ''], ['waivers', 'Waivers', claims, 'n'],
    ['trades', 'Trades', incoming, '']];
  const close = () => setSheet(undefined);
  const finish = (words: string) => { setSheet(undefined); setDone(words); setRevision(n => n + 1); };
  const move = (player: MovesPlayer) => setSheet({ kind: player.window ? 'claim' : 'add', player });
  const blind = /blind/i.test(reading.waivers.value?.method ?? '');
  return <section className="mv4" aria-label="Moves">
    <div className="mv4-seg" role="tablist">
      {segments.map(([id, label, count, tone]) => <HQAct key={id} role="tab" selected={segment === id}
        className={segment === id ? 'on' : ''} onAction={() => setSegment(id)}>
        {label}{count ? <i className={tone}>{count}</i> : null}</HQAct>)}
    </div>
    {done && <p className="mv4-done" role="status">{done}</p>}
    {segment === 'players' && <Players reading={reading.pool} league={league.id} port={port}
      search={search} setSearch={setSearch}
      position={position} setPosition={setPosition} open={player => setSheet({ kind: 'player', player })}
      move={move} canAdd={Boolean(reading.roster?.writable)} openBench={reading.openBench}
      games={games} runAt={reading.waivers.value?.runAt} />}
    {segment === 'waivers' && <Waivers reading={reading.waivers} league={league.id} mine={mine} port={port}
      onChanged={() => setRevision(n => n + 1)} />}
    {segment === 'trades' && <Trades reading={reading.trades} league={league.id} mine={mine} port={port}
      onChanged={() => setRevision(n => n + 1)} onTrade={deal => setSheet({ kind: 'trade', deal })} />}
    {sheet && !context && (contextError
      ? unavailableLine(contextError, league.id, port, contextErrorCode)
      : <p role="status">Opening…</p>)}
    {sheet && context && sheet.kind === 'player' && <PlayerSheet context={context} player={sheet.player}
      onMove={() => move(sheet.player)} onClose={close} />}
    {sheet && context && sheet.kind === 'add' && reading.roster && <AddSheet context={context}
      player={sheet.player} roster={reading.roster} onDone={finish} onClose={close} />}
    {sheet && context && sheet.kind === 'claim' && reading.waivers.value && <ClaimSheet context={context}
      player={sheet.player} revision={reading.waivers.value.revision} blind={blind} onDone={finish}
      onClose={close} />}
    {sheet && context && sheet.kind === 'trade' && <TradeSheet context={context} start={sheet.deal}
      onDone={finish} onClose={close} />}
  </section>;
}

function Players({ reading, league, port, search, setSearch, position, setPosition, open, move, canAdd,
  openBench, games, runAt }: {
  reading: MovesReading['pool']; league: string; port: HostedPort;
  search: string; setSearch: (v: string) => void;
  position: string; setPosition: (v: string) => void; open: (p: MovesPlayer) => void;
  move: (p: MovesPlayer) => void; canAdd: boolean; openBench?: number; games?: Game[]; runAt?: string }) {
  const shown = useMemo(() => (reading.value ?? []).filter(p =>
    (position === 'ALL' || p.position === position)
    && p.name.toLowerCase().includes(search.trim().toLowerCase())), [reading, search, position]);
  if (!reading.value) return unavailableLine(reading.error, league, port, reading.errorCode);
  return <>
    <div className="mv4-search"><input type="search" aria-label="Search players" placeholder="Search players"
      value={search} onChange={event => setSearch(event.target.value)} /></div>
    <div className="mv4-chips">{POSITIONS.map(p => <HQAct key={p} className={position === p ? 'on' : ''}
      selected={position === p} onAction={() => setPosition(p)}>{p}</HQAct>)}</div>
    {openBench !== undefined && <p className="mv4-note">{benchWords(openBench)}</p>}
    {runAt && <p className="mv4-note">Waivers run {dayTime(runAt)}</p>}
    {!reading.value.length && <p className="mv4-empty">No free agents in the pool.</p>}
    {reading.value.length > 0 && !shown.length &&
      <p className="mv4-empty">No one matches. Clear the search or pick ALL.</p>}
    <div className="mv4-list">{shown.map(p => <div className="mv4-fa" key={p.id}>
      <span className="mv4-ps">{p.position || '—'}</span>
      <HQAct className="mv4-who" label={`Open ${p.name}`} onAction={() => open(p)}>
        <b>{p.name}</b>{p.team && <em className="tm">{gameWords(p.team, games, undefined).words}</em>}
        {p.waiver && <em className="wv">{p.waiver}</em>}</HQAct>
      <span className="mv4-avg" aria-label={p.average ? `Season average ${p.average}` : 'No season average'}>
        <b>{p.average ?? '—'}</b><em>AVG</em></span>
      <HQAct className="mv4-act" label={`${p.window ? 'Claim' : 'Add'} ${p.name}`}
        disabled={!p.window && !canAdd} onAction={() => move(p)}>{p.window ? 'CLAIM' : 'ADD'}</HQAct>
    </div>)}</div>
  </>;
}

function Waivers({ reading, league, mine, port, onChanged }: { reading: MovesReading['waivers'];
  league: string; mine: string; port: HostedPort; onChanged: () => void }) {
  const room = useMemo(() => createHostedLeagueRoom(league, port), [league, port]);
  const state = room.use();
  const [error, setError] = useState('');
  const [errorCode, setErrorCode] = useState<string>();
  useEffect(() => { void room.refresh(); return room.dispose; }, [room]);
  if (!reading.value) return unavailableLine(reading.error, league, port, reading.errorCode);
  const w = reading.value;
  const change = (c: Parameters<typeof reviewHostedClaimChange>[4]) => {
    setError('');
    setErrorCode(undefined);
    void reviewHostedClaimChange(room, league, mine, w.revision, c).catch(cause => {
      setError(message(cause));
      setErrorCode(errorCodeFor(cause));
    });
  };
  const ids = w.claims.map(c => c.id);
  const swap = (i: number, j: number) => {
    const order = [...ids];
    [order[i], order[j]] = [order[j], order[i]];
    change({ command: 'waivers.claim.order.set', order });
  };
  const off = !room.canChange() || Boolean(state.review);
  const activeError = error || state.error;
  const activeCode = error ? errorCode : state.errorCode;
  return <>
    {!w.claims.length && <p className="mv4-empty">No waiver claims. Players with a CLAIM key in Players
      can be claimed until waivers run.</p>}
    {w.claims.length > 0 && <p className="mv4-note">Claims run top first. {w.place
      ? `You are ${ordinal(w.place)} of ${w.total} in waiver order`
      : `You are not in the waiver order of ${w.total}`}
      {w.source ? ` · ${w.source}` : ''}.</p>}
    <div className="mv4-list">{w.claims.map((c, i) => <div className="mv4-wc" key={c.id}>
      <b>{i + 1}</b><span><b>+ {c.name}</b><em>{c.words}</em></span>
      <span className="mv4-ud">
        <HQAct label="Move up" disabled={off || i === 0} onAction={() => swap(i, i - 1)}>▲</HQAct>
        <HQAct label="Move down" disabled={off || i === ids.length - 1}
          onAction={() => swap(i, i + 1)}>▼</HQAct>
        <HQAct label="Cancel claim" disabled={off}
          onAction={() => change({ command: 'waivers.claim.cancel', claim: c.id })}>✕</HQAct></span>
    </div>)}</div>
    {activeError && <FailureAlert message={activeError} code={activeCode} league={league}
      screen="Phone · Moves" route="phone/moves" port={port} />}
    {state.review && <Review room={room} onConfirmed={onChanged} />}
  </>;
}

function Trades({ reading, league, mine, port, onChanged, onTrade }: { reading: MovesReading['trades'];
  league: string; mine: string; port: HostedPort; onChanged: () => void;
  onTrade: (deal?: WarRoomDeal) => void }) {
  const room = useMemo(() => createHostedLeagueRoom(league, port), [league, port]);
  const state = room.use();
  const [error, setError] = useState('');
  const [errorCode, setErrorCode] = useState<string>();
  useEffect(() => { void room.refresh(); return room.dispose; }, [room]);
  if (!reading.value) return unavailableLine(reading.error, league, port, reading.errorCode);
  const t = reading.value;
  const act = (offer: Offer, action: 'trade.accept' | 'trade.decline' | 'trade.withdraw') => {
    setError('');
    setErrorCode(undefined);
    void reviewHostedOwnerTrade(room, league, mine, offer.id, t.revision, action).catch(cause => {
      setError(message(cause));
      setErrorCode(errorCodeFor(cause));
    });
  };
  const counter = (offer: Offer) => {
    try { setError(''); setErrorCode(undefined); onTrade(dealFromWarRoomOffer(offer, mine)); }
    catch (cause) { setError(message(cause)); setErrorCode(errorCodeFor(cause)); }
  };
  const activeError = error || state.error;
  const activeCode = error ? errorCode : state.errorCode;
  return <>
    <HQAct className="mv4-key mv4-start" onAction={() => onTrade()}>Start a trade</HQAct>
    {t.notes.map(note => <p className="mv4-note" key={note}>{note}</p>)}
    {!t.offers.length && <p className="mv4-empty">No trade offers.</p>}
    {t.offers.map(offer => {
      const names = new Map(offer.teams);
      const other = offer.parties.filter(id => id !== mine).map(id => names.get(id) ?? id).join(' · ');
      const open = offer.status === 'offered' && offer.parties.includes(mine);
      const incoming = open && offer.proposer !== mine;
      const need = incoming && !offer.accepted.includes(mine);
      const give = offer.legs.filter(leg => leg.from === mine).map(leg => leg.label);
      const get = offer.legs.filter(leg => leg.to === mine).map(leg => leg.label);
      const side = (title: string, labels: string[]) => <div><small>{title}</small>
        {labels.length ? labels.map((label, i) => <p key={i}>{label}</p>)
          : <p className="none">Nothing</p>}</div>;
      return <article className={`mv4-to${need ? ' need' : ''}`} key={offer.id}>
        <h4><span>{offer.proposer === mine ? 'Your offer to' : 'Offer from'} {other || offer.id}</span>
          <em>{offer.expires}</em></h4>
        <div className="mv4-sides">{side('YOU GIVE', give)}{side('YOU GET', get)}</div>
        {(offer.reasons ?? []).filter(reason => reason.startsWith('Review held')).map(reason =>
          <p className="mv4-note mv4-hold" key={reason}>{reason}</p>)}
        {incoming ? <div className="mv4-acts">
          {need && <HQAct className="yes" disabled={!room.canChange()}
            onAction={() => act(offer, 'trade.accept')}>Accept</HQAct>}
          <HQAct onAction={() => counter(offer)}>Counter</HQAct>
          <HQAct disabled={!room.canChange()} onAction={() => act(offer, 'trade.decline')}>Decline</HQAct>
        </div> : open ? <div className="mv4-res wait"><span>{offer.words}</span>
          <HQAct disabled={!room.canChange()} onAction={() => act(offer, 'trade.withdraw')}>Withdraw</HQAct>
        </div> : <p className="mv4-res">{offer.words}</p>}
      </article>;
    })}
    {activeError && <FailureAlert message={activeError} code={activeCode} league={league}
      screen="Phone · Moves" route="phone/moves" port={port} />}
    {state.message && <p role="status">{state.message}</p>}
    {state.review && <Review room={room} onConfirmed={onChanged} />}
  </>;
}

export default MovesTab;
