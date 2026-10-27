import { createReadQuery } from '../data/query';
import type { Comms } from './contract';
import { commsAPI, type CommsReader } from './api';

export function createCommsState(reader: CommsReader) {
  type Selection = { comms?: Comms; team: string; kind: string; limit: number };
  const query = createReadQuery('comms.TargetComms', {
    initial: (): Selection => ({ team: '', kind: '', limit: 50 }),
    read: (scope) => reader.read(scope.team, scope.kind),
    accept: comms => ({ comms, team: comms.team, kind: comms.kind }),
  });
  const load = (refresh = false) => query.load(() => {
    const { team, kind } = query.peek();
    return [{ team, kind }];
  }, { before: refresh ? () => reader.refresh(true) : undefined });
  async function choose(action: 'refresh' | 'team' | 'kind' | 'more' | 'open', value: string) {
    if (action === 'open') { await reader.open(value); return; }
    if (action === 'more') { query.set(s => ({ limit: s.limit + 50 })); return; }
    if (action === 'refresh') { await load(true); return; }
    query.set({ [action]: value, limit: 50 });
    await load();
  }
  return { use: query.use, peek: query.peek, load, choose };
}
export const commsState = createCommsState(commsAPI);
