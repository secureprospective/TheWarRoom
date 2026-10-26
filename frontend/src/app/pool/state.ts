import { createStore } from 'zustand/vanilla';
import { useStore } from 'zustand';
import { commands } from '../commands/registry';
import {
  planOpen,
  type BidFormState, type BoardReading, type FormField, type Plan, type PoolFilter, type PoolReading,
  type PoolScope, type Preview,
} from './contract';
import { poolProvider, type PoolProvider } from './provider';

export type PoolState = {
  scope: PoolScope;
  reading?: PoolReading;
  board?: BoardReading;
  refreshing: boolean;
  error: string;
  notice: string;
  busy: string;
  form?: BidFormState;
  preview?: Preview;
  // The plan whose card is open above the table, and the last request to bring it into view.
  openPlan: string;
  focus: { id: string; seq: number };
};

// The window-close box shows local time; the wire carries UTC.
export function localInput(iso: string): string {
  const at = new Date(iso);
  if (!iso || Number.isNaN(at.getTime())) return '';
  const pad = (n: number) => String(n).padStart(2, '0');
  return `${at.getFullYear()}-${pad(at.getMonth() + 1)}-${pad(at.getDate())}` +
    `T${pad(at.getHours())}:${pad(at.getMinutes())}`;
}

async function copyText(value: string): Promise<boolean> {
  try {
    await navigator.clipboard.writeText(value);
    return true;
  } catch {
    const box = document.createElement('textarea');
    box.value = value;
    document.body.appendChild(box);
    box.select();
    const done = document.execCommand('copy');
    box.remove();
    return done;
  }
}

type Deps = {
  provider: PoolProvider;
  franchise: () => string;
  copy: (value: string) => Promise<boolean>;
  delay: number;
};
const blank = {
  conditionalDrop: '', conditionalOpen: false, years: 1, snipe: false, beat: '', evidence: '', salary: '',
  closesOpen: false, closesDate: '', closesTime: '',
};

export function createPoolState(supplied: Partial<Deps> = {}) {
  const deps: Deps = {
    provider: poolProvider, franchise: () => commands.read().franchiseId ?? '', copy: copyText,
    delay: 200, ...supplied,
  };
  const store = createStore<PoolState>(() => ({
    scope: { position: '', sort: 'engine', desc: true, showNoTeam: false },
    refreshing: false, error: '', notice: '', busy: '',
    openPlan: '', focus: { id: '', seq: 0 },
  }));
  let poolToken = 0;
  let previewToken = 0;
  let timer: ReturnType<typeof setTimeout> | undefined;
  const set = store.setState;
  const fail = (cause: unknown) => set({ error: String(cause), busy: '' });

  async function readPool() {
    const token = ++poolToken;
    try {
      const reading = await deps.provider.pool(deps.franchise(), store.getState().scope);
      if (token === poolToken) set({ reading });
    } catch (cause) {
      if (token === poolToken) fail(cause);
    }
  }
  async function readBoard() {
    const franchiseId = deps.franchise();
    if (!franchiseId) return;
    try {
      const board = await deps.provider.plans(franchiseId);
      if (deps.franchise() === franchiseId) set({ board });
    } catch (cause) {
      fail(cause);
    }
  }
  async function refresh(force: boolean) {
    set({ refreshing: true });
    try {
      await deps.provider.refresh(force);
    } catch (cause) {
      fail(cause);
    }
    set({ refreshing: false });
    await readPool();
  }
  // Opening the pool shows what is held at once, then asks MFL (skipped when the copy is fresh).
  async function open() {
    set({ error: '' });
    await Promise.all([readPool(), readBoard()]);
    await refresh(false);
  }
  async function reload() {
    await Promise.all([readPool(), readBoard()]);
  }

  async function runPreview() {
    const { form } = store.getState();
    const franchiseId = deps.franchise();
    if (!form || !franchiseId) return;
    const token = ++previewToken;
    try {
      const preview = await deps.provider.preview(franchiseId, form);
      // The previous answer stays on screen until this one lands, so typing never blanks it.
      if (token === previewToken && store.getState().form === form) set({ preview });
    } catch (cause) {
      if (token === previewToken) fail(cause);
    }
  }
  function schedulePreview() {
    clearTimeout(timer);
    timer = setTimeout(() => void runPreview(), deps.delay);
  }
  async function openForm(form: BidFormState) {
    if (!deps.franchise()) {
      set({ error: 'Choose my franchise first; a bid plan belongs to a franchise.' });
      return;
    }
    set({ form, preview: undefined, error: '', notice: '' });
    await runPreview();
  }
  function plan(id: string): Plan {
    const found = store.getState().board?.value.plans.find((p) => p.id === id);
    if (!found) throw new Error('That bid plan is not in the list; reload My moves.');
    return found;
  }
  // Opens a plan's card above the table and asks the screen to scroll it into view.
  function show(id: string) {
    set({ openPlan: id, focus: { id, seq: store.getState().focus.seq + 1 } });
  }
  async function act(id: string, run: () => Promise<string>) {
    set({ busy: id, error: '' });
    try {
      set({ notice: await run(), busy: '' });
    } catch (cause) {
      fail(cause);
    }
    await reload();
  }
  async function save() {
    const { form } = store.getState();
    const franchiseId = deps.franchise();
    if (!form) return;
    await act('save', async () => {
      const saved = form.planId
        ? await deps.provider.revise(form.planId, franchiseId, form)
        : await deps.provider.draft(franchiseId, form);
      set({ form: undefined, preview: undefined });
      set({ openPlan: saved.id });
      return `${saved.player}: ${saved.state}. ${saved.status}`;
    });
  }
  async function handOff(id: string) {
    const current = plan(id);
    await act(id, async () => {
      const copied = await deps.copy(current.post);
      const done = await deps.provider.handOff(id);
      return copied
        ? `Post copied for ProBoards. Nothing was posted for you. ${done.status}`
        : 'Copying failed here: select the post text above and copy it yourself. ' +
          'The plan is marked handed off; nothing was posted for you.';
    });
  }

  function run(filter: PoolFilter): Promise<void> {
    const { scope } = store.getState();
    switch (filter.field) {
      case 'pool-no-team':
        set({ scope: { ...scope, showNoTeam: !scope.showNoTeam } });
        return readPool();
      case 'pool-position':
        set({ scope: { ...scope, position: filter.value } });
        return readPool();
      case 'pool-sort': {
        const same = scope.sort === filter.value;
        const sort = filter.value as PoolScope['sort'];
        const desc = same ? !scope.desc : ['engine', 'ppg', 'lab'].includes(sort);
        set({ scope: { ...scope, sort, desc } });
        return readPool();
      }
      case 'pool-refresh':
        return refresh(true);
      case 'pool-plan': {
        const row = store.getState().reading?.value.rows.find((r) => r.id === filter.value);
        // One plan per player: a player who has an open plan gets that plan, not a second form.
        const held = store.getState().board?.value.plans.find(
          (p) => p.playerId === filter.value && planOpen(p.stateCode),
        )?.id ?? row?.planId;
        if (held) {
          show(held);
          return Promise.resolve();
        }
        return openForm({ ...blank, planId: '', playerId: filter.value,
          playerName: row?.name ?? filter.value });
      }
      case 'pool-revise': {
        const p = plan(filter.value);
        const closes = localInput(p.closes).split('T');
        return openForm({
          conditionalDrop: p.conditionalDrop, conditionalOpen: false,
          planId: p.id, playerId: p.playerId, playerName: p.player, salary: p.form.salary, years: p.years,
          snipe: p.snipe, beat: p.form.beat, evidence: p.evidence, closesOpen: Boolean(p.closes),
          closesDate: closes[0], closesTime: closes[1] ?? '',
        });
      }
      case 'pool-open':
        show(filter.value);
        return Promise.resolve();
      case 'pool-toggle':
        if (store.getState().openPlan === filter.value) set({ openPlan: '' });
        else show(filter.value);
        return Promise.resolve();
      case 'pool-conditional-change':
      case 'pool-conditional-clear': {
        const { form } = store.getState();
        if (!form) return Promise.resolve();
        const clear = filter.field === 'pool-conditional-clear';
        set({ form: { ...form, conditionalOpen: !clear, ...(clear && { conditionalDrop: '' }) } });
        if (clear) schedulePreview();
        return Promise.resolve();
      }
      case 'pool-closes-set':
      case 'pool-closes-clear': {
        const { form } = store.getState();
        if (!form) return Promise.resolve();
        const clear = filter.field === 'pool-closes-clear';
        set({ form: { ...form, closesOpen: !clear, ...(clear && { closesDate: '', closesTime: '' }) } });
        if (clear) schedulePreview();
        return Promise.resolve();
      }
      case 'pool-cancel':
        set({ form: undefined, preview: undefined });
        return Promise.resolve();
      case 'pool-save':
        return save();
      case 'pool-handoff':
        return handOff(filter.value);
      case 'pool-posted':
        return act(filter.value, async () => (await deps.provider.posted(filter.value)).status);
      case 'pool-withdraw':
      case 'pool-unposted':
        return act(filter.value, async () => { await deps.provider.withdraw(filter.value); return ''; });
      case 'pool-copy':
        return act(filter.value, async () =>
          (await deps.copy(plan(filter.value).post)) ? 'Post copied again.' : 'Copying failed here.');
    }
  }
  // Every filter lands here; a failure becomes the screen's error, never an unhandled rejection.
  async function change(filter: PoolFilter) {
    set({ error: '' });
    try {
      await run(filter);
    } catch (cause) {
      fail(cause);
    }
  }
  function edit(field: FormField, value: string | number | boolean) {
    const { form } = store.getState();
    if (!form) return;
    set({ form: { ...form, [field]: value,
      ...(field === 'conditionalDrop' && { conditionalOpen: false }) } });
    schedulePreview();
  }
  return {
    use: () => useStore(store),
    peek: store.getState,
    open, reload, change, edit, flush: runPreview, loadBoard: readBoard,
  };
}
export const poolState = createPoolState();
