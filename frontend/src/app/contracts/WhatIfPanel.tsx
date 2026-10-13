import { WhatIfFigures } from './WhatIfFigures';
import { RowMenu } from './RowMenu';
import { useContractMove } from './contractActs';
import { ContractActMenu } from './ContractActs';
import { Act } from '../commands/Act';
import { localDate } from '../clock/labels';
import type { Reading } from './contract';
import type { WhatIf } from './whatif';
import { ContractReading } from './Contracts';

export function WhatIfActs({ mine, player, value, menu = false }: {
  mine: string; player: string; value?: WhatIf; menu?: boolean;
}) {
  const target = useContractMove();
  const move = value?.moves.find(move => move.player === player);
  if (move) return <small className="contract-whatif-tag">{move.tag}</small>;
  if (target.mine === mine && target.player === player) {
    const opened = `Opened: ${target.label || 'contract move'}; see Contract move at the top`;
    return <p className="contract-opened" role="status" aria-label={opened} title={menu ? opened : undefined}>
      {menu ? 'Opened above ↑' : opened}</p>;
  }
  const choice = value?.choices.find(choice => choice.player === player);
  if (!choice || !value) return null;
  const acts = <div className="contract-whatif-acts">
    <Act verb="contracts.set" variant="text"
      args={{ field: 'whatif', value: { mine, action: 'cut', player } }}
      disabled={value.stale} label={choice.cutLabel}>{choice.cutLabel}</Act>
    <Act verb="contracts.set" variant="text"
      args={{ field: 'whatif', value: { mine, action: 'buyout', player } }}
      disabled={value.stale || !choice.buyoutAllowed}
      label={choice.buyoutAllowed ? choice.buyoutLabel : choice.reason}>{choice.buyoutLabel}</Act>
    <p>{choice.reason}</p>
    <ContractActMenu mine={mine} player={player} />
    {value.stale && <p>{value.banner}; Reset first.</p>}
  </div>;
  return menu ? <RowMenu>{acts}</RowMenu> : acts;
}

export function WhatIfPanel({ mine, reading, error }: {
  mine: string; reading?: Reading<WhatIf>; error?: string;
}) {
  const value = reading?.value;
  const empty = value?.moves.length === 0;
  return (
    <ContractReading title="What-if" endpoint="C-06.1" reading={reading} error={error}
      metadata={!empty}>
      {value && <div className="whatif-panel">
        {(!empty || value.stale) && <div className="whatif-heading">
          {value.stale && <p role="status">{value.banner}</p>}
          <Act verb="contracts.set" variant="text" label="Reset"
            args={{ field: 'whatif', value: { mine, action: 'reset' } }}>Reset</Act>
        </div>}
        {empty && <p>{value.empty}</p>}
        {value.notice && <p role="alert">{value.notice}</p>}
        {!empty && <><ol className="whatif-moves">
          {value.moves.map(move => <li key={move.id}>
            <div><b>{move.label}</b><span className="contract-money">{move.charge.text}</span>
              <Act verb="contracts.set" variant="text"
                args={{ field: 'whatif', value: { mine, action: 'undo', move: move.id } }}
                disabled={value.stale} label={`Undo ${move.label}`}>Undo</Act>
            </div>
            <p>{move.explanation}</p>
            <div className="whatif-plan-line">
              {move.plan ? <p>{move.plan}</p> : <Act verb="contracts.set" variant="text"
              label={`Make this a plan: ${move.label}`}
              args={{ field: 'release', value: { mine, action: 'draft', move: move.id } }}>
              Make this a plan</Act>}</div>
          </li>)}
        </ol>
        <WhatIfFigures figures={value.figures} label="Before and after this what-if" />
        <p className="contract-source">Started from MFL rosters as of{' '}
          {reading?.baseline ? localDate(reading.baseline) : 'Not recorded'}
        </p></>}
      </div>}
    </ContractReading>
  );
}
