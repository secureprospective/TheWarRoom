import { createReadQuery } from '../data/query';
import { newKey } from '../commissioner/forms';
import { parseReceipt, refusalText, staleWords } from '../commissioner/proxy';
import { parseDraft, type Draft } from './draft';

// Season 0 asks Go for the draft that is next, or with startup the league's startup draft; a time of zero
// asks it to read at its own now.
type Scope = { league: string; season: number; franchise: string; at: string; startup?: boolean };
type Choice = { league: string; season: number; franchise: string; player: string; request: string;
  expected: number; startup?: boolean };
export type DraftRoomPort = {
  read(scope: Scope): Promise<unknown>; pick(choice: Choice): Promise<unknown>;
};


const unavailable = async (): Promise<never> => {
  throw new Error('The draft room requires the desktop app.');
};
const live: DraftRoomPort = { read: unavailable, pick: unavailable };
const zeroTime = '0001-01-01T00:00:00Z';

export type PickOutcome = { status: 'idle' } | { status: 'sending' } | { status: 'done'; message: string }
  | { status: 'refused'; message: string };
type RoomState = { draft?: Draft; scope?: Scope; error: string; pending: boolean; pick: PickOutcome;
  startup: boolean };

// One draft room: the reading Go sent, the owner's pick, and why anything was refused.
export function createDraftRoom(port: DraftRoomPort = live) {
  const query = createReadQuery('commissioner.TargetNativeDraft', {
    initial: (): RoomState => ({ error: '', pending: false, pick: { status: 'idle' }, startup: false }),
    retainErrorWhilePending: true, failure: refusalText,
    read: async (scope: Scope) => parseDraft(await port.read(scope)),
    accept: draft => ({ draft }),
  });
  const store = query.store;
  let held = { signature: '', key: '' };
  function read(league: string, franchise: string, season = 0) {
    const { draft: held, startup } = store.getState();
    if (held && (held.league !== league || held.startup !== startup)) store.setState({ draft: undefined });
    const scope = { league, season, franchise, at: zeroTime, ...(startup ? { startup } : {}) };
    store.setState({ scope });
    return query.load([scope]);
  }
  // choose says which draft the room shows: the league's startup draft, or the rookie draft that is next.
  const choose = (startup: boolean) => store.setState({ startup });
  // A retry of the same pick repeats its request key, so it cannot be recorded twice.
  function requestKey(signature: string) {
    if (held.signature !== signature) held = { signature, key: newKey() };
    return held.key;
  }
  async function pick(franchise: string, player: string) {
    const draft = store.getState().draft;
    if (!draft) return;
    const choice = { league: draft.league, season: draft.season, franchise, player,
      expected: draft.revision, ...(draft.startup ? { startup: true } : {}),
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
  return { read, pick, forget, reset, choose, use: query.use, peek: store.getState };
}
export const draftRoom = createDraftRoom();
