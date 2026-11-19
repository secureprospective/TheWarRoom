// Thin-mode transport seam; browser sessions keep credentials in the server's HttpOnly cookie.
import { observeView, prepareHostedAction } from '../data/hostedReads';
import type { NativeRequest } from '../commands/native';
import { object, text } from '../data/parseValues';
import type { ReadTags } from '../data/session';
import { nativeTransport } from './transport';
export type NativeAction = 'update' | 'retry' | 'status' | 'recover' | 'enroll' | 'login' | 'enter' | 'forget'
  | 'revoke-device' | 'revoke-all' | 'totp-begin' | 'totp-confirm' | 'totp-step-up' | 'recovery-codes';
export type HelperAction = 'helper-issue' | 'helper-revoke' | 'helper-check' | 'helper-confirm';
export type NativeSeat = { franchise: string; name: string };
export type NativeLeagueMembership = { id: string; name: string; role: string;
  franchise: string; franchiseName: string };
export type NativeView = {
  helperPreview?: string;
  update?: 'app' | 'server';
  authGeneration: string; server: string; league: string; message: string; name: string; role: string;
  franchise: string; franchiseName: string; seed: string;
  // A helper's connection line is shown once; a confirmation code covers one change.
  connection: string; helperId: string; confirmation: string; seats: NativeSeat[];
  // Browser session membership summaries; desktop admission remains a single configured league.
  leagues?: NativeLeagueMembership[];
  // New recovery codes are shown once, right after they are printed.
  recoveryCodes?: string[];
};
const updateListeners = new Set<(view: NativeView) => void>();
export function subscribeNativeUpdate(listener: (view: NativeView) => void): () => void {
  updateListeners.add(listener);
  return () => { updateListeners.delete(listener); };
}
const textKeys = ['authGeneration', 'server', 'league', 'message', 'name', 'role',
  'franchise', 'franchiseName', 'seed',
  'connection', 'helperId', 'confirmation'];
const isSeat = (seat: unknown) => Boolean(seat) && typeof seat === 'object'
  && Object.keys(seat as object).length === 2
  && typeof (seat as NativeSeat).franchise === 'string' && typeof (seat as NativeSeat).name === 'string';
function parseBrowserMemberships(value: unknown, allowEmpty = false): NativeLeagueMembership[] {
  if (!Array.isArray(value) || (!allowEmpty && value.length === 0))
    throw new Error('Unreadable browser memberships.');
  const seen = new Set<string>();
  return value.map(entry => {
    if (!entry || typeof entry !== 'object' || Array.isArray(entry))
      throw new Error('Unreadable browser membership.');
    const row = entry as Record<string, unknown>;
    const keys = ['id', 'name', 'role', 'franchise', 'franchiseName'];
    if (Object.keys(row).length !== keys.length || keys.some(key => typeof row[key] !== 'string')
      || !String(row.id).startsWith('native.') || seen.has(String(row.id)))
      throw new Error('Unreadable browser membership.');
    seen.add(String(row.id));
    return row as NativeLeagueMembership;
  });
}
export function parseNativeView(raw: unknown): NativeView {
  if (!raw || typeof raw !== 'object' || Array.isArray(raw)) throw new Error('Unreadable desktop response.');
  const value = raw as Record<string, unknown>;
  if (Object.keys(value).length !== textKeys.length + 1 + ('helperPreview' in value ? 1 : 0)
      + ('update' in value ? 1 : 0) + ('recoveryCodes' in value ? 1 : 0)
      + ('leagues' in value ? 1 : 0)
    || ('update' in value && value.update !== 'app' && value.update !== 'server')
    || ('helperPreview' in value && typeof value.helperPreview !== 'string')
    || ('recoveryCodes' in value && (!Array.isArray(value.recoveryCodes) || value.recoveryCodes.length !== 8
      || !value.recoveryCodes.every(c => typeof c === 'string' && c.length === 43)))
    || ('leagues' in value && !parseBrowserMemberships(value.leagues, value.league === ''))
    || textKeys.some(key => typeof value[key] !== 'string')
    || !/^\d+$/.test(String(value.authGeneration))
    || !Array.isArray(value.seats) || !value.seats.every(isSeat)
    || !validServer(value.server) || (value.league !== '' && !String(value.league).startsWith('native.'))
    || (!value.league && (value.name || value.role)))
    throw new Error('Unreadable desktop response.');
  return value as NativeView;
}
function validServer(value: unknown): boolean {
  if (typeof value !== 'string') return false;
  try {
    const url = new URL(value);
    return url.protocol === 'https:' || (url.protocol === 'http:'
      && ['localhost', '127.0.0.1', '::1'].includes(url.hostname));
  } catch { return false; }
}
let browserGeneration = 0;
let browserIdentity = '';
let browserCurrent: NativeView | undefined;
function createBrowserView(selected: NativeLeagueMembership | undefined, leagues: NativeLeagueMembership[],
  seats: NativeSeat[], message: string, forceGeneration: boolean): NativeView {
  const identity = selected ? JSON.stringify({ selected, leagues }) : '';
  if (identity !== browserIdentity || forceGeneration) { browserIdentity = identity; browserGeneration++; }
  const view = parseNativeView({
    authGeneration: String(browserGeneration), server: window.location.origin,
    league: selected?.id ?? '', message, name: selected?.name ?? '', role: selected?.role ?? '',
    franchise: selected?.franchise ?? '', franchiseName: selected?.franchiseName ?? '',
    seats, leagues, seed: '', connection: '', helperId: '', confirmation: '',
  });
  nativeTransport.setIdentity(view.league, view.authGeneration);
  browserCurrent = view;
  return view;
}
function browserView(raw?: unknown, message = '', forceGeneration = false): NativeView {
  if (raw === undefined) return createBrowserView(undefined, [], [], message, forceGeneration);
  if (!raw || typeof raw !== 'object' || Array.isArray(raw)) throw new Error('Unreadable browser session.');
  const response = raw as Record<string, unknown>;
  if (response.protocol !== 1 || Object.keys(response).length !== 3
    || !response.admission || typeof response.admission !== 'object' || Array.isArray(response.admission))
    throw new Error('Unreadable browser session.');
  const admission = response.admission as Record<string, unknown>;
  const keys = ['league', 'name', 'franchise', 'franchiseName', 'role', 'seats'];
  if (Object.keys(admission).length !== keys.length || keys.some(key => !(key in admission))
    || typeof admission.league !== 'string' || !admission.league.startsWith('native.')
    || ['name', 'franchise', 'franchiseName', 'role'].some(key => typeof admission[key] !== 'string')
    || !Array.isArray(admission.seats) || !admission.seats.every(isSeat))
    throw new Error('Unreadable browser admission.');
  const leagues = parseBrowserMemberships(response.leagues);
  const admitted = leagues.find(league => league.id === admission.league);
  if (!admitted || admitted.name !== admission.name || admitted.role !== admission.role
    || admitted.franchise !== admission.franchise || admitted.franchiseName !== admission.franchiseName)
    throw new Error('Browser admission is not in the membership list.');
  const selected = (!forceGeneration && browserCurrent
    ? leagues.find(league => league.id === browserCurrent?.league) : undefined) ?? admitted;
  return createBrowserView(selected, leagues,
    selected.id === admission.league ? admission.seats as NativeSeat[] : [], message, forceGeneration);
}
export function selectBrowserLeague(league: string): NativeView {
  if (nativeTransport.kind !== 'web' || !browserCurrent?.leagues)
    throw new Error('Browser league selection is unavailable.');
  const selected = browserCurrent.leagues.find(item => item.id === league);
  if (!selected) throw new Error('This account is not admitted to that league.');
  const view = createBrowserView(selected, browserCurrent.leagues, [], '', false);
  observeView(view);
  return view;
}
async function browserActionResult(action: string, response: Response): Promise<NativeView> {
  const raw: unknown = await response.json();
  if (!raw || typeof raw !== 'object' || Array.isArray(raw)) throw new Error('Unreadable account response.');
  const body = raw as Record<string, unknown>;
  if (body.protocol !== 1 || 'bearer' in body || 'recent_auth' in body || 'recentAuth' in body)
    throw new Error('Unreadable account response.');
  if (['login', 'status', 'enter'].includes(action))
    return browserView(raw, '', action === 'login');
  throw new Error('This account feature is not available in the browser app.');
}
function responseMessage(response: Response, fallback: string): Promise<string> {
  return response.text().then(text => text.trim().slice(0, 4096) || fallback).catch(() => fallback);
}
export async function nativeStartup(): Promise<NativeView | null> {
  const response = await nativeTransport.startup();
  if (response.status === 404 && nativeTransport.kind === 'wails') return null;
  if (nativeTransport.kind === 'web' && response.status === 401) {
    const view = browserView(); observeView(view); return view;
  }
  if (!response.ok) throw new Error('Desktop startup unavailable.');
  // Vite's local development fallback serves index.html for unknown desktop routes.
  if (response.headers.get('Content-Type')?.startsWith('text/html')) {
    if (nativeTransport.kind === 'web') throw new Error('The browser session endpoint is unavailable.');
    return null;
  }
  const view = nativeTransport.kind === 'web'
    ? browserView(await response.json()) : parseNativeView(await response.json());
  observeView(view);
  return view;
}
async function post(body: Record<string, string>): Promise<NativeView> {
  prepareHostedAction(body.action);
  const response = await nativeTransport.action(body);
  if (response.status === 426) {
    const view = parseNativeView(await response.json());
    observeView(view);
    if (!view.update) throw new Error('The update request could not be read. Close and reopen the app.');
    updateListeners.forEach(listener => listener(view));
    return view;
  }
  if (nativeTransport.kind === 'web' && response.status === 401
    && ['status', 'enter'].includes(body.action)) {
    const view = browserView(undefined, 'Your sign-in expired. Sign in again.');
    observeView(view);
    return view;
  }
  if (!response.ok) throw new Error(await responseMessage(response, 'The account request was refused.'));
  if (nativeTransport.kind === 'web' && body.action === 'forget') {
    const view = browserView(undefined, 'Signed out on this browser.');
    observeView(view);
    return view;
  }
  const view = nativeTransport.kind === 'web'
    ? await browserActionResult(body.action, response) : parseNativeView(await response.json());
  observeView(view);
  return view;
}
export function nativeAction(
  action: NativeAction, username = '', password = '', code = '',
): Promise<NativeView> {
  return post({ action, username, password, code });
}
export async function helperAction(
  action: HelperAction, fields: { helper?: string; franchise?: string; code?: string; change?: string },
): Promise<NativeView> {
  const response = await nativeTransport.helper({ action, ...fields });
  if (!response.ok) throw new Error(await responseMessage(response, 'The helper action was refused.'));
  const view = parseNativeView(await response.json());
  observeView(view);
  return view;
}
// Only recognize the envelope here; Go validates the change and the league describes it.
export function readConfirmRequest(text: string): boolean {
  const trimmed = text.trim();
  if (!trimmed.startsWith('twrc1.')) return false;
  try {
    const json = atob(trimmed.slice(6).replace(/-/g, '+').replace(/_/g, '/'));
    const request = JSON.parse(json) as { tokenId?: unknown; change?: { command?: unknown } };
    return typeof request.tokenId === 'string' && typeof request.change?.command === 'string';
  } catch { return false; }
}
export type HostedResponse = {
  label: string; writable: boolean; football: string; data: unknown; tags?: ReadTags;
};
function parseWorkflowTags(value: unknown): ReadTags {
  const row = object(value, 'Hosted read tags', ['serverEpoch', 'league', 'football', 'live', 'values']);
  if (!row.league || typeof row.league !== 'object' || Array.isArray(row.league))
    throw new Error('Unreadable hosted read tags.');
  const league: Record<string, string> = {};
  for (const [key, version] of Object.entries(row.league)) {
    if (!/^(?:league(?::\d+)?|team:.+|scores:\d+:\d+|values:\d+:.+|clock)$/.test(key)
      || typeof version !== 'string' || !/^\d+$/.test(version))
      throw new Error('Unreadable hosted read tags.');
    league[key] = version;
  }
  return { serverEpoch: text(row.serverEpoch, 'Server epoch'), league,
    football: text(row.football, 'Football version'), live: text(row.live, 'Live version'),
    values: text(row.values, 'Values version') };
}

export type HostedPlayerReadingUnavailable = { unavailable: true; reason: 'input-not-held' };
export function hostedPlayerReadingUnavailable(data: unknown): HostedPlayerReadingUnavailable | undefined {
  if (!data || typeof data !== 'object' || Array.isArray(data)) return undefined;
  const value = data as Record<string, unknown>;
  return Object.keys(value).length === 2 && value.unavailable === true
    && value.reason === 'input-not-held' ? value as HostedPlayerReadingUnavailable : undefined;
}
export async function hostedWorkflow(request: NativeRequest, football = ''): Promise<HostedResponse> {
  const response = await nativeTransport.workflow(request, football);
  if (!response.ok) {
    const fallback = 'The league could not accept this action. '
      + 'Refresh before retrying; check your connection, seat and authenticator.';
    if (request.mode === 'execute' && response.status === 409
      && response.headers.get('Content-Type') === 'text/plain; charset=utf-8') {
      const message = (await response.text()).trim();
      if (message.length > 0 && message.length <= 4096) throw new Error(message);
    }
    throw new Error(fallback);
  }
  const r = object(await response.json(), 'Hosted league', ['label', 'writable', 'football', 'data', 'tags']);
  if (typeof r.writable !== 'boolean') throw new Error('Unreadable league authority. Refresh to try again.');
  const tags = r.tags == null ? undefined : parseWorkflowTags(r.tags);
  return {
    label: text(r.label, 'League location'), writable: r.writable,
    football: text(r.football, 'Football evidence'), data: r.data, tags,
  };
}

export type ChangeTopic = {
  kind: 'league' | 'team' | 'scores' | 'values' | 'clock'; version: string;
  franchise?: string; season?: number; week?: number; run?: string; nextBoundary?: string;
};
export type NativeChanges = {
  authGeneration: string; seq: number; topics: ChangeTopic[];
  state: 'connected' | 'reconnecting' | 'signed-out' | 'refused';
  serverEpoch?: string; league?: string;
};
function parseChanges(raw: unknown): NativeChanges {
  const r = object(raw, 'Desktop changes',
    ['authGeneration', 'seq', 'topics', 'state', 'serverEpoch', 'league']);
  if (typeof r.authGeneration !== 'string' || !/^\d+$/.test(r.authGeneration)
    || !Number.isSafeInteger(r.seq) || Number(r.seq) < 0 || !Array.isArray(r.topics)
    || !['connected', 'reconnecting', 'signed-out', 'refused'].includes(String(r.state)))
    throw new Error('Unreadable desktop changes.');
  for (const topic of r.topics) {
    const t = object(topic, 'Change topic',
      ['kind', 'version', 'franchise', 'season', 'week', 'run', 'nextBoundary']);
    if (!['league', 'team', 'scores', 'values', 'clock'].includes(String(t.kind))
      || typeof t.version !== 'string' || !/^\d+$/.test(t.version))
      throw new Error('Unreadable change topic.');
    for (const key of ['franchise', 'run', 'nextBoundary']) {
      if (key in t && typeof t[key] !== 'string') throw new Error('Unreadable topic scope.');
    }
    for (const key of ['season', 'week']) {
      if (key in t && (!Number.isSafeInteger(t[key]) || Number(t[key]) < 1))
        throw new Error('Unreadable topic scope.');
    }
    if ((t.kind === 'team' && !t.franchise)
      || (t.kind === 'scores' && (!t.season || !t.week))
      || (t.kind === 'values' && (!t.season || !t.run))) throw new Error('Missing topic scope.');
  }
  for (const key of ['serverEpoch', 'league']) {
    if (key in r && typeof r[key] !== 'string') throw new Error('Unreadable change identity.');
  }
  return r as NativeChanges;
}

// The read owner subscribes once, invalidates first, then notifies its screen listeners.
export function subscribeChanged(listener: (change: NativeChanges) => void): () => void {
  return nativeTransport.subscribeChanged(value => {
    try { listener(parseChanges(value)); } catch {
      // Invalidations are hints; reads validate their own replies.
    }
  });
}
