import { useMemo } from 'react';
import type { AssetBoard } from '../assets/contract';
import type { Snapshot } from '../data/contract';
import type { HQCap, HQSettings } from './hqModel';
import { capacity, money, unavailable } from './hqModel';
import type { Injury } from './hqPlayerData';
import type { Readings } from '../contracts/contract';
import type { RosterFacts } from './hqRosterFacts';
import { researchFact } from './hqPlayerData';
import { commands } from '../commands/registry';
import { useContractRead } from '../contracts/Contracts';
import { whatIfProvider } from '../contracts/whatif';
import { contractsProvider } from '../contracts/provider';
import { hqEvents } from './hqReads';

export function CapRunway({ years }: { years?: HQCap[] }) {
  return <div className="run">{years?.length ? years.map(year => <div className={`y ${(year.room ?? 0) < 0 ? 'tight' : ''}`} key={year.season}>
    <span>{year.season}</span><div className="bar" title={`Committed ${money(year.committed)} · Dead ${money(year.dead)} · Room ${money(year.room)} of ${money(year.max)}`}>
      {([['seg-c', year.committed], ['seg-d', year.dead]] as const).map(([css, value]) => <span key={css} className={css}
        style={{ width: year.max && value !== undefined ? `${Math.max(0, Math.min(100, value / year.max * 100))}%` : '0%' }} />)}
    </div><b>{money(year.room)}</b></div>) : <p className="empty">{unavailable}</p>}</div>;
}
function ContractRow({ row, season, mine, years, facts, select, assets }: { assets?: AssetBoard; row: { id: string; name: string; position: string; salary: number; status: string };
  season: number; mine: string; years: number[]; facts?: RosterFacts; select: (id: string) => void }) {
  const reading = useContractRead(row.id, () => contractsProvider().player(row.id), hqEvents);
  const asset = assets?.values.find(v => v.id === row.id);
  const dynasty = asset?.dynasty.held ? asset.dynastyValue : undefined;
  return <tr tabIndex={0} onClick={() => select(row.id)} onKeyDown={e => { if (e.key === 'Enter') select(row.id); }}>
    <td><span className="pg">{row.position}</span></td><td>{row.name}</td>
    {years.map(year => { const value = reading.reading?.value?.years.find(y => y.season === year);
      return <td key={year} className="yr" title={value?.reason}>{value ? money(value.cost.cents) : year === season ? money(row.salary) : unavailable}</td>; })}
    <td>{researchFact(facts?.get(row.id)?.research, 'Dead cap if cut today') ?? unavailable}</td><td title={asset?.dynasty.detail}>{asset?.dynasty.held ? asset.dynasty.text : unavailable}</td><td>{dynasty && dynasty > 0 ? (row.salary / 1_000_000 / dynasty).toFixed(1) : unavailable}</td>
  </tr>;
}
export function HQContracts({ snapshot, mine, contracts, runway, facts, select, native = false, assets }: {
  snapshot: Snapshot; mine: string; contracts?: Readings; runway?: HQCap[]; facts?: RosterFacts; select: (id: string) => void; native?: boolean; assets?: AssetBoard;
}) {
  const season = snapshot.league.value.season;
  const roster = snapshot.rosters.value.find(r => r.franchiseId === mine)?.players ?? [];
  const years = runway?.length ? runway.map(y => y.season) : Array.from(new Set([season,
    ...contracts?.cap.value?.future.map(y => y.season) ?? []]));
  const rows = contracts?.table.value?.rows.map(r => ({ id: r.id, name: r.name, position: r.position,
    salary: r.salary.cents, status: r.status })) ?? roster.map(r => ({ id: r.id,
      name: snapshot.players.value.find(p => p.id === r.id)?.name ?? r.id,
      position: snapshot.players.value.find(p => p.id === r.id)?.position ?? '—', salary: r.salary, status: r.contractStatus }));
  return <section className="ws capws on" id="ws-cap"><div className="ctable">
    <div className="ch"><b>CONTRACTS · {rows.length}</b><span>final year ends in<i className="key-cap rfa" />RFA · team can match<i className="key-cap ufa" />UFA · walks free</span></div>
    <div className="ct-wrap"><table className="ct"><thead><tr><th>POS</th><th>PLAYER</th>{years.map(y => <th key={y}>{y}</th>)}<th>DEAD IF CUT</th><th>VALUE</th><th>$ / VALUE</th></tr></thead>
      <tbody>{rows.map(row => native ? <tr key={row.id} tabIndex={0} onClick={() => select(row.id)} onKeyDown={e => { if (e.key === 'Enter') select(row.id); }}>
        <td><span className="pg">{row.position}</span></td><td>{row.name}</td>{years.map(y => <td className="yr" key={y}>{y === season ? money(row.salary) : unavailable}</td>)}<td>{unavailable}</td><td>{unavailable}</td><td>{unavailable}</td>
      </tr> : <ContractRow key={row.id} {...{ row, season, mine, years, facts, select, assets }} />)}</tbody></table>
      {!rows.length && <p className="empty">{unavailable}</p>}</div>
  </div><div className="cside"><div className="ch"><b>CAP RUNWAY</b><span>committed · dead · room</span></div><CapRunway years={runway} />
    <div className="ch"><b>DECISIONS DUE</b><span>soonest first</span></div><div className="dec" style={{ flex: 'none' }}><p className="empty">{unavailable}</p></div>
    <div className="ch" style={{ borderTop: '2px solid var(--black)' }}><b>DEAD CAP</b><span>money on players you no longer have</span></div>
    <div className="dec"><p className="empty">{unavailable}</p></div></div></section>;
}
export function HQReserves({ kind, snapshot, mine, settings, injuries, close, native = false }: {
  kind: 'ir' | 'taxi'; snapshot: Snapshot; mine: string; settings: HQSettings; injuries?: Injury[]; close: () => void; native?: boolean;
}) {
  const roster = snapshot.rosters.value.find(r => r.franchiseId === mine)?.players ?? [];
  const players = roster.filter(p => p.rosterStatus === (kind === 'ir' ? 'IR' : 'TAXI_SQUAD'));
  const active = roster.filter(p => p.rosterStatus === 'ROSTER');
  const limit = kind === 'ir' ? settings.ir : settings.taxi;
  const candidates = active.filter(p => snapshot.eligibility.players[p.id]?.[kind].verdict === 'eligible');
  const returns = players.flatMap(p => {
    const injury = injuries?.find(i => i.id === p.id);
    return injury?.returnWeek && injury.returnSeason === snapshot.league.value.season ? [{ ...injury, name: snapshot.players.value.find(v => v.id === p.id)?.name ?? p.id }] : [];
  }).sort((a, b) => a.returnWeek! - b.returnWeek!);
  const action = (id: string) => { void commands.dispatch(kind === 'ir' ? 'roster.ir' : 'roster.taxi', { subject: { kind: 'player', id, franchiseId: mine } }); };
  return <section className="bdrawer open" aria-label="Roster drawer"><div className="bd-head"><h2>{kind === 'ir' ? 'Injured reserve' : 'Taxi squad'}</h2>
    <span className="sub">{capacity(players.length, limit)}</span><button className="x" onClick={close}>ESC</button></div>
    <div className="bd-body"><div className="bd-main"><table className="ct bt"><thead><tr>{(kind === 'ir'
      ? ['POS', 'PLAYER', 'INJURY', 'BACK', 'IN', 'SALARY', 'NOTE', ''] : ['POS', 'PLAYER', 'NFL YR', 'TAXI LEFT', 'SALARY', 'VALUE', 'NOTE', '']).map((label, i) => <th key={i}>{label}</th>)}</tr></thead>
      <tbody>{players.map(p => { const player = snapshot.players.value.find(v => v.id === p.id); const injury = injuries?.find(i => i.id === p.id);
        return <tr key={p.id}><td><span className="pg">{player?.position ?? '—'}</span></td><td>{player?.name ?? p.id}<small>{player?.team}</small></td>
          {kind === 'ir' ? <><td>{injury?.details || unavailable}</td><td title={injury?.returnReason}>{injury?.returnWeek ? `${injury.returnSeason} · Wk ${injury.returnWeek}` : unavailable}</td><td>{unavailable}</td><td>{money(p.salary)}</td><td>{injury?.returnReason || unavailable}</td></>
            : <><td>{unavailable}</td><td>{unavailable}</td><td>{money(p.salary)}</td><td>{unavailable}</td><td>{snapshot.eligibility.players[p.id]?.taxi.reason ?? unavailable}</td></>}
          <td><button className="btn" disabled={native || snapshot.eligibility.players[p.id]?.[kind].verdict === 'ineligible'} onClick={() => action(p.id)}>{kind === 'ir' ? 'ACTIVATE' : 'PROMOTE'}</button></td></tr>;
      })}</tbody></table>{!players.length && <p className="empty">{unavailable}</p>}
      {kind === 'ir' && <><div className="ch" style={{ borderTop: '1px solid var(--edge)' }}><b>RETURNS</b><span>plan the active spot before they come back</span></div>
        {returns.length ? <div className="tl" style={{ gridTemplateColumns: `repeat(${returns.length},1fr)` }}>{returns.map(r => <div key={r.id}>WK {r.returnWeek}<b>{r.name}</b></div>)}</div> : <p className="empty">{unavailable}</p>}</>}
    </div><div className="bd-side"><div className="sum"><div><b>{capacity(players.length, limit)}</b><span>{kind.toUpperCase()} SLOTS</span></div><div><b>{capacity(active.length, settings.active)}</b><span>ACTIVE</span></div>
      <div><b>{kind === 'taxi' ? money(players.reduce((n, p) => n + p.salary, 0)) : unavailable}</b><span>{kind === 'ir' ? 'IR PAY · ON CAP' : 'TAXI PAY'}</span></div>
      <div><b>{kind === 'ir' ? returns[0] ? `Wk ${returns[0].returnWeek}` : unavailable : limit === undefined ? unavailable : limit - players.length}</b><span>{kind === 'ir' ? 'NEXT RETURN' : 'OPEN TAXI SLOT'}</span></div></div>
      <div className="ch"><b>{kind === 'ir' ? 'IR CANDIDATES' : 'COULD STASH'} · {candidates.length}</b><span>{kind === 'ir' ? 'on your roster' : 'active, still eligible'}</span></div>
      {candidates.map(p => <div className="drow" key={p.id}><span className="pg">{snapshot.players.value.find(v => v.id === p.id)?.position ?? '—'}</span><span>{snapshot.players.value.find(v => v.id === p.id)?.name ?? p.id}</span>
        <button className="btn" disabled={native} onClick={() => action(p.id)}>TO {kind.toUpperCase()}</button><span className="why">{snapshot.eligibility.players[p.id]?.[kind].reason}</span></div>)}
    </div></div></section>;
}
export function HQWhatIf({ mine, close }: { mine: string; close: () => void }) {
  const provider = useMemo(whatIfProvider, []);
  const { reading, error } = useContractRead(mine, () => provider.query(mine), provider);
  const value = reading?.value;
  return <section className="bdrawer open" aria-label="Roster drawer"><div className="bd-head"><h2>What-if</h2><span className="sub">Tick moves to stack them. Nothing is saved or sent.</span><button className="x" onClick={close}>ESC</button></div>
    <div className="bd-body"><div className="bd-main">{value?.choices.map(choice => {
      const move = value.moves.find(m => m.player === choice.player);
      return <button key={choice.player} className={`wi ${move ? 'on' : ''}`} disabled={value.stale} onClick={() => { void commands.dispatch('contracts.set', { field: 'whatif', value: move ? { mine, action: 'undo', move: move.id } : { mine, action: 'cut', player: choice.player } }); }}>
        <span className="box" /><span className="k">CUT</span><span>{choice.cutLabel}</span><span className="d">{move?.charge.text ?? ''}</span></button>;
    })}{(!value || !value.choices.length) && <p className="empty" title={error}>{unavailable}</p>}</div>
      <div className="bd-side"><div className="sum">{value?.figures.map(figure => <div key={figure.label}><b>{figure.after}</b><span>{figure.label}</span><span className="delta">BEFORE {figure.before} ┆ AFTER {figure.after}</span></div>)}</div>
        <div className="tray" style={{ background: 'none', marginTop: 'auto' }}><span className="msg">{value?.notice || `${value?.moves.length ?? 0} stacked`}</span>
          <button className="btn sp" disabled={!value?.moves.length} onClick={() => { void commands.dispatch('contracts.set', { field: 'whatif', value: { mine, action: 'reset' } }); }}>CLEAR</button>
          <button className="btn g2" disabled={!value?.moves.length || value.stale} onClick={() => { for (const move of value?.moves.filter(m => !m.plan) ?? []) void commands.dispatch('contracts.set', { field: 'release', value: { mine, action: 'draft', move: move.id } }); }}>SEND TO MY MOVES</button>
        </div></div></div></section>;
}
