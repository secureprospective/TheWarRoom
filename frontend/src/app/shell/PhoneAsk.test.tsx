// @vitest-environment jsdom
import { act } from 'react';
import { createRoot, type Root } from 'react-dom/client';
import { afterEach, expect, it } from 'vitest';
import type { HomeBrief } from './homeReads';
import { answerPhoneQuestion, PhoneAsk } from './PhoneAsk';

const brief = {
  results: { week: 8, games: [{ home: 'mine', away: 'other', homeName: 'My Team', awayName: 'Rivals' }],
    scores: [{ franchise: 'mine', points: 28.4 }, { franchise: 'other', provisional: 24.1 }] },
  at: { results: '2026-11-18T12:00:00Z' }, errors: {},
} as unknown as HomeBrief;
let root: Root | undefined;
let host: HTMLDivElement | undefined;
afterEach(async () => {
  if (root) await act(async () => root!.unmount());
  host?.remove(); root = undefined; host = undefined;
});

it('answers supported score questions from the held Home reading only', () => {
  expect(answerPhoneQuestion('Am I winning?', brief, 'mine')).toEqual({ label: 'FROM HOME READS',
    text: 'You are ahead of Rivals, 28.4–24.1.' });
  expect(answerPhoneQuestion('Scores right now', brief, 'mine')).toEqual({ label: 'FROM HOME READS',
    text: 'Week 8 · My Team 28.4–24.1 Rivals' });
});

it('labels unknown and missing score readings without saving or requesting anything', () => {
  expect(answerPhoneQuestion('Who is injured?', brief, 'mine')).toMatchObject({ label: 'NOT ANSWERED',
    text: expect.stringContaining('Nothing was sent or saved') });
  expect(answerPhoneQuestion('Am I winning?', { at: {}, errors: {} }, 'mine'))
    .toMatchObject({ label: 'READING UNAVAILABLE', text: expect.stringContaining('No score reading') });
});

it('opens a local Ask sheet and shows a score answer', async () => {
  Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true });
  host = document.createElement('div'); document.body.append(host); root = createRoot(host);
  await act(async () => root!.render(<PhoneAsk brief={brief} mine="mine" />));
  const open = host.querySelector('.phone-ask__open') as HTMLButtonElement;
  await act(async () => open.click());
  expect(host.querySelector('[role="dialog"]')).not.toBeNull();
  const sample = [...host.querySelectorAll('.phone-ask__suggestions button')][0] as HTMLButtonElement;
  await act(async () => sample.click());
  expect(host.querySelector('[role="status"]')?.textContent).toContain('You are ahead of Rivals');
});
