import { array, choice, id, integer, object, text, timestamp } from '../data/parseValues';
import type { NativeRequest } from '../commands/native';
import { newKey } from '../commissioner/forms';

export const CONTRACT_DECISION_ACTIONS = ['extension', 'tag', 'rfa_tender', 'resign'] as const;
export type ContractDecisionAction = (typeof CONTRACT_DECISION_ACTIONS)[number];
export type ContractDecisionQuote = {
  priceable: true;
  allowed: boolean;
  reason: string;
  salaryCents: number;
  salaries: { season: number; cents: number }[];
  cap?: { roomAfterCents?: number; words: string };
};
export type ContractDecisionChoice = {
  years: number;
  level: number;
  quote: ContractDecisionQuote;
};
export type ContractDecisionOption = {
  action: ContractDecisionAction;
  deadline: string;
  opensAt?: string;
  open: boolean;
  pastDeadline: boolean;
  choices: ContractDecisionChoice[];
};
export type ContractDecisionPlayer = {
  player: string;
  name: string;
  expiration: number;
  decision?: { action: ContractDecisionAction; years: number; level: number; at: string };
  options: ContractDecisionOption[];
};
export type ContractDecisions = {
  league: string;
  franchise: string;
  revision: number;
  season: number;
  words: string;
  players: ContractDecisionPlayer[];
};
export type ContractDecisionSubmit = { player: string; action: ContractDecisionAction;
  years: number; level: number; reason: string };

export function contractDecisionRequest(league: string, franchise: string, revision: number,
  selection: ContractDecisionSubmit): NativeRequest {
  return { command: 'contract.decision.set', mode: 'preview', arguments: { league,
    command: 'contract.decision.set', args: { contractDecision: { franchise, player: selection.player,
      action: selection.action, years: selection.years, level: selection.level } },
    reason: selection.reason, expected: revision, request: newKey(), effective: 'now',
      visibility: 'private' } };
}

function boolean(value: unknown, path: string): boolean {
  if (typeof value !== 'boolean') throw new Error(`${path}: expected boolean`);
  return value;
}
function schedule(value: unknown, path: string) {
  return array(value, path, (item, p) => {
    const row = object(item, p, ['season', 'cents']);
    return { season: integer(row.season, p), cents: integer(row.cents, p) };
  });
}
function quote(value: unknown, path: string, league: string, player: string,
  action: ContractDecisionAction, revision: number): ContractDecisionQuote {
  const row = object(value, path, ['league', 'revision', 'season', 'player', 'name', 'franchise', 'action',
    'actionName', 'allowed', 'priceable', 'reason', 'needs', 'years', 'salaryCents', 'salaries',
    'deadMoney', 'reliefCents', 'cap', 'basis', 'notes', 'source']);
  if (row.league !== league || row.revision !== revision || row.player !== player || row.action !== action)
    throw new Error(`${path}: quote belongs to another decision`);
  if (!boolean(row.priceable, `${path}.priceable`)) throw new Error(`${path}: unpriceable choice`);
  if (row.salaryCents === null || row.salaryCents === undefined)
    throw new Error(`${path}.salaryCents: expected a priced salary`);
  const salaries = schedule(row.salaries, `${path}.salaries`);
  if (salaries.length === 0) throw new Error(`${path}.salaries: expected a priced schedule`);
  const cap = row.cap == null ? undefined : object(row.cap, `${path}.cap`,
    ['beforeCents', 'afterCents', 'changeCents', 'roomAfterCents', 'words']);
  return { priceable: true, allowed: boolean(row.allowed, `${path}.allowed`),
    reason: text(row.reason, `${path}.reason`), salaryCents: integer(row.salaryCents, `${path}.salaryCents`),
    salaries,
    ...(cap ? { cap: { roomAfterCents: cap.roomAfterCents == null ? undefined : integer(cap.roomAfterCents,
      `${path}.cap.roomAfterCents`, Number.MIN_SAFE_INTEGER),
      words: text(cap.words, `${path}.cap.words`) } } : {}) };
}

export function parseContractDecisions(value: unknown, league: string, franchise: string,
  revision: number): ContractDecisions {
  const row = object(value, 'Contract decisions',
    ['league', 'franchise', 'revision', 'season', 'words', 'players']);
  if (row.league !== league || row.franchise !== franchise || row.revision !== revision)
    throw new Error('Contract decisions belong to another league, owner or revision');
  const players = array(row.players, 'Contract decision players', (entry, path) => {
    const player = object(entry, path, ['player', 'name', 'expiration', 'decision', 'options']);
    const playerID = id(player.player, `${path}.player`, true);
    const decision = player.decision == null ? undefined : (() => {
      const current = object(player.decision, `${path}.decision`, ['action', 'years', 'level', 'at']);
      return { action: choice(current.action, `${path}.decision.action`, CONTRACT_DECISION_ACTIONS),
        years: integer(current.years, `${path}.decision.years`), level: integer(current.level,
          `${path}.decision.level`), at: timestamp(current.at, `${path}.decision.at`) };
    })();
    const actions = new Set<ContractDecisionAction>();
    const options = array(player.options, `${path}.options`, (entry, optionPath) => {
      const option = object(entry, optionPath,
        ['action', 'deadline', 'opensAt', 'open', 'pastDeadline', 'choices']);
      const action = choice(option.action, `${optionPath}.action`, CONTRACT_DECISION_ACTIONS);
      if (actions.has(action)) throw new Error(`${optionPath}: duplicate action`);
      actions.add(action);
      const choices = array(option.choices, `${optionPath}.choices`, (entry, choicePath) => {
        const candidate = object(entry, choicePath, ['years', 'level', 'quote']);
        return { years: integer(candidate.years, `${choicePath}.years`), level: integer(candidate.level,
          `${choicePath}.level`), quote: quote(candidate.quote, `${choicePath}.quote`, league, playerID,
          action, revision) };
      });
      if (choices.length === 0) throw new Error(`${optionPath}: expected a priceable choice`);
      const terms = new Set<string>();
      for (const candidate of choices) {
        const key = `${candidate.years}:${candidate.level}`;
        if (terms.has(key)) throw new Error(`${optionPath}: duplicate terms`);
        terms.add(key);
      }
      if (typeof option.open !== 'boolean' || typeof option.pastDeadline !== 'boolean')
        throw new Error(`${optionPath}: unreadable deadline state`);
      const opensAt = option.opensAt == null ? undefined : timestamp(option.opensAt, `${optionPath}.opensAt`);
      const deadline = timestamp(option.deadline, `${optionPath}.deadline`);
      if (!deadline) throw new Error(`${optionPath}.deadline: expected a finite deadline`);
      const open = boolean(option.open, `${optionPath}.open`);
      const pastDeadline = boolean(option.pastDeadline, `${optionPath}.pastDeadline`);
      if (open && pastDeadline) throw new Error(`${optionPath}: a past deadline cannot be open`);
      return { action, deadline, opensAt, open, pastDeadline, choices };
    });
    if (decision && !decision.at) throw new Error(`${path}.decision.at: expected a recorded time`);
    return { player: playerID, name: text(player.name, `${path}.name`),
      expiration: integer(player.expiration, `${path}.expiration`), decision, options };
  });
  if (new Set(players.map(player => player.player)).size !== players.length)
    throw new Error('Contract decisions: duplicate player');
  return { league, franchise, revision, season: integer(row.season, 'Contract decisions season'),
    words: text(row.words, 'Contract decision words'), players };
}
