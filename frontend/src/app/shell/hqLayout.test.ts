import { readFileSync } from 'node:fs';
import { expect, it } from 'vitest';

it('bounds the HQ workspace grid track independently of its content', () => {
  const css = readFileSync(new URL('./hq.css', import.meta.url), 'utf8');
  expect(css).toMatch(/\.hq-v10 \.work\{[^}]*grid-template-columns:\s*minmax\(0,\s*1fr\)/);
});
