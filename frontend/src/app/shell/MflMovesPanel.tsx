import { useCallback, useEffect, useState } from 'react';
import { hostedWorkflow } from '../commissioner/api';
import type { HostedPort } from '../commissioner/hostedRoom';
import type { NativeRequest } from '../commands/native';

// My MFL moves: the owner's helper for a league that still lives on MFL. It reads the held public copy,
// checks a plan against the league's rules and keeps the plan private. Nothing here posts to MFL; the
// "Open on MFL" button only opens the page the server names, and the owner makes the move there.

type Obj = Record<string, unknown>;
type Team = { id: string; name: string };
type PoolRow = { id: string; name: string; position: string; team: string; age: string; minimum: string;
  value: string };
type Pool = { rows: PoolRow[]; positions: string[]; sorts: string[]; ageSeconds: number };
type Plan = { id: string; kind: string; state: string; words: string; canDone: boolean;
  canWithdraw: boolean };
type Verdict = { ok: boolean; words: string; problems: string[] };
export type MflRosterPlayer = { id: string; name: string };

const NO_LEAGUE = /no MFL league/i;
const NO_LEAGUE_LINE = 'This league does not follow an MFL league yet. The commissioner sets "MFL league number" in league settings.';

const obj = (v: unknown): Obj => (v && typeof v === 'object' && !Array.isArray(v) ? v as Obj : {});
const str = (v: unknown) => typeof v === 'string' ? v : '';
const list = (v: unknown): Obj[] => Array.isArray(v) ? v.map(obj) : [];
const say = (e: unknown) => e instanceof Error ? e.message : 'That did not work. Try again.';

export function ageWords(seconds: number): string {
  if (seconds < 60) return 'held just now';
  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) return `held ${minutes} minute${minutes === 1 ? '' : 's'} ago`;
  const hours = Math.floor(minutes / 60);
  if (hours < 48) return `held ${hours} hour${hours === 1 ? '' : 's'} ago`;
  const days = Math.floor(hours / 24);
  return `held ${days} days ago`;
}

function parseTeams(data: unknown): { teams: Team[]; yours: string } {
  const row = obj(data);
  return { teams: list(row.teams).map(t => ({ id: str(t.id), name: str(t.name) })), yours: str(row.yours) };
}
function parsePool(data: unknown): Pool {
  const row = obj(data);
  return { ageSeconds: typeof row.ageSeconds === 'number' ? row.ageSeconds : 0,
    positions: Array.isArray(row.positions) ? row.positions.map(str) : [],
    sorts: Array.isArray(row.sorts) ? row.sorts.map(str) : [],
    rows: list(row.rows).map(r => ({ id: str(r.id), name: str(r.name), position: str(r.position),
      team: str(r.team), age: str(r.age), minimum: str(r.minimum), value: str(r.value) })) };
}
function parsePlans(data: unknown): Plan[] {
  return list(obj(data).plans).map(p => ({ id: str(p.id), kind: str(p.kind), state: str(p.state),
    words: str(p.words), canDone: p.canDone === true, canWithdraw: p.canWithdraw === true }));
}
function parseVerdict(data: unknown, lineup: boolean): Verdict {
  const row = obj(data);
  const problems = lineup ? list(row.problems).map(p => str(p.message))
    : (Array.isArray(row.problems) ? row.problems.map(str) : []);
  return { ok: row.ok === true, words: str(row.words), problems };
}

const box = 'rounded border border-neutral-700 p-3 mb-3';
const btn = 'rounded border border-neutral-500 px-2 py-1 text-sm disabled:opacity-50';
const field = 'ml-2 rounded border border-neutral-600 bg-transparent px-1 py-0.5';

export default function MflMovesPanel({ league, roster = [], week, port = hostedWorkflow }: {
  league: string; roster?: MflRosterPlayer[]; week?: number; port?: HostedPort;
}) {
  const [state, setState] = useState<'loading' | 'no-league' | 'pick' | 'main'>('loading');
  const [teams, setTeams] = useState<Team[]>([]);
  const [error, setError] = useState('');
  const [note, setNote] = useState('');
  const [pool, setPool] = useState<Pool>();
  const [plans, setPlans] = useState<Plan[]>([]);
  const [position, setPosition] = useState('');
  const [sort, setSort] = useState('');

  const call = useCallback((command: NativeRequest['command'], mode: 'read' | 'execute',
    args: Obj = {}) => port({ command, mode, arguments: { league, ...args } } as NativeRequest)
    .then(r => r.data), [port, league]);

  const readPlans = useCallback(async () => {
    setPlans(parsePlans(await call('mflmoves.plans.read', 'read')));
  }, [call]);
  const readPool = useCallback(async () => {
    setPool(parsePool(await call('mflmoves.pool.read', 'read',
      { ...(position ? { position } : {}), ...(sort ? { sort } : {}) })));
  }, [call, position, sort]);

  useEffect(() => {
    let live = true;
    void (async () => {
      try {
        const found = parseTeams(await call('mflmoves.teams.read', 'read'));
        if (!live) return;
        setTeams(found.teams);
        setState(found.yours ? 'main' : 'pick');
      } catch (e) {
        if (!live) return;
        if (NO_LEAGUE.test(say(e))) setState('no-league');
        else { setError(say(e)); setState('pick'); }
      }
    })();
    return () => { live = false; };
  }, [call]);

  useEffect(() => {
    if (state !== 'main') return;
    let live = true;
    readPool().catch(e => { if (live) setError(say(e)); });
    readPlans().catch(e => { if (live) setError(say(e)); });
    return () => { live = false; };
  }, [state, readPool, readPlans]);

  const pickTeam = async (team: string) => {
    setError('');
    try {
      await call('mflmoves.team.set', 'execute', { team });
      setState('main');
    } catch (e) { setError(say(e)); }
  };
  const refresh = async () => {
    setError(''); setNote('');
    try {
      const row = obj(await call('mflmoves.refresh', 'execute'));
      setNote(row.askedMfl === false
        ? 'MFL was not asked again: the held copy is under 5 minutes old.' : str(row.words) || 'Refreshed from MFL.');
      await readPool();
    } catch (e) { setError(say(e)); }
  };
  const act = async (command: 'mflmoves.plan.done' | 'mflmoves.plan.withdraw', plan: string) => {
    setError('');
    try { await call(command, 'execute', { plan }); await readPlans(); } catch (e) { setError(say(e)); }
  };
  const open = async (plan: string) => {
    setError('');
    try {
      const url = str(obj(await call('mflmoves.plan.link', 'read', { plan })).url);
      if (url) window.open(url, '_blank', 'noopener');
    } catch (e) { setError(say(e)); }
  };

  if (state === 'loading') return <p role="status">Opening My MFL moves…</p>;
  if (state === 'no-league') return <p role="status">{NO_LEAGUE_LINE}</p>;
  if (state === 'pick') return <section className={box} aria-label="Pick your MFL team">
    <h3>Which MFL team is yours?</h3>
    {error && <p role="alert">{error}</p>}
    <ul>{teams.map(t => <li key={t.id}>{t.name}{' '}
      <button className={btn} onClick={() => { void pickTeam(t.id); }}>This is my team</button></li>)}</ul>
  </section>;

  return <section aria-label="My MFL moves">
    {error && <p role="alert">{error}</p>}
    <div className={box}>
      <button className={btn} onClick={() => { void refresh(); }}>Refresh from MFL</button>
      {pool && <span className="ml-2">{ageWords(pool.ageSeconds)}</span>}
      {note && <p role="status">{note}</p>}
    </div>
    <div className={box}>
      <h3>Free agents</h3>
      {pool && <>
        <label>Position<select className={field} value={position}
          onChange={e => setPosition(e.target.value)}><option value="">All</option>
          {pool.positions.map(p => <option key={p} value={p}>{p}</option>)}</select></label>{' '}
        <label>Sort<select className={field} value={sort} onChange={e => setSort(e.target.value)}>
          <option value="">Default</option>
          {pool.sorts.map(s => <option key={s} value={s}>{s}</option>)}</select></label>
        <table><thead><tr><th>Name</th><th>Pos</th><th>Team</th><th>Age</th><th>Minimum</th><th>Value</th>
        </tr></thead><tbody>{pool.rows.map(r => <tr key={r.id}><td>{r.name}</td><td>{r.position}</td>
          <td>{r.team}</td><td>{r.age}</td><td>{r.minimum}</td><td>{r.value}</td></tr>)}</tbody></table></>}
    </div>
    <BidForm call={call} pool={pool?.rows ?? []} roster={roster} saved={readPlans} />
    <LineupForm call={call} roster={roster} week={week} saved={readPlans} />
    <div className={box}>
      <h3>My plans</h3>
      {plans.length === 0 && <p>No plans yet.</p>}
      <ul>{plans.map(p => <li key={p.id}><b>{p.state}</b> {p.words}{' '}
        <button className={btn} onClick={() => { void open(p.id); }}>Open on MFL</button>
        {p.canDone && <button className={btn} onClick={() => { void act('mflmoves.plan.done', p.id); }}>
          I did it</button>}
        {p.canWithdraw && <button className={btn} onClick={() => {
          void act('mflmoves.plan.withdraw', p.id); }}>Withdraw</button>}</li>)}</ul>
    </div>
  </section>;
}

type Call = (command: NativeRequest['command'], mode: 'read' | 'execute', args?: Obj) => Promise<unknown>;

function Verdicts({ v }: { v?: Verdict }) {
  if (!v) return null;
  return <div role="status">{v.words && <p>{v.words}</p>}
    {v.problems.length > 0 && <ul>{v.problems.map(p => <li key={p}>{p}</li>)}</ul>}</div>;
}

function BidForm({ call, pool, roster, saved }: { call: Call; pool: PoolRow[]; roster: MflRosterPlayer[];
  saved: () => Promise<void> }) {
  const [add, setAdd] = useState('');
  const [amount, setAmount] = useState('');
  const [years, setYears] = useState('');
  const [drop, setDrop] = useState<string[]>([]);
  const [verdict, setVerdict] = useState<Verdict>();
  const [message, setMessage] = useState('');
  const body = () => ({ add, amount: Number(amount), ...(years ? { years: Number(years) } : {}),
    ...(drop.length ? { drop } : {}) });
  const check = async () => {
    setMessage('');
    try { setVerdict(parseVerdict(await call('mflmoves.bid.check', 'read', body()), false)); }
    catch (e) { setVerdict(undefined); setMessage(say(e)); }
  };
  const save = async () => {
    setMessage('');
    try { await call('mflmoves.plan.save', 'execute', { kind: 'bid', body: body() });
      setMessage('Plan saved.'); await saved(); }
    catch (e) { setMessage(say(e)); }
  };
  return <div className={box}><h3>Plan a waiver bid</h3>
    <label>Player<select className={field} value={add} onChange={e => setAdd(e.target.value)}>
      <option value="">Choose a player</option>
      {pool.map(r => <option key={r.id} value={r.id}>{r.name} · {r.position}</option>)}</select></label>{' '}
    <label>Amount ($)<input className={field} inputMode="decimal" value={amount}
      onChange={e => setAmount(e.target.value)} /></label>{' '}
    <label>Years<input className={field} inputMode="numeric" value={years}
      onChange={e => setYears(e.target.value)} /></label>
    <fieldset><legend>Drops</legend>{roster.map(p => <label key={p.id} className="mr-3">
      <input type="checkbox" checked={drop.includes(p.id)} onChange={e => setDrop(d => e.target.checked
        ? [...d, p.id] : d.filter(x => x !== p.id))} />{p.name}</label>)}</fieldset>
    <button className={btn} disabled={!add || !amount} onClick={() => { void check(); }}>Check</button>{' '}
    <button className={btn} disabled={!add || !amount} onClick={() => { void save(); }}>Save plan</button>
    {message && <p role="status">{message}</p>}<Verdicts v={verdict} /></div>;
}

function LineupForm({ call, roster, week, saved }: { call: Call; roster: MflRosterPlayer[];
  week?: number; saved: () => Promise<void> }) {
  const [wk, setWk] = useState(week ? String(week) : '');
  const [starters, setStarters] = useState<string[]>([]);
  const [verdict, setVerdict] = useState<Verdict>();
  const [message, setMessage] = useState('');
  const body = () => ({ week: Number(wk), starters });
  const check = async () => {
    setMessage('');
    try { setVerdict(parseVerdict(await call('mflmoves.lineup.check', 'read', body()), true)); }
    catch (e) { setVerdict(undefined); setMessage(say(e)); }
  };
  const save = async () => {
    setMessage('');
    try { await call('mflmoves.plan.save', 'execute', { kind: 'lineup', body: body() });
      setMessage('Plan saved.'); await saved(); }
    catch (e) { setMessage(say(e)); }
  };
  return <div className={box}><h3>Plan a lineup</h3>
    <label>Week<input className={field} inputMode="numeric" value={wk}
      onChange={e => setWk(e.target.value)} /></label>
    <fieldset><legend>Starters</legend>{roster.map(p => <label key={p.id} className="mr-3">
      <input type="checkbox" checked={starters.includes(p.id)} onChange={e => setStarters(s => e.target.checked
        ? [...s, p.id] : s.filter(x => x !== p.id))} />{p.name}</label>)}</fieldset>
    <button className={btn} disabled={!wk || starters.length === 0} onClick={() => { void check(); }}>
      Check</button>{' '}
    <button className={btn} disabled={!wk || starters.length === 0} onClick={() => { void save(); }}>
      Save plan</button>
    {message && <p role="status">{message}</p>}<Verdicts v={verdict} /></div>;
}
