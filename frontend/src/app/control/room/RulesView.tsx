import { useRoomSurface } from './useRoomSurface';
import { Act } from '../../commands/Act';
import { useMemo, useState } from 'react';
import { entriesOf, type Entry } from '../../commissioner/controls';
import { pendingWhen, type RuleRow } from '../../commissioner/rules';
import { Kv, NotYet, useDefaultInspector, useRoom, when, type Room } from './room';
import { useFit } from './useFit';
import { PresetCompare } from './PresetCompare';

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
  if (l.planned)
    return (
      <span className="cr-st plan" title="In the control census, not built yet">
        PLANNED
      </span>
    );
  if (l.row!.pending.length)
    return (
      <span className="cr-st pend" title="Saved; takes effect at the next rollover">
        NEXT ROLLOVER
      </span>
    );
  if (l.row!.state === 'Needs a ruling')
    return (
      <span className="cr-st need" title="No value set: the app will not guess. Click the row to set it.">
        NEEDS A RULING
      </span>
    );
  return <span className="cr-st set">SET</span>;
}

export function inspectRule(room: Room, l: Line, owner?: Entry) {
  const r = l.row;
  if (!r) {
    room.inspect(
      <>
        <div className="sub">
          {familyOf(l.section) ?? l.section} ›{l.section}
        </div>
        <h3>{l.label}</h3>
        <div className="big">—</div>
        <div className="foot">
          <div className="lock">Planned: {l.planned?.note || 'not built yet.'}</div>
        </div>
      </>,
    );
    return;
  }
  const choices = r.schema.choices.map((c) => c.label);
  const parts = r.schema.fields.map((f) => f.label);
  room.inspect(
    <>
      <div className="sub">
        {familyOf(l.section) ?? l.section} ›{l.section}
      </div>
      <h3>{r.label}</h3>
      <div className="big">{r.value}</div>
      <Kv
        rows={[
          ['State', r.pending.length ? 'Pending' : r.state],
          ['Source', r.citation || '—'],
          ['Command', r.field],
          ['Competitive', r.competitive ? 'yes' : 'no'],
        ]}
      />
      {choices.length > 0 && (
        <>
          <div className="lbl">CHOICES</div>
          <div className="sub">{choices.join(' · ')}</div>
        </>
      )}
      {parts.length > 0 && (
        <>
          <div className="lbl">PARTS</div>
          <div className="sub">{parts.join(' · ')}</div>
        </>
      )}
      <div className="lbl">CHANGES THESE OWNER SCREENS</div>
      <div className="gov">
        {owner ? (
          <span>
            {owner.groupLabel} · {owner.name}
          </span>
        ) : (
          <NotYet />
        )}
      </div>
      <div className="lbl">HISTORY</div>
      <div className="hist">
        <div>
          {r.governed
            ? `${when(r.governed.when)} · ${r.governed.who} · ${r.governed.change}`
            : `Season start · ${r.citation || 'set'}`}
        </div>
        {r.pending.map((p) => (
          <div key={p.entry}>
            Pending: {p.value} · {pendingWhen(p)}
          </div>
        ))}
      </div>
      <div className="foot">
        <Act
          className="cr-btn pri"
          verb="commissioner.surface"
          variant="surface"
          label={'CHANGE ›'}
          args={{ screen: 'control-room', action: 'check', value: r.field }}
          attributes={{ 'data-edit': String(r.field) }}
        >
          CHANGE ›
        </Act>
        <Act
          className="cr-btn"
          verb="commissioner.surface"
          variant="surface"
          label={'ONE-OFF EXCEPTION ›'}
          args={{ screen: 'control-room', action: 'open', value: 'rule.exception' }}
        >
          ONE-OFF EXCEPTION ›
        </Act>
      </div>
    </>,
  );
}

export function RulesView() {
  const room = useRoom();
  const { rules, controls } = room.state;
  const [chip, setChip] = useState('all');
  const [page, setPage] = useState(0);
  const [sel, setSel] = useState('');
  const { panel, per } = useFit(19);
  const entries = useMemo(() => (controls ? entriesOf(controls) : []), [controls]);
  const lines: Line[] = useMemo(
    () => [
      ...(rules?.sections ?? []).flatMap((s) =>
        s.rows.map((r) => ({ key: r.field, section: s.label, label: r.label, value: r.value, row: r })),
      ),
      ...entries
        .filter((e) => !e.built)
        .map((e) => ({ key: e.id, section: e.groupLabel, label: e.name, value: '—', planned: e })),
    ],
    [rules, entries],
  );
  const need = lines.filter((l) => l.row?.state === 'Needs a ruling');
  const pend = lines.filter((l) => l.row?.pending.length);
  const plan = lines.filter((l) => l.planned);
  const rows =
    chip === 'all'
      ? lines
      : chip === 'need'
        ? need
        : chip === 'pend'
          ? pend
          : chip === 'plan'
            ? plan
            : lines.filter((l) => familyOf(l.section) === chip);
  const pages = Math.max(1, Math.ceil(rows.length / per));
  const at = Math.min(page, pages - 1);
  const vis = rows.slice(at * per, at * per + per);
  useDefaultInspector(
    <>
      <h3>Rules</h3>
      <div className="sub">
        All {lines.length} league rules on one list. The chips filter; they are not menus. Click any rule to
        see what it governs and change it.
      </div>
      <div className="lbl">PRESETS</div>
      <div className="sub">
        A preset is a bundle of setting values. Compare with a preset lists every setting where this league
        differs; applying some or all of them is one ordinary rule change, checked like a change made here.
      </div>
    </>,
    [lines.length],
  );
  const pick = (l: Line) => {
    setSel(l.key);
    inspectRule(
      room,
      l,
      entries.find((e) => e.fields.some((f) => f.field === l.key)),
    );
  };
  useRoomSurface('room-rules', ({ action, value }) => {
    if (action === 'group') {
      setChip(value);
      setPage(0);
      setSel('');
    }
    if (action === 'pick') setPage(Number(value));
    if (action === 'view' && value === 'presets') {
      setSel('');
      room.inspect(<PresetCompare league={room.league} />);
    }
    if (action === 'open') {
      const line = lines.find((l) => l.key === value);
      if (line) pick(line);
    }
  });
  const c = (k: string, label: string, n: number) => (
    <Act
      key={k}
      className="cr-chip"
      active={chip === k}
      verb="commissioner.surface"
      variant="surface"
      label={label}
      args={{ screen: 'room-rules', action: 'group', value: k }}
      attributes={{ 'data-fam': String(k) }}
    >
      {label}
      <span className="c">{n}</span>
    </Act>
  );
  return (
    <div className="cr-col" style={{ flex: 1 }}>
      <div className="cr-chips">
        {c('all', 'ALL', lines.length)}
        {FAMILIES.map(([f]) => c(f, f, lines.filter((l) => familyOf(l.section) === f).length))}
        {c('need', 'NEEDS A RULING', need.length)}
        {c('pend', 'PENDING', pend.length)}
        {c('plan', 'PLANNED (BREADTH)', plan.length)}
        <Act
          className="cr-btn sm"
          verb="commissioner.surface"
          variant="surface"
          label="COMPARE WITH A PRESET ›"
          args={{ screen: 'room-rules', action: 'view', value: 'presets' }}
        >
          COMPARE WITH A PRESET ›
        </Act>
      </div>
      <div className="cr-panel" style={{ flex: 1 }} ref={panel}>
        {!rules && <div className="cr-note">{room.state.error || 'Reading the league rules…'}</div>}
        <table className="cr-t">
          <thead>
            <tr>
              <th style={{ width: '15%' }}>SECTION</th>
              <th style={{ width: '27%' }}>RULE</th>
              <th>VALUE</th>
              <th style={{ width: 150 }}>STATE</th>
              <th style={{ width: 44 }}>⚖</th>
            </tr>
          </thead>
          <tbody>
            {vis.map((l) => (
              <tr key={l.key} className={`rw${sel === l.key ? ' sel' : ''}`} data-rule={l.key}>
                <td style={{ color: 'var(--ink3)' }}>{l.section}</td>
                <td>
                  <Act
                    verb="commissioner.surface"
                    variant="surface"
                    label={l.label}
                    args={{ screen: 'room-rules', action: 'open', value: l.key }}
                  >
                    {l.label}
                  </Act>
                </td>
                <td className="v">{l.value}</td>
                <td>{stateOf(l)}</td>
                <td title="competitive: changes default to next rollover">{l.row?.competitive ? '⚖' : ''}</td>
              </tr>
            ))}
          </tbody>
        </table>
        <div className="cr-pager">
          <span>
            {rows.length} rules · page {at + 1} of {pages}
          </span>
          <span className="sp" />
          <Act
            className="cr-btn sm"
            disabled={at === 0}
            verb="commissioner.surface"
            variant="surface"
            label={'‹ PREV'}
            args={{ screen: 'room-rules', action: 'pick', value: String(at - 1) }}
          >
            ‹ PREV
          </Act>
          <Act
            className="cr-btn sm"
            disabled={at >= pages - 1}
            verb="commissioner.surface"
            variant="surface"
            label={'NEXT ›'}
            args={{ screen: 'room-rules', action: 'pick', value: String(at + 1) }}
          >
            NEXT ›
          </Act>
          <span>⚖ = competitive: changes wait for next rollover unless you choose now</span>
        </div>
      </div>
    </div>
  );
}
