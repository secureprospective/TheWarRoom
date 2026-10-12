import { array, id, object, text } from '../data/parseValues';

// C-01, the asset board: Go words every cell; Held is false when the text says why there is no value.
export type AssetCell = { text: string; sub: string; detail: string; held: boolean };
export type AssetValue = { id: string; now: AssetCell; dynasty: AssetCell; market: AssetCell;
  nowValue: number; dynastyValue: number };
export type AssetBoard = { values: AssetValue[]; nowSource: string; dynastySource: string;
  how: string };

// Points per game can be negative (a kicker's or defender's bad weeks), so any finite number reads.
function finite(value: unknown, path: string): number {
  if (typeof value !== 'number' || !Number.isFinite(value))
    throw new Error(`${path}: expected number`);
  return value;
}
function cell(value: unknown, path: string): AssetCell {
  const r = object(value, path, ['text', 'sub', 'detail', 'held']);
  if (typeof r.held !== 'boolean') throw new Error(`${path}.held: expected boolean`);
  return { text: text(r.text, `${path}.text`), sub: text(r.sub, `${path}.sub`),
    detail: text(r.detail, `${path}.detail`), held: r.held };
}
function assetValue(value: unknown, path: string): AssetValue {
  const r = object(value, path, ['id', 'now', 'dynasty', 'market', 'nowValue', 'dynastyValue']);
  return { id: id(r.id, `${path}.id`, true), now: cell(r.now, `${path}.now`),
    dynasty: cell(r.dynasty, `${path}.dynasty`), market: cell(r.market, `${path}.market`),
    nowValue: finite(r.nowValue, `${path}.nowValue`),
    dynastyValue: finite(r.dynastyValue, `${path}.dynastyValue`) };
}
export function parseAssetBoard(value: unknown): AssetBoard {
  const p = 'Asset board';
  const r = object(value, p, ['values', 'nowSource', 'dynastySource', 'how']);
  return { values: array(r.values, `${p}.values`, assetValue), nowSource: text(r.nowSource, `${p}.nowSource`),
    dynastySource: text(r.dynastySource, `${p}.dynastySource`), how: text(r.how, `${p}.how`) };
}
