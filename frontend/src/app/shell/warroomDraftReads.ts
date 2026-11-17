import { hostedWorkflow } from '../commissioner/api';
import { parseDraft, type Draft } from '../draftroom/draft';
import { provider as reports } from '../draftreports/api';
import { array, object, text } from '../data/parseValues';
import type { Board } from '../reports/contract';
import type { WarRoomNative, WarRoomContext } from './warroomReads';

function parseQueue(value: unknown): string[] {
  const response = object(value, 'Draft queue', ['players']);
  const players = array(response.players, 'Draft queue players', (item, path) => {
    const id = text(item, path);
    if (!/^\d{4,}$/.test(id)) throw new Error(`${path}: invalid player id`);
    return id;
  });
  if (players.length > 500 || new Set(players).size !== players.length)
    throw new Error('Draft queue is too large or has duplicate players');
  return players;
}

export async function readWarRoomDraft(c: WarRoomContext): Promise<{
  draft?: Draft; board?: Board; queue?: string[]; queueError?: string;
}> {
  if (!c.native) return { board: await reports.read() };
  const port = c.native.port ?? hostedWorkflow;
  const [draftResult, queueResult] = await Promise.allSettled([
    port({ command: 'draft.read', mode: 'read', arguments: { league: c.native.league,
      franchise: c.mine, season: 0, at: '0001-01-01T00:00:00Z' } }),
    port({ command: 'draft.queue.read', mode: 'read', arguments: { league: c.native.league } }),
  ]);
  if (draftResult.status === 'rejected') throw draftResult.reason;
  const draft = parseDraft(draftResult.value.data);
  if (draft.league !== c.native.league) throw new Error('Draft belongs to another league');
  if (queueResult.status === 'rejected') return { draft, queueError: 'Queue unavailable' };
  try {
    return { draft, queue: parseQueue(queueResult.value.data) };
  } catch {
    return { draft, queueError: 'Queue unavailable' };
  }
}

export async function saveWarRoomDraftQueue(native: WarRoomNative, players: string[]): Promise<string[]> {
  const { data } = await (native.port ?? hostedWorkflow)({ command: 'draft.queue.set', mode: 'execute',
    arguments: { league: native.league, players } });
  return parseQueue(data);
}
