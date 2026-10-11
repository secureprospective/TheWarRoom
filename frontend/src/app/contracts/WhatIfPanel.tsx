import { Act } from '../commands/Act';
import { localDate } from '../clock/labels';
import type { Reading } from './contract';
import type { WhatIf } from './whatif';
import { ContractReading } from './Contracts';

export function WhatIfActs({ mine, player, value, menu = false }: {
  mine: string; player: string; value?: WhatIf; menu?: boolean;
}) {
  const move = value?.moves.find(move => move.player === player);
  if (move) return <small className="contract-whatif-tag">{move.tag}</small>;
  const choice = value?.choices.find(choice => choice.player === player);
  if (!choice || !value) return null;
  const acts = <div className="contract-whatif-acts">
    <Act verb="contracts.set" variant="text"
      args={{ field: 'whatif', value: { mine, action: 'cut', player } }}
      disabled={value.stale}>{choice.cutLabel}</Act>
    <Act verb="contracts.set" variant="text"
      args={{ field: 'whatif', value: { mine, action: 'buyout', player } }}
      disabled={value.stale || !choice.buyoutAllowed}
      label={choice.buyoutAllowed ? choice.buyoutLabel : choice.reason}>{choice.buyoutLabel}</Act>
    <p>{choice.reason}</p>
    {value.stale && <p>{value.banner}; Reset first.</p>}
  </div>;
  return menu ? <details className="contract-menu"><summary>What if…</summary>{acts}</details> : acts;
}

export function WhatIfPanel({ mine, reading, error }: {
  mine: string; reading?: Reading<WhatIf>; error?: string;
}) {
  const value = reading?.value;
  const empty = value?.moves.length === 0;
  return (
    <ContractReading title="What-if" endpoint="C-06.1" reading={reading} error={error}
      metadata={!empty}>
      {value && (empty ? <p>{value.empty}</p> : <div className="whatif-panel">
        <div className="whatif-heading">
          {value.stale && <p role="status">{value.banner}</p>}
          <Act verb="contracts.set" variant="command"
            args={{ field: 'whatif', value: { mine, action: 'reset' } }}>Reset</Act>
        </div>
        {value.notice && <p role="alert">{value.notice}</p>}
        <ol className="whatif-moves">
          {value.moves.map(move => <li key={move.id}>
            <div><b>{move.label}</b><span className="contract-money">{move.charge.text}</span>
              <Act verb="contracts.set" variant="command"
                args={{ field: 'whatif', value: { mine, action: 'undo', move: move.id } }}
                disabled={value.stale} label={`Undo ${move.label}`}>Undo</Act>
            </div>
            <p>{move.explanation}</p>
            <div className="whatif-plan-line">
              {move.plan ? <p>{move.plan}</p> : <Act verb="contracts.set" variant="command"
              label={`Make this a plan: ${move.label}`}
              args={{ field: 'release', value: { mine, action: 'draft', move: move.id } }}>
              Make this a plan</Act>}</div>
          </li>)}
        </ol>
        <table className="whatif-figures" aria-label="Before and after this what-if">
          <thead><tr><th>Result</th><th>Before</th><th>After</th><th>Change</th></tr></thead>
          <tbody>{value.figures.map(figure => <tr key={figure.label}>
            <th scope="row">{figure.label}</th>
            <td className="contract-money">{figure.before}</td>
            <td className="contract-money">{figure.after}</td>
            <td className="contract-money">{figure.change}</td>
          </tr>)}</tbody>
        </table>
        <p className="contract-source">Started from MFL rosters as of{' '}
          {reading?.baseline ? localDate(reading.baseline) : 'Not recorded'}
        </p>
      </div>)}
    </ContractReading>
  );
}
