import { releaseAction, onReleaseChange, useReleaseConfirmation, type ReleasePlan } from './releases';
import { useEffect, useState } from 'react';
import { Act } from '../commands/Act';
import { commands } from '../commands/registry';
import { RailStrip } from '../shell/EnvelopeRail';
import { localDate } from '../clock/labels';
import type { Reading } from './contract';
import './contracts.css';
import '../pool/pool.css';

export function ReleaseCard({ plan, mine }: { plan: ReleasePlan; mine: string }) {
  const token = `${plan.id}:${plan.charge}:${plan.receipt.audit.at(-1)?.at}`;
  const confirm = useReleaseConfirmation() === token;
  const finished = ['withdrawn', 'landed', 'failed'].includes(plan.receipt.state);
  const post = <pre className="bid-post" aria-label="Release request text">{plan.post}</pre>;
  return <article className="bid-plan" aria-label={plan.label} data-state={plan.receipt.state}>
    <h5>{plan.label}</h5>
    {!finished && <p>Planned dead cap this season: <span className="contract-money">{plan.charge}</span></p>}
    <RailStrip receipt={plan.receipt} />
    {!finished && <><p>{plan.channel}</p>{post}</>}
    <div className="bid-actions">{plan.canHandOff && (plan.cut && !confirm ?
      <Act verb="contracts.set" variant="button" label={`Open MFL drop page for ${plan.player}`}
        args={{ field: 'release-confirm', value: token }}>Open MFL drop page…</Act> :
      <div className="release-confirm" data-primary="handoff">
        {plan.cut && <p role="alert">{plan.confirmation}</p>}
        <Act verb="contracts.set" variant="button"
          label={plan.cut ? `Confirm release of ${plan.player}` : `Copy buyout request for ${plan.player}`}
          args={{ field: 'release', value: {
            mine, action: 'handoff', id: plan.id, confirmed: plan.cut && confirm,
          } }}>
          {plan.cut ? 'Confirm and open MFL' : 'Copy request and hand off'}
        </Act>
        {plan.cut && <Act verb="contracts.set" variant="button" label="Cancel"
          args={{ field: 'release-confirm', value: '' }}>Cancel</Act>}
      </div>)}
    {plan.canRebase && <Act verb="contracts.set" variant="button" label={plan.rebaseLabel}
      args={{ field: 'release', value: { mine, action: 'rebase', id: plan.id } }}>
      {plan.rebaseLabel}</Act>}
    {plan.withdrawLabel && <Act verb="contracts.set" variant="button" label={plan.withdrawLabel}
      args={{ field: 'release', value: { mine, action: 'withdraw', id: plan.id } }}>
      {plan.withdrawLabel}</Act>}
    </div><details className="bid-details"><summary>History</summary>{finished && post}
      <ol>{plan.receipt.audit.map((entry, i) =>
      <li key={i}><time dateTime={entry.at}>{localDate(entry.at)}</time> {entry.note}</li>)}</ol></details>
  </article>;
}
export default function ReleasePlans({ mine }: { mine: string }) {
  const [result, setResult] = useState<{
    mine: string; reading?: Reading<ReleasePlan[]>; error?: string;
  }>({ mine });
  useEffect(() => {
    let active = true, sequence = 0;
    const load = () => {
      const request = ++sequence;
      void releaseAction({ mine, action: 'read' }).then(reading => {
        if (active && request === sequence) setResult({ mine, reading });
      }, error => {
        if (active && request === sequence) setResult(old => ({ ...old, mine, error: String(error) }));
      });
    };
    if (mine) load();
    const off = onReleaseChange(load), movesOff = commands.onMovesChange(load);
    return () => { active = false; off(); movesOff(); };
  }, [mine]);
  const plans = result.mine === mine ? result.reading?.value : undefined;
  return <section className="release-plans" data-endpoint="C-06.3" aria-label="Cut and buyout plans">
    <h4>Cut and buyout plans</h4>
    {result.error && <p role="alert">{result.error}</p>}
    {!plans && !result.error && <p role="status">Reading release plans…</p>}
    {plans?.length === 0 && <p>Make a plan from a move in the What-if panel.</p>}
    {plans?.map(plan => <ReleaseCard key={plan.id} mine={mine} plan={plan} />)}
    {result.reading && <details><summary>How this is counted</summary>
      <p>{result.reading.howCounted}</p></details>}
  </section>;
}
