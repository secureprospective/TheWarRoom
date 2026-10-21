import type { LineupCheck, LineupReading, Receipt } from './contract';
import { parseLineup, parseLineupCheck } from './parseLineup';
import { parseReceipt } from './parseEnvelope';
import { targetHandOff, targetWithdrawRoster } from './api';
import { targetCheckLineup, targetDraftLineup, targetLineup } from '../contracts/api';

export async function checkLineup(franchiseId: string, starters: string[]): Promise<LineupCheck> {
  return parseLineupCheck(await targetCheckLineup(franchiseId, starters), 'lineup.check');
}
export async function draftLineup(franchiseId: string, starters: string[]): Promise<Receipt> {
  return parseReceipt(await targetDraftLineup(franchiseId, starters));
}
export async function handOff(correlationId: string): Promise<Receipt> {
  return parseReceipt(await targetHandOff(correlationId));
}

export async function lineup(franchiseId: string): Promise<LineupReading> {
  return parseLineup(await targetLineup(franchiseId));
}

export async function withdrawRoster(correlationId: string): Promise<Receipt> {
  return parseReceipt(await targetWithdrawRoster(correlationId));
}
