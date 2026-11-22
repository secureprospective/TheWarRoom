import { createElement } from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import { afterEach, expect, it, vi } from 'vitest';
import { commands } from '../commands/registry';
import { TargetApp } from './TargetApp';

afterEach(() => { commands.dispatch('surface.close', {}); vi.restoreAllMocks(); });
it('opened Reports and Status overlays carry the CSS visibility contract', () => {
  vi.spyOn(commands, 'use').mockImplementation(() => commands.read());
  for (const place of ['Report drawer', 'Status strip'] as const) {
    commands.dispatch('surface.open', { place });
    const html = renderToStaticMarkup(createElement(TargetApp));
    expect(html).toContain(`class="shell-overlay registry-panel" data-open="true" aria-label="${place}"`);
  }
  commands.dispatch('surface.close', {});
  expect(renderToStaticMarkup(createElement(TargetApp))).not.toContain('shell-overlay registry-panel');
});
