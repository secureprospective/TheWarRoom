import type { PlayerCardModel } from './playerModel';

export const matrixColumns = [
  { key: 'position', label: 'Pos', width: '5%' },
  { key: 'name', label: 'Player', width: '25%' },
  { key: 'team', label: 'Team', width: '6%' },
  { key: 'age', label: 'Age', width: '5%' },
  { key: 'now', label: 'On-field-now', width: '14%' },
  { key: 'dynasty', label: 'Dynasty value', width: '14%' },
  { key: 'salary', label: 'Salary', width: '11%' },
  { key: 'years', label: 'Years', width: '6%' },
  { key: 'status', label: 'Status', width: '14%' },
] as const;

// assetColumn is the C-01 cell a matrix column shows, drawn by AssetNumber rather than as text.
export function assetColumn(index: number): 'now' | 'dynasty' | undefined {
  const key = matrixColumns[index]?.key;
  return key === 'now' || key === 'dynasty' ? key : undefined;
}

export function matrixCells(model: PlayerCardModel): string[] {
  return [
    model.position ?? '—', model.name, model.team ?? '—', String(model.age ?? '—'),
    '—', '—', model.salary, model.years?.split(' ')[0] ?? '—',
    [model.contractStatus, model.rosterNote].filter(Boolean).join(' · '),
  ];
}
