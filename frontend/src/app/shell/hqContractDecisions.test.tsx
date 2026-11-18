import { afterEach, expect, it, vi } from 'vitest';
import {
  all, button, click, control, flush, mount, rerender, type, unmount,
} from '../commissioner/commissionerDom';
import type { HostedPort } from '../commissioner/hostedRoom';
import { nativeEnvelope, type NativeRequest } from '../commands/native';
import { heldSnapshot } from '../data/lineupTestData';
import { HQContracts } from './HQPanels';
import { contractDecisionRequest, parseContractDecisions, type ContractDecisionSubmit,
  type ContractDecisions } from './hqContractDecisions';

const league = 'native.test';
const franchise = 'franchise.01';
const revision = 17;
const deadline = '2026-10-31T04:00:00Z';
const schedule = (season: number, cents: number) => [{ season, cents }];
function quote(player: string, action: string, years: number) {
  return { league, revision, season: 2026, player, name: 'Decision receiver', franchise: 'My team', action,
    actionName: action, allowed: true, priceable: true, reason: '', needs: [], years,
    salaryCents: 1_000_000_000, salaries: schedule(2027, 1_000_000_000), deadMoney: [], reliefCents: 0,
    cap: { beforeCents: 5_000_000_000, afterCents: 5_000_000_000, changeCents: 0,
      roomAfterCents: 5_000_000_000, words: 'Counted cap stays unchanged this season.' },
      basis: [], notes: [],
    source: 'Held league rules' };
}
function payload(decision?: { action: string; years: number; level: number }, past = false) {
  const player = '0101';
  const actions = [
    ['extension', [{ years: 1, level: 0 }, { years: 2, level: 0 }]],
    ['tag', [{ years: 0, level: 0 }]],
    ['rfa_tender', [{ years: 0, level: 1 }, { years: 0, level: 2 }]],
    ['resign', [{ years: 1, level: 0 }, { years: 2, level: 0 }]],
  ] as const;
  return { league, franchise, revision, season: 2026,
    words: 'Only rule-priced owner choices with dated league deadlines.',
    players: [{ player, name: 'Decision receiver', expiration: 2026,
      ...(decision ? { decision: { ...decision, at: '2026-10-12T10:00:00Z' } } : {}),
      options: actions.map(([action, terms]) => ({ action, deadline, open: !past, pastDeadline: past,
        choices: terms.map(({ years, level }) =>
          ({ years, level, quote: quote(player, action, years) })) })) }] };
}
function panel(reading: ContractDecisions, decide: (selection: ContractDecisionSubmit) => void,
  decisionReady = true) {
  const snapshot = heldSnapshot();
  snapshot.league.value.season = 2026;
  return <HQContracts native snapshot={snapshot} mine={franchise} runway={[]} contractDecisions={reading}
    decisionReady={decisionReady} decide={decide} select={() => {}} />;
}
const read = (decision?: { action: string; years: number; level: number }, past = false) =>
  parseContractDecisions(payload(decision, past), league, franchise, revision);
afterEach(async () => { await unmount(); vi.restoreAllMocks(); });

it('reads dated choices, sends the selected terms through the hosted command contract, '
  + 'and allows replacement',
  async () => {
    const port = vi.fn<HostedPort>(async request => {
      nativeEnvelope(request);
      const data = request.command === 'contract.decisions.read' ? payload() : {
        question: 'Record this contract choice?', target: { league, name: 'League', version: revision },
        fields: [], affects: { readings: [], count: 0, words: '' }, blockers: [], safeguard: 'hold' };
      return { label: 'Hosted league', writable: true, football: 'football-pin', data };
    });
    const readRequest: NativeRequest = { command: 'contract.decisions.read', mode: 'read',
      arguments: { league, viewerZone: 'UTC' } };
    const response = await port(readRequest);
    const reading = parseContractDecisions(response.data, league, franchise, revision);
    port.mockClear();
    const submit = (selection: { player: string; action: 'extension' | 'tag' | 'rfa_tender' | 'resign';
      years: number; level: number; reason: string }) => {
      const request = contractDecisionRequest(league, franchise, revision, selection);
      void port(request);
    };
    const host = await mount(panel(reading, submit));
    await flush();
    expect(host.textContent).toContain('DECISIONS DUE');
    expect(host.textContent).toContain('Decision receiver');
    expect(host.textContent).toContain('Deadline');
    expect(host.textContent).toContain('OPEN');
    expect(host.textContent).toContain('2027 $10.00M');
    await type(control(host, 'Reason for this choice'), 'Keep a core receiver under contract');
    const extension = all(host, 'select').find(field =>
      field.getAttribute('aria-label') === 'Decision receiver Extension terms');
    expect(extension).toBeDefined();
    await type(extension!, '2:0');
    await click(button(host, 'REVIEW CHOICE'));
    await flush();
    expect(port).toHaveBeenCalledTimes(1);
    const first = port.mock.calls[0][0];
    expect(first).toMatchObject({ command: 'contract.decision.set', mode: 'preview', arguments: {
      league, command: 'contract.decision.set', expected: revision, visibility: 'private',
      reason: 'Keep a core receiver under contract', args: { contractDecision: { franchise, player: '0101',
        action: 'extension', years: 2, level: 0 } } } });

    await rerender(panel(read({ action: 'extension', years: 2, level: 0 }), submit));
    expect(button(host, 'CURRENT CHOICE').disabled).toBe(true);
    await type(extension!, '1:0');
    await click(button(host, 'CHANGE CHOICE'));
    await flush();
    expect(port).toHaveBeenCalledTimes(2);
    expect(port.mock.calls[1][0].arguments).toMatchObject({ args: { contractDecision: {
      franchise, player: '0101', action: 'extension', years: 1, level: 0 } } });

    await rerender(panel(read({ action: 'extension', years: 1, level: 0 }, true), submit));
    expect(all(host, 'select').every(field => field.disabled)).toBe(true);
    expect(all(host, 'button').filter(field => field.textContent === 'CHANGE CHOICE' ||
      field.textContent === 'CURRENT CHOICE').every(field => field.disabled)).toBe(true);
    expect(host.textContent).toContain('PAST DEADLINE');
    expect(host.querySelector('input[aria-label="Reason for Decision receiver\'s contract choice"]'))
      .toBeNull();
  });

it('rejects a mismatched private reading and an invented owner action', () => {
  expect(() => parseContractDecisions({ ...payload(), franchise: 'franchise.02' }, league, franchise,
    revision)).toThrow('another league, owner or revision');
  expect(() => parseContractDecisions({ ...payload(), players: [{ ...payload().players[0], options: [
    { ...payload().players[0].options[0], action: 'walk' }] }] }, league, franchise, revision)).toThrow();
  const selection = { player: '0101', action: 'extension' as const, years: 2, level: 0, reason: 'Reason' };
  const request: NativeRequest = contractDecisionRequest(league, franchise, revision, selection);
  expect(request.arguments).toMatchObject({ visibility: 'private', reason: 'Reason', expected: revision });
});
