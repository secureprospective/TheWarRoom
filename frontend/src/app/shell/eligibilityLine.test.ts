import { describe, expect, it } from 'vitest';
import type { EligibilityDecision, Provenance } from '../data/contract';
import { eligibilityLine, ineligibleLabel } from './eligibilityLine';

const feeds: Record<'injuries' | 'players' | 'rosters', Provenance> = {
  injuries: {
    source: 'injuries', kind: 'live',
    freshness: { state: 'live', fetchedAt: '2026-10-10T16:02:00Z', note: '' },
  },
  players: {
    source: 'mfl-players', kind: 'live',
    freshness: { state: 'live', fetchedAt: '2026-10-05T19:59:41Z', note: '' },
  },
  rosters: {
    source: 'mfl-mirror', kind: 'live',
    freshness: { state: 'live', fetchedAt: '2026-10-10T16:02:00Z', note: '' },
  },
};

function decision(overrides: Partial<EligibilityDecision>): EligibilityDecision {
  return { direction: 'to_ir', verdict: 'eligible', source: 'injuries', ...overrides };
}

// The panel shows the feed's local clock time; the test computes it the way the app does so
// it holds in any locale, like the rail's own time assertions.
const clock = (at: string) =>
  new Intl.DateTimeFormat(undefined, { hour: '2-digit', minute: '2-digit' }).format(new Date(at));

describe('eligibility lines', () => {
  it('names the reason, feed and as-of time on an ineligible line', () => {
    const line = eligibilityLine('IR', decision({
      verdict: 'ineligible', reason: 'not on NFL injured reserve',
    }), feeds);
    expect(line).toBe(`IR: not eligible, not on NFL injured reserve ` +
      `(MFL injuries, ${clock('2026-10-10T16:02:00Z')})`);
  });
  it('names the direction on eligible activations and promotions', () => {
    expect(eligibilityLine('IR', decision({
      direction: 'off_ir', source: 'rosters',
    }), feeds)).toBe(`IR: eligible to activate (MFL rosters, ${clock('2026-10-10T16:02:00Z')})`);
    expect(eligibilityLine('Taxi', decision({
      direction: 'off_taxi', source: 'rosters',
    }), feeds)).toBe(`Taxi: eligible to promote (MFL rosters, ${clock('2026-10-10T16:02:00Z')})`);
  });
  it('keeps a plain eligible line for deactivation and demotion', () => {
    expect(eligibilityLine('Taxi', decision({
      direction: 'to_taxi', source: 'players',
    }), feeds)).toBe(`Taxi: eligible (MFL players, ${clock('2026-10-05T19:59:41Z')})`);
  });
  it('says MFL will decide on an unknown line', () => {
    const unknown = eligibilityLine('Taxi', decision({
      direction: 'to_taxi', verdict: 'unknown', reason: 'stale', source: 'players',
    }), feeds);
    expect(unknown).toBe(`Taxi: not checked, stale; MFL will decide ` +
      `(MFL players, ${clock('2026-10-05T19:59:41Z')})`);
  });
  it('carries the stale copy\'s own time on a not-checked line, like every line', () => {
    const stale = structuredClone(feeds);
    stale.injuries.kind = 'live';
    stale.injuries.freshness = {
      state: 'stale', fetchedAt: '2026-10-05T19:59:41Z',
      note: 'live fetch failed, showing last known data: mfl unreachable',
    };
    expect(eligibilityLine('IR', decision({
      verdict: 'unknown', reason: 'stale',
    }), stale)).toBe(`IR: not checked, stale; MFL will decide ` +
      `(MFL injuries, ${clock('2026-10-05T19:59:41Z')})`);
  });
  it('drops the time when the feed holds none', () => {
    const failed = structuredClone(feeds);
    failed.injuries.freshness.fetchedAt = '';
    expect(eligibilityLine('IR', decision({
      verdict: 'ineligible', reason: 'not on NFL injured reserve',
    }), failed)).toBe('IR: not eligible, not on NFL injured reserve (MFL injuries)');
  });
  it('labels disabled Acts with the reason and leaves other labels alone', () => {
    expect(ineligibleLabel('Draft IR placement', decision({
      verdict: 'ineligible', reason: 'IR full (12/12)',
    }))).toBe('Draft IR placement: IR full (12/12)');
    expect(ineligibleLabel('Draft IR placement', decision({}))).toBeUndefined();
    expect(ineligibleLabel('Draft IR placement', undefined)).toBeUndefined();
  });
});
