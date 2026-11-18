import { expect, it } from 'vitest';
import { readFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';

it('constrains the embedded trade desk to a 390px phone without horizontal overflow', () => {
  const css = readFileSync(fileURLToPath(new URL('./warroom.css', import.meta.url)), 'utf8');
  const phoneRules = css.slice(css.lastIndexOf('@media (max-width: 699px)'));
  expect(phoneRules).toContain('.wr-v6.embedded { width: 100%; min-width: 0;');
  expect(phoneRules).toContain('.wr-v6.embedded .desk { display: flex; ' +
    'flex-direction: column; min-width: 0;');
  expect(phoneRules).toContain('.wr-v6.embedded .desk .trays { grid-template-columns: minmax(0,1fr); }');
  expect(phoneRules).toContain('min-height: 44px');
  expect(phoneRules).toContain('.wr-v6.embedded .desk .shelf .list { max-height: 240px; ' +
    'overflow-y: auto; }');
});
