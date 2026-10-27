import { draftBindings as bindings } from '../commissioner/api';
import { createReadQuery } from '../data/query';
import { newKey } from '../commissioner/forms';
import { parseReceipt, refusalText, staleWords } from '../commissioner/proxy';
import { parseDraft, type Draft } from './draft';
import type * as Bindings from '../commissioner/api';

// Season 0 asks Go for the draft that is next; a time of zero asks it to read at its own now.
type Scope = { league: string; season: number; franchise: string; at: string };
type Choice = { league: string; season: number; franchise: string; player: string; request: string;
  expected: number };
type Port = { read(scope: Scope): Promise<unknown>; pick(choice: Choice): Promise<unknown> };


// The generated classes only add a converter to plain data; Wails sends the data as JSON.
type Arg<F> = F extends (arg: infer A) => unknown ? A : never;
const asArg = <F>(value: object) => value as unknown as Arg<F>;
const live: Port = {
  read: scope => bindings().TargetNativeDraft(asArg<typeof Bindings.TargetNativeDraft>(scope)),
  pick: choice => bindings().TargetNativeDraftPick(asArg<typeof Bindings.TargetNativeDraftPick>(choice)),
};
const zeroTime = '0001-01-01T00:00:00Z';

export type PickOutcome = { status: 'idle' } | { status: 'sending' } | { status: 'done'; message: string }
  | { status: 'refused'; message: string };
type RoomState = { draft?: Draft; error: string; pending: boolean; pick: PickOutcome };

// One draft room: the reading Go sent, the owner's pick, and why anything was refused.
export function createDraftRoom(port: Port = live) {
  const query = createReadQuery('commissioner.TargetNativeDraft', {
    initial: (): RoomState => ({ error: '', pending: false, pick: { status: 'idle' } }),
    retainErrorWhilePending: true, failure: refusalText,
    read: async (scope: Scope) => parseDraft(await port.read(scope)),
    accept: draft => ({ draft }),
  });
  const store = query.store;
  let held = { signature: '', key: '' };
  function read(league: string, franchise: string, season = 0) {
    const held = store.getState().draft;
    if (held && held.league !== league) store.setState({ draft: undefined });
    return query.load([{ league, season, franchise, at: zeroTime }]);
  }
  // A retry of the same pick repeats its request key, so it cannot be recorded twice.
  function requestKey(signature: string) {
    if (held.signature !== signature) held = { signature, key: newKey() };
    return held.key;
  }
  async function pick(franchise: string, player: string) {
    const draft = store.getState().draft;
    if (!draft) return;
    const choice = { league: draft.league, season: draft.season, franchise, player,
      expected: draft.revision,
      request: requestKey(`${draft.league}/${draft.season}/${franchise}/${player}/${draft.revision}`) };
    store.setState({ pick: { status: 'sending' } });
    try {
      const receipt = parseReceipt(await port.pick(choice));
      store.setState({ pick: { status: 'done', message: receipt.message } });
    } catch (cause) {
      const said = refusalText(cause);
      const stale = said.includes('reread before changing it');
      store.setState({ pick: { status: 'refused', message: stale ? staleWords : said } });
    }
    await read(draft.league, franchise, draft.season);
  }
  const forget = () => store.setState({ pick: { status: 'idle' } });
  // reset puts the room back as it was before anything was read.
  const reset = query.reset;
  return { read, pick, forget, reset, use: query.use, peek: store.getState };
}
export const draftRoom = createDraftRoom();
