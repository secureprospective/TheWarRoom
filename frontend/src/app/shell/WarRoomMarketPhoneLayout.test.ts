import { expect, it } from 'vitest';
import { readFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';

it('keeps the market searchable, touch-sized and vertically scrollable at 390px', () => {
  const css = readFileSync(fileURLToPath(new URL('./warroom.css', import.meta.url)), 'utf8');
  const phoneRules = css.slice(css.indexOf('@media(max-width:699px)'));
  expect(phoneRules).toContain('.wr-v6 .mk3{display:flex;flex-direction:column;min-height:0;' +
    'overflow-y:auto;overflow-x:hidden}');
  expect(phoneRules).toContain('.wr-v6 .mk3 .phone-market-tools{display:grid');
  expect(phoneRules).toContain('.wr-v6 .mk3 .fa-search input{width:100%;min-width:0;' +
    'min-height:44px');
  expect(phoneRules).toContain('.wr-v6 .mk3 .waiver-priority{display:grid');
  expect(phoneRules).toContain('.wr-v6 .mk3 .fa-r{height:auto;min-height:106px;' +
    'grid-template-columns:36px 36px minmax(0,1fr) auto');
  expect(phoneRules).toContain('.wr-v6 .mk3 .fa-r .add-player{display:flex');
  expect(phoneRules).toContain('min-width:88px;min-height:44px');
  expect(phoneRules).toContain('.wr-v6 .mk3 .bidb button,.wr-v6 .mk3 .bidb select,' +
    '.wr-v6 .mk3 .bidb input,');
  expect(phoneRules).toContain('min-width:0;min-height:44px');
  expect(css).toContain('.wr-v6 .mk{grid-template-columns:minmax(0,1fr) 400px');
});
