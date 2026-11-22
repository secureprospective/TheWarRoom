import { describe, expect, it } from 'vitest';
import { FixtureProvider } from '../data/provider';
import { parseAlerts } from '../data/parseAlerts';
import { heldLineup } from '../data/lineupTestData';

function alertReading() {
  return { alerts: [{ kind: 'lineup', urgency: 'U2', title: 'Check your lineup', detail: 'Go',
    node: 'hq', workspace: 'lineup-and-roster', at: '2026-10-08T23:15:00Z',
    provenance: heldLineup().provenance }], unavailable: [] };
}

describe('Home and League Pulse boundary', () => {
  it('parses the Go readings and rejects malformed fields at their paths', () => {
    expect(parseAlerts(JSON.parse(JSON.stringify(alertReading())))).toEqual(alertReading());
    const badAlert = JSON.parse(JSON.stringify(alertReading()));
    badAlert.alerts[0].urgency = 'U9';
    expect(() => parseAlerts(badAlert)).toThrow('alerts[0].urgency');
  });
  it('says what needs the desktop app in the fixture build', async () => {
    const fixture = new FixtureProvider();
    await expect(fixture.alerts('0025')).rejects.toThrow('Alerts need the desktop app');
  });
});
