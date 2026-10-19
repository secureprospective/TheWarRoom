import type { ReactNode } from 'react';
import type { Preview } from './preview';

// The shared review order stays the same for every commissioner action. When values share their sources, as
// with starting assets, each pair of sources is named once with how many values it covers, not on every row.
export function PreviewReview({ preview: p, children, confirm }: {
  preview?: Preview; children: ReactNode; confirm: ReactNode;
}) {
  const pairs = new Map<string, { before: string; after: string; count: number }>();
  for (const f of p?.fields ?? []) {
    const key = JSON.stringify([f.beforeSource, f.afterSource]);
    const pair = pairs.get(key) ?? { before: f.beforeSource, after: f.afterSource, count: 0 };
    pairs.set(key, { ...pair, count: pair.count + 1 });
  }
  const shared = !!p && pairs.size < p.fields.length;
  const covers = (count: number) =>
    pairs.size === 1 ? 'Every value' : count === 1 ? '1 value' : `${count} values`;
  return (
    <>
      {p && <>
        <h4>{p.question}</h4>
        <p className="tray-target">{p.target.name}</p>
        <section aria-label="Values and sources">
          <h5>Values and sources</h5>
          {shared ? [...pairs.values()].map((pair, i) => <dl key={i} className="rule-current">
            <dt>{covers(pair.count)} in force now</dt><dd>{pair.before}</dd>
            <dt>{covers(pair.count)} proposed</dt><dd>{pair.after}</dd>
          </dl>) : p.fields.map((f, i) => <dl key={i} className="rule-current">
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
