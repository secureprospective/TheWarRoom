import {
  ENVELOPE_EVENTS, ENVELOPE_STATES, GRAVITIES, UNDO_CLASSES, ROSTER_STATUSES,
} from './contract';
import type { AuditEntry, EnvelopeDemo, EnvelopeSpec, Receipt } from './contract';
import { object, choice, text, id, array, requiredText, utc, integer } from './parseValues';

function target(value: unknown, path: string): EnvelopeSpec['target'] {
  const r = object(value, path, ['kind', 'url']);
  const kind = choice(r.kind, `${path}.kind`, ['mapped', 'unmapped'] as const);
  if (kind === 'unmapped') {
    if (r.url !== undefined) throw new Error(`${path}.url: unmapped target cannot have URL`);
    return { kind };
  }
  const url = requiredText(r.url, `${path}.url`);
  let parsed: URL;
  try {
    parsed = new URL(url);
  } catch {
    throw new Error(`${path}.url: expected HTTPS URL`);
  }
  if (parsed.protocol !== 'https:' || parsed.username || parsed.password) {
    throw new Error(`${path}.url: expected HTTPS URL`);
  }
  return { kind, url };
}
function expectedChange(
  value: unknown, path: string, intent: string, players: string[], franchiseId: string,
): EnvelopeSpec['expected'] {
  const r = object(value, path, ['player', 'rosterStatus', 'lineup', 'trade']);
  if (intent === 'trade.accept') {
    for (const key of ['player', 'rosterStatus', 'lineup']) {
      if (r[key] !== undefined) throw new Error(`${path}.${key}: forbidden for trade.accept`);
    }
    const p = `${path}.trade`;
    const t = object(r.trade, p, ['tradeId', 'offering', 'accepting', 'offeringGives', 'acceptingGives']);
    const offering = id(t.offering, `${p}.offering`);
    const accepting = id(t.accepting, `${p}.accepting`);
    if (offering === accepting) throw new Error(`${p}.offering: must differ from accepting`);
    if (accepting !== franchiseId) throw new Error(`${p}.accepting: must equal franchiseId`);
    const offeringGives = array(t.offeringGives, `${p}.offeringGives`, requiredText);
    const acceptingGives = array(t.acceptingGives, `${p}.acceptingGives`, requiredText);
    if (!offeringGives.length) throw new Error(`${p}.offeringGives: expected nonempty array`);
    if (!acceptingGives.length) throw new Error(`${p}.acceptingGives: expected nonempty array`);
    return { trade: {
      tradeId: requiredText(t.tradeId, `${p}.tradeId`), offering, accepting, offeringGives, acceptingGives,
    } };
  }
  if (r.trade !== undefined) throw new Error(`${path}.trade: not a trade intent`);
  if (intent !== 'lineup.set') {
    if (r.lineup !== undefined) throw new Error(`${path}.lineup: not a lineup intent`);
    const player = id(r.player, `${path}.player`, true);
    if (!players.includes(player)) throw new Error(`${path}.player: must be a subject`);
    return { player, rosterStatus: choice(r.rosterStatus, `${path}.rosterStatus`, ROSTER_STATUSES) };
  }
  for (const key of ['player', 'rosterStatus']) {
    if (r[key] !== undefined) throw new Error(`${path}.${key}: forbidden for lineup.set`);
  }
  const p = `${path}.lineup`;
  const lineup = object(r.lineup, p, ['week', 'starters', 'baseline']);
  const starters = array(lineup.starters, `${p}.starters`, (v, at) => id(v, at, true));
  if (!starters.length || new Set(starters).size !== starters.length) {
    throw new Error(`${p}.starters: expected nonempty unique players`);
  }
  if (players.length !== starters.length || new Set(players).size !== starters.length ||
    !players.every((player) => starters.includes(player))) {
    throw new Error(`${path.replace(/expected$/, 'subject.players')}: must equal starters`);
  }
  return { lineup: {
    week: integer(lineup.week, `${p}.week`, 1), starters,
    baseline: array(lineup.baseline, `${p}.baseline`, (v, at) => id(v, at, true)),
  } };
}
function envelopeSpec(value: unknown, path: string): EnvelopeSpec {
  const r = object(value, path, [
    'intent', 'leagueId', 'franchiseId', 'subject', 'expected',
    'gravity', 'undo', 'target', 'deadline',
  ]);
  const subject = object(r.subject, `${path}.subject`, ['players', 'picks']);
  const players = array(subject.players, `${path}.subject.players`, (v, p) => id(v, p, true));
  const intent = requiredText(r.intent, `${path}.intent`);
  const franchiseId = id(r.franchiseId, `${path}.franchiseId`);
  const expected = expectedChange(r.expected, `${path}.expected`, intent, players, franchiseId);
  return {
    intent, leagueId: id(r.leagueId, `${path}.leagueId`),
    franchiseId,
    subject: { players, picks: array(subject.picks, `${path}.subject.picks`, requiredText) },
    expected,
    gravity: choice(r.gravity, `${path}.gravity`, GRAVITIES),
    undo: choice(r.undo, `${path}.undo`, UNDO_CLASSES),
    target: target(r.target, `${path}.target`),
    ...(r.deadline === undefined ? {} : { deadline: utc(r.deadline, `${path}.deadline`) }),
  };
}
function auditEntry(value: unknown, path: string): AuditEntry {
  const r = object(value, path, ['at', 'from', 'event', 'to', 'note']);
  return {
    at: utc(r.at, `${path}.at`), from: choice(r.from, `${path}.from`, ENVELOPE_STATES),
    event: choice(r.event, `${path}.event`, ENVELOPE_EVENTS),
    to: choice(r.to, `${path}.to`, ENVELOPE_STATES), note: text(r.note, `${path}.note`),
  };
}
function receipt(value: unknown, path: string): Receipt {
  const r = object(value, path, ['correlationId', 'spec', 'state', 'audit']);
  const spec = envelopeSpec(r.spec, `${path}.spec`);
  const state = choice(r.state, `${path}.state`, ENVELOPE_STATES);
  if (spec.target.kind === 'unmapped' && state !== 'draft' && state !== 'blocked') {
    throw new Error(`${path}.state: unmapped target must stay draft or blocked`);
  }
  const audit = array(r.audit, `${path}.audit`, auditEntry);
  let previous = 'draft';
  let at = -Infinity;
  for (const entry of audit) {
    if (entry.from !== previous || Date.parse(entry.at) < at) {
      throw new Error(`${path}.audit: discontinuous trail`);
    }
    previous = entry.to;
    at = Date.parse(entry.at);
  }
  if (previous !== state) throw new Error(`${path}.state: must match audit head`);
  return {
    correlationId: requiredText(r.correlationId, `${path}.correlationId`), spec, state, audit,
  };
}
export function parseReceipt(value: unknown): Receipt {
  return receipt(value, 'receipt');
}
export function parseEnvelopeDemo(value: unknown): EnvelopeDemo {
  const r = object(value, 'envelope', ['kind', 'receipt']);
  return {
    kind: choice(r.kind, 'envelope.kind', ['fixture'] as const),
    receipt: receipt(r.receipt, 'envelope.receipt'),
  };
}
