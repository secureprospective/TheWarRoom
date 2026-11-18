import { hostedReads } from '../data/hostedReads';
import { createReadQuery } from '../data/query';
import type { TaggedReply } from '../data/session';
import { hostedWorkflow, type HostedResponse } from '../control/nativeApi';
import { hostedCommandAvailable, type NativeRequest } from '../commands/native';
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

export function createHostedLedgerQuery(league: string, port: HostedPort = hostedWorkflow) {
  const readLedger = async (scope: string, filter: string) => {
    const response: HostedResponse = await port({ command: filter
      ? 'commish.ledger.filtered.read' : 'commish.ledger.read', mode: 'read',
    arguments: filter ? { league: scope, filter: { person: '', kind: filter, from: '', before: '' } }
      : { league: scope } } as NativeRequest);
    const ledger = parseLedger(response.data);
    if (ledger.league !== league || scope !== league)
      throw new Error('Ledger belongs to another league.');
    return { ledger, football: response.football };
  };
  return createReadQuery<Ledger, { ledger?: Ledger }, [string, string]>(
    'commissioner.TargetCommissionerLedger', {
    initial: () => ({}),
    hosted: { session: hostedReads, schemaVersion: '1', dependencies: [{ kind: 'league' }],
      policy: 'memory' },
    read: async ([scope, filter]): Promise<TaggedReply<Ledger>> => {
      const result = await readLedger(scope, filter);
      const key = JSON.parse(hostedReads.key('commissioner.TargetCommissionerLedger', '1',
        [scope, filter])) as unknown;
      if (!Array.isArray(key) || typeof key[1] !== 'string')
        throw new Error('The hosted league identity is unavailable. Refresh before reading the ledger.');
      return { data: result.ledger, tags: { serverEpoch: key[1],
        league: { league: String(result.ledger.revision) }, football: result.football,
        live: '', values: '' } };
    },
    accept: ledger => ({ ledger }),
  });
}

export function firstHostedJobTool(job: Job): ActDefinition | undefined {
  for (const control of job.controls) {
    const definition = actsFor(control).find(candidate => hostedCommandAvailable(candidate.command));
    if (definition) return definition;
  }
  return undefined;
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
