import { createElement } from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import { expect, it, vi } from 'vitest';
import { commands } from '../commands/registry';
import { renderState } from '../shell/state';
import { revealEndpoint } from '../registry/EndpointIndex';
import { PoolScreen } from './Pool';
import { poolState } from './state';

it('opening M-039 reveals the rendered pool marker and opens enclosing disclosures', () => {
  commands.dispatch('endpoint.open', { id: 'M-039' });
  expect(commands.read().node).toBe('war');
  expect(commands.read().workspace.war).toBe('valuations-and-pool');
  const spy = vi.spyOn(commands, 'use').mockReturnValue(renderState(commands.read()));
  try {
    const markup = renderToStaticMarkup(createElement(PoolScreen, {
      state: poolState.peek(), franchiseId: '',
    }));
    expect(markup).toMatch(/class="pool-controls"[^>]*data-endpoint="M-039"/);
    const ids = [...markup.matchAll(/data-endpoint="([^"]+)"/g)].map(match => match[1]);
    const work = { scrollTop: 25, clientTop: 0, clientHeight: 600,
      classList: { contains: (name: string) => name === 'wbody' },
      getBoundingClientRect: () => ({ top: 50 }), querySelectorAll: () => [board, ...surfaces] };
    const disclosure = { tagName: 'DETAILS', open: false, parentElement: work };
    const board = { dataset: { endpoint: 'C-01' }, focus: vi.fn() };
    const surfaces = ids.map(id => ({ dataset: { endpoint: id }, parentElement: disclosure,
      tabIndex: 0, focus: vi.fn(),
      closest: (selector: string) => selector === '.wbody' ? work : null,
      getBoundingClientRect: () => ({ top: 850, bottom: 950 }) }));
    const fallback = { focus: vi.fn() };
    const index = { closest: () => work, querySelector: () => fallback };
    expect(revealEndpoint(index as unknown as HTMLElement, commands.read().endpointIds)).toBe(true);
    expect(surfaces.find(surface => surface.dataset.endpoint === 'M-039')!.focus)
      .toHaveBeenCalledWith({ preventScroll: true });
    expect(disclosure.open).toBe(true);
    expect(work.scrollTop).toBe(825);
    expect(board.focus).not.toHaveBeenCalled();
    expect(fallback.focus).not.toHaveBeenCalled();
  } finally { spy.mockRestore(); }
});
