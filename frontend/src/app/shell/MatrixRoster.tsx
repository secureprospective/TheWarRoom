import { Act } from '../commands/Act';
import { matrixColumns, matrixCells } from '../cards/matrix';
import type { PlayerCardModel } from '../cards/playerModel';
import { SignalChip } from '../look/Slots';
import type { PlayerSubject } from './state';

type Row = { model: PlayerCardModel; provenance: 'own' | 'inherited' };

export function MatrixRoster({ cards, franchiseId, selected }: {
  cards: readonly Row[];
  franchiseId: string;
  selected: PlayerSubject | null;
}) {
  return (
    <table className="matrix-roster" aria-label="Roster">
      <colgroup>
        {matrixColumns.map((column) => <col key={column.key} style={{ width: column.width }} />)}
      </colgroup>
      <thead>
        <tr>
          {matrixColumns.map((column) => (
            <th key={column.key} scope="col">
              {column.label}{'note' in column && <small>{column.note}</small>}
            </th>
          ))}
        </tr>
      </thead>
      <tbody>
        {cards.map(({ model, provenance }) => (
          <tr key={model.id} data-selected={
            selected?.id === model.id && selected.franchiseId === franchiseId
          }>
            {matrixCells(model).map((value, index) => (
              <td key={matrixColumns[index].key} title={value}>
                {index === 1 ? (
                  <Act verb="inspector.open" args={{
                    subject: { kind: 'player', id: model.id, franchiseId },
                  }} label={`Inspect ${model.name}`}>
                    {value}
                  </Act>
                ) : value}
                {index === 1 && provenance === 'own' && <SignalChip {...model.provenance} />}
              </td>
            ))}
          </tr>
        ))}
      </tbody>
    </table>
  );
}
