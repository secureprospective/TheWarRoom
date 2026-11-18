import { createReadQuery } from '../data/query';
import { array, integer, object, text, timestamp } from '../data/parseValues';
import { hostedWorkflow, type HostedResponse } from '../control/nativeApi';
import { hostedCommandAvailable, type NativeRequest } from '../commands/native';
import { parsePulseDirectory } from '../data/nativePulse';
import { actsFor, actTable, type ActDefinition } from './actTable';
import { parseJobs, type Job, type Jobs } from './jobs';
import { parseLedger, type Ledger } from './proxy';
import type { HostedPort } from './hostedRoom';

export const hostedControlTabs = [
  ['jobs', 'Jobs'], ['situations', 'Situations'], ['rules', 'Rules'], ['season', 'Season'],
  ['fix', 'Fix'], ['picks', 'Picks'], ['people', 'People'], ['league', 'League'], ['server', 'Server'],
] as const;
export type HostedControlTab = (typeof hostedControlTabs)[number][0];

// Job visibility comes from the server read; do not infer per-deputy scope in the client.
export function createHostedJobsQuery(league: string, port: HostedPort = hostedWorkflow) {
  return createReadQuery('commissioner.TargetCommissionerJobs', {
    initial: (): { jobs?: Jobs } => ({}),
    read: async (scope: string): Promise<Jobs> => {
      const response: HostedResponse = await port({ command: 'commish.jobs.read', mode: 'read',
        arguments: { league: scope } } as NativeRequest);
      const jobs = parseJobs(response.data);
      if (jobs.league !== league || scope !== league) throw new Error('Jobs belong to another league.');
      return jobs;
    },
    accept: (jobs: Jobs) => ({ jobs }),
  });
}

export function firstHostedJobTool(job: Job): ActDefinition | undefined {
  for (const control of job.controls) {
    const definition = actsFor(control).find(candidate => hostedCommandAvailable(candidate.command));
    if (definition) return definition;
  }
  return undefined;
}

export const hostedControlRoomNotOnServer = 'This part of the Control Room is not on the server app yet.';
export type HostedCorrection = {
  entry: string; who: string; when: string; what: string;
  franchise?: string; rule?: string; until?: string; scope?: { franchise?: string };
};
export type HostedCorrections = { league: string; revision: number; items: HostedCorrection[] };
export type HostedCalendarException = {
  action: string; franchise: string; start: string; until: string; permanent: boolean;
};
export type HostedFixReadings = {
  corrections?: HostedCorrections; ledger?: Ledger;
  calendarExceptions?: HostedCalendarException[]; franchises?: { franchise: string; name: string }[];
  correctionsError: string; ledgerError: string; calendarError: string;
};

export function parseHostedCorrections(value: unknown): HostedCorrections {
  const result = object(value, 'League corrections', ['league', 'revision', 'items']);
  return {
    league: text(result.league, 'League'), revision: integer(result.revision, 'Version'),
    items: array(result.items, 'Corrections', (value, path) => {
      const row = object(value, path, ['entry', 'who', 'when', 'why', 'what', 'player', 'franchise',
        'pick', 'rule', 'scope', 'until']);
      let scope: HostedCorrection['scope'];
      if (row.scope !== undefined) {
        const raw = object(row.scope, `${path}.scope`, ['franchise', 'player', 'transaction']);
        scope = raw.franchise === undefined ? {} : {
          franchise: text(raw.franchise, `${path}.scope.franchise`),
        };
        for (const key of ['player', 'transaction'])
          if (raw[key] !== undefined) text(raw[key], `${path}.scope.${key}`);
      }
      return {
        entry: text(row.entry, path), who: text(row.who, path), when: timestamp(row.when, path),
        what: text(row.what, path),
        ...(row.franchise === undefined ? {} : { franchise: text(row.franchise, path) }),
        ...(row.rule === undefined ? {} : { rule: text(row.rule, path) }),
        ...(row.until === undefined ? {} : { until: timestamp(row.until, path) }),
        ...(scope === undefined ? {} : { scope }),
      };
    }),
  };
}

async function readHostedFixSource(port: HostedPort, command: string, league: string): Promise<unknown> {
  if (!hostedCommandAvailable(command, 'read')) throw new Error(hostedControlRoomNotOnServer);
  const request = { command, mode: 'read', arguments: { league } } as NativeRequest;
  return (await port(request)).data;
}

function readError(cause: unknown): string {
  return cause instanceof Error ? cause.message : String(cause);
}

async function readHostedCalendarExceptions(port: HostedPort, league: string) {
  if (!hostedCommandAvailable('native.directory.read', 'read') ||
      !hostedCommandAvailable('calendar.read', 'read'))
    throw new Error(hostedControlRoomNotOnServer);
  const directoryRequest = { command: 'native.directory.read', mode: 'read',
    arguments: { league } } as NativeRequest;
  const directory = parsePulseDirectory((await port(directoryRequest)).data, league);
  const franchise = directory.seats[0]?.franchise;
  if (!franchise) throw new Error('No franchise is available to read calendar exceptions.');
  const calendarRequest = { command: 'calendar.read', mode: 'read', arguments: { league,
    season: directory.season, franchise, action: 'lineup',
    localZone: Intl.DateTimeFormat().resolvedOptions().timeZone } } as NativeRequest;
  const value = object((await port(calendarRequest)).data, 'League calendar',
    ['league', 'revision', 'clock', 'decision']);
  if (value.league !== league) throw new Error('Calendar belongs to another league.');
  const clock = object(value.clock, 'Action calendar',
    ['windows', 'exceptions', 'tradeDeadline', 'secondsToTradeDeadline']);
  const exceptions = array(clock.exceptions, 'Calendar exceptions', (value, path) => {
    const row = object(value, path, ['permanent', 'action', 'franchise', 'start', 'until', 'reason']);
    if (row.permanent !== undefined && typeof row.permanent !== 'boolean')
      throw new Error(`${path}.permanent: expected true or false`);
    return { action: text(row.action, path), franchise: text(row.franchise, path),
      start: timestamp(row.start, path), until: timestamp(row.until, path),
      permanent: row.permanent === true };
  });
  const franchises = directory.seats.map(({ franchise, name }) => ({ franchise, name }));
  return { exceptions, franchises };
}

export function createHostedFixQuery(league: string, port: HostedPort = hostedWorkflow) {
  return createReadQuery('commissioner.TargetCorrections', {
    initial: (): { fixes?: HostedFixReadings } => ({}),
    read: async (scope: string): Promise<HostedFixReadings> => {
      const [corrections, ledger, calendar] = await Promise.allSettled([
        readHostedFixSource(port, 'commish.corrections.read', scope).then(parseHostedCorrections),
        readHostedFixSource(port, 'commish.ledger.read', scope).then(parseLedger),
        readHostedCalendarExceptions(port, scope),
      ]);
      const result: HostedFixReadings = { correctionsError: '', ledgerError: '', calendarError: '' };
      if (corrections.status === 'fulfilled' && corrections.value.league === league && scope === league)
        result.corrections = corrections.value;
      else result.correctionsError = corrections.status === 'rejected'
        ? readError(corrections.reason) : 'Corrections belong to another league.';
      if (ledger.status === 'fulfilled' && ledger.value.league === league && scope === league)
        result.ledger = ledger.value;
      else result.ledgerError = ledger.status === 'rejected'
        ? readError(ledger.reason) : 'The ledger belongs to another league.';
      if (calendar.status === 'fulfilled') {
        result.calendarExceptions = calendar.value.exceptions;
        result.franchises = calendar.value.franchises;
      } else result.calendarError = readError(calendar.reason);
      return result;
    },
    accept: fixes => ({ fixes }),
  });
}

export type SituationRoute =
  | { kind: 'tab'; tab: HostedControlTab }
  | { kind: 'tool'; definition: ActDefinition }
  | { kind: 'unavailable'; words: string }
  | { kind: 'unknown'; route: string };

const branchTabs: Record<string, HostedControlTab> = {
  Jobs: 'jobs', Situations: 'situations', Rules: 'rules', Season: 'season', Fix: 'fix',
  Picks: 'picks', People: 'people', League: 'league', Server: 'server',
};
const unavailableCommands = new Set([
  'account.totp.reset', 'draft.change', 'draft.revert', 'draft.timer', 'draft.skip', 'draft.assign',
  'draft.insert', 'draft.archive', 'draft.start', 'draft.pick', 'pick.grant', 'stat.queue.hold',
  'commish.notice.publish', 'commish.newsletter.publish', 'pick.forfeit', 'pick.swap',
  'pick.condition', 'pick.void', 'pick.create', 'rules.text.add', 'league.alignment',
]);

/** Resolves every guide step to a hosted tab, a server-backed actTable row or an explicit seam. */
export function resolveSituationStep(step: string, mark: 'B' | 'R' | 'G'): SituationRoute {
  if (step.includes('›')) {
    const branch = step.split('›')[0]!.trim();
    const tab = branchTabs[branch];
    return tab ? { kind: 'tab', tab } : { kind: 'unknown', route: step };
  }
  const command = step.trim().split(/\s+/)[0] ?? '';
  if (command === '—') return { kind: 'unavailable', words: 'This route has no server action.' };
  if (command === 'stats.queue') {
    const queue = actTable.find(candidate => candidate.command === 'commish.stat.queue.approve');
    return queue && hostedCommandAvailable(queue.command) ? { kind: 'tool', definition: queue }
      : { kind: 'unavailable', words: 'The statistics queue is not on the server app yet.' };
  }
  if (command === 'commish.rules.set') return { kind: 'tab', tab: 'rules' };
  const definition = actTable.find(candidate => candidate.command === command);
  if (mark === 'B' && definition && hostedCommandAvailable(command)) return { kind: 'tool', definition };
  if (unavailableCommands.has(command) || command.startsWith('draft.'))
    return { kind: 'unavailable', words: 'This action is not on the server app yet.' };
  if (definition && !hostedCommandAvailable(command))
    return { kind: 'unavailable', words: 'This action is not on the server app yet.' };
  return { kind: 'unknown', route: step };
}
