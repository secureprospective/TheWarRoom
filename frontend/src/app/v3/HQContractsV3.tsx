// Desktop v3 Franchise HQ, Contracts and cap, after TheWarRoom-mock-v3-FINAL.html (screens.js hq.contracts):
// the contract table, cap runway, contracts expiring, what-if, dead cap and committed contract moves. Every
// figure is Franchise HQ's own read (readNativeHQ: roster contracts, annual costs, runway, decisions, moves).
import { useState, type CSSProperties, type ReactNode } from 'react';
import { HQAct } from '../shell/HQAct';
import type { NativeHQ } from '../commissioner/hqReading';
import type { HQCap } from '../shell/hqModel';

const cash = (v?: number | null) => v == null ? '—' : `$${(v / 100_000_000).toFixed(1)}M`;
const mil = (v?: number | null) => v == null ? '' : (v / 100_000_000).toFixed(1);
const St = ({ children, k }: { children: ReactNode; k?: string }) =>
  <div className="st">{k && <span className="k">{k}</span>}<div className="eng">{children}</div></div>;
const Tag = ({ status }: { status: string }) => ['RFA', 'UFA'].includes(status)
  ? <span className="tag" style={{ background: `var(--${status.toLowerCase()})` }}>{status}</span>
  : <span className="tag">{status}</span>;

function Panel({ title, badge, children, foot, style, tight }: { title: string; badge: ReactNode;
  children: ReactNode; foot?: ReactNode; style?: CSSProperties; tight?: boolean }) {
  return <section className={`pc panel sz-L${tight ? ' tight' : ''}`} style={style}>
    <div className="ph">{badge}<b className="pt">{title}</b><span className="sp" /></div>
    <div className="pb">{children}</div>
    {foot && <div className="pf"><span className="ell">{foot}</span></div>}</section>;
}

export type ContractRow = { id: string; name: string; pos: string; salary: number | null; status: string;
  years: Map<number, number>; ends?: number };

/** One row per rostered player: each season's cost, the final season, and how the deal ends. */
export function contractRows(reading: NativeHQ, mine: string): ContractRow[] {
  const { snapshot } = reading;
  const season = snapshot.league.value.season;
  const roster = snapshot.rosters.value.find(r => r.franchiseId === mine)?.players ?? [];
  const people = new Map(snapshot.players.value.map(p => [p.id, p]));
  return roster.map(r => {
    const annual = reading.contracts?.get(r.id) ?? [];
    const years = new Map(annual.map(y => [y.season, y.cents]));
    if (!years.has(season) && r.salary !== null) years.set(season, r.salary);
    const ends = annual.length ? Math.max(...annual.map(y => y.season))
      : r.yearsRemaining !== undefined ? season + r.yearsRemaining - 1 : undefined;
    return { id: r.id, name: people.get(r.id)?.name ?? r.id, pos: people.get(r.id)?.position ?? '—',
      salary: r.salary, status: r.contractStatus, years, ends };
  }).sort((a, b) => (b.salary ?? 0) - (a.salary ?? 0));
}

export function HQContractsV3({ reading, mine, badge, select, whatIf, release, decide, chosen }: {
  reading: NativeHQ; mine: string; badge: ReactNode; select: (id: string) => void; whatIf: () => void;
  release?: () => void; decide?: () => void; chosen?: string }) {
  const [filter, setFilter] = useState(0);
  const season = reading.snapshot.league.value.season;
  const all = contractRows(reading, mine);
  const rows = filter === 1 ? all.filter(r => r.ends === season) : filter === 2 ? all.slice(0, 12) : all;
  const seasons = [season, season + 1, season + 2, season + 3];
  const runway: HQCap[] = reading.runway ?? [];
  const top = Math.max(1, ...runway.map(y => y.max ?? 0));
  const expiring = all.filter(r => r.ends === season);
  const due = reading.decisions?.find(d => d.at);
  const dueWords = due?.at ? `due ${new Date(due.at).toLocaleDateString('en-US', { month: 'short',
    day: 'numeric', timeZone: 'America/Chicago' })}` : 'no deadline on the league calendar';
  const moves = (reading.moves ?? []).filter(m => /release|cut|contract|extend|restructure|tag/i
    .test(m.intent));
  const owner = Boolean(decide);
  const picked = all.find(r => r.id === chosen);
  return <div className="wbody scr scr-hq"><div className="lay" style={{
    gridTemplateColumns: 'minmax(0,1.5fr) minmax(0,1fr)',
    gridTemplateRows: 'minmax(0,1.1fr) minmax(0,1fr) minmax(0,.8fr)' }}>
    <Panel title="Contracts" badge={badge} style={{ gridRow: '1/4' }}
      foot="RFA lime · UFA yellow: classification only">
      <div className="fbar">{['All', 'Expiring', 'Top salaries'].map((w, i) => <HQAct key={w} className="chip"
        selected={filter === i} onAction={() => setFilter(i)}>{w}</HQAct>)}</div>
      {!rows.length
        ? <St>{filter === 1 ? `No contract ends in ${season}.` : 'No contracts on this roster.'}</St>
        : <table className="t"><tbody><tr><th>Player</th><th>Pos</th>
          {seasons.map(y => <th key={y} className="n">{y}</th>)}<th>Ends</th></tr>
          {rows.map(r => <tr key={r.id}>
            <td className={r.id === chosen ? 'w' : undefined}>
              <HQAct className="lk" onAction={() => select(r.id)}>{r.name}</HQAct></td>
            <td><span className="pos">{r.pos}</span></td>
            {seasons.map(y => <td key={y} className="n">{r.years.has(y) ? mil(r.years.get(y))
              : r.ends !== undefined && y === r.ends + 1 ? <Tag status={r.status} /> : ''}</td>)}
            <td className="n d">{r.ends ?? '—'}</td></tr>)}</tbody></table>}
    </Panel>
    <div className="lay" style={{ gridTemplateColumns: '1fr 1fr', padding: 0 }}>
      <Panel title="Cap runway" badge={badge} foot="solid committed · striped dead · figure = room">
        {!runway.length ? <St>The cap runway is not available for this league.</St>
          : <div className="capr">{runway.slice(0, 5).map(y => <div key={y.season}
            title={`Committed ${cash(y.committed)} · dead ${cash(y.dead)} · room ${cash(y.room)} of ${
              cash(y.max)}`}><b>{cash(y.room)}</b>
            <i className="dead" style={{ height: `${(y.dead ?? 0) / top * 100}%` }} />
            <i style={{ height: `${(y.committed ?? 0) / top * 100}%` }} />{y.season}</div>)}</div>}
      </Panel>
      <Panel title="Contracts expiring / decisions due" badge={badge} tight
        foot={`${dueWords} · RFA can be matched · UFA walks`}>
        {!expiring.length ? <St>No contract ends this season.</St>
          : expiring.map(r => <div key={r.id} className="row"
            style={{ gridTemplateColumns: '30px minmax(0,1fr) 54px 44px auto' }}>
            <span className="pos">{r.pos}</span>
            <span className="ell"><HQAct className="lk" onAction={() => select(r.id)}>{r.name}</HQAct></span>
            <span className="v">{cash(r.salary)}</span><Tag status={r.status} />
            <span className="acts"><HQAct className="btn sm" disabled={!owner} onAction={decide}
              title={owner ? undefined : 'Only this franchise\'s owner decides its contracts'}>
              Decide ▾</HQAct>
            </span></div>)}
      </Panel>
    </div>
    <div className="lay" style={{ gridTemplateColumns: '1fr 1fr', padding: 0 }}>
      <Panel title="What-if" badge={badge} tight foot="quotes only · commit in Franchise HQ">
        <St k="TRY A MOVE">Stack cuts and trades against the league's own cap rules and see the room they
          make. Nothing is committed.</St>
        <div className="tbar" style={{ justifyContent: 'center' }}>
          <HQAct className="btn sm" onAction={whatIf}>Open what-if ▸</HQAct></div>
      </Panel>
      <Panel title="Dead-cap ledger" badge={badge} tight foot="money on your books for players who are gone">
        {!runway.some(y => y.dead) ? <St>No dead money on the books.</St>
          : runway.filter(y => y.dead).map(y => <div key={y.season} className="row"
            style={{ gridTemplateColumns: 'minmax(0,1fr) 44px 54px' }}>
            <span className="ell">Dead cap charged</span><span className="m">{y.season}</span>
            <span className="v r">{cash(y.dead)}</span></div>)}
      </Panel>
    </div>
    <Panel title="Committed contract moves" badge={badge}
      foot="each move is final at the deadline · undo until then">
      <div className="tbar">
        <HQAct className="btn sm" disabled={!release || !picked} onAction={release}
          title={!release ? 'Only this franchise\'s owner releases' : 'Pick the player in the table first'}>
          {picked ? `Release ${picked.name}` : 'Release a player'}</HQAct>
        <span className="m sp">{moves.length ? `${moves.length} pending` : ''}</span></div>
      {!moves.length ? <St k="NO MOVES THIS SEASON">{owner
        ? 'Release a player from the button above, or decide'
        + ' an expiring contract.' : 'Moves are visible only to this franchise\'s owner.'}</St>
        : moves.map(m => <div key={m.id} className="row"
          style={{ gridTemplateColumns: 'minmax(0,1fr) auto' }}>
          <span className="ell">{m.note || m.intent}</span><span className="m">{m.state}</span></div>)}
    </Panel>
  </div></div>;
}
