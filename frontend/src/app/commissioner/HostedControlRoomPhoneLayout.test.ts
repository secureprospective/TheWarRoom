import { expect, it } from 'vitest';
import { readFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';

it('keeps Control Room rules, season and league usable at 390px without page overflow', () => {
  const path = new URL('../control/room/controlRoom.css', import.meta.url);
  const css = readFileSync(fileURLToPath(path), 'utf8');
  const phoneRules = css.slice(css.lastIndexOf('@media (max-width: 699px)'));
  expect(phoneRules).toContain('.crx.hosted-control-room {');
  expect(phoneRules).toContain('width: 100%; min-width: 0;');
  expect(phoneRules).toContain('.ws-tabs {');
  expect(phoneRules).toContain('overflow-x: auto');
  expect(phoneRules).toContain('min-height: 44px');
  expect(phoneRules).toContain('.hosted-season .cr-panel li');
  expect(phoneRules).toContain('.league-form textarea { width: 100%; min-width: 0; }');
  expect(phoneRules).toContain('.cr-led > div:not(.cr-chips)');
  expect(css).not.toContain('@media (min-width: 700px)');
});
