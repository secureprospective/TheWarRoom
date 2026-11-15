import { parseNativeView, subscribeChanged,
  type NativeChanges, type NativeView } from '../control/nativeApi';
import { createPublicReadStorage } from './clientCache';
import { createReadSession, type ReadIdentity, type ReadTopic } from './session';

export const hostedReads = createReadSession({ storage: createPublicReadStorage() });
const listeners = new Set<() => void>();
let current: NativeView | undefined;
let epoch = '';
// Only a restart changes the render key; learning the first epoch must not remount the screen.
let fenceEpoch = '';
let subscribed = false;
let revision = 0;
let refreshing = false;
let trailing = false;
let pendingTopics: ReadTopic[] = [];
let snapshot = { key: '', blocked: true, view: current };
export const hostedSnapshot = () => snapshot;
export function onHostedChange(listener: () => void): () => void {
  listeners.add(listener);
  return () => { listeners.delete(listener); };
}
function notify() { for (const listener of [...listeners]) listener(); }
const topicListeners = new Set<(topics: ReadTopic[]) => void>();
/** onHostedTopics hears each change's topics after the read owner has applied them (C3 order). It is
 *  silent while the identity is suspended and for empty resync hints. */
export function onHostedTopics(listener: (topics: ReadTopic[]) => void): () => void {
  topicListeners.add(listener);
  return () => { topicListeners.delete(listener); };
}
function identity(view: NativeView): ReadIdentity {
  return { serverId: view.server, serverEpoch: epoch, accountId: view.authGeneration,
    authGeneration: view.authGeneration, league: view.league, view: `${view.role}/${view.franchise}` };
}
export function observeView(view: NativeView) {
  if (current && view.server === current.server && view.league === current.league
    && BigInt(view.authGeneration) < BigInt(current.authGeneration)) return;
  if (current && (view.server !== current.server || view.league !== current.league)) {
    epoch = ''; fenceEpoch = '';
  }
  if (current && (view.server !== current.server || view.league !== current.league
    || view.authGeneration !== current.authGeneration)) pendingTopics = [];
  revision++;
  current = view;
  const next = identity(view);
  hostedReads.setIdentity(next);
  hostedReads.applyTopics(pendingTopics);
  pendingTopics = [];
  snapshot = { key: JSON.stringify({ ...next, serverEpoch: fenceEpoch }),
    blocked: !view.name || !view.role, view };
  if (!subscribed && view.name && view.role) {
    subscribed = true;
    subscribeChanged(consume);
  }
  notify();
}
export function prepareHostedAction(action: string) {
  if (['login', 'enter', 'forget', 'revoke-device', 'revoke-all', 'recover'].includes(action)) {
    suspend();
    notify();
  }
}
function suspend() {
  revision++;
  hostedReads.endSession('desktop admission changed');
  snapshot = { ...snapshot, blocked: true };
}
async function refreshView() {
  if (refreshing) { trailing = true; return; }
  refreshing = true;
  do {
    trailing = false;
    const started = revision;
    try {
      // Unlike startup's saved-session probe, status rechecks the current admission in Go.
      const response = await fetch('/desktop/native', {
        method: 'POST', cache: 'no-store', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ action: 'status', username: '', password: '', code: '' }),
      });
      if (!response.ok) throw new Error('Desktop admission unavailable');
      const view = parseNativeView(await response.json());
      if (started === revision) observeView(view);
    } catch { /* A suspended identity stays hidden; ordinary refresh failures leave the screen intact. */ }
  } while (trailing);
  refreshing = false;
}
function consume(change: NativeChanges) {
  if (!current || BigInt(change.authGeneration) < BigInt(current.authGeneration)) return;
  const ended = change.state === 'signed-out' || change.state === 'refused';
  const changed = change.authGeneration !== current.authGeneration
    || (!!change.serverEpoch && !!epoch && change.serverEpoch !== epoch);
  const admission = change.topics.some(topic => topic.kind === 'league' || topic.kind === 'team');
  if (ended || changed) suspend();
  if (change.serverEpoch) {
    if (epoch && change.serverEpoch !== epoch) fenceEpoch = change.serverEpoch;
    epoch = change.serverEpoch;
    if (!snapshot.blocked) hostedReads.setIdentity(identity(current));
  }
  if (ended) {
    pendingTopics = [];
    observeView({ ...current, authGeneration: change.authGeneration,
      name: '', role: '', franchise: '', franchiseName: '', seats: [], seed: '',
      connection: '', helperId: '', confirmation: '', message: 'Sign in to reconnect.' });
  } else {
    const topics = change.topics.map(topic => ({ ...topic, league: change.league ?? current!.league }));
    if (snapshot.blocked) pendingTopics.push(...topics);
    else hostedReads.applyTopics(topics);
    if (!snapshot.blocked && topics.length > 0) for (const listener of [...topicListeners]) listener(topics);
    if (snapshot.blocked || admission) void refreshView();
  }
  notify();
}
