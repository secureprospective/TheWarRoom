import { readFileSync } from 'node:fs';
import { expect, it } from 'vitest';

it('fits hosted Pulse into a 390px phone viewport without changing desktop rules', () => {
  const css = readFileSync(new URL('./pulse.css', import.meta.url), 'utf8');
  const phone = css.slice(css.lastIndexOf('@media (max-width: 699px)'));
  expect(phone).toContain('.lp-v14.embedded {');
  expect(phone).toContain('min-width: 0; min-height: 0; height: auto');
  expect(phone).toContain('.lp-v14.embedded #ws-live,');
  expect(phone).toContain('.lp-v14.embedded #ws-race { grid-template-columns: minmax(0, 1fr); }');
  expect(phone).toContain('min-height: 44px');
  expect(phone).toContain('.lp-v14.embedded .scroll { max-width: 100%; overflow-x: auto; }');
  expect(css).toContain('.lp-v14.embedded{position:relative;inset:auto;z-index:auto;height:100%;'
    + 'min-height:650px;min-width:1040px}');
  expect(css).toContain('.lp-v14.hosted .shell{grid-template-columns:minmax(0,1fr) 260px}');
});
