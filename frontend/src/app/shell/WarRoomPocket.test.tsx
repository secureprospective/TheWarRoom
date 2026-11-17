// @vitest-environment jsdom
import { act, useState } from 'react';
import { createRoot, type Root } from 'react-dom/client';
import { afterEach, beforeEach, expect, it, vi } from 'vitest';
import type { HostedPort } from '../commissioner/hostedRoom';
import type { HostedResponse } from '../control/nativeApi';
import type { NativeRequest } from '../commands/native';
import { heldSnapshot } from '../data/lineupTestData';
import { WarRoomPocketSave } from './WarRoomPocketSave';
import { WarRoomDrawers } from './WarRoomDrawers';
import { WarRoomTrade } from './WarRoomTrade';
import type { WarRoomContext } from './warroomReads';
import type { WarRoomDeal, WarRoomCatalog } from './warroomTradeReads';
import type { WarRoomSavedDraft } from './warroomDrafts';

vi.mock('./warroomTradeReads', async importOriginal => {
  const original = await importOriginal<typeof import('./warroomTradeReads')>();
  return { ...original, readWarRoomPartner: vi.fn(async () => undefined),
    readWarRoomDelta: vi.fn(async () => undefined) };
});

const saved: WarRoomSavedDraft = { id: 'a'.repeat(32), partner: 'franchise.02', give: ['0999'], get: [],
  label: 'Old deal', at: '2026-11-17T12:00:00Z' };
const emptyCatalog: WarRoomCatalog = { revision: 4, teams: [
  { id: 'franchise.01', name: 'Mine', assets: [] },
  { id: 'franchise.02', name: 'Other', assets: [] },
] };
let host: HTMLDivElement;
let root: Root;
beforeEach(() => {
  Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true });
  host = document.createElement('div'); document.body.append(host); root = createRoot(host);
});
afterEach(async () => {
  await act(async () => root.unmount()); host.remove(); vi.restoreAllMocks();
});
const settle = () => act(async () => { await new Promise(resolve => setTimeout(resolve, 50)); });
const buttons = (label: string) => [...host.querySelectorAll('button')]
  .filter(button => button.textContent?.trim() === label || button.getAttribute('aria-label') === label);
async function click(label: string) {
  expect(buttons(label), label).toHaveLength(1);
  await act(async () => { buttons(label)[0].focus(); buttons(label)[0].click(); });
  await settle();
}
function context(port: HostedPort): WarRoomContext {
  return { snapshot: heldSnapshot(), mine: 'franchise.01', week: 1, native: {
    league: 'native.test', mine: 'franchise.01', owner: true, port, leave: () => {},
  } };
}
function reply(data: unknown): HostedResponse {
  return { data, label: 'Server', writable: true, football: 'pin' };
}
function fakePort(initial: WarRoomSavedDraft[] = []) {
  const rows = [...initial];
  const calls: NativeRequest[] = [];
  const port: HostedPort = vi.fn(async request => {
    calls.push(request);
    switch (request.command) {
      case 'trade.draft.read': return reply({ drafts: [...rows] });
      case 'trade.draft.save': {
        const args = request.arguments as { partner: string; give: string[]; get: string[]; label: string };
        const proposal = { ...saved, id: 'b'.repeat(32), partner: args.partner, give: args.give,
          get: args.get, label: args.label, at: '2026-11-17T12:01:00Z' };
        rows.push(proposal); return reply(proposal);
      }
      case 'trade.draft.delete': {
        const id = (request.arguments as { id: string }).id;
        const index = rows.findIndex(row => row.id === id);
        if (index >= 0) rows.splice(index, 1);
        return reply({ id });
      }
      case 'trade.offer': return reply({ question: 'Preview',
        target: { league: 'native.test', name: 'Test League', version: 4 }, fields: [],
        affects: { readings: [], count: 0, words: '' },
        blockers: ['The saved player is no longer available'], safeguard: '' });
      default: throw new Error(`Unexpected workflow ${request.command}`);
    }
  });
  return { port, calls };
}

it('enables hosted SAVE TO POCKET for a complete proposal and leaves local mode disabled', async () => {
  const service = fakePort();
  const saveContext = context(service.port);
  const opened = vi.fn();
  await act(async () => root.render(<WarRoomPocketSave context={saveContext}
    deal={{ partner: 'franchise.02', give: ['0101'], get: [] }} onSaved={opened} />));
  const saveButton = buttons('SAVE TO POCKET')[0];
  expect(saveButton.disabled).toBe(false);
  const label = host.querySelector('input[aria-label="Saved proposal label"]') as HTMLInputElement;
  await act(async () => {
    Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, 'value')!.set!.call(label, 'My offer');
    label.dispatchEvent(new Event('input', { bubbles: true }));
  });
  await click('SAVE TO POCKET');
  expect(service.calls.map(call => call.command)).toEqual(['trade.draft.save']);
  expect((service.calls[0].arguments as { label: string }).label).toBe('My offer');
  expect(opened).toHaveBeenCalledOnce();

  await act(async () => root.render(<WarRoomPocketSave context={{ ...saveContext, native: undefined }}
    deal={{ partner: 'franchise.02', give: ['0101'], get: [] }} onSaved={opened} />));
  expect(buttons('SAVE TO POCKET')[0].disabled).toBe(true);
  expect(service.calls).toHaveLength(1);
});

function PocketDesk({ context: room }: { context: WarRoomContext }) {
  const [deal, setDeal] = useState<WarRoomDeal>({ partner: '', give: [], get: [] });
  const [reopen, setReopen] = useState(false);
  const [nonce, setNonce] = useState(0);
  return <>
    {reopen && <WarRoomTrade context={room} catalog={emptyCatalog} deal={deal} change={setDeal}
      select={() => {}} previewNonce={nonce} />}
    <WarRoomDrawers context={room} drawer="pocket" close={() => {}} openOffer={() => {}}
      onOpenDraft={draft => {
        setDeal({ partner: draft.partner, give: [...draft.give], get: [...draft.get] });
        setNonce(value => value + 1); setReopen(true);
      }} />
  </>;
}

it('reopens intact stale assets for a fresh preview and rereads the pocket after delete', async () => {
  const service = fakePort([saved]);
  const room = context(service.port);
  await act(async () => root.render(<PocketDesk context={room} />));
  await settle();
  expect(host.textContent).toContain('Old deal');
  await click('REOPEN ON DESK');
  await settle();
  expect(service.calls.some(call => call.command === 'trade.offer' && call.mode === 'preview')).toBe(true);
  const preview = service.calls.find(call => call.command === 'trade.offer');
  const args = preview?.arguments as { args: { trade: { assets: { player?: string }[] } } };
  expect(args.args.trade.assets).toEqual([{ from: 'franchise.01', to: 'franchise.02', player: '0999' }]);
  expect(host.textContent).toContain('No longer available');
  expect(host.textContent).toContain('The saved player is no longer available');

  await click('DELETE');
  expect(service.calls.filter(call => call.command === 'trade.draft.read')).toHaveLength(2);
  expect(service.calls.at(-2)?.command).toBe('trade.draft.delete');
  expect(service.calls.at(-1)?.command).toBe('trade.draft.read');
  expect(host.textContent).toContain('No saved proposals yet');
});
