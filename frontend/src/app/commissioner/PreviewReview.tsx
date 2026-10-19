import type { ReactNode } from 'react';
import type { Preview } from './preview';

// The shared review order stays the same for every commissioner action. When every value comes from the
// same sources, as with starting assets, the sources are named once rather than on every row.
export function PreviewReview({ preview: p, children, confirm }: {
  preview?: Preview; children: ReactNode; confirm: ReactNode;
}) {
  const first = p?.fields[0];
  const shared = !!first && p.fields.length > 1 && p.fields.every(f =>
    f.beforeSource === first.beforeSource && f.afterSource === first.afterSource);
  return (
    <>
      {p && <>
        <h4>{p.question}</h4>
        <p className="tray-target">{p.target.name}</p>
        <section aria-label="Values and sources">
          <h5>Values and sources</h5>
          {shared ? <dl className="rule-current">
            <dt>Every value in force now</dt><dd>{first.beforeSource}</dd>
            <dt>Every value proposed</dt><dd>{first.afterSource}</dd>
          </dl> : p.fields.map((f, i) => <dl key={i} className="rule-current">
            <dt>{f.label} — in force now</dt><dd>{f.beforeSource}</dd>
            <dt>{f.label} — proposed</dt><dd>{f.afterSource}</dd>
          </dl>)}
        </section>
      </>}
      {children}
      {p && <>
        <section aria-label="Before and after">
          <h5>Before → after</h5>
          {p.fields.map((f, i) => <p key={i}>{f.label}: {f.before} → {f.after} ({f.when})</p>)}
        </section>
        <section aria-label="Affects">
          <h5>Affects</h5>
          <p>{p.affects.count === null ? 'Not known yet' : p.affects.words}</p>
          <ul>{p.affects.readings.map((r, i) => <li key={i}>{r}</li>)}</ul>
        </section>
        <section aria-label="Blockers">
          <h5>Before you confirm</h5>
          {p.blockers.length === 0 ? <p>No blockers.</p> :
            <ul role="alert">{p.blockers.map((b, i) => <li key={i}>{b}</li>)}</ul>}
        </section>
      </>}
      {confirm}
    </>
  );
}
