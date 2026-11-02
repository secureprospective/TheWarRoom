import { useMemo, useState } from 'react';
import { entriesOf, type Entry } from '../../commissioner/controls';
import { pendingWhen, type RuleRow } from '../../commissioner/rules';
import { Kv, NotYet, useDefaultInspector, useRoom, when, type Room } from './room';
import { useFit } from './useFit';

/** One line of the rules list: a league rule from the rules reading, or a census control that is not
 *  built yet (the mock's PLANNED rows). */
/** The mock's eight families over the rules reading's sections (control-room-mock-v1 FAM). */
const FAMILIES: [string, string[]][] = [
  ['League & teams', ['League shape', 'Player pools']],
  ['Rosters & lineups', ['Roster', 'Lineup', 'IR', 'Taxi squad']],
  ['Scoring & standings', ['Scoring', 'Standings']],
  ['Cap & contracts', ['Salary cap', 'Contracts']],
  ['Player movement', ['Waivers', 'Free-agent bids', 'Trades']],
  ['Season shape', ['Schedule', 'Playoffs', 'Calendar']],
  ['Draft', ['Draft']],
  ['Rulebook & notifications', ['House rules', 'Notifications']],
];
export const familyOf = (section: string) => FAMILIES.find(([, s]) => s.includes(section))?.[0];

type Line = { key: string; section: string; label: string; value: string; row?: RuleRow; planned?: Entry };

function stateOf(l: Line) {
  if (l.planned) return <span className="cr-st plan" title="In the control census, not built yet">PLANNED</span>;
  if (l.row!.pending.length) return <span className="cr-st pend" title="Saved; takes effect at the next rollover">
    NEXT ROLLOVER</span>;
  if (l.row!.state === 'Needs a ruling') return <span className="cr-st need"
    title="No value set: the app will not guess. Click the row to set it.">NEEDS A RULING</span>;
  return <span className="cr-st set">SET</span>;
}

export function inspectRule(room: Room, l: Line, owner?: Entry) {
  const r = l.row;
  if (!r) {
    room.inspect(<><div className="sub">{familyOf(l.section) ?? l.section} › {l.section}</div><h3>{l.label}</h3>
      <div className="big">—</div>
      <div className="foot"><div className="lock">Planned: {l.planned?.note || 'not built yet.'}</div></div></>);
    return;
  }
  const choices = r.schema.choices.map(c => c.label);
  const parts = r.schema.fields.map(f => f.label);
  room.inspect(<><div className="sub">{familyOf(l.section) ?? l.section} › {l.section}</div><h3>{r.label}</h3><div className="big">{r.value}</div>
    <Kv rows={[['State', r.pending.length ? 'Pending' : r.state], ['Source', r.citation || '—'],
      ['Command', r.field], ['Competitive', r.competitive ? 'yes' : 'no']]} />
    {choices.length > 0 && <><div className="lbl">CHOICES</div><div className="sub">{choices.join(' · ')}</div></>}
    {parts.length > 0 && <><div className="lbl">PARTS</div><div className="sub">{parts.join(' · ')}</div></>}
    <div className="lbl">CHANGES THESE OWNER SCREENS</div>
    <div className="gov">{owner ? <span>{owner.groupLabel} · {owner.name}</span> : <NotYet />}</div>
    <div className="lbl">HISTORY</div><div className="hist">
      <div>{r.governed ? `${when(r.governed.when)} · ${r.governed.who} · ${r.governed.change}`
        : `Season start · ${r.citation || 'set'}`}</div>
      {r.pending.map(p => <div key={p.entry}>Pending: {p.value} · {pendingWhen(p)}</div>)}
    </div>
    <div className="foot">
      <button className="cr-btn pri" data-edit={r.field} onClick={() => room.rule(r.field)}>CHANGE ›</button>
      <button className="cr-btn" onClick={() => room.run('rule.exception')}>ONE-OFF EXCEPTION ›</button>
    </div></>);
}

export function RulesView() {
  const room = useRoom();
  const { rules, controls } = room.state;
  const [chip, setChip] = useState('all');
  const [page, setPage] = useState(0);
  const [sel, setSel] = useState('');
  const { panel, per } = useFit(19);
  const entries = useMemo(() => (controls ? entriesOf(controls) : []), [controls]);
  const lines: Line[] = useMemo(() => [
    ...(rules?.sections ?? []).flatMap(s => s.rows.map(r => ({ key: r.field, section: s.label, label: r.label,
      value: r.value, row: r }))),
    ...entries.filter(e => !e.built).map(e => ({ key: e.id, section: e.groupLabel, label: e.name, value: '—',
      planned: e })),
  ], [rules, entries]);
  const need = lines.filter(l => l.row?.state === 'Needs a ruling');
  const pend = lines.filter(l => l.row?.pending.length);
  const plan = lines.filter(l => l.planned);
  const rows = chip === 'all' ? lines : chip === 'need' ? need : chip === 'pend' ? pend : chip === 'plan' ? plan
    : lines.filter(l => familyOf(l.section) === chip);
  const pages = Math.max(1, Math.ceil(rows.length / per));
  const at = Math.min(page, pages - 1);
  const vis = rows.slice(at * per, at * per + per);
  useDefaultInspector(<><h3>Rules</h3><div className="sub">All {lines.length} league rules on one list. The chips
    filter; they are not menus. Click any rule to see what it governs and change it.</div>
    <div className="lbl">PRESETS</div><div className="sub">A preset is a bundle of suggested values applied when a
      league is created; after that, each rule changes on its own here.</div></>, [lines.length]);
  const pick = (l: Line) => {
    setSel(l.key);
    inspectRule(room, l, entries.find(e => e.fields.some(f => f.field === l.key)));
  };
  const c = (k: string, label: string, n: number) => <button key={k} className="cr-chip" data-fam={k}
    aria-pressed={chip === k} onClick={() => { setChip(k); setPage(0); setSel(''); }}>{label}
    <span className="c">{n}</span></button>;
  return <div className="cr-col" style={{ flex: 1 }}>
    <div className="cr-chips">{c('all', 'ALL', lines.length)}
      {FAMILIES.map(([f]) => c(f, f, lines.filter(l => familyOf(l.section) === f).length))}
      {c('need', 'NEEDS A RULING', need.length)}{c('pend', 'PENDING', pend.length)}
      {c('plan', 'PLANNED (BREADTH)', plan.length)}</div>
    <div className="cr-panel" style={{ flex: 1 }} ref={panel}>
      {!rules && <div className="cr-note">{room.state.error || 'Reading the league rules…'}</div>}
      <table className="cr-t"><thead><tr><th style={{ width: '15%' }}>SECTION</th><th style={{ width: '27%' }}>RULE</th>
        <th>VALUE</th><th style={{ width: 150 }}>STATE</th><th style={{ width: 44 }}>⚖</th></tr></thead>
        <tbody>{vis.map(l => <tr key={l.key} className={`rw${sel === l.key ? ' sel' : ''}`} data-rule={l.key}
          tabIndex={0} onClick={() => pick(l)} onKeyDown={e => { if (e.key === 'Enter') pick(l); }}>
          <td style={{ color: 'var(--ink3)' }}>{l.section}</td><td>{l.label}</td><td className="v">{l.value}</td>
          <td>{stateOf(l)}</td><td title="competitive: changes default to next rollover">{l.row?.competitive ? '⚖' : ''}</td>
        </tr>)}</tbody></table>
      <div className="cr-pager"><span>{rows.length} rules · page {at + 1} of {pages}</span><span className="sp" />
        <button className="cr-btn sm" disabled={at === 0} onClick={() => setPage(at - 1)}>‹ PREV</button>
        <button className="cr-btn sm" disabled={at >= pages - 1} onClick={() => setPage(at + 1)}>NEXT ›</button>
        <span>⚖ = competitive: changes wait for next rollover unless you choose now</span></div>
    </div>
  </div>;
}
