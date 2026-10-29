import { restoreWorkAnchor, type WorkAnchor } from '../shell/workScroll';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { createElement, act } from 'react';
import { createRoot } from 'react-dom/client';
import { renderToStaticMarkup } from 'react-dom/server';
import { readFileSync } from 'node:fs';
import { localDate } from '../clock/labels';
import { clockDom, TestElement } from '../clock/testDom';
import { heldSnapshot } from '../data/lineupTestData';
import { commands } from '../commands/registry';
import Contracts, { ContractReading, ContractPanel } from './Contracts';
import { ContractActCard, ContractMovePanel } from './ContractActs';
import { WhatIfActs } from './WhatIfPanel';
import { reconcileContractMove, termsKey, useContractMove, type ContractActPlan } from './contractActState';
import { parseContracts, parsePlayerContract } from './provider';
import { setContractsFilter } from './state';
import type { WhatIf, WhatIfProvider } from './whatif';
import type { Reading, Readings, ContractsProvider, PlayerContract, Scope } from './contract';

function reading<T>(value: T): Reading<T> {
  return {
    value,
    status: 'ok',
    absenceReason: null,
    count: 1,
    threshold: 0,
    baseline: 'Mirror revision',
    cutoff: '2026-10-10T00:00:00Z',
    method: 'Integer cents',
    inputSHA: '',
    buildStamp: 'dev ring2',
    howCounted: 'Go computed; flat-filled years are inferred.',
    provenance: {
      source: 'MFL mirror',
      kind: 'live',
      freshness: { state: 'live', fetchedAt: '2026-10-10T00:00:00Z', note: 'Held mirror' },
    },
  };
}
const player: PlayerContract = {
  player: {
    id: '0042',
    franchiseId: '0001',
    team: 'Mine',
    name: 'Example player',
    position: 'QB',
    salary: { cents: 150000000, text: '$1,500,000' },
    expiration: 2027,
    status: 'RFA',
    info: 'MFL original info',
  },
  years: [
    {
      season: 2026,
      cost: { cents: 150000000, text: '$1,500,000' },
      inferred: false,
      tag: 'Observed',
      reason: 'MFL annual salary',
    },
    {
      season: 2027,
      cost: { cents: 150000000, text: '$1,500,000' },
      inferred: true,
      tag: 'Inferred',
      reason: 'Flat-filled; not observed',
    },
  ],
};
function readings(): Readings {
  return {
    table: reading({
      directorySource: 'MFL players archive',
      directoryAsOf: '2026-10-05T00:00:00Z',
      rows: [player.player],
      teams: [
        { id: '0001', name: 'Mine' },
        { id: '0002', name: 'Other' },
      ],
      positions: ['QB', 'WR'],
    }),
    cap: reading({
      team: 'Mine',
      season: 2026,
      cap: { cents: 12500000000, text: '$125,000,000' },
      used: { cents: 12345, text: '$123.45' },
      room: { cents: 12499987655, text: '$124,999,876.55' },
      capSource: 'Active MFL config; as-of not held',
      future: [player.years[1]],
    }),
    adjustments: reading({
      rows: [
        {
          id: '7',
          franchiseId: '0001',
          team: 'Mine',
          amount: { cents: -123, text: '-$1.23' },
          description: 'MFL literal credit',
        },
      ],
    }),

  };
}
function provider(): ContractsProvider {
  return {
    contracts: vi.fn().mockResolvedValue(readings()),
    player: vi.fn().mockResolvedValue(reading(player)),
    onChange: () => () => {},
  };
}
const globals = ['window', 'document', 'IS_REACT_ACT_ENVIRONMENT'].map((key) => ({
  key,
  descriptor: Object.getOwnPropertyDescriptor(globalThis, key),
}));
const mounted: ReturnType<typeof createRoot>[] = [];
afterEach(async () => {
  await act(async () => {
    for (const root of mounted.splice(0)) root.unmount();
  });
  vi.restoreAllMocks();
  for (const { key, descriptor } of globals) {
    if (descriptor) Object.defineProperty(globalThis, key, descriptor);
    else Reflect.deleteProperty(globalThis, key);
  }
});
function nodes(root: TestElement, tag: string): TestElement[] {
  return root.children.flatMap((child): TestElement[] =>
    child instanceof TestElement
      ? [...(child.tagName === tag ? [child] : []), ...nodes(child, tag)]
      : [],
  );
}
function props(node: TestElement): Record<string, unknown> {
  const key = Object.keys(node).find((key) => key.startsWith('__reactProps'))!;
  return (node as unknown as Record<string, Record<string, unknown>>)[key];
}
async function mount(view: ReturnType<typeof createElement>, host: Record<string, unknown> = {}) {
  const dom = clockDom([]);
  const create = dom.document.createElement.bind(dom.document);
  dom.document.createElement = (tag) => {
    const node = create(tag);
    Object.assign(node, { closest: () => null });
    if (tag === 'select')
      Object.defineProperty(node, 'options', { get: () => nodes(node, 'OPTION') });
    return node;
  };
  vi.stubGlobal('IS_REACT_ACT_ENVIRONMENT', true);
  vi.stubGlobal('document', dom.document);
  vi.stubGlobal('window', { document: dom.document, HTMLIFrameElement: class {}, ...host });
  const root = createRoot(dom.element);
  mounted.push(root);
  await act(async () => {
    root.render(view);
  });
  return { ...dom, renderer: root };
}
function shell() {
  setContractsFilter({ field: 'team', value: '' });
  setContractsFilter({ field: 'position', value: '' });
  setContractsFilter({ field: 'sort', value: 'name' });
  setContractsFilter({ field: 'desc', value: false });
  vi.spyOn(commands, 'use').mockReturnValue({ ...commands.read(), franchiseId: '0001' });
}

describe('Contracts and cap read surfaces', () => {
  it('renders loading, failed, not-wired and successful states without invented figures', () => {
    const html = (r?: Reading<unknown>, error?: string) =>
      renderToStaticMarkup(
        createElement(ContractReading, { title: 'Test', reading: r, error }, 'FIGURE'),
      );
    expect(html()).toContain('No figures yet');
    expect(html()).not.toContain('FIGURE');
    expect(html(undefined, 'offline')).toContain('Failed');
    const failed = {
      ...reading({}),
      status: 'failed' as const,
      absenceReason: 'Mirror unavailable',
    };
    expect(html(failed)).not.toContain('FIGURE');
    expect(html(reading({}))).toContain('FIGURE');
    expect(html(reading({}))).toContain('Source: MFL mirror');
    expect(html(reading({}))).toContain(localDate('2026-10-10T00:00:00Z'));
    expect(html(reading({}))).not.toContain('2026-10-10T00:00:00Z');
  });
  it('shows Go-authored cap, inferred commitments, literal adjustments and honest M-025', async () => {
    shell();
    const test = await mount(
      createElement(Contracts, { snapshot: heldSnapshot(), provider: provider() }),
    );
    const text = test.element.textContent!;
    for (const value of [
      '$125,000,000',
      '$123.45',
      '$124,999,876.55',
      'Flat-filled',
      '-$1.23',
      'MFL literal credit',
      'dead cap from drops, retirements and commissioner changes',
      'What-if models the charge; Make this a plan keeps a checked move',
    ])
      expect(text).toContain(value);
    expect(text).not.toContain("Not wired");
    expect(text).not.toContain("Committed by future season");
    expect(text).not.toContain("Inferred");
    expect(text).not.toContain("configuration as-of");
    const endpoints = test.root
      .querySelectorAll('[data-endpoint]')
      .map((node) => node.getAttribute('data-endpoint'));
    expect(endpoints).toEqual(['C-06.1', 'M-078', 'M-031']);
    expect(text).toContain('MFL players archive');
    expect(text).toContain(localDate('2026-10-05T00:00:00Z'));
    expect(text).not.toContain('2026-10-05T00:00:00Z');
    expect(text).toContain('My team first.');
    expect(text).not.toContain('Sorting and filters re-query Go');
    expect(text).not.toContain('React does not calculate');
  });
  it('covers empty table, adjustments, commitments and missing cap configuration', async () => {
    shell();
    const value = readings();
    value.table.value!.rows = [];
    value.adjustments.value!.rows = [];
    value.cap.value!.future = [];
    value.cap.value!.cap = null;
    value.cap.value!.room = null;
    const p = provider();
    p.contracts = vi.fn().mockResolvedValue(value);
    const test = await mount(createElement(Contracts, { snapshot: heldSnapshot(), provider: p }));
    for (const text of [
      'No contracts match',
      'No MFL salary adjustments',
      'No future commitments',
      'Cap configuration unavailable',
      'Unavailable',
    ])
      expect(test.element.textContent).toContain(text);
    expect(test.element.textContent).not.toContain('$125,000,000');
  });
  it('re-queries Go on team, position, sort and direction; does not sort or total in React', async () => {
    shell();
    const p = provider();
    const test = await mount(createElement(Contracts, { snapshot: heldSnapshot(), provider: p }));
    const selects = nodes(test.root, 'SELECT');
    const change = async (index: number, value: string) => {
      await act(async () => {
        (props(selects[index]).onChange as (e: { target: { value: string } }) => void)({
          target: { value },
        });
      });
    };
    await change(0, '0002');
    await change(1, 'WR');
    await change(2, 'salary');
    const button = nodes(test.root, 'BUTTON').find((n) => n.textContent === 'Ascending')!;
    await act(async () => {
      (props(button).onClick as () => void)();
    });
    expect(p.contracts).toHaveBeenLastCalledWith({
      mine: '0001',
      team: '0002',
      position: 'WR',
      sort: 'salary',
      desc: true,
    });
    expect(p.contracts).toHaveBeenCalledTimes(5);
    const source = readFileSync('src/app/contracts/Contracts.tsx', 'utf8');
    expect(source).not.toMatch(/\.sort\(|\.reduce\(|\.cents\s*[+*/-]/);
  });
  it('reads the player contract in the Inspector, observed/inferred reasons and failure', async () => {
    const p = provider();
    const test = await mount(createElement(ContractPanel, { id: '0042', provider: p }));
    expect(p.player).toHaveBeenCalledWith('0042');
    expect(test.element.textContent).toContain('MFL annual salary');
    expect(test.element.textContent).toContain('Flat-filled; not observed');
    expect(test.element.textContent).toContain('MFL original info');
    expect(test.root.querySelector('[data-endpoint]')?.getAttribute('data-endpoint')).toBe(
      'C-06.5',
    );
    p.player = vi.fn().mockRejectedValue(new Error('not rostered'));
    await act(async () => {
      test.renderer.render(createElement(ContractPanel, { id: '9999', provider: p }));
    });
    expect(test.element.textContent).toContain('not rostered');
    expect(test.element.textContent).not.toContain('$1,500,000');
  });
  it('hides stale readings while a query changes and ignores out-of-order responses', async () => {
    shell();
    let resolve!: (v: Readings) => void;
    const p = provider();
    p.contracts = vi.fn((scope: Scope) =>
      scope.team === '0002'
        ? new Promise<Readings>((yes) => {
            resolve = yes;
          })
        : Promise.resolve(readings()),
    );
    const test = await mount(createElement(Contracts, { snapshot: heldSnapshot(), provider: p }));
    const select = nodes(test.root, 'SELECT')[0];
    await act(async () => {
      (props(select).onChange as (e: { target: { value: string } }) => void)({
        target: { value: '0002' },
      });
    });
    expect(test.element.textContent).toContain('No figures yet');
    expect(test.element.textContent).not.toContain('$125,000,000');
    await act(async () => {
      (props(select).onChange as (e: { target: { value: string } }) => void)({
        target: { value: '' },
      });
    });
    const stale = readings();
    stale.cap.value!.team = 'STALE TEAM';
    await act(async () => {
      resolve(stale);
    });
    expect(test.element.textContent).not.toContain('STALE TEAM');
  });
});

it('rejects fractional or unsafe money at the wire boundary and preserves signed cents', () => {
  expect(parseContracts(readings()).adjustments.value!.rows[0].amount.cents).toBe(-123);
  for (const cents of [1.5, Number.MAX_SAFE_INTEGER + 1]) {
    const r = readings();
    r.table.value!.rows[0] = { ...player.player, salary: { cents, text: 'invalid' } };
    expect(() => parseContracts(r)).toThrow(/safe integer/);
  }
  expect(parsePlayerContract(reading(player)).value!.years[1].inferred).toBe(true);
  expect(() => parsePlayerContract({ ...reading(player), extra: true })).toThrow(
    /unexpected field/,
  );
});

it('ships narrow/wide container layouts and a slim closed Inspector rail without raising the budget', () => {
  const css = readFileSync('src/app/contracts/contracts.css', 'utf8');
  expect(css).toContain('container-type: inline-size');
  expect(css).toContain('@container (max-width: 760px)');
  expect(css).toContain('white-space: nowrap');
  expect(css).not.toMatch(/overflow-wrap:\s*anywhere|word-break:\s*break-all/);
  const shell = readFileSync('src/app/shell/shell.css', 'utf8');
  expect(shell).toMatch(
    /target-frame\[data-inspector="closed"\]\s*\{\s*grid-template-columns: 192px minmax\(0, 1fr\) 40px 48px/,
  );
  expect(shell).toContain('grid-template-columns: 192px minmax(0, 1fr) 320px 48px');
  // At 1280px, closing frees 280px of workspace, rather than leaving a 320px strip.
  expect(1280 - 192 - 40 - 48).toBe(1000);
  const app = readFileSync('src/app/shell/TargetApp.tsx', 'utf8');
  expect(app).toContain("lazy(() => import('../contracts/Contracts'))");
  expect(app).toContain('label="Open inspector"');
  expect(readFileSync('scripts/bundle-budget.mjs', 'utf8')).toContain('const BUDGET = 321_818;');
});

it('routes built endpoints and preserves Inspector close/reopen state', async () => {
  const { createCommands } = await import('../commands/registry');
  const { endpointById } = await import('../registry');
  const c = createCommands();
  for (const id of ['M-078', 'M-031', 'C-06.5'] as const) {
    c.dispatch('endpoint.open', { id });
    expect(c.read().node).toBe('hq');
    expect(c.read().workspace.hq).toBe('contracts-and-cap');
    expect(endpointById(id).status).not.toBe('spec');
  }
  c.dispatch('inspector.close', {});
  expect(c.read().inspector).toBe('closed');
  c.dispatch('inspector.toggle', {});
  expect(c.read().inspector).toBe('rest');
});

it('covers each failed surface and refreshes on held-feed change', async () => {
  shell();
  const p = provider();
  let resolve!: (v: Readings) => void;
  let changed!: () => void;
  const unsubscribe = vi.fn();
  p.contracts = vi.fn(
    () =>
      new Promise<Readings>((yes) => {
        resolve = yes;
      }),
  );
  p.onChange = (listener) => {
    changed = listener;
    return unsubscribe;
  };
  const test = await mount(createElement(Contracts, { snapshot: heldSnapshot(), provider: p }));
  expect(test.element.textContent?.match(/No figures yet/g)).toHaveLength(3);
  const value = readings();
  for (const key of ['table', 'cap', 'adjustments'] as const) {
    value[key].status = 'failed';
    value[key].absenceReason = `${key} unavailable`;
  }
  await act(async () => {
    resolve(value);
  });
  for (const key of ['table', 'cap', 'adjustments']) {
    expect(test.element.textContent).toContain(`${key} unavailable`);
  }
  expect(test.element.textContent).not.toContain('$125,000,000');
  await act(async () => {
    changed();
  });
  expect(p.contracts).toHaveBeenCalledTimes(2);
  expect(test.element.textContent?.match(/No figures yet/g)).toHaveLength(3);
  await act(async () => {
    test.renderer.unmount();
  });
  expect(unsubscribe).toHaveBeenCalledTimes(1);
});

 it('shows each year reason once per section, with only Go-authored tags in rows', async () => {
  shell();
  const value = readings();
  value.cap.value!.future.push({ ...player.years[1], season: 2028 });
  const p = provider();
  p.contracts = vi.fn().mockResolvedValue(value);
  const cap = await mount(createElement(Contracts, { snapshot: heldSnapshot(), provider: p }));
  const years = nodes(cap.root, 'TABLE')[0];
  expect(nodes(years, 'TR')).toHaveLength(3);
  expect(years.textContent).not.toContain('Flat-filled');
  expect(cap.element.textContent?.match(/Flat-filled; not observed/g)).toHaveLength(1);
  const contract = { ...player, years: [...player.years, { ...player.years[1], season: 2028 }] };
  p.player = vi.fn().mockResolvedValue(reading(contract));
  const panel = await mount(createElement(ContractPanel, { id: '0042', provider: p }));
  expect(panel.element.textContent?.match(/MFL annual salary/g)).toHaveLength(1);
  expect(panel.element.textContent?.match(/Flat-filled; not observed/g)).toHaveLength(1);
  expect(nodes(panel.root, 'TABLE')[0].textContent).not.toContain('annual salary');
  expect(panel.element.textContent).toContain(localDate('2026-10-10T00:00:00Z'));
  expect(panel.element.textContent).not.toContain('Observed · Observed');
  expect(panel.element.textContent).not.toContain('Inferred · Inferred');
});

it('labels a viewed team as selected, not mine, and offers no what-if for it', async () => {
  shell();
  vi.spyOn(commands, 'useMFLKey').mockReturnValue({ ...commands.readMFLKey(), myFranchise: '0025' });
  const value = readings();
  value.cap.value!.team = 'Arizona Cardinals';
  const p = provider();
  p.contracts = vi.fn().mockResolvedValue(value);
  const simulation: WhatIfProvider = {
    query: vi.fn().mockRejectedValue(new Error('what-if plans are private')),
    change: async () => {}, onChange: () => () => {},
  };
  const test = await mount(createElement(Contracts, { snapshot: heldSnapshot(), provider: p,
    whatIf: simulation }));
  const text = test.element.textContent ?? '';
  expect(text).toContain('Team cap');
  expect(text).toContain('Selected team first.');
  expect(text).toContain('What-ifs and contract moves are for your own team.');
  expect(text).not.toContain('My franchise cap');
  expect(text).not.toContain('My team');
  expect(text).not.toContain('Failed');
  test.renderer.unmount();
  vi.restoreAllMocks();
});

it('keeps twelve one-line my-team rows and counts filtered groups without repeating team cells', async () => {
  shell();
  const value = readings();
  value.cap.value!.team = 'Arizona Cardinals';
  value.table.value!.rows = Array.from({ length: 12 }, (_, index) => ({
    ...player.player, id: String(index + 1000), team: 'Arizona Cardinals', name: `Player ${index}`,
  }));
  value.table.value!.rows.push({ ...player.player, id: '0099', franchiseId: '0002', team: 'Other' });
  const p = provider();
  p.contracts = vi.fn().mockResolvedValue(value);
  const simulation: WhatIfProvider = {
    query: vi.fn().mockResolvedValue(reading<WhatIf>({ stale: false, banner: '',
      empty: 'No moves yet.', notice: '', moves: [], figures: [],
      choices: value.table.value!.rows.map(row => ({ player: row.id, cutLabel: 'Cut him',
        buyoutLabel: 'Buy him out', buyoutAllowed: false, reason: 'Needs two years' })) })),
    change: async () => {}, onChange: () => () => {},
  };
  const test = await mount(createElement(Contracts, { snapshot: heldSnapshot(), provider: p,
    whatIf: simulation }));
  expect(test.element.textContent).toContain('My team · Arizona Cardinals (12)');
  expect(test.element.textContent).toContain('Rest of the league (1)');
  const tables = nodes(test.root, 'TABLE');
  const mine = tables[1];
  const rest = tables[2];
  expect(nodes(mine, 'TH').map((n) => n.textContent))
    .toEqual(['Player', 'Annual salary', 'Expires / status', 'What-if']);
  const rows = nodes(mine, 'TR').slice(1);
  expect(rows).toHaveLength(12);
  for (const row of rows) {
    expect(nodes(row, 'TD')).toHaveLength(4);
    expect(row.textContent).not.toContain('Arizona Cardinals');
    expect(nodes(row, 'SMALL')[0].textContent).toBe('QB');
    const menu = nodes(row, 'DETAILS')[0];
    expect(nodes(menu, 'SUMMARY')[0].textContent).toBe('What if…');
    expect(menu.getAttribute('open')).toBeNull();
    expect(nodes(menu, 'BUTTON')).toHaveLength(3);
    expect(nodes(menu, 'BUTTON')[2].textContent).toBe('Contract Acts…');
    expect(nodes(menu, 'P')[0].textContent).toBe('Needs two years');
    expect(nodes(row, 'TD')[0].textContent).not.toContain('Needs two years');
  }
  expect(nodes(rest, 'TH').map((n) => n.textContent)).toContain('Team');
  expect(nodes(rest, 'TD')).toHaveLength(4);
  const css = readFileSync('src/app/contracts/contracts.css', 'utf8');
  expect(css).not.toMatch(/display: block|content: attr|grid-row: 3/);
  expect(css).toContain('padding: 2px 4px');
  expect(css).toContain('line-height: 18px');
  expect(css).toContain('text-overflow: ellipsis');
  value.table.value!.rows = value.table.value!.rows.slice(-1);
  await act(async () => {
    test.renderer.render(createElement(Contracts, { snapshot: heldSnapshot(), provider: { ...p } }));
  });
  expect(test.element.textContent).toContain('My team · Arizona Cardinals (0)');
  expect(test.element.textContent).toContain('Rest of the league (1)');
});

it('retains the what-if during replay and restores the clicked row position before paint', async () => {
  shell();
  const value: WhatIf = { stale: false, banner: '', notice: '', empty: 'No moves yet.',
    figures: [], moves: [],
    choices: [{ player: '0042', cutLabel: 'Cut him', buyoutLabel: 'Buy him out',
      buyoutAllowed: false, reason: 'Needs two years' }] };
  let changed!: () => void;
  let resolve!: (value: Reading<WhatIf>) => void;
  const simulation: WhatIfProvider = { query: vi.fn().mockResolvedValue(reading(value)),
    change: async () => {}, onChange: listener => { changed = listener; return () => {}; } };
  const test = await mount(createElement(Contracts, { snapshot: heldSnapshot(),
    provider: provider(), whatIf: simulation }));
  const outer = nodes(test.root, 'DIV').find(node => node.getAttribute('class') === 'contracts')!;
  const row = { isConnected: true, getBoundingClientRect: vi.fn().mockReturnValue({ top: 400 }) };
  const scroll = { scrollTop: 100, clientHeight: 600, getBoundingClientRect: () => ({ top: 60 }) };
  const anchor = { row, top: 400, scroll };
  simulation.query = vi.fn(() => new Promise<Reading<WhatIf>>(yes => { resolve = yes; }));
  await act(async () => { changed(); });
  expect(test.element.textContent).toContain('No moves yet.');
  row.getBoundingClientRect.mockReturnValue({ top: 650 });
  const modelled = { ...value, moves: [{ id: '1', player: '0042', tag: 'Cut in what-if',
    status: 'Modelled: cut Example player, $700,000; see What-if at the top',
      label: 'Cut Example player', charge: { cents: 70000000, text: '$700,000' },
        explanation: 'Go rule' }], choices: [] };
  await act(async () => { resolve({ ...reading(modelled), baseline: '2026-10-11T07:27:22Z' }); });
  restoreWorkAnchor(anchor as unknown as WorkAnchor);
  expect(scroll.scrollTop).toBe(350);
  expect(test.element.textContent).toContain(modelled.moves[0].status);
  expect(test.element.textContent).toContain('Cut in what-if');
  const myRow = nodes(test.root, 'TR').find(node => node.textContent.includes('Example player'))!;
  expect(nodes(myRow, 'DETAILS')).toHaveLength(0);
});

it('offers the same Go Acts in my player Inspector, never another franchise', async () => {
  shell();
  const value: WhatIf = { stale: false, banner: '', notice: '', empty: '', figures: [], moves: [],
    choices: [{ player: '0042', cutLabel: 'What if I cut him',
      buyoutLabel: 'What if I buy him out', buyoutAllowed: false, reason: 'Needs two years' }] };
  const simulation: WhatIfProvider = { query: vi.fn().mockResolvedValue(reading(value)),
    change: async () => {}, onChange: () => () => {} };
  const test = await mount(createElement(ContractPanel, { id: '0042', provider: provider(),
    whatIf: simulation }));
  expect(test.element.textContent).toContain('What if I cut him');
  expect(test.element.textContent).toContain('What if I buy him out');
  expect(test.element.textContent).toContain('Needs two years');
  await act(async () => { test.renderer.render(createElement(ContractPanel, { id: '0042',
    mine: '0002', provider: provider(), whatIf: simulation })); });
  expect(test.element.textContent).not.toContain('What if I cut him');
});

it('both table routes use the shared work anchor rather than row-menu-only capture', () => {
  const code = readFileSync('src/app/contracts/Contracts.tsx', 'utf8');
  expect(code).toContain("useWorkAnchor(root, '.contract-table tbody tr, .bid-plan')");
});

it('keeps an opened row on one line with the full sentence as its label', async () => {
  useContractMove.setState({ mine: '0001', player: '0042', label: 'Extend Oluokun, Foyesade' });
  const sentence = 'Opened: Extend Oluokun, Foyesade; see Contract move at the top';
  const row = await mount(createElement(WhatIfActs, { mine: '0001', player: '0042', menu: true }));
  const cell = nodes(row.root, 'P')[0];
  expect(cell.getAttribute('aria-label')).toBe(sentence);
  expect(cell.getAttribute('title')).toBe(sentence);
  expect(cell.textContent).toBe('Opened above ↑');
  const inspector = await mount(createElement(WhatIfActs, { mine: '0001', player: '0042' }));
  expect(nodes(inspector.root, 'P')[0].textContent).toBe(sentence);
  const css = readFileSync('src/app/contracts/contracts.css', 'utf8');
  expect(css).toContain('.contract-table .contract-opened { white-space: nowrap; overflow: hidden; '
    + 'text-overflow: ellipsis;');
  await act(async () => { useContractMove.setState({ mine: '', player: '', label: '' }); });
});

function actPlan(id = ''): ContractActPlan {
  return { id, intent: 'contract.extend', act: 'Extend him', levels: [], years: [1, 2, 3],
    player: 'Oluokun, Foyesade',
    subject: 'Oluokun, Foyesade · LB · $11,000,000 a year through 2028 · UFA',
    label: 'Extend him · Oluokun, Foyesade', price: '$16,500,000', post: 'Adds 1 year', how: '§10',
    teamFigures: [], reasons: [], figures: [], canHandOff: false, canPost: false, canWithdraw: false,
    canRebase: false };
}
it('names the contract under the heading and holds Kept until the terms differ', async () => {
  const wire = actPlan();
  const go = { contracts: { API: { TargetContractActs: vi.fn(async () =>
    ({ ...reading([{ ...wire, receipt: {} }]) })) } } };
  useContractMove.setState({ mine: '0001', player: '0042', kept: undefined, keptTerms: '', error: '' });
  const test = await mount(createElement(ContractMovePanel, { mine: '0001' }), { go });
  await act(async () => {});
  const heading = nodes(test.root, 'H4')[0];
  const subject = nodes(heading.parentNode!.parentNode!, 'P').find(node =>
    node.getAttribute('class') === 'contract-move-subject')!;
  expect(subject.textContent).toBe('Oluokun, Foyesade · LB · $11,000,000 a year through 2028 · UFA');
  const keep = () => nodes(test.root, 'BUTTON').find(b => /Keep|Kept/.test(b.textContent))!;
  expect(keep().textContent).toBe('Keep plan');
  const request = { mine: '0001', player: '0042', action: 'preview' as const, intent: 'contract.extend',
    level: 1, years: 1, move: '1', source: 0, destination: 0 };
  await act(async () => { useContractMove.setState({ kept: { ...actPlan('contract-1'),
    receipt: { state: 'ready', spec: { intent: 'contract.extend' }, audit: [] } },
  keptTerms: termsKey(request) }); });
  expect(keep().textContent).toBe('Kept');
  expect(props(keep()).disabled).toBe(true);
  const years = nodes(test.root, 'SELECT')[1];
  await act(async () => { (props(years).onChange as (e: unknown) => void)({ target: { value: '2' } }); });
  expect(keep().textContent).toBe('Keep plan');
  expect(test.element.textContent).toContain('Plan: Ready');
  await act(async () => { (props(years).onChange as (e: unknown) => void)({ target: { value: '1' } }); });
  expect(keep().textContent).toBe('Kept');
  await act(async () => {
    useContractMove.setState({ mine: '', player: '', kept: undefined, keptTerms: '' });
  });
});

it('scrolls the revealed plan card into view in My moves and focuses it, once', async () => {
  useContractMove.setState({ reveal: '' });
  const test = await mount(createElement(ContractActCard, { plan: actPlan('contract-1'), mine: '0001',
    revealing: true }));
  const card = nodes(test.root, 'ARTICLE')[0];
  const focus = vi.fn();
  const scroll = { scrollTop: 0, clientTop: 0, clientHeight: 700,
    getBoundingClientRect: () => ({ top: 0 }) };
  Object.assign(card, { closest: () => scroll, getBoundingClientRect: () => ({ top: 400 }), focus });
  await act(async () => { useContractMove.setState({ reveal: 'contract-other' }); });
  expect(scroll.scrollTop).toBe(0);
  await act(async () => { useContractMove.setState({ reveal: 'contract-1' }); });
  expect(scroll.scrollTop).toBe(400);
  expect(focus).toHaveBeenCalledWith({ preventScroll: true });
  expect(card.getAttribute('tabindex')).toBe('-1');
  expect(useContractMove.getState().reveal).toBe('');
});

it.each(['withdrawn', 'landed', 'failed', 'missing'])
('re-enables Keep and clears feedback for %s', async state => {
  const wire = actPlan();
  const go = { contracts: { API: { TargetContractActs: vi.fn(async () => reading([wire])) } } };
  useContractMove.setState({ mine: '0001', player: '0042', kept: undefined, keptTerms: '' });
  const test = await mount(createElement(ContractMovePanel, { mine: '0001' }), { go });
  const kept = { ...actPlan('contract-1'), receipt: {
    state: 'ready' as const, spec: { intent: 'contract.extend' }, audit: [],
  } };
  const request = { mine: '0001', action: 'preview' as const, intent: 'contract.extend',
    level: 1, years: 1, move: '1', source: 0, destination: 0 };
  await act(async () => { useContractMove.setState({ kept, keptTerms: termsKey(request) }); });
  expect(test.element.textContent).toContain('Kept');
  await act(async () => { reconcileContractMove('0001', state === 'missing' ? [] :
    [{ ...kept, receipt: { ...kept.receipt, state: state as 'withdrawn' | 'landed' | 'failed' } }]); });
  const keep = nodes(test.root, 'BUTTON').find(node => node.textContent === 'Keep plan')!;
  expect(props(keep).disabled).toBe(false);
  expect(test.element.textContent).not.toContain('Plan:');
});

it('Close is registered, removes the panel and clears the row opened state', async () => {
  useContractMove.setState({ mine: '0001', player: '0042' });
  const wire = actPlan();
  const go = { contracts: { API: { TargetContractActs: vi.fn(async () => reading([wire])) } } };
  const test = await mount(createElement(ContractMovePanel, { mine: '0001' }), { go });
  const close = nodes(test.root, 'BUTTON').find(node => node.textContent === 'Close')!;
  await act(async () => { (props(close).onClick as () => void)(); });
  expect(useContractMove.getState().player).toBe('');
  expect(test.element.textContent).not.toContain('Contract move');
  const row = renderToStaticMarkup(createElement(WhatIfActs, {
    mine: '0001', player: '0042', menu: true,
  }));
  expect(row).not.toContain('Opened above');
});

it('re-reads the kept plan on moves change and clears terminal feedback', async () => {
  let changed!: () => void;
  vi.spyOn(commands, 'onMovesChange').mockImplementation(listener => {
    changed = listener; return () => {};
  });
  const wire = actPlan();
  let live: ContractActPlan[] = [];
  const go = { contracts: { API: { TargetContractActs: vi.fn(async (raw: { action: string }) =>
    reading(raw.action === 'read' ? live : [wire])) } } };
  useContractMove.setState({ mine: '0001', player: '0042', kept: undefined, keptTerms: '' });
  const test = await mount(createElement(ContractMovePanel, { mine: '0001' }), { go });
  const kept = { ...actPlan('contract-1'), receipt: {
    state: 'ready' as const, spec: { intent: 'contract.extend' }, audit: [],
  } };
  await act(async () => { useContractMove.setState({ kept, keptTerms: 'terms' }); });
  expect(test.element.textContent).toContain('Plan: Ready');
  const before = go.contracts.API.TargetContractActs.mock.calls.length;
  await act(async () => { changed(); });
  expect(go.contracts.API.TargetContractActs.mock.calls.length).toBeGreaterThan(before);
  expect(test.element.textContent).not.toContain('Plan:');
  expect(useContractMove.getState().kept).toBeUndefined();
});

it('Contract Acts menu click renders the row form, scrolls to it and focuses it', async () => {
  shell(); useContractMove.setState({ mine: '', player: '' });
  const wire = actPlan();
  const go = { contracts: { API: { TargetContractActs: vi.fn(async () => reading([wire])) } } };
  const scroll = { scrollTop: 0, clientTop: 0, clientHeight: 700,
    getBoundingClientRect: () => ({ top: 0 }) };
  const focus = vi.spyOn(TestElement.prototype, 'focus');
  const test = await mount(createElement('div', null,
    createElement(WhatIfActs, { mine: '0001', player: '0042', menu: true, value: {
      stale: false, banner: '', empty: '', notice: '', moves: [], figures: [],
      choices: [{ player: '0042', cutLabel: 'Cut him', buyoutLabel: 'Buy him out',
        buyoutAllowed: false, reason: 'No buyout' }],
    } }), createElement(ContractMovePanel, { mine: '0001' })), { go });
  const create = document.createElement.bind(document);
  vi.spyOn(document, 'createElement').mockImplementation(tag => {
    const node = create(tag);
    if (tag === 'section') Object.assign(node, { closest: () => scroll,
      getBoundingClientRect: () => ({ top: -300 }) });
    return node;
  });
  const menu = nodes(test.root, 'BUTTON').find(b => b.textContent === 'Contract Acts…')!;
  await act(async () => { (props(menu).onClick as () => void)(); });
  const panel = nodes(test.root, 'SECTION').find(s => s.getAttribute('aria-label') === 'Contract move');
  expect(panel?.textContent).toContain(wire.subject);
  expect(panel?.textContent).toContain('Extend him');
  expect(panel?.getAttribute('tabindex')).toBe('-1');
  expect(scroll.scrollTop).toBe(-300);
  expect(focus).toHaveBeenCalledWith({ preventScroll: true });
  expect(go.contracts.API.TargetContractActs.mock.calls.length).toBeGreaterThan(0);
  await act(async () => { useContractMove.setState({ mine: '', player: '' }); });
});

it.each(['keydown', 'pointerdown', 'scroll'])('wires open RowMenu dismissal to document %s', async name => {
  const events = new EventTarget();
  const test = await mount(createElement(WhatIfActs, { mine: '0001', player: '0042', menu: true,
    value: { stale: false, banner: '', empty: '', notice: '', moves: [], figures: [],
      choices: [{ player: '0042', cutLabel: 'Cut', buyoutLabel: 'Buyout',
        buyoutAllowed: false, reason: '' }] } }));
  const details = nodes(test.root, 'DETAILS')[0], summary = nodes(details, 'SUMMARY')[0];
  const focus = vi.spyOn(summary, 'focus');
  Object.assign(details, { open: true, querySelector: () => summary, contains: () => false });
  vi.spyOn(document, 'addEventListener').mockImplementation((type, listener, options) => {
    events.addEventListener(type, listener, options);
  });
  vi.spyOn(document, 'removeEventListener').mockImplementation((type, listener, options) => {
    events.removeEventListener(type, listener, options);
  });
  await act(async () => {
    (props(details).onToggle as (e: unknown) => void)({ currentTarget: details });
  });
  const event = new Event(name);
  Object.defineProperty(event, 'key', { value: 'Escape' });
  await act(async () => { events.dispatchEvent(event); });
  expect((details as unknown as HTMLDetailsElement).open).toBe(false);
  expect(focus).toHaveBeenCalledWith({ preventScroll: true });
  await act(async () => { test.renderer.unmount(); });
  Object.assign(details, { open: true }); events.dispatchEvent(new Event('scroll'));
  expect((details as unknown as HTMLDetailsElement).open).toBe(true);
});

it('groups a team filter under the chosen team without offering another team what-ifs', async () => {
  shell();
  setContractsFilter({ field: 'team', value: '0002' });
  const value = readings();
  value.table.value!.rows = [{ ...player.player, franchiseId: '0002', team: 'Other' }];
  const p = provider();
  p.contracts = vi.fn().mockResolvedValue(value);
  const test = await mount(createElement(Contracts, { snapshot: heldSnapshot(), provider: p }));
  expect(test.element.textContent).toContain('Selected team · Other (1)');
  expect(test.element.textContent).toContain('Rest of the league (0)');
  expect(test.element.textContent).not.toContain('My team · Mine');
  const group = nodes(test.root, 'SECTION').find(n => n.getAttribute('aria-label') === 'Selected team · Other (1)')!;
  expect(group.textContent).not.toContain('What-if');
});
