import { createElement } from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import { expect, it } from 'vitest';
import { BidForm } from './BidForm';
import { createPoolState } from './state';
import { parseBoard, parsePool, parsePreview, rawBid, type PoolProvider } from './provider';
import type { BidFormState, Preview } from './contract';
import wire from './wire.testdata.json';

const form: BidFormState = {
  planId: '', playerId: '13593', playerName: 'Smith, John', salary: '0.6', years: 1,
  beat: '', evidence: '', snipe: false, closesOpen: false, closesDate: '', closesTime: '',
};
const preview = parsePreview(wire.preview);
const over: Preview = { ...preview, blocked: true, rules: [{
  rule: 'cap-room', label: 'Cap room', verdict: 'fail', mark: 'Fails', reason: 'Still $450,500 short.',
}], conditionalDrops: [
  { id: '0042', name: 'Rostered, Rick (QB, BUF)', salary: 90000000, deadCap: 30000000,
    freed: 60000000, label: 'Rick · $900,000 salary · 2 years · $300,000 dead cap if dropped',
    covers: true, available: true },
] };
const render = (f: BidFormState, p: Preview) => renderToStaticMarkup(
  createElement(BidForm, { form: f, preview: p, busy: '' }),
);

it('shows a secondary reveal only over room, then one roster select with the priced label', () => {
  expect(render(form, preview)).not.toContain('Name a conditional drop');
  const fittingOpen = render({ ...form, conditionalOpen: true }, preview);
  expect(fittingOpen).not.toContain('<select aria-label="Conditional');
  const closed = render(form, over);
  expect(closed).toContain('Over your cap room? Name a conditional drop');
  expect(closed).not.toContain('<select aria-label="Conditional');
  const opened = render({ ...form, conditionalOpen: true }, over);
  expect(opened).toContain('<select aria-label="Conditional drop if you win"');
  expect(opened).toContain('$900,000 salary · 2 years · $300,000 dead cap if dropped');
  const chosen = render({ ...form, conditionalDrop: '0042' }, preview);
  expect(chosen).toContain('Conditional drop if you win:');
  expect(chosen).toContain('Change');
  expect(chosen).toContain('Clear');
});

it('reveals, selects, sends, revises and clears the condition through the existing form state', async () => {
  const ready = parseBoard(wire.board).value.plans.find(p => p.stateCode === 'ready')!;
  const provider: PoolProvider = {
    refresh: async () => {}, pool: async () => parsePool(wire.pool),
    draft: async () => ready, revise: async () => ready, handOff: async () => ready,
    posted: async () => ready, withdraw: async () => ready,
    preview: async () => over,
    plans: async () => ({ ...parseBoard(wire.board), value: {
      plans: [{ ...ready, conditionalDrop: '0042' }], summary: '' } }),
  };
  const state = createPoolState({ provider, franchise: () => '0001', delay: 0 });
  await state.change({ field: 'pool-plan', value: '13594' });
  await state.change({ field: 'pool-conditional-change', value: '' });
  expect(state.peek().form?.conditionalOpen).toBe(true);
  state.edit('conditionalDrop', '0042');
  expect(state.peek().form?.conditionalOpen).toBe(false);
  expect(rawBid('0001', state.peek().form!).conditionalDrop).toBe('0042');
  await state.loadBoard();
  await state.change({ field: 'pool-revise', value: ready.id });
  expect(state.peek().form?.conditionalDrop).toBe('0042');
  await state.change({ field: 'pool-conditional-clear', value: '' });
  expect(rawBid('0001', state.peek().form!).conditionalDrop).toBe('');
  await state.flush();
});
