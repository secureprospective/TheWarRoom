import { readFileSync } from 'node:fs';
import { expect, it } from 'vitest';
import postcss, { type AtRule } from 'postcss';

it('keeps hosted People, Picks and Fix usable without page overflow at 390px', () => {
  const css = readFileSync(new URL('../control/room/controlRoom.css', import.meta.url), 'utf8');
  const root = postcss.parse(css);
  const phone = root.nodes.find(node => node.type === 'atrule' &&
    node.name === 'media' && node.params === '(max-width:699px)');
  expect(phone, 'hosted Control Room needs a phone-only layout').toBeDefined();
  const rules = (phone as AtRule).nodes?.filter(node => node.type === 'rule')
    .map(node => node.toString()).join('\n') ?? '';
  expect(rules).toContain('.crx.hosted-control-room{width:100%');
  expect(rules).toContain('.hosted-control-content{display:block;width:100%;min-width:0');
  expect(rules).toContain('.hosted-people-table,.crx .hosted-picks .ct-wrap');
  expect(rules).toContain('overflow-x:auto');
  expect(rules).toContain('.hosted-people-table .cr-t{min-width:560px}');
  expect(rules).toContain('.hosted-picks .ct-wrap .cr-t{min-width:680px}');
  expect(rules).toContain('.crx .hosted-fix .cr-fix{grid-template-columns:minmax(0,1fr)');
  expect(rules).toContain('min-height:44px');
  expect(rules).not.toContain('min-width:1040px');
});
