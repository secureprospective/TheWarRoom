import { readFileSync } from 'node:fs';
import { expect, it } from 'vitest';

it('bounds the HQ workspace grid track independently of its content', () => {
  const css = readFileSync(new URL('./hq.css', import.meta.url), 'utf8');
  expect(css).toMatch(/\.hq-v10 \.work\{[^}]*grid-template-columns:\s*minmax\(0,\s*1fr\)/);
});

it('reserves the native inspector only once and protects narrow embedded rows', () => {
  const shell = readFileSync(new URL('./shell.css', import.meta.url), 'utf8');
  expect(shell).toContain('.twr-app[data-native-workspace="true"] > .ws { grid-column: 2 / 4; }');
  expect(shell).toContain('.twr-app > .insp {');
  const hq = readFileSync(new URL('./hq.css', import.meta.url), 'utf8');
  expect(hq).toContain('.hq-v10.embedded .work{grid-template-rows:auto minmax(0,1fr) auto}');
  expect(hq).toContain('.hq-v10.embedded .col:first-child .slot{grid-template-columns:40px 28px minmax(80px,1fr)');
  const war = readFileSync(new URL('./warroom.css', import.meta.url), 'utf8');
  expect(war).toContain('.wr-v6.embedded .work{grid-template-columns:minmax(0,1fr);grid-template-rows:auto minmax(0,1fr) auto auto}');
  expect(war).toContain('.wr-v6.embedded .mk3{grid-template-columns:minmax(0,1fr);overflow:auto}');
});
