import { AssetCellView } from '../assets/AssetNumber';
import type { TradeAsset } from '../data/contract';
import { useEffect } from 'react';
import { Act } from '../commands/Act';
import { commands } from '../commands/registry';
import { WhatIfFigures } from '../contracts/WhatIfFigures';
import { gravityClasses } from '../look/channels';
import { RailStrip } from './EnvelopeRail';
import { useBuilder, builderAction, type BuilderPlan } from './tradeBuilder';
import '../contracts/contracts.css';
import '../pool/pool.css';

export function BuilderAsset({ asset }: { asset: TradeAsset }) {
  return <span className="builder-asset">
    <b>{asset.name}</b>
    {asset.position && <span>{asset.position} · {asset.team || 'No NFL team'} · {asset.contract}</span>}
    {asset.now && <span>On-field-now: <AssetCellView cell={asset.now} /></span>}
    {asset.dynasty && <span>{asset.position ? 'Dynasty value' : 'Lab pick value'}:
      {' '}<AssetCellView cell={asset.dynasty} /></span>}
  </span>;
}
function Deal({ plan, mine, busy }: { plan: BuilderPlan; mine: string; busy: boolean }) {
  return <article className={`bid-plan ${gravityClasses.G3}`} aria-label="Trade proposal plan">
    <h5>Both rosters after this deal</h5>
    {plan.receipt && <RailStrip receipt={plan.receipt} />}
    {plan.reasons.length > 0 && <ul>{plan.reasons.map((reason, i) => <li key={i}>{reason}</li>)}</ul>}
    {plan.resets.map(r => <Act key={r.franchise} verb="contracts.set" variant="text" label={r.label}
      args={{ field: 'whatif', value: { mine: r.franchise, action: 'reset' } }}
      disabled={busy}>{r.label}</Act>)}
    <div className="trade-teams">{plan.teams.map(team => <section key={team.team}>
      <h6>{team.team}</h6><WhatIfFigures figures={team.figures} label={`${team.team} trade pre-flight`} />
      <details><summary>Roster after the deal · {team.roster.length}</summary>
        <ul>{team.roster.map(p => <li key={p.token}><BuilderAsset asset={p} /></li>)}</ul></details>
    </section>)}</div>
    <p>Agree on ProBoards first. After acceptance there, this deal cannot be withdrawn.</p>
    <pre className="bid-post">{plan.post}</pre>
    {plan.receipt?.state === 'ready' && <Act verb="trade.builder"
      args={{ action: 'handoff', mine, id: plan.id, post: plan.post }} disabled={busy}
      label="Copy the deal and open MFL's trade desk" />}
    {!plan.receipt && <Act verb="trade.builder" args={{ action: 'save', mine }} disabled={busy}
      label="Keep this local trade plan" />}
  </article>;
}
export default function TradeBuilder({ mine }: { mine: string }) {
  const s = useBuilder();
  useEffect(() => {
    void builderAction({ action: 'read', mine });
    return commands.onMovesChange(() => { void builderAction({ action: 'read', mine }); });
  }, [mine]);
  useEffect(() => {
    if (!s.other || !s.give.length || !s.get.length) return;
    const timer = setTimeout(() => { void builderAction({ action: 'preview', mine }); }, 250);
    return () => clearTimeout(timer);
  }, [mine, s.other, s.give, s.get, s.rationale]);
  const view = s.reading?.value;
  return <section className="trade-builder" aria-label="Trade builder">
    <h4>Build a deal</h4>
    <p>Local what-if only. Choose a franchise, then what each side gives.</p>
    <label>Other franchise <select value={s.other} onChange={e => commands.dispatch('trade.builder',
      { action: 'other', value: e.target.value })} disabled={s.busy}>
      <option value="">Choose a franchise</option>
      {view?.teams.filter(t => t.id !== mine).map(t => <option key={t.id} value={t.id}>{t.name}</option>)}
    </select></label>
    {s.other && <div className="trade-assets">{(['give', 'get'] as const).map(side => {
      const team = view?.teams.find(t => t.id === (side === 'give' ? mine : s.other));
      return <fieldset key={side}><legend>{side === 'give' ? 'You give' : 'You get'} · {team?.name}</legend>
        <div className="trade-choices">{team?.assets.map(p => <label key={p.token}>
          <input type="checkbox" checked={s[side].includes(p.token)} disabled={s.busy}
            onChange={() => commands.dispatch('trade.builder', { action: side, value: p.token })} />
          <BuilderAsset asset={p} /></label>)}</div>
      </fieldset>;
    })}</div>}
    <label>Why this deal helps my team <textarea value={s.rationale} disabled={s.busy}
      onChange={e => commands.dispatch('trade.builder',
        { action: 'rationale', value: e.target.value })} /></label>
    <Act verb="trade.builder" args={{ action: 'preview', mine }}
      disabled={s.busy || !s.other || !s.give.length || !s.get.length} label="Check both sides" />
    {s.busy && <p role="status">Checking held rosters and cap…</p>}
    {s.error && <p role="alert">{s.error}</p>}
    {view?.preview?.teams.length ? <Deal plan={view.preview} mine={mine} busy={s.busy} /> : null}
    {s.get.filter(token => !token.startsWith('FP_')).map(player => <Act key={player}
      verb="acquisition" args={{ mine, player }} label={`What does ${view?.teams
        .find(t => t.id === s.other)?.assets.find(p => p.token === player)?.name
        ?? 'this add'} cost me?`} />)}
    {view?.plans.length ? <section><h4>Saved proposals</h4>
      {view.plans.map(plan => <Deal key={plan.id} plan={plan} mine={mine} busy={s.busy} />)}</section> : null}
    <details><summary>How this is counted</summary><p>{s.reading?.howCounted}</p></details>
  </section>;
}
