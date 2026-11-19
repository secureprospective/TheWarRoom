// @vitest-environment jsdom
import { act } from 'react';
import { createRoot, type Root } from 'react-dom/client';
import { afterEach, beforeEach, expect, it, vi } from 'vitest';
import HostedLeague from './HostedLeague';
import type { HostedPort } from './hostedRoom';
import type { NativeRequest } from '../commands/native';
import { hostedReads, observeView } from '../data/hostedReads';

const league = 'native.test';
const ruleReading = {
  league, name: 'Test League', revision: 7, rollover: 'No rollover date is set', sections: [{
    label: 'Lineup', rows: Array.from({ length: 32 }, (_, i) => ({
      field: `lineup.test_${i + 1}`, label: `Setting ${i + 1}`, value: `Value ${i + 1}`,
      state: 'Set', citation: '', changedAt: '', current: `Value ${i + 1}`, unit: '',
      competitive: false, nowOnly: false, pending: [],
      schema: { kind: 'string', choices: [], fields: [] },
      safeguards: { nextRollover: 'confirm', now: 'hold' },
      governed: i === 31 ? { entry: 'ledger.32', who: 'commissioner',
        when: '2026-11-17T12:00:00Z', change: 'Rule set' } : null,
    })),
  }],
};
function jobs(count: number) {
  return { league, name: 'Test League', revision: 7, jobs: Array.from({ length: count }, (_, i) => ({
    id: i === 0 ? 'rollover' : `job-${i + 1}`, name: i === 0 ? 'Rollover' : `Job ${i + 1}`,
    state: 'upcoming', words: 'Open the commissioner tool.', opens: '', closes: '',
    controls: i === 0 ? ['CM-027'] : [],
  })) };
}
const backupPreview = { question: 'Save and verify a league backup now?',
  target: { league, name: 'Test League', version: 7 }, fields: [{ label: 'Off-site league backup',
    before: 'No manual backup started', beforeSource: 'Hosted server',
    after: 'Save and check the league backup in Cloudflare R2',
    afterSource: 'Hosted server backup service', when: 'now' }],
  affects: { readings: [], count: null, words: 'No league settings or records change' },
  blockers: [], safeguard: 'confirm' };
function ledger() {
  return { league, name: 'Test League', revision: 7,
    entries: Array.from({ length: 32 }, (_, i) => ({
      id: `entry-${i}`, who: 'Commissioner', what: `Change ${i + 1}`, why: 'Test reason',
      when: '2026-11-17T12:00:00Z', gravity: 'G1', undo: '',
    })) };
}
function port(count: number): HostedPort & ReturnType<typeof vi.fn> {
  return vi.fn(async (request: NativeRequest) => {
    let data: unknown;
    if (request.command === 'commish.rules.read') data = ruleReading;
    else if (request.command === 'commish.jobs.read') data = jobs(count);
    else if (request.command === 'job.snoozes.read' || request.command === 'job.snooze.set'
      || request.command === 'job.snooze.wake') data = { snoozes: [] };
    else if (request.command === 'commish.ledger.read') data = ledger();
    else if (request.command === 'backup.now' && request.mode === 'preview') data = backupPreview;
    else if (request.command === 'backup.now') data = { league, captured: '2026-11-18T12:00:00Z',
      verified: '2026-11-18T12:00:01Z', where: 'Cloudflare R2', checked: true,
      message: 'League backup saved 2026-11-18 12:00 UTC and checked in Cloudflare R2.' };
    else throw new Error(`Unexpected hosted request: ${request.command}`);
    return { data, label: 'League server', writable: true, football: 'held-pin' };
  }) as HostedPort & ReturnType<typeof vi.fn>;
}
let host: HTMLDivElement;
let root: Root;
let authGeneration = 0;
beforeEach(() => {
  hostedReads.endSession();
  authGeneration++;
  observeView({ authGeneration: String(authGeneration), server: 'https://league.test', league,
    message: '', name: 'Commissioner', role: 'Commissioner', franchise: '', franchiseName: '',
    seed: '', connection: '', helperId: '', confirmation: '', seats: [] });
  Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true });
  host = document.createElement('div'); document.body.append(host); root = createRoot(host);
});
afterEach(async () => {
  await act(async () => root.unmount());
  hostedReads.endSession(); host.remove(); vi.restoreAllMocks();
});
async function settle() { await act(async () => { await new Promise(resolve => setTimeout(resolve, 30)); }); }
function controlRoomButton() {
  return host.querySelector<HTMLButtonElement>(
    '.thin-league-shell .node>.nb[aria-label="Control Room"]');
}

const teams = Array.from({ length: 32 }, (_, index) => ({ franchise: `franchise.${index + 1}`,
  name: `Team ${index + 1}`, abbreviation: `T${index + 1}`, owner: `owner.${index + 1}`,
  caretaker: '', division: '' }));
const fixLedger = { league, name: 'Test League', revision: 7, entries: [{ id: 'fix.1', who: 'commissioner',
  what: 'Corrected a score', why: 'Recorded reason', when: '2026-11-17T12:00:00Z', gravity: 'G3',
  undo: 'league' }, { id: 'fix.2', who: 'commissioner', what: 'Already replaced', why: 'Recorded reason',
  when: '2026-11-16T12:00:00Z', gravity: 'G3', undo: 'league', replacedAt: '2026-11-17T12:00:00Z' }] };
const corrections = { league, revision: 7, items: [{ entry: 'fix.1', who: 'commissioner',
  what: 'Roster limit exception', why: 'Recorded reason', when: '2026-11-17T12:00:00Z',
  franchise: 'franchise.1', rule: 'roster.limits' }] };
const preview = { question: 'Record the correction?', target: { league, name: 'Test League', version: 7 },
  fields: [{ label: 'Change', before: 'Before', beforeSource: 'held', after: 'After',
    afterSource: 'commissioner', when: 'now' }],
  affects: { readings: ['league'], count: 32, words: '32 teams' },
  blockers: [], safeguard: 'hold' };
function assets() {
  return { league, revision: 7, seeded: true, source: 'held', pool: [], identities: [], snapshot: {
    season: 2026, franchises: teams.map(team => ({ id: team.franchise, heldId: team.franchise,
      name: team.name, abbreviation: team.abbreviation, division: '' })),
    players: [{ id: '0042', name: 'Josh Allen', franchise: 'franchise.2', status: 'ROSTER',
      salaryCents: 100, expirationYear: 2027 }], picks: [],
    pool: [{ player: '0044', name: 'Free Agent', season: 2026 }],
  } };
}
function fixPort(): HostedPort & ReturnType<typeof vi.fn> {
  let revision = 7;
  return vi.fn(async (request: NativeRequest) => {
    let data: unknown;
    if (request.command === 'commish.rules.read') data = { ...ruleReading, revision };
    else if (request.command === 'commish.jobs.read') data = jobs(32);
    else if (request.command === 'job.snoozes.read' || request.command === 'job.snooze.set'
      || request.command === 'job.snooze.wake') data = { snoozes: [] };
    else if (request.command === 'commish.corrections.read') data = { ...corrections, revision };
    else if (request.command === 'commish.ledger.read') data = { ...fixLedger, revision };
    else if (request.command === 'native.directory.read') data = { league, name: 'Test League',
      revision, rosterSeason: 2026, source: 'held', franchises: teams };
    else if (request.command === 'calendar.read') data = { league, revision,
      clock: { windows: [], exceptions: [{ action: 'lineup', franchise: 'franchise.2',
        start: '2026-11-17T12:00:00Z', until: '0001-01-01T00:00:00Z', permanent: true,
        reason: 'Recorded reason' }], tradeDeadline: null, secondsToTradeDeadline: null }, decision: {} };
    else if (request.command === 'native.assets.read') data = { ...assets(), revision };
    else if (request.command === 'week.read') data = { schedule: { league, franchise: '', season: 2026,
      revision, message: '', games: [] }, current: null, nextWeek: 1, message: '' };
    else if (request.command === 'standings.read') data = { league, name: 'Test League', season: 2026,
      revision, status: '', words: '', through: 0, decided: 0, pending: 0, criteria: [], noData: [],
      computed: [] };
    else if (request.command === 'bracket.read') data = { league, name: 'Test League', season: 2026,
      revision, status: '', words: '', started: false, fields: [], projected: [], rules: {
        words: '', perConference: 0, divisionWinners: 0, wildCards: 0, byeSeeds: 0,
        seeding: '', reseed: false, startWeek: 1, finalWeek: 1, homeBonus: 0, shape: [],
      } };
    else if (request.command === 'odds.read') data = { league, name: 'Test League', season: 2026,
      revision, status: '', words: '', powerWords: '', gameWords: '', oddsWords: '', week: 0,
      runs: 0, seed: 0, rows: [], games: [] };
    else if (request.mode === 'preview') data = { ...preview,
      target: { ...preview.target, version: revision } };
    else if (request.mode === 'execute') data = { entry: 'fix.done', league, revision: ++revision,
      recordedAt: '2026-11-17T12:30:00Z', message: 'Correction recorded.' };
    else throw new Error(`Unexpected hosted request: ${request.command}`);
    return { data, label: 'League server', writable: true, football: 'held-pin' };
  }) as HostedPort & ReturnType<typeof vi.fn>;
}
function setValue(element: HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement, value: string) {
  const setter = Object.getOwnPropertyDescriptor(Object.getPrototypeOf(element), 'value')?.set;
  setter?.call(element, value);
  element.dispatchEvent(new Event(element instanceof HTMLSelectElement ? 'change' : 'input',
    { bubbles: true }));
}
const fixCommands = ['commish.stat.correct', 'commish.score.player', 'commish.score.franchise',
  'commish.lineup.correct', 'commish.player.position', 'commish.standings.override',
  'commish.standings.record.adjust', 'commish.contract.edit', 'commish.correction.revert'];
it('finds a player rostered on another team from the owner Home screen', async () => {
  const read = vi.fn(async (request: NativeRequest) => {
    if (request.command === 'native.assets.read')
      return { data: assets(), label: 'League server', writable: false, football: '' };
    throw new Error(`Unexpected hosted request: ${request.command}`);
  }) as HostedPort & ReturnType<typeof vi.fn>;
  await act(async () => root.render(<HostedLeague league={league} franchise="franchise.1"
    name="Team 1" role="Owner" port={read} />));
  await settle();
  const input = host.querySelector<HTMLInputElement>('#thin-player-search');
  expect(input).not.toBeNull();
  await act(async () => setValue(input!, 'Josh Allen'));
  expect(host.textContent).toContain('Josh Allen');
  expect(host.querySelector('button[aria-label="Open Josh Allen player card"]')).not.toBeNull();
  await act(async () => setValue(input!, 'Free Agent'));
  expect(host.querySelector('button[aria-label="Open Free Agent player card"]')).not.toBeNull();
});

async function openControlRoom(read: HostedPort, role = 'Commissioner') {
  await act(async () => root.render(<HostedLeague league={league} franchise="" name="Test League"
    role={role} port={read} />));
  expect(host.querySelector('.thin-league-shell')).not.toBeNull();
  const controlRoom = controlRoomButton();
  expect(controlRoom).not.toBeNull();
  await act(async () => controlRoom!.click());
  await vi.waitFor(async () => {
    await settle();
    expect(host.querySelector('[data-testid="hosted-control-room"]')).not.toBeNull();
  });
}
it.each(['Commissioner', 'Co-commissioner'])(
  'opens the hosted Control Room from the Bundle 4 shell for %s', async role => {
    const read = port(0);
    await act(async () => root.render(<HostedLeague league={league} franchise=""
      name="Test League" role={role} port={read} />));
    expect(host.querySelector('.thin-league-shell')).not.toBeNull();
    const controlRoom = controlRoomButton();
    expect(controlRoom).not.toBeNull();
    await act(async () => controlRoom!.click());
    await vi.waitFor(async () => {
      await settle();
      expect(host.querySelector('[data-testid="hosted-control-room"]')).not.toBeNull();
    });
    expect(host.querySelector('[role="tab"][aria-label="Jobs"]')).not.toBeNull();
  });

async function openFixTab() {
  const tab = [...host.querySelectorAll<HTMLButtonElement>('[role="tab"]')]
    .find(button => button.textContent?.includes('Fix'));
  expect(tab).toBeDefined();
  await act(async () => tab!.click());
  await vi.waitFor(async () => {
    await settle();
    expect(host.querySelector('[data-testid="hosted-control-room-fix"]')).not.toBeNull();
  });
}
async function fillFixForm(command: string) {
  await vi.waitFor(async () => {
    await settle();
    expect(host.querySelector('.hosted-act-tray select, .hosted-act-tray input')).not.toBeNull();
  });
  const selects = [...host.querySelectorAll<HTMLSelectElement>('.hosted-act-tray select')];
  await act(async () => {
    selects.forEach((select, index) => {
      const option = command === 'commish.standings.override'
        ? select.options[index + 1] : select.options[1];
      if (option) setValue(select, option.value);
    });
    const inputs = host.querySelectorAll<HTMLInputElement>('.hosted-act-tray input:not([type="checkbox"])');
    for (const input of inputs)
      setValue(input, input.closest('label')?.textContent?.startsWith('Season') ? '2026' : '1');
    for (const checkbox of host.querySelectorAll<HTMLInputElement>('.hosted-act-tray input[type="checkbox"]'))
      checkbox.click();
    const reason = host.querySelector<HTMLTextAreaElement>('.hosted-act-tray textarea');
    if (reason) setValue(reason, 'Correction reason');
  });
}

it.each(['Commissioner', 'Co-commissioner'])(
  'mounts the v3 hosted tabs for %s and reuses one jobs read for the count', async role => {
  const read = port(32);
  await act(async () => root.render(<HostedLeague league={league} franchise="" name="Test League"
    role={role} port={read} />));
  expect(host.querySelector('.thin-league-shell')).not.toBeNull();
  const controlRoom = controlRoomButton();
  expect(controlRoom).not.toBeNull();
  await act(async () => controlRoom!.click());
  await vi.waitFor(async () => {
    await settle();
    expect(host.querySelector('[data-testid="hosted-control-room"]')).not.toBeNull();
  });
  const tabs = [...host.querySelectorAll('[role="tab"]')].map(button =>
    button.textContent?.replace(/\s+\d+$/, '').trim());
  expect(tabs).toEqual([
    'Jobs', 'Situations', 'Rules', 'Season', 'Fix', 'Picks', 'People', 'League', 'Server',
  ]);
  expect(host.textContent).toContain('32 from the league job read');
  expect(host.querySelector('[aria-label="League health"]')?.textContent).toContain('32');
  expect(read.mock.calls.filter(([request]) => request.command === 'commish.jobs.read')).toHaveLength(1);

  const rollover = host.querySelector<HTMLButtonElement>('[data-job="rollover"]');
  expect(rollover).not.toBeNull();
  await act(async () => rollover!.click());
  await settle();
  expect(host.textContent).toContain('Preview move to the next season');
  },
);

it('shows the hosted server ledger at 32 entries and leaves operator actions unavailable', async () => {
  const read = port(32);
  await act(async () => root.render(<HostedLeague league={league} franchise=""
    name="Test League" role="Commissioner" port={read} />));
  const controlRoom = controlRoomButton();
  await act(async () => controlRoom!.click());
  await settle();
  const serverTab = [...host.querySelectorAll<HTMLButtonElement>('[role="tab"]')]
    .find(button => button.textContent?.includes('Server'));
  await act(async () => serverTab!.click());
  await vi.waitFor(async () => {
    await settle();
    expect(host.textContent).toContain('Change 32');
  });
  expect(read.mock.calls.some(([request]) => request.command === 'commish.ledger.read')).toBe(true);
  expect(host.textContent).toContain('operator-only');
  expect(host.querySelectorAll('[aria-label="Server status"] ol li')).toHaveLength(32);
  const previewBackup = [...host.querySelectorAll<HTMLButtonElement>('button')]
    .find(button => button.textContent === 'Preview backup now')!;
  await act(async () => previewBackup.click());
  await vi.waitFor(() => expect(host.textContent).toContain('Save and verify a league backup now?'));
  expect(read.mock.calls.filter(([request]) => request.command === 'backup.now')
    .map(([request]) => request.mode)).toEqual(['preview']);
  const confirm = [...host.querySelectorAll<HTMLButtonElement>('button')]
    .find(button => button.textContent?.includes('Confirm back up now'))!;
  await act(async () => {
    confirm.dispatchEvent(new KeyboardEvent('keydown', { key: ' ', bubbles: true }));
    await new Promise(resolve => setTimeout(resolve, 650));
    confirm.dispatchEvent(new KeyboardEvent('keyup', { key: ' ', bubbles: true }));
  });
  await vi.waitFor(() => expect(host.textContent).toContain('checked in Cloudflare R2'));
  expect(read.mock.calls.filter(([request]) => request.command === 'backup.now')
    .map(([request]) => request.mode)).toEqual(['preview', 'execute']);
});

it('opens the server-backed Rules board and renders all 32 settings with their recorded source', async () => {
  const read = port(0);
  await act(async () => root.render(<HostedLeague league={league} franchise="" name="Test League"
    role="Commissioner" port={read} />));
  await settle();
  await act(async () => controlRoomButton()!.click());
  await settle();
  await act(async () => [...host.querySelectorAll<HTMLButtonElement>('[role="tab"]')]
    .find(button => button.textContent?.includes('Rules'))!.click());
  expect(read.mock.calls.some(([request]) => request.command === 'commish.rules.read')).toBe(true);
  expect(host.querySelector('[aria-label="Rosters rules"]')).not.toBeNull();
  expect(host.textContent).toContain('Setting 32');
  expect(host.textContent).toContain('YOU');
  expect(host.textContent).not.toContain('not on the server app yet');
});

it('opens the hosted frame for an assigned commissioner too', async () => {
  const read = port(0);
  await act(async () => root.render(<HostedLeague league={league} franchise="franchise.01"
    name="Franchise One" role="Commissioner" port={read} />));
  const controlRoom = controlRoomButton();
  expect(controlRoom).not.toBeNull();
  await act(async () => controlRoom!.click());
  await vi.waitFor(async () => {
    await settle();
    expect(host.querySelector('[data-testid="hosted-control-room"]')).not.toBeNull();
  });
  expect(read.mock.calls.some(([request]) => request.command === 'commish.jobs.read')).toBe(true);
});

it('does not mount or read Control Room jobs for a plain hosted owner', async () => {
  const read = port(0);
  await act(async () => root.render(<HostedLeague league={league} franchise="franchise.01"
    name="Franchise One" role="Owner" port={read} />));
  await settle();
  expect(host.querySelector('[data-testid="hosted-control-room"]')).toBeNull();
  const visibleControlRoom = [...host.querySelectorAll('button')]
    .some(button => button.textContent?.includes('Control Room'));
  expect(visibleControlRoom).toBe(false);
  expect(read.mock.calls.some(([request]) => request.command === 'commish.jobs.read')).toBe(false);
});

it('renders hosted corrections and all 32 franchises in the Fix tab', async () => {
  const read = fixPort();
  await openControlRoom(read);
  await openFixTab();
  await vi.waitFor(async () => {
    await settle();
    expect(host.textContent).toContain('Roster limit exception');
    expect(host.textContent).toContain('Corrected a score');
    expect(host.textContent).toContain('lineup outside its window');
    expect(host.textContent).toContain('Team 2');
  });
  expect([...host.querySelectorAll('[data-fix-command]')].map(row => row.getAttribute('data-fix-command')))
    .toEqual(fixCommands);
  expect(read.mock.calls.filter(([request]) => request.command === 'commish.corrections.read'))
    .toHaveLength(1);
  expect(read.mock.calls.filter(([request]) => request.command === 'commish.ledger.read'))
    .toHaveLength(1);

  const order = host.querySelector<HTMLButtonElement>(
    '[data-fix-command="commish.standings.override"] button');
  await act(async () => order!.click());
  await vi.waitFor(async () => {
    await settle();
    expect(host.querySelectorAll('.hosted-act-tray select')).toHaveLength(32);
  });
  expect(host.querySelector('.hosted-act-tray select:last-child')?.textContent).toContain('Team 32');

  const end = [...host.querySelectorAll<HTMLButtonElement>('.hosted-fix-tables button')]
    .find(button => button.textContent === 'END NOW');
  await act(async () => end!.click());
  await vi.waitFor(async () => {
    await settle();
    expect(host.querySelector<HTMLSelectElement>('.hosted-act-tray select')?.value).toBe('fix.1');
  });
});

it('routes each of the nine Fix writes through hosted preview, review and execute', async () => {
  const read = fixPort();
  await openControlRoom(read);
  await openFixTab();
  for (const command of fixCommands) {
    const card = host.querySelector<HTMLButtonElement>(`[data-fix-command="${command}"] button`);
    expect(card, command).not.toBeNull();
    await act(async () => card!.click());
    await fillFixForm(command);
    const previewButton = [...host.querySelectorAll<HTMLButtonElement>('.hosted-act-tray button')]
      .find(button => button.textContent?.startsWith('Preview'));
    expect(previewButton, `${command} preview`).toBeDefined();
    await act(async () => previewButton!.click());
    await vi.waitFor(async () => {
      await settle();
      expect(host.querySelector('.hosted-review')).not.toBeNull();
    });
    const confirm = [...host.querySelectorAll<HTMLButtonElement>('.hosted-review button')]
      .find(button => button.textContent?.includes('Confirm league change'));
    expect(confirm, `${command} review`).toBeDefined();
    const hold = host.querySelector<HTMLElement>('.hosted-review .hold-to-fire');
    await act(async () => {
      hold!.dispatchEvent(new MouseEvent('pointerdown', { bubbles: true, button: 0 }));
      await new Promise(resolve => setTimeout(resolve, 650));
      hold!.dispatchEvent(new MouseEvent('pointerup', { bubbles: true, button: 0 }));
    });
    await vi.waitFor(async () => {
      await settle();
      expect(read.mock.calls.some(([request]) => request.command === command && request.mode === 'execute'))
        .toBe(true);
      expect(host.querySelector<HTMLButtonElement>('.hosted-act-tray button')?.disabled).toBe(false);
    });
  }
  const requests = read.mock.calls.map(([request]) => request);
  const writes = requests.filter(request => fixCommands.includes(request.command) &&
    (request.mode === 'preview' || request.mode === 'execute'))
    .map(request => [request.command, request.mode]);
  expect(writes).toEqual(fixCommands.flatMap(command => [[command, 'preview'], [command, 'execute']]));
  for (const command of fixCommands) {
    const sentPreview = requests.find(request => request.command === command && request.mode === 'preview');
    expect(JSON.stringify(sentPreview?.arguments), command).toContain('Correction reason');
  }
}, 15000);

it('snoozes a shared job privately, lists it, and wakes it without a league write', async () => {
  let state: { snoozes: { league: string; jobId: string; until: string }[] } = { snoozes: [] };
  const read = vi.fn(async (request: NativeRequest) => {
    let data: unknown;
    if (request.command === 'commish.rules.read') data = ruleReading;
    else if (request.command === 'commish.jobs.read') data = jobs(2);
    else if (request.command === 'job.snoozes.read') data = state;
    else if (request.command === 'job.snooze.set') {
      const args = request.arguments as { league: string; jobId: string; until: string };
      state = { snoozes: [{ ...args }] };
      data = state;
    } else if (request.command === 'job.snooze.wake') {
      state = { snoozes: [] };
      data = state;
    } else throw new Error(`Unexpected hosted request: ${request.command}`);
    return { data, label: 'League server', writable: true, football: 'held-pin' };
  }) as HostedPort & ReturnType<typeof vi.fn>;
  await openControlRoom(read);
  expect(host.querySelector('[data-job="rollover"]')).not.toBeNull();
  const summary = host.querySelector<HTMLElement>('summary[aria-label="Snooze Rollover"]');
  expect(summary).not.toBeNull();
  await act(async () => summary!.click());
  const snooze = host.querySelector<HTMLButtonElement>(
    'button[aria-label="Snooze Rollover for 3 hours"]');
  expect(snooze).not.toBeNull();
  await act(async () => snooze!.click());
  await vi.waitFor(() => expect(host.querySelector('[data-job="rollover"]')).toBeNull());
  expect(host.querySelector('[aria-label="Snoozed jobs"]')?.textContent).toContain('Rollover');
  expect(read.mock.calls.some(([request]) => request.command === 'job.snooze.set')).toBe(true);

  const wake = host.querySelector<HTMLButtonElement>('button[aria-label="Wake Rollover"]');
  expect(wake).not.toBeNull();
  await act(async () => wake!.click());
  await vi.waitFor(() => expect(host.querySelector('[data-job="rollover"]')).not.toBeNull());
  expect(read.mock.calls.some(([request]) => request.command === 'job.snooze.wake')).toBe(true);
  expect(read.mock.calls.some(([request]) => request.command === 'commish.jobs.read'
    && request.mode === 'execute')).toBe(false);
});
