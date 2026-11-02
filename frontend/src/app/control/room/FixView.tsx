import { fixes } from './guide';
import { openFix } from './openers';
import { useDefaultInspector, useRoom, when } from './room';
import { teamName } from './roomReads';

const iso = (t: unknown) => (typeof t === 'string' ? t : '');

export function FixView() {
  const room = useRoom();
  const { data } = room;
  const ledger = room.state.ledger;
  const items = data.corrections?.items ?? [];
  // Exceptions in force: the corrections that carry an end, and the calendar's window exceptions.
  const inForce = [
    ...items.filter(c => c.until || c.rule).map(c => ({ key: c.entry, what: c.what,
      who: c.franchise ? teamName(data, c.franchise) : c.scope?.franchise ? teamName(data, c.scope.franchise) : '—',
      ends: c.until ? when(c.until) : 'until cleared', entry: c.entry })),
    ...(data.calendar?.clock.exceptions ?? []).map(x => ({ key: `${x.action}:${x.franchise}:${iso(x.start)}`,
      what: `${x.action} outside its window`, who: teamName(data, x.franchise),
      ends: x.permanent ? 'until cleared' : when(iso(x.until)), entry: '' })),
  ];
  const recent = (ledger?.entries ?? []).filter(e => e.undo !== '' && !e.replacedAt).slice(0, 10);
  useDefaultInspector(<><h3>Fix</h3><div className="sub">Something went wrong with one thing. Pick the thing. If you
    don&apos;t know which, try Situations.</div></>);
  return <div className="cr-col" style={{ flex: 1 }}>
    <div className="cr-fix">{fixes.map((f, i) => <button key={f[0]} data-fix={i} onClick={() => openFix(room, i)}>
      <b>{f[0]}</b><span>{f[1]}</span><i>{f[2]}</i></button>)}</div>
    <div className="cr-note">Every fix: preview first (who it touches, does a result flip), a typed reason, and it lands
      on the ledger owners can read. Picks have their own page.</div>
    <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 12, flex: 1, minHeight: 0 }}>
      <div className="cr-panel"><div className="cr-ph">EXCEPTIONS IN FORCE<span className="sp" />
        <span className="ct">they end on their own</span></div><div className="cr-pb">
        <table className="cr-t"><thead><tr><th>WHAT</th><th>WHO</th><th style={{ width: 110 }}>ENDS</th>
          <th style={{ width: 100 }} /></tr></thead><tbody>
          {inForce.map(x => <tr key={x.key}><td>{x.what}</td><td>{x.who}</td><td>{x.ends}</td>
            <td>{x.entry && <button className="cr-btn sm" onClick={() => room.undo(x.entry)}>END NOW</button>}</td></tr>)}
          {!inForce.length && <tr><td colSpan={4} style={{ color: 'var(--ink3)' }}>
            {data.corrections ? 'None in force.' : data.errors.corrections ?? 'Reading the corrections…'}</td></tr>}
        </tbody></table></div></div>
      <div className="cr-panel"><div className="cr-ph">RECENT FIXES<span className="sp" />
        <span className="ct">click to undo</span></div><div className="cr-pb">
        <table className="cr-t"><thead><tr><th style={{ width: 90 }}>WHEN</th><th>FIX</th><th style={{ width: 90 }}>BY</th>
          <th style={{ width: 70 }} /></tr></thead><tbody>
          {recent.map(e => <tr key={e.id}><td>{when(e.when).split(',')[0]}</td><td>{e.what}</td><td>{e.who}</td>
            <td><button className="cr-btn sm" onClick={() => room.undo(e.id)}>UNDO</button></td></tr>)}
          {!recent.length && <tr><td colSpan={4} style={{ color: 'var(--ink3)' }}>
            {ledger ? 'Nothing on the ledger can be undone.' : 'Reading the ledger…'}</td></tr>}
        </tbody></table></div></div>
    </div>
  </div>;
}
