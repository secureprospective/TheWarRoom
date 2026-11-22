import { array, object, text, utc } from '../data/parseValues';
import { hostedWorkflow } from '../commissioner/api';
import type { HostedPort } from '../commissioner/hostedRoom';
import type { WarRoomContext } from './warroomReads';
import type { WarRoomDeal } from './warroomTradeReads';

export type WarRoomSavedDraft = {
  id: string; partner: string; give: string[]; get: string[]; label: string; at: string;
};

function parseDraft(value: unknown): WarRoomSavedDraft {
  const row = object(value, 'Saved proposal', ['id', 'partner', 'give', 'get', 'label', 'at']);
  const id = text(row.id, 'Proposal id');
  if (!/^[0-9a-f]{32}$/.test(id)) throw new Error('Unreadable saved proposal id');
  return { id, partner: text(row.partner, 'Proposal partner'),
    give: array(row.give, 'Assets sent', text), get: array(row.get, 'Assets received', text),
    label: row.label === undefined ? '' : text(row.label, 'Proposal label'), at: utc(row.at, 'Saved at') };
}

function portFor(context: WarRoomContext): HostedPort {
  if (!context.native) throw new Error('Saved proposals require the hosted league');
  return context.native.port ?? hostedWorkflow;
}

export async function readWarRoomDrafts(context: WarRoomContext): Promise<WarRoomSavedDraft[]> {
  const response = await portFor(context)({ command: 'trade.draft.read', mode: 'read',
    arguments: { league: context.native!.league } });
  const row = object(response.data, 'Saved proposals', ['drafts']);
  return array(row.drafts, 'Saved proposals', parseDraft);
}

export async function saveWarRoomDraft(context: WarRoomContext, deal: WarRoomDeal, label: string) {
  const native = context.native;
  if (!native) throw new Error('Saved proposals require the hosted league');
  const response = await portFor(context)({ command: 'trade.draft.save', mode: 'execute',
    arguments: { league: native.league, partner: deal.partner, give: deal.give, get: deal.get, label } });
  const saved = parseDraft(response.data);
  if (saved.partner !== deal.partner || JSON.stringify(saved.give) !== JSON.stringify(deal.give) ||
    JSON.stringify(saved.get) !== JSON.stringify(deal.get))
    throw new Error('Saved proposal does not match the current trade desk');
  return saved;
}

export async function deleteWarRoomDraft(context: WarRoomContext, id: string) {
  const native = context.native;
  if (!native) throw new Error('Saved proposals require the hosted league');
  const response = await portFor(context)({ command: 'trade.draft.delete', mode: 'execute',
    arguments: { league: native.league, id } });
  const deleted = object(response.data, 'Deleted proposal', ['id']);
  if (text(deleted.id, 'Deleted proposal id') !== id) throw new Error('A different proposal was deleted');
  return readWarRoomDrafts(context);
}
