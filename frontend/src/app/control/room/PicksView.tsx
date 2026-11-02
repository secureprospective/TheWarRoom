import { useState } from 'react';
import type { commissioner } from '../../commissioner/api';
import { pickActs } from './guide';
import { openPickAct } from './openers';
import { routeCommand } from './route';
import { Kv, ruleValue, useDefaultInspector, useRoom } from './room';
import { teamName } from './roomReads';
import { useFit } from './useFit';

type Pick = commissioner.NativePick;
const how = (p: Pick) => p.correction ? 'Correction' : p.compensatory ? 'Compensatory'
  : p.owner !== p.original ? 'Traded' : 'Original';
const status = (p: Pick) => p.player ? 'USED' : p.skipped ? 'SKIPPED' : 'ACTIVE';

export function PicksView() {
  const room = useRoom();
  const { data } = room;
  const picks = data.assets?.snapshot?.picks ?? [];
  const seasons = Array.from(new Set(picks.map(p => p.season))).sort();
  const [season, setSeason] = useState<number>();
  const [team, setTeam] = useState('All');
  const [page, setPage] = useState(0);
  const [sel, setSel] = useState('');
  const { panel, per } = useFit(16);
  const shown = season ?? seasons.find(s => s > (data.assets?.snapshot?.season ?? 0)) ?? seasons[0];
  const rows = picks.filter(p => p.season === shown && (team === 'All' || p.owner === team || p.original === team))
    .sort((a, b) => a.round - b.round || a.slot - b.slot);
  const pages = Math.max(1, Math.ceil(rows.length / per));
  const at = Math.min(page, pages - 1);
  const vis = rows.slice(at * per, at * per + per);
  const odd = picks.filter(p => p.season === shown && (p.owner !== p.original || p.correction || p.compensatory));
  const name = (f: string) => teamName(data, f);
  useDefaultInspector(<><h3>Picks</h3><div className="sub">Every pick in the league, any season. Click a pick for its
    history; use the actions on the left to change anything.</div><div className="lbl">LIMITS THAT STILL APPLY</div>
    <div className="sub">Draft rules: most picks per team {ruleValue(room, 'draft.maximumPicks')}; extras
      {' '}{ruleValue(room, 'draft.forfeitExtra')} (you can change that in Rules › Draft).</div></>);
  const key = (p: Pick) => `${p.season}:${p.round}:${p.original}:${p.slot}`;
  const history = (p: Pick) => {
    setSel(key(p));
    room.inspect(<><div className="sub">{p.season} round {p.round}</div>
      <h3>{name(p.original)} R{p.round} → {name(p.owner)}</h3>
      <Kv rows={[['Status', status(p)], ['Condition', 'none'], ['How', how(p)]]} />
      <div className="lbl">HISTORY</div><div className="hist">
        <div>Issued as {name(p.original)}&apos;s pick</div>
        {p.owner !== p.original && <div>{how(p)} to {name(p.owner)}</div>}</div>
      <div className="foot">
        <button className="cr-btn" onClick={() => openPickAct(room, 1)}>MOVE ›</button>
        <button className="cr-btn" onClick={() => openPickAct(room, 4)}>ADD CONDITION ›</button>
        <button className="cr-btn g3" onClick={() => openPickAct(room, 3)}>FORFEIT ›</button></div></>);
  };
  return <>
    <div className="cr-col" style={{ width: 280, flex: 'none' }}>
      <div className="cr-panel"><div className="cr-ph">DO SOMETHING WITH A PICK</div><div className="cr-pb cr-launch">
        {pickActs.map((p, i) => <button key={p[2]} data-pickact={i} onClick={() => openPickAct(room, i)}>
          <span><b>{p[0]}</b>{routeCommand(p[2]).kind === 'unbuilt' &&
            <> <span className="cr-st plan">RULED · NOT BUILT</span></>}<br /><span className="sub">{p[1]}</span></span>
          <span className="kb">›</span></button>)}</div></div>
      <div className="cr-note">Commissioner has full power over picks (ruling 10-30). New pick commands are <b>ruled, not
        built</b>; moving a pick exists today.</div>
    </div>
    <div className="cr-col" style={{ flex: 1 }}>
      <div className="cr-chips">{seasons.map(s => <button key={s} className="cr-chip" aria-pressed={s === shown}
        onClick={() => { setSeason(s); setPage(0); }}>{s}</button>)}<span style={{ width: 12 }} />
        <select aria-label="Team" value={team} onChange={e => { setTeam(e.target.value); setPage(0); }}
          style={{ background: 'var(--sunken)', border: '1px solid var(--edge)', color: 'var(--ink)', padding: '3px 6px' }}>
          <option>All</option>{(data.directory?.franchises ?? []).map(f => <option key={f.franchise}
            value={f.franchise}>{f.name}</option>)}</select>
        <span className="sub" style={{ marginLeft: 'auto' }}>{odd.length} non-standard picks in {shown ?? '—'}</span></div>
      <div className="cr-panel" style={{ flex: 1 }} ref={panel}>
        {!data.assets && <div className="cr-note">{data.errors.assets ?? 'Reading the picks…'}</div>}
        <table className="cr-t"><thead><tr><th style={{ width: 70 }}>ROUND</th><th>ORIGINAL</th><th>OWNED BY</th>
          <th>CONDITION</th><th>HOW</th><th style={{ width: 110 }}>STATUS</th></tr></thead>
          <tbody>{vis.map(p => <tr key={key(p)} className={`rw${sel === key(p) ? ' sel' : ''}`} onClick={() => history(p)}>
            <td className="v">R{p.round}</td><td>{name(p.original)}</td>
            <td>{p.owner !== p.original ? <b>{name(p.owner)}</b> : name(p.owner)}</td><td>—</td>
            <td style={{ color: 'var(--ink2)' }}>{how(p)}</td>
            <td>{status(p) === 'ACTIVE' ? <span className="cr-st set">ACTIVE</span>
              : <span className="cr-st need">{status(p)}</span>}</td></tr>)}</tbody></table>
        <div className="cr-pager"><span>{rows.length} picks · page {at + 1} of {pages}</span><span className="sp" />
          <button className="cr-btn sm" disabled={at === 0} onClick={() => setPage(at - 1)}>‹ PREV</button>
          <button className="cr-btn sm" disabled={at >= pages - 1} onClick={() => setPage(at + 1)}>NEXT ›</button></div>
      </div>
    </div>
  </>;
}
