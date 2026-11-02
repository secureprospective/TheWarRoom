import { useState } from 'react';
import { situations } from './guide';
import { openSituation } from './openers';
import { useDefaultInspector, useRoom } from './room';

export function SituationsView() {
  const room = useRoom();
  const [cat, setCat] = useState('ALL');
  const cats = ['ALL', ...Array.from(new Set(situations.map(s => s[0])))];
  useDefaultInspector(<><h3>Situations</h3><div className="sub">Real league trouble arrives as a story. Pick the
    story; you get the tools in order, what is still up to you, and what to check afterwards.</div>
    <div className="lbl">MISSING ONE?</div><div className="sub">Every tool is also reachable from Fix, Rules and the
      command line. Situations are the short way in.</div></>);
  return <div className="cr-col" style={{ flex: 1 }}>
    <div className="cr-chips">{cats.map(c => <button key={c} className="cr-chip" aria-pressed={c === cat}
      onClick={() => setCat(c)}>{c}<span className="c">{c === 'ALL' ? situations.length
        : situations.filter(s => s[0] === c).length}</span></button>)}</div>
    <div className="cr-sit">{situations.map((s, i) => cat === 'ALL' || s[0] === cat ? <button key={s[1]}
      data-sit={i} onClick={() => openSituation(room, i)}><span className="cat">{s[0]}</span><b>{s[1]}</b>
      <span className="fs">{s[2].length} step{s[2].length > 1 ? 's' : ''} · first: {s[2][0]![0]}</span></button>
      : null)}</div>
  </div>;
}
