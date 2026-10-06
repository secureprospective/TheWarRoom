import { describe, expect, it } from 'vitest';
import { POSITIONS, FRESHNESS_STATES } from '../data/contract';
import { GRAVITIES, URGENCIES, SIGNALS, gravityClasses, urgencyClasses, signalClasses } from './channels';
import { glyphPaths, TIERS, POSTURES, PRIVATE_MARKERS } from './Glyph';
import { freshnessSignals } from '../cards/playerModel';
import { readFileSync } from 'node:fs';

const css = readFileSync(new URL('./look.css', import.meta.url), 'utf8');

describe('total channel maps', () => {
  it('has a CSS treatment for every gravity, urgency and signal', () => {
    for (const [values, map] of [[GRAVITIES, gravityClasses], [URGENCIES, urgencyClasses], [SIGNALS, signalClasses]] as const) {
      for (const value of values) {
        const className = (map as Record<string, string>)[value];
        expect(className).toBeTruthy();
        expect(css).toContain(`.${className} {`);
      }
    }
  });
  it('has a distinct path for every contract position and classification', () => {
    const names = [...POSITIONS, ...TIERS, ...POSTURES, ...PRIVATE_MARKERS];
    for (const name of names) expect(glyphPaths[name]).toMatch(/^M/);
    expect(new Set(names.map(name => glyphPaths[name])).size).toBe(names.length);
  });
  it('maps every contract freshness to a locked signal', () => {
    for (const state of FRESHNESS_STATES) expect(SIGNALS).toContain(freshnessSignals[state]);
  });
  it('keeps the board ramp and readable tx3 value scoped', () => {
    expect(css).toContain('.twr-app {');
    expect(css).not.toContain(':root');
    expect(css).toContain('--surface-tx3: hsl(220 8% 57%)');
    for (const signal of SIGNALS) {
      expect(css).toContain(`--signal-${signal}: hsl(`);
      expect(css).toContain(`--signal-${signal}-text: var(--signal-${signal})`);
      expect(css).toContain(`--signal-${signal}-edge: var(--signal-${signal})`);
    }
  });
});
