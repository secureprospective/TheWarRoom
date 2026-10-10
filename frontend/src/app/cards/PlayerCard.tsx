import type { Snapshot } from '../data/contract';
import type { Gravity } from '../look/channels';
import { Glyph } from '../look/Glyph';
import { SignalChip } from '../look/Slots';
import { Card } from './Card';
import { matrixColumns, matrixCells } from './matrix';
import { playerCardModel, type PlayerCardModel } from './playerModel';

export type PlayerCardProps = {
  snapshot: Snapshot;
  franchiseId: string;
  playerId: string;
  asOf: Date;
  provenance?: 'own' | 'inherited';
  gravity?: Gravity;
  model?: PlayerCardModel;
};

export function PlayerCard({
  snapshot,
  franchiseId,
  playerId,
  asOf,
  provenance = 'own',
  gravity = 'G0',
  model: suppliedModel,
}: PlayerCardProps) {
  const model = suppliedModel ?? playerCardModel(snapshot, franchiseId, playerId, asOf);
  return (
    <>
      <div className="player-matrix">
        {matrixCells(model).map((value, index) => (
          <span key={matrixColumns[index].key} title={value}>
            {value}
            {index === 1 && provenance === 'own' && <SignalChip {...model.provenance} />}
          </span>
        ))}
      </div>
      <Card
        gravity={gravity}
        status={model.status}
        provenance={provenance === 'own' ? model.provenance : undefined}
        header={
          <div className="player-identity">
            {model.position && (
              <span className="classification-label">
                <Glyph name={model.position} />
                {model.position}
              </span>
            )}
            <h3 className="card-title">
              {model.name}
            </h3>
            <span className="player-context">
              {[model.team, model.age === undefined ? undefined : `${model.age}y`]
                .filter(Boolean)
                .join(' · ')}
            </span>
          </div>
        }
        primaryZone={
          <div className="two-numbers">
            {model.numbers.map((number) => (
              <div key={number.label}>
                <small>{number.label}</small>
                <b>{number.state}</b>
              </div>
            ))}
          </div>
        }
      >
        <div className="player-contract">
          <span className="player-contract-data">
            {model.salary}
            {model.years && ` · ${model.years}`}
          </span>
          {' · '}
          {model.contractStatus}
        </div>
        <p className="player-unwired-rows not-wired">{model.unwiredRows}</p>
      </Card>
    </>
  );
}
