import type { LineupCheck, LineupReading, Receipt } from './contract';
import { parseLineup, parseLineupCheck } from './parseLineup';
import { parseReceipt } from './parseEnvelope';
import {
  TargetLineup, TargetCheckLineup, TargetDraftLineup, TargetHandOff,
} from '../../../wailsjs/go/main/App';

export async function checkLineup(franchiseId: string, starters: string[]): Promise<LineupCheck> {
  return parseLineupCheck(await TargetCheckLineup(franchiseId, starters), 'lineup.check');
}
export async function draftLineup(franchiseId: string, starters: string[]): Promise<Receipt> {
  return parseReceipt(await TargetDraftLineup(franchiseId, starters));
}
export async function handOff(correlationId: string): Promise<Receipt> {
  return parseReceipt(await TargetHandOff(correlationId));
}

export async function lineup(franchiseId: string): Promise<LineupReading> {
  return parseLineup(await TargetLineup(franchiseId));
}
