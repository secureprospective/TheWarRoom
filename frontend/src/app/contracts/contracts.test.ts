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
import { parseContracts, parsePlayerContract } from './provider';
import { setContractsFilter } from './state';
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
    cantCut: {
      ...reading<string[]>([]),
      status: 'not_wired',
      absenceReason: "No can't-cut verdict in held exports; owner pages are not scraped",
    },
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
async function mount(view: ReturnType<typeof createElement>) {
  const dom = clockDom([]);
  const create = dom.document.createElement.bind(dom.document);
  dom.document.createElement = (tag) => {
    const node = create(tag);
    if (tag === 'select')
      Object.defineProperty(node, 'options', { get: () => nodes(node, 'OPTION') });
    return node;
  };
  vi.stubGlobal('IS_REACT_ACT_ENVIRONMENT', true);
  vi.stubGlobal('document', dom.document);
  vi.stubGlobal('window', { document: dom.document, HTMLIFrameElement: class {} });
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
    expect(html(readings().cantCut)).toContain('Not wired');
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
      'Inferred',
      'Flat-filled',
      '-$1.23',
      'MFL literal credit',
      'not proven itemised dead cap',
      'owner pages are not scraped',
      'Dead cap · not wired',
    ])
      expect(text).toContain(value);
    const endpoints = test.root
      .querySelectorAll('[data-endpoint]')
      .map((node) => node.getAttribute('data-endpoint'));
    expect(endpoints).toEqual(['M-078', 'M-031', 'M-025']);
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
  for (const id of ['M-078', 'M-031', 'M-025', 'C-06.5'] as const) {
    c.dispatch('endpoint.open', { id });
    expect(c.read().node).toBe('hq');
    expect(c.read().workspace.hq).toBe('contracts-and-cap');
    expect(endpointById(id).status).toBe('built');
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
  expect(test.element.textContent?.match(/No figures yet/g)).toHaveLength(4);
  const value = readings();
  for (const key of ['table', 'cap', 'adjustments', 'cantCut'] as const) {
    value[key].status = 'failed';
    value[key].absenceReason = `${key} unavailable`;
  }
  await act(async () => {
    resolve(value);
  });
  for (const key of ['table', 'cap', 'adjustments', 'cantCut']) {
    expect(test.element.textContent).toContain(`${key} unavailable`);
  }
  expect(test.element.textContent).not.toContain('$125,000,000');
  await act(async () => {
    changed();
  });
  expect(p.contracts).toHaveBeenCalledTimes(2);
  expect(test.element.textContent?.match(/No figures yet/g)).toHaveLength(4);
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
  const test = await mount(createElement(Contracts, { snapshot: heldSnapshot(), provider: p }));
  expect(test.element.textContent).toContain('My team · Arizona Cardinals (12)');
  expect(test.element.textContent).toContain('Rest of the league (1)');
  const tables = nodes(test.root, 'TABLE');
  const mine = tables[1];
  const rest = tables[2];
  expect(nodes(mine, 'TH').map((n) => n.textContent))
    .toEqual(['Player', 'Annual salary', 'Expires / status']);
  const rows = nodes(mine, 'TR').slice(1);
  expect(rows).toHaveLength(12);
  for (const row of rows) {
    expect(nodes(row, 'TD')).toHaveLength(3);
    expect(row.textContent).not.toContain('Arizona Cardinals');
    expect(nodes(row, 'SMALL')[0].textContent).toBe('QB');
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
