import type { Snapshot } from '../data/contract';
import type { Gravity } from '../look/channels';
import { Glyph } from '../look/Glyph';
import { Card } from './Card';
import type { Density } from './Card';
import { playerCardModel } from './playerModel';

export type PlayerCardProps = {
  snapshot: Snapshot;
  franchiseId: string;
  playerId: string;
  asOf: Date;
  density: Density;
  gravity?: Gravity;
};

export function PlayerCard({ snapshot, franchiseId, playerId, asOf, density, gravity = 'G0' }: PlayerCardProps) {
  const model = playerCardModel(snapshot, franchiseId, playerId, asOf);
  return <Card gravity={gravity} density={density} status={model.status} provenance={model.provenance}
    header={<div className="player-identity">
      {model.position && <span className="classification-label"><Glyph name={model.position} />{model.position}</span>}
      <h3 className="card-title" title={model.name}>{model.name}</h3>
      <span className="player-context">{[model.team, model.age === undefined ? undefined : `${model.age}y`].filter(Boolean).join(' · ')}</span>
    </div>}
    primaryZone={<div className="two-numbers">
      {model.numbers.map(number => <div key={number.label}><small>{number.label}</small><b>{number.state}</b></div>)}
    </div>}>
    <div className="player-contract">
      <span className="player-contract-data">{model.salary}{model.years && ` · ${model.years}`}</span>
      {' · '}{model.contractStatus}
    </div>
    <p className="player-unwired-rows not-wired">{model.unwiredRows}</p>
  </Card>;
}
