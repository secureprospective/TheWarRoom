import { useEffect } from 'react';
import { Act } from '../commands/Act';
import { commands } from '../commands/registry';
import { WhatIfFigures } from './WhatIfFigures';
import { useAcquisition, openAcquisition, refreshAcquisition } from './acquisition';

export default function Acquisitions({ mine }: { mine: string }) {
  const s = useAcquisition();
  useEffect(() => {
    if (s.request?.mine !== mine) void openAcquisition({ mine });
    return commands.onMovesChange(() => { void refreshAcquisition(mine); });
  }, [mine]);
  const view = s.request?.mine === mine ? s.reading?.value : undefined;
  const request = s.request ?? { mine };
  return <section className="acquisition-panel" aria-label="Acquisitions">
    <h4>What does the add cost me?</h4>
    <p>Choose an incoming contract or a saved bid. This previews the cost; it does not add or cut anyone.</p>
    <label>Incoming player <select disabled={s.busy} value={request.bid || request.player || ''}
      onChange={e => {
        const source = view?.incoming.find(p => (p.bid || p.player) === e.target.value);
        commands.dispatch('acquisition', { mine, player: source?.player, bid: source?.bid });
      }}><option value="">Choose an incoming player</option>
      {view?.incoming.map(p => <option key={p.bid || p.player} value={p.bid || p.player}>{p.name}</option>)}
    </select></label>
    {s.busy && <p role="status">Pricing the add…</p>}
    {s.error && <p role="alert">{s.error}</p>}
    {view?.player && <>
      <h5>{view.player}</h5><p>{view.spot}</p>
      <WhatIfFigures figures={view.seasons} label="Incoming cap hit by season" />
      <label>What if I cut <select disabled={s.busy} value={request.cut ?? ''} onChange={e =>
        commands.dispatch('acquisition', { ...request, mine, cut: e.target.value })}>
        <option value="">Nobody</option>{view.cuts.map(p => <option key={p.token} value={p.token}>
          {p.name}</option>)}
      </select></label>
      {view.cutName && <p>{view.cutName} · cut dead cap this season: {view.deadCap.text}</p>}
      <WhatIfFigures figures={view.team} label="My cap after the add and selected cut" />
      <ul>{view.reasons.map(reason => <li key={reason}>{reason}</li>)}</ul>
    </>}
    <Act variant="button"
      verb="acquisition" args={request} disabled={s.busy} label="Recheck acquisition cost" />
    <details><summary>How this is counted</summary><p>{s.reading?.howCounted}</p></details>
  </section>;
}
