import { afterEach, describe, expect, it, vi } from 'vitest';
import { readFileSync } from 'node:fs';
import { act, createElement } from 'react';
import { createRoot } from 'react-dom/client';
import { renderToStaticMarkup } from 'react-dom/server';
import fixture from '../data/fixtures/snapshot.json';
import { parseEnvelopeDemo, parseSnapshot } from '../data/parse';
import type { AuditEntry, EligibilityDecision, PlayerEligibility } from '../data/contract';
import { lineupReceipt } from '../data/lineupReceiptTestData';
import { FixtureProvider } from '../data/provider';
import { createCommands } from '../commands/registry';
import { EnvelopeRail, RailStrip, railStages } from './EnvelopeRail';
import ReceiptStatus from './ReceiptStatus';
import { stateLabel } from './moveLabels';
import PlayerAct from './PlayerAct';
import MyMoves from './MyMoves';
import { renderState } from './state';
import { clockDom } from '../clock/testDom';

const demoImports = vi.fn();
vi.mock('../data/fixtures/envelope-demo.json', () => {
  demoImports();
  return { default: JSON.parse(readFileSync('src/app/data/fixtures/envelope-demo.json', 'utf8')) };
});
const snapshot = parseSnapshot(fixture);
const receipt = parseEnvelopeDemo(JSON.parse(readFileSync(
  'src/app/data/fixtures/envelope-demo.json', 'utf8',
))).receipt;
const subject = { kind: 'player' as const, id: '11675', franchiseId: '0001' };
// The demo receipt describes its own player — tools/fixtures picks the first one the held rules
// leave unchecked, 11675 in franchise 0001 — so the inspector
// tests need a receipt for the player under inspection.
function forSubject(r: typeof receipt, spec: Partial<typeof receipt.spec> = {}): typeof receipt {
  return {
    ...r,
    spec: {
      ...r.spec, ...spec,
      franchiseId: subject.franchiseId,
      subject: { players: [subject.id], picks: [] },
    },
  };
}
const serverExecutor = (executor: ReturnType<typeof createCommands>) => ({
  ...executor,
  use: () => renderState(executor.read()),
  useMoves: <T,>(selector: (state: ReturnType<typeof executor.readMoves>) => T) =>
    selector(executor.readMoves()),
});
const render = (executor: ReturnType<typeof createCommands>) =>
  renderToStaticMarkup(createElement(MyMoves, { snapshot, executor: serverExecutor(executor) }));

const globals = ['window', 'document', 'IS_REACT_ACT_ENVIRONMENT'].map((key) => ({
  key, descriptor: Object.getOwnPropertyDescriptor(globalThis, key),
}));
afterEach(() => {
  for (const { key, descriptor } of globals) {
    if (descriptor) Object.defineProperty(globalThis, key, descriptor);
    else Reflect.deleteProperty(globalThis, key);
  }
});

describe('move rail', () => {
  it('renders the real demo in order with local times, notes and landed current', () => {
    expect(railStages(receipt)).toEqual(['draft', 'ready', 'handed_off', 'not_yet_done', 'landed']);
    const html = renderToStaticMarkup(createElement(EnvelopeRail, { snapshot, receipt }));
    const demoPlayer = snapshot.players.value.find(
      (p) => p.id === receipt.spec.subject.players[0])!;
    expect(html).toContain(demoPlayer.name);
    expect(html).toContain('IR placement');
    expect(html).toContain('G2 · reversible');
    expect(html).toMatch(/aria-current="step"[^>]*><span[^>]*>Landed/);
    let index = -1;
    for (const label of ['Draft', 'Ready', 'Handed off', 'Not yet done', 'Landed']) {
      const next = html.indexOf(`>${label}<`);
      expect(next).toBeGreaterThan(index);
      index = next;
    }
    for (const entry of receipt.audit) {
      expect(html).toContain(entry.note);
      expect(html).toContain(`dateTime="${entry.at}"`);
      expect(html).toContain(new Intl.DateTimeFormat(undefined, { timeStyle: 'medium' })
        .format(new Date(entry.at)));
    }
  });
  it('replaces supersession IDs with owner-facing text', () => {
    const ready = lineupReceipt();
    const correlationId = 'lineup-0123456789abcdef0123456789abcdef';
    const stale = {
      ...ready, state: 'stale' as const, audit: [...ready.audit, {
        at: '2026-10-07T14:01:00Z', from: 'ready' as const, event: 'invalidate' as const,
        to: 'stale' as const, note: `superseded by ${correlationId}`,
      }],
    };
    const html = renderToStaticMarkup(createElement(EnvelopeRail, { snapshot, receipt: stale }));
    expect(html).toContain('<p>Replaced by a newer plan</p>');
    expect(html).not.toContain(correlationId);
    expect(html).not.toContain('superseded by');
  });
  it.each(['not_verified', 'landed'] as const)(
    'keeps Not verified in audit order with its time and note when currently %s', (state) => {
      const ready = lineupReceipt();
      const audit: AuditEntry[] = [ready.audit[0], {
        at: '2026-10-07T14:01:00Z', from: 'ready' as const, event: 'hand_off' as const,
        to: 'handed_off' as const, note: 'Opened MFL',
      }, {
        at: '2026-10-07T14:02:00Z', from: 'handed_off' as const, event: 'partial' as const,
        to: 'not_verified' as const, note: 'Saved lineup not verified',
      }];
      if (state === 'landed') audit.push({
        at: '2026-10-07T14:03:00Z', from: 'not_verified', event: 'match',
        to: 'landed', note: 'MFL saved the drafted starters',
      });
      const plan = { ...ready, state, audit };
      const stages = ['draft', 'ready', 'handed_off', 'not_verified'];
      if (state === 'landed') stages.push('not_yet_done', 'landed');
      expect(railStages(plan)).toEqual(stages);
      const html = renderToStaticMarkup(createElement(EnvelopeRail, { snapshot, receipt: plan }));
      expect(html.match(/>Not verified</g)).toHaveLength(1);
      expect(html.indexOf('>Not verified<')).toBeGreaterThan(html.indexOf('>Handed off<'));
      expect(html).toContain('<p>Saved lineup not verified</p>');
      expect(html).toContain('dateTime="2026-10-07T14:02:00Z"');
      expect(html).toContain(new Intl.DateTimeFormat(undefined, { timeStyle: 'medium' })
        .format(new Date('2026-10-07T14:02:00Z')));
    },
  );
  it.each(['roster.ir', 'roster.taxi'])(
    'omits the %s summary when the player is unavailable', (intent) => {
      const unknown = { ...receipt, spec: {
        ...receipt.spec, intent, expected: { player: 'missing', rosterStatus: 'TAXI_SQUAD' as const },
      } };
      const html = renderToStaticMarkup(createElement(EnvelopeRail, { snapshot, receipt: unknown }));
      expect(html).toContain('<h4>Player unavailable</h4>');
      expect(html).not.toContain('<p>IR Player unavailable</p>');
      expect(html).not.toContain('<p>+Taxi Player unavailable</p>');
    },
  );
  it('ends at blocked with the exact Go note and no later stages', () => {
    const blocked = {
      ...receipt,
      state: 'blocked' as const,
      spec: { ...receipt.spec, target: { kind: 'unmapped' as const } },
      audit: [{
        at: receipt.audit[0].at, from: 'draft' as const, event: 'checks_block' as const,
        to: 'blocked' as const, note: 'MFL target not verified',
      }],
    };
    expect(railStages(blocked)).toEqual(['draft', 'blocked']);
    const html = renderToStaticMarkup(createElement(EnvelopeRail, { snapshot, receipt: blocked }));
    expect(html).toContain('MFL target not verified');
    expect(html).toMatch(/aria-current="step"[^>]*><span[^>]*signal-red[^>]*>Blocked/);
    expect(html).not.toContain('>Ready<');
    expect(html).not.toContain('>Landed<');
  });
  it('inserts pending stages only when the trail visits them and mutes future stages', () => {
    const pending = {
      ...receipt, state: 'dot_review' as const,
      audit: [receipt.audit[0], {
        at: receipt.audit[1].at, from: 'ready' as const, event: 'await_dot' as const,
        to: 'dot_review' as const, note: 'Awaiting DOT',
      }],
    };
    expect(railStages(pending)).toEqual([
      'draft', 'ready', 'dot_review', 'handed_off', 'not_yet_done', 'landed',
    ]);
    const html = renderToStaticMarkup(createElement(EnvelopeRail, { snapshot, receipt: pending }));
    expect(html).toContain('Awaiting DOT');
    expect(html.match(/data-reached="false"/g)).toHaveLength(3);
    expect(html).not.toContain('Bid pending');
    expect(html).not.toContain('Waiver pending');
  });
});

describe('My moves and inspector', () => {






  it('labels the fixture landing as simulated and never offers an enabled draft', async () => {
    const executor = createCommands(() => undefined, new FixtureProvider());
    executor.loadSnapshot(snapshot);
    await executor.loadMoves('0001');
    const html = render(executor);
    expect(demoImports).toHaveBeenCalledTimes(1);
    expect(html).toContain(
      'Fixture · demo envelope · landing simulated from a real roster snapshot',
    );
    expect(html).toContain('>Landed<');
    const inspector = renderToStaticMarkup(createElement(PlayerAct, {
      subject, executor: serverExecutor(executor),
    }));
    expect(inspector).toContain('disabled=""');
    expect(inspector).toContain('Drafting needs the desktop app');
    expect(inspector).toContain('My moves');
  });


});
