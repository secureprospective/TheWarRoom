import { useRef } from 'react';
import type { Safeguard } from './rules';

// What a form sends through commissioner.execute. Go validates, authorises and refuses; the form only
// builds this body. `request` is added last so a retry of the same submit repeats the same key.
export type Change = {
  league: string; command: string; args: object; reason: string; expected: number;
  effective: 'now' | 'next_rollover'; visibility: 'league' | 'private';
};
export function newKey(): string {
  const bytes = new Uint8Array(16);
  globalThis.crypto.getRandomValues(bytes);
  return Array.from(bytes, b => b.toString(16).padStart(2, '0')).join('');
}
export const bodyOf = (change: Change, request: string) => JSON.stringify({ ...change, request });

// useRequestKey keeps one key while the change is the same, so retrying a submit cannot record it twice.
// Anything that differs (including the league version, which moves on every receipt) gets a fresh key.
export function useRequestKey(change: Change | null): string {
  const held = useRef({ signature: '', key: '' });
  const signature = change ? JSON.stringify(change) : '';
  if (held.current.signature !== signature) held.current = { signature, key: newKey() };
  return held.current.key;
}

export const safeguardWords: Record<Safeguard, string> = {
  confirm: 'You will be asked to confirm once. A reason is optional.',
  hold: 'Hold the confirm button to record it. A reason is optional.',
  reason: 'Needs a typed reason before it can be recorded.',
};
export const noReasonWords = 'Type a reason to continue.';

export const roleChoices = [
  { value: 'co-commissioner', label: 'Co-commissioner' }, { value: 'DOT', label: 'Trade review seat' },
] as const;

// Go names a person by their sign-in; this desktop's own sign-in reads as "Local operator", as in the ledger.
export const personLabel = (person: string) => (person.startsWith('local.') ? 'Local operator' : person);
