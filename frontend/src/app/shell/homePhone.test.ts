import { readFileSync } from 'node:fs';
import { expect, it } from 'vitest';

const css = readFileSync(new URL('./homePhone.css', import.meta.url), 'utf8');
const phone = css.slice(css.indexOf('@media (max-width: 699px)'));

it('stacks Home boards and keeps phone controls reachable without touching desktop rules', () => {
  expect(phone).toContain('.hb .grid { display: flex; flex-direction: column;');
  expect(phone).toContain('.hb .t .ed .move { display: inline-flex; }');
  expect(phone).toContain('min-height: 44px');
  expect(phone).toContain('overflow-x: hidden');
  expect(phone).not.toContain('@media (min-width');
});
