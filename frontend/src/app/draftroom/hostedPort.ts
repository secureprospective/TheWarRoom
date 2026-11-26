import type { NativeRequest } from '../commands/native';
import type { DraftRoomPort } from './state';

type HostedDraftTransport = (request: NativeRequest) => Promise<{ data: unknown }>;

export function hostedDraftPort(port: HostedDraftTransport): DraftRoomPort {
  return {
    read: async scope => (await port({ command: 'draft.read', mode: 'read', arguments: scope })).data,
    pick: async choice => (await port({ command: 'draft.pick', mode: 'execute', arguments: choice })).data,
  };
}
