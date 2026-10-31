import { hostedWorkflow } from '../commissioner/api';
import { parseDraft, type Draft } from '../draftroom/draft';
import { provider as reports } from '../draftreports/api';
import type { Board } from '../reports/contract';
import type { WarRoomContext } from './warroomReads';
export async function readWarRoomDraft(c: WarRoomContext): Promise<{ draft?: Draft; board?: Board }> {
  if (!c.native) return { board: await reports.read() };
  const { data } = await (c.native.port ?? hostedWorkflow)({ command: 'draft.read', mode: 'read',
    arguments: { league: c.native.league, franchise: c.mine, season: 0,
      at: '0001-01-01T00:00:00Z' } });
  const draft = parseDraft(data);
  if (draft.league !== c.native.league) throw new Error('Draft belongs to another league');
  return { draft };
}
