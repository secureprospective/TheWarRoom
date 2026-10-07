import {
  ENVELOPE_EVENTS, ENVELOPE_STATES, GRAVITIES, UNDO_CLASSES, ROSTER_STATUSES,
} from './contract';
import type { AuditEntry, EnvelopeDemo, EnvelopeSpec, Receipt } from './contract';
import { object, choice, text, id, array, requiredText, utc } from './parseValues';

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
function envelopeSpec(value: unknown, path: string): EnvelopeSpec {
  const r = object(value, path, [
    'intent', 'leagueId', 'franchiseId', 'subject', 'expected',
    'gravity', 'undo', 'target', 'deadline',
  ]);
  const subject = object(r.subject, `${path}.subject`, ['players', 'picks']);
  const expected = object(r.expected, `${path}.expected`, ['player', 'rosterStatus']);
  const players = array(subject.players, `${path}.subject.players`, (v, p) => id(v, p, true));
  const player = id(expected.player, `${path}.expected.player`, true);
  if (!players.includes(player)) throw new Error(`${path}.expected.player: must be a subject`);
  return {
    intent: requiredText(r.intent, `${path}.intent`), leagueId: id(r.leagueId, `${path}.leagueId`),
    franchiseId: id(r.franchiseId, `${path}.franchiseId`),
    subject: { players, picks: array(subject.picks, `${path}.subject.picks`, requiredText) },
    expected: {
      player,
      rosterStatus: choice(expected.rosterStatus, `${path}.expected.rosterStatus`, ROSTER_STATUSES),
    },
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
