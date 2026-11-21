import { plainFailure } from './plainFailure';

export const ERROR_CODE_HEADER = 'X-TWR-Error-Code';

const ALPHABET = '23456789ABCDEFGHJKLMNPQRSTUVWXYZ';
const CODE_RE = /^TWR-[23456789ABCDEFGHJKLMNPQRSTUVWXYZ]{4}-[23456789ABCDEFGHJKLMNPQRSTUVWXYZ]{4}$/;
const ROUTE_RE = /^[a-z0-9-]+\/[a-z0-9-]+$/;
const MAX_RECENT = 64;
const REPORT_LIMIT = 2000;

export type LeagueInfo = { league: string; name: string };
export type FailureMeta = {
  screenLabel: string;
  route: string;
  appRelease?: number;
  appVersion: string;
  league: string;
  leagueName: string;
};

const errorCodes = new WeakMap<Error, string>();
const recentByMessage = new Map<string, string>();
let currentScreen = { label: 'Home', route: 'home/overview' };
let lastAppRelease: number | undefined;
let leagueProvider: (() => LeagueInfo | undefined) | undefined;

export function validErrorCode(value: string): boolean {
  return CODE_RE.test(value);
}

export function newClientErrorCode(): string {
  const bytes = new Uint8Array(7);
  if (typeof crypto !== 'undefined' && typeof crypto.getRandomValues === 'function') {
    crypto.getRandomValues(bytes);
  } else {
    for (let i = 0; i < bytes.length; i++) bytes[i] = Math.floor(Math.random() * 256);
  }
  let head = 'C';
  for (let i = 0; i < 3; i++) head += ALPHABET[bytes[i] & 31];
  let tail = '';
  for (let i = 3; i < 7; i++) tail += ALPHABET[bytes[i] & 31];
  return `TWR-${head}-${tail}`;
}

function rememberMessageCode(message: string, code: string): void {
  const key = plainFailure(message).trim();
  if (!key) return;
  if (recentByMessage.has(key)) recentByMessage.delete(key);
  recentByMessage.set(key, code);
  if (recentByMessage.size > MAX_RECENT) {
    const oldest = recentByMessage.keys().next().value;
    if (oldest !== undefined) recentByMessage.delete(oldest);
  }
}

export function attachErrorCode(error: Error, code?: string): Error {
  const resolved = code && validErrorCode(code) ? code : errorCodes.get(error) ?? newClientErrorCode();
  errorCodes.set(error, resolved);
  rememberMessageCode(error.message, resolved);
  return error;
}

export function createFailureError(message: string, code?: string): Error {
  return attachErrorCode(new Error(plainFailure(message)), code);
}

export function errorCodeFor(cause: unknown): string | undefined {
  if (cause instanceof Error) {
    return errorCodes.get(cause) ?? lookupErrorCode(cause.message);
  }
  return typeof cause === 'string' ? lookupErrorCode(cause) : undefined;
}

export function lookupErrorCode(message: string): string | undefined {
  const key = plainFailure(message).trim();
  return key ? recentByMessage.get(key) : undefined;
}

export function setFailureScreen(label: string, route: string): void {
  const cleanLabel = label.trim() || 'App';
  const cleanRoute = ROUTE_RE.test(route) ? route : 'home/overview';
  currentScreen = { label: cleanLabel, route: cleanRoute };
}

export function recordAppRelease(release: number | undefined): void {
  if (typeof release === 'number' && Number.isSafeInteger(release) && release >= 1 && release <= 999999) {
    lastAppRelease = release;
  }
}

export function registerLeagueProvider(provider: () => LeagueInfo | undefined): void {
  leagueProvider = provider;
}

export function baseAppVersion(): string {
  return typeof __TWR_APP_VERSION__ === 'string' && __TWR_APP_VERSION__
    ? __TWR_APP_VERSION__ : '0.5.0';
}

export function getFailureMeta(overrides?: {
  screen?: string;
  route?: string;
  league?: string;
  leagueName?: string;
  appRelease?: number;
}): FailureMeta {
  const active = leagueProvider?.();
  const league = overrides?.league ?? active?.league ?? '';
  const leagueName = overrides?.leagueName ?? (league === active?.league ? active?.name ?? '' : '');
  const release = overrides?.appRelease ?? lastAppRelease;
  const route = overrides?.route && ROUTE_RE.test(overrides.route)
    ? overrides.route : currentScreen.route;
  const screenLabel = overrides?.screen?.trim() || currentScreen.label;
  const version = baseAppVersion();
  const appVersion = release ? `${version} (release ${release})` : version;
  return { screenLabel, route, appRelease: release, appVersion, league, leagueName };
}

export function formatCentralTime(date: Date = new Date()): string {
  try {
    return new Intl.DateTimeFormat('en-US', {
      timeZone: 'America/Chicago',
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
      hour: 'numeric',
      minute: '2-digit',
      second: '2-digit',
      hour12: true,
      timeZoneName: 'short',
    }).format(date);
  } catch {
    return date.toISOString();
  }
}

function stripUnsupportedControlChars(value: string): string {
  let out = '';
  for (const char of value) {
    const code = char.codePointAt(0) ?? 0;
    const allowed = char === '\n' || char === '\r' || char === '\t';
    if (!allowed && (code < 0x20 || (code >= 0x7f && code <= 0x9f))) continue;
    out += char;
  }
  return out;
}

export function formatFailureDetails(input: {
  code: string;
  message: string;
  at?: Date;
  screen: string;
  appVersion: string;
  league: string;
}): string {
  const lines = [
    `Error code: ${input.code}`,
    `Time (Central): ${formatCentralTime(input.at ?? new Date())}`,
    `Screen: ${input.screen}`,
    `App version: ${input.appVersion}`,
    `League: ${input.league || 'None selected'}`,
    `Message: ${plainFailure(input.message).trim()}`,
  ];
  const clean = stripUnsupportedControlChars(lines.join('\n'));
  return Array.from(clean).slice(0, REPORT_LIMIT).join('');
}

export function resetFailureStateForTest(): void {
  recentByMessage.clear();
  currentScreen = { label: 'Home', route: 'home/overview' };
  lastAppRelease = undefined;
  leagueProvider = undefined;
}
