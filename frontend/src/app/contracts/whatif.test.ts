import { afterEach, describe, expect, it, vi } from 'vitest';
import { createElement } from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import { readFileSync } from 'node:fs';
import { WhatIfPanel, WhatIfActs } from './WhatIfPanel';
import { parseWhatIf, whatIfProvider } from './whatif';
import { localDate } from '../clock/labels';
import type { Reading } from './contract';
import type { WhatIf } from './whatif';
import { commands } from '../commands/registry';
import * as providers from '../data/provider';
import * as bindings from '../../../wailsjs/go/main/App';

function reading(): Reading<WhatIf> {
  return {
    status: 'ok', absenceReason: null, count: 0, threshold: 0,
    baseline: '2026-10-11T07:27:22.069367355Z',
    cutoff: '2026-10-10T00:00:00Z',
    method: 'Coordinator handlers; integer cents', inputSHA: '', buildStamp: 'dev ring2',
    howCounted: 'Future cells are inferred; 35% assumed.',
    provenance: { source: 'whatif.db; pinned mirror', kind: 'live',
      freshness: { state: 'live', fetchedAt: '2026-10-10T00:00:00Z',
        note: 'Baseline held until Reset' } },
    value: { stale: false, banner: '', empty: 'No moves yet. Nothing sent.', notice: '',
      moves: [], choices: [], figures: [{ label: 'Used', before: '$123.45', after: '$999.99',
        change: '−$30,000' }] },
  };
}
function html(r?: Reading<WhatIf>, error?: string) {
  return renderToStaticMarkup(createElement(WhatIfPanel, { mine: '0001', reading: r, error }));
}
afterEach(() => { vi.restoreAllMocks(); });
describe('What-if', () => {
  it('shows loading, empty, moves, stale and failed states, with Go text untouched', () => {
    expect(html()).toContain('No figures yet');
    const r = reading();
    expect(html(r)).toContain('No moves yet. Nothing sent.');
    expect(html(r)).not.toContain('<table');
    expect(html(r)).not.toContain('Reset');
    expect(html(r)).not.toContain('How this is counted');
    expect(html(r).match(/<p>/g)).toHaveLength(1);
    r.value!.moves = [{ id: '1', player: '0042', tag: 'Cut in what-if',
      status: 'Modelled: cut Example player, $900.01; see What-if at the top',
        label: 'Cut Example player', charge: { cents: 999, text: '$900.01' },
      explanation: 'Final paid year: $0' }];
    expect(html(r)).toContain('Cut Example player');
    expect(html(r)).toContain('$900.01');
    expect(html(r)).toContain('Change');
    expect(html(r)).toContain('−$30,000');
    expect(html(r)).toContain(`Started from MFL rosters as of ${localDate(r.baseline)}`);
    expect(html(r)).not.toContain(r.baseline);
    expect(html(r)).toContain('act-command');
    expect(html(r)).toContain('Undo Cut Example player');
    r.value!.stale = true; r.value!.banner = 'MFL changed since this what-if started';
    expect(html(r)).toContain(r.value!.banner);
    expect(html(r)).toContain('disabled');
    expect(html(r)).toContain('Reset');
    expect(html(r)).not.toContain('Filter or sort contracts');
    r.status = 'failed'; r.absenceReason = 'Ledger unavailable';
    expect(html(r)).toContain('Ledger unavailable');
    expect(html(r)).not.toContain('$999.99');
    expect(html(undefined, 'offline')).toContain('Failed');
  });
  it('Acts call Go and notify a reader that re-queries Go rather than updating numbers locally', async () => {
    const fixture = new providers.FixtureProvider();
    vi.spyOn(providers, 'selectProvider').mockReturnValue({
      ...fixture, kind: 'live', onSeasonChange: () => () => {},
    } as unknown as providers.Provider);
    const binding = vi.spyOn(bindings, 'TargetWhatIf').mockResolvedValue(reading() as never);
    const p = whatIfProvider();
    const off = p.onChange(() => { void p.query('0001'); });
    commands.dispatch('contracts.set', {
      field: 'whatif', value: { mine: '0001', action: 'cut', player: '0042' },
    });
    await new Promise(resolve => setTimeout(resolve, 30));
    expect(binding).toHaveBeenCalledTimes(2);
    expect(binding.mock.calls[0][0]).toEqual({ mine: '0001', action: 'cut', player: '0042', move: '' });
    expect(binding.mock.calls[1][0].action).toBe('read');
    off();
    await p.change({ mine: '0001', action: 'undo', move: '1' });
    await p.change({ mine: '0001', action: 'reset' });
    expect(binding.mock.calls.map(c => c[0].action)).toEqual(['cut', 'read', 'undo', 'reset']);
  });
  it('rejects malformed payloads and reports change failures without inventing state', async () => {
    const r = reading();
    r.value!.stale = 'false' as unknown as boolean;
    expect(() => parseWhatIf(r)).toThrow('expected boolean');
    vi.spyOn(providers, 'selectProvider').mockReturnValue({
      kind: 'live', onSeasonChange: () => () => {},
    } as unknown as providers.Provider);
    vi.spyOn(bindings, 'TargetWhatIf').mockResolvedValue({
      ...reading(), status: 'failed', absenceReason: 'Read failed',
    } as never);
    const p = whatIfProvider();
    const changed = vi.fn(); const off = p.onChange(changed);
    await expect(p.change({ mine: '0001', action: 'buyout', player: '0042' })).rejects.toThrow('Read failed');
    expect(changed).toHaveBeenCalledOnce(); off();
  });
  it('keeps arithmetic and rule sentences in Go, with nonbreaking money and lazy code', () => {
    for (const file of ['WhatIfPanel.tsx', 'whatif.ts']) {
      const source = readFileSync(new URL(file, import.meta.url), 'utf8');
      expect(source).not.toMatch(/\.reduce\(|parseFloat\(|toFixed\(|Intl\.NumberFormat|\.cents\s*[+*/-]/);
    }
    const css = readFileSync(new URL('contracts.css', import.meta.url), 'utf8');
    expect(css).toContain('white-space: nowrap');
    expect(css).toContain('grid-template-columns: 1fr 1fr 1fr');
    const state = readFileSync(new URL('state.ts', import.meta.url), 'utf8');
    expect(state).toContain("import('./whatif')");
  });
});

it('puts both Acts and a disabled reason in the menu, and Go move tags in the row or Inspector', () => {
  const value = reading().value!;
  value.choices = [{ player: '0042', cutLabel: 'What if I cut him',
    buyoutLabel: 'What if I buy him out', buyoutAllowed: false, reason: 'Needs 2–4 paid years' }];
  const view = (menu = true) => renderToStaticMarkup(createElement(WhatIfActs, { mine: '0001',
    player: '0042', value, menu }));
  expect(view()).toContain('<summary>What if…</summary>');
  expect(view()).toContain('disabled');
  expect(view()).toContain('<p>Needs 2–4 paid years</p>');
  expect(view(false)).not.toContain('class="contract-menu"');
  expect(view(false)).toContain('Contract Acts…');
  expect(view(false)).toContain('What if I buy him out');
  expect(view()).not.toContain('Filter or sort contracts');
  for (const tag of ['Cut in what-if', 'Bought out in what-if']) {
    value.moves = [{ id: '1', player: '0042', tag, status: 'Go status', label: 'Go move',
      charge: { cents: 0, text: '$0' }, explanation: '' }];
    expect(view()).toContain(tag);
    expect(view()).not.toContain('<button');
  }
});
