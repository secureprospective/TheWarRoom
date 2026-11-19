import { readFileSync } from 'node:fs';
import { expect, it } from 'vitest';

it('stacks player card and draft picks for a 390px phone without changing desktop layout', () => {
  const css = readFileSync(new URL('./warroom.css', import.meta.url), 'utf8');
  const phone = css.slice(css.indexOf('/* Player card and picks: stack their existing desktop panels'));

  expect(phone).toContain('.wr-v6.embedded .insp {');
  expect(phone).toContain('grid-template-columns: minmax(0,1fr)');
  expect(phone).toContain('.wr-v6.embedded .dr5 { display: flex; flex-direction: column;');
  expect(phone).toContain('.wr-v6.embedded .native-draft { width: 100%; min-width: 0;');
  expect(phone).toContain('min-height: 44px');
  expect(phone).toContain('.wr-v6.embedded .dr5 .olist { flex: none; min-height: 0;');
  expect(phone).toContain('.wr-v6.embedded .insp textarea { width: 100%;');
  expect(css).toContain('.wr-v6 .dr5{grid-template-columns:340px minmax(0,1fr) 300px;');
  expect(css).toContain('.wr-v6 .insp{background:var(--tile);border-left:3px solid var(--black);');
});
