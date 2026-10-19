import { vi } from 'vitest';
import { flush, mount, render, settle, type App, type Element } from '../commissioner/commissionerDom';
import { commissionerPeople, ownerPeople, receipt, stub } from '../commissioner/commissionerStub';
import { leagueState } from '../commissioner/proxy';
import { active, modelFocusApis } from '../commissioner/trayHarness';
import DraftRoom from './DraftRoom';
import { draftRoom } from './state';
import wire from './draft.wire.testdata.json';

// The Go bindings as the draft room sees it: the reading and the previews are the Go-written fixtures, and
// every change the room sends is kept so a test can read exactly what Go would have been given.
export type Sent = { command: string; league: string; request: string; reason: string; expected: number;
  effective: string; visibility: string; args: { draft: { season: number; act?: Record<string, unknown> } } };
export type Choice = { league: string; season: number; franchise: string; player: string; request: string;
  expected: number };

function previewFor(sent: Sent) {
  const act = sent.args.draft.act;
  const key = sent.command === 'draft.timer' ? (act?.timer === 'set' ? 'set' : 'pause')
    : { 'draft.revert': 'restore', 'draft.archive': 'close' }[sent.command] ?? sent.command.slice(6);
  const source = wire.previews[key as keyof typeof wire.previews];
  return { ...source, target: { ...source.target, league: sent.league, version: sent.expected } };
}

export const people = { commissioner: commissionerPeople, owner: ownerPeople };
export async function openRoom(over: {
  reading?: unknown | (() => unknown); people?: unknown; pick?: (choice: Choice) => Promise<unknown>;
  execute?: (body: string) => Promise<unknown>; leagues?: unknown;
  // Fake timers cannot wait on a timer, so the room is settled on promises alone.
  fakeTimers?: boolean;
} = {}) {
  leagueState.forget();
  draftRoom.reset();
  // The tray and the verb that opens it are loaded on demand; a cold load must not outlast a test's wait.
  await Promise.all([import('./DraftTray'), import('../commissioner/surfaceActions')]);
  const restore = modelFocusApis();
  const sent: Sent[] = [];
  const picks: Choice[] = [];
  const reads: { league: string; season: number; franchise: string; at: string }[] = [];
  const { app, calls } = stub({
    TargetCommissionerPeople: async () => over.people ?? commissionerPeople,
    TargetCommissionerPreview: async body => previewFor(JSON.parse(body) as Sent),
    TargetCommissionerExecute: over.execute ??
      (async body => { sent.push(JSON.parse(body)); return receipt(); }),
    ...(over.leagues ? { TargetCommissionerLeagues: async () => over.leagues } : {}),
  });
  const draftApp = {
    ...app,
    TargetNativeDraft: async (scope: object) => {
      const asked = scope as (typeof reads)[number];
      reads.push(asked);
      const reading = typeof over.reading === 'function' ? (over.reading as () => unknown)()
        : over.reading ?? wire.running;
      return { ...(reading as object), league: asked.league };
    },
    TargetNativeDraftPick: async (choice: Choice) => {
      picks.push(choice);
      return over.pick ? over.pick(choice) : receipt({ message: 'Your pick is recorded.' });
    },
  } as unknown as Partial<App>;
  const host: Element = await mount(render(DraftRoom), draftApp);
  active(host);
  await (over.fakeTimers ? flush() : settle());
  return { host, sent, picks, reads, calls, restore };
}
export const at = (iso: string) => vi.spyOn(Date, 'now').mockReturnValue(Date.parse(iso));
export { wire };
