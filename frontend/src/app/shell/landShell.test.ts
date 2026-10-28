import { createElement, type ReactElement } from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import { readFileSync, readdirSync } from 'node:fs';
import { resolve, join } from 'node:path';
import { expect, it } from 'vitest';
import ts from 'typescript';
import { Act } from '../commands/Act';
import { commands } from '../commands/registry';
import { ReportText, ReportView } from '../reports/ReportView';
import type { Report } from '../reports/contract';
import { parseNow } from '../now/contract';
import nowWire from '../now/wire.testdata.json';
import { parseSnapshot } from '../data/parse';
import fixture from '../data/fixtures/snapshot.json';
import { PlaySpine } from '../now/LiveGame';
import { parseLive } from '../now/liveContract';
import liveWire from '../now/live.testdata.json';

const css = (file: string) => readFileSync(resolve('src/app', file), 'utf8');
function declarations(source: string, selector: string) {
  const rules = [...source.replace(/\/\*[\s\S]*?\*\//g, '').matchAll(/([^{}]+)\{([^{}]*)\}/g)]
    .filter(match => match[1].trim() === selector);
  return rules.map(match => match[2]).join(';');
}
function views(root: string): string[] {
  return readdirSync(root, { withFileTypes: true }).flatMap(entry => {
    const file = join(root, entry.name);
    return entry.isDirectory() ? views(file) : file.endsWith('.tsx') ? [file] : [];
  });
}
it('every Act declares its presentation intent', () => {
  const missing: string[] = [];
  for (const file of views(resolve('src/app'))) {
    const tree = ts.createSourceFile(file, readFileSync(file, 'utf8'), ts.ScriptTarget.Latest,
      true, ts.ScriptKind.TSX);
    function visit(node: ts.Node) {
      if ((ts.isJsxOpeningElement(node) || ts.isJsxSelfClosingElement(node)) &&
          node.tagName.getText(tree) === 'Act' && !node.attributes.properties.some(attr =>
            ts.isJsxAttribute(attr) && attr.name.text === 'variant')) missing.push(file);
      ts.forEachChild(node, visit);
    }
    visit(tree);
  }
  expect(missing).toEqual([]);
});
it('commands render the bordered button face with hover, keyboard focus and disabled states', () => {
  const html = renderToStaticMarkup(createElement(Act, {
    variant: 'button', verb: 'refresh.run', args: { scope: 'signals' }, children: 'Refresh Signals',
  }));
  expect(html).toContain('class="act act-button"');
  const source = css('shell/shell.css');
  expect(declarations(source, '.twr-app .act-button')).toContain('border: 1px solid');
  expect(declarations(source, '.twr-app .act-button')).toContain('background: var(--surface-2)');
  expect(declarations(source, '.twr-app .act-button:hover:not(:disabled)')).toContain('background:');
  expect(declarations(source, '.twr-app .act-button:focus-visible')).toContain('outline:');
  expect(declarations(source, '.twr-app .act-button:disabled')).toContain('opacity: .6');
  expect(declarations(source, '.twr-app .act-text')).toContain('text-decoration: underline');
});
it('expanded Inspector reserves its width; density wraps instead of being overlaid or clipped', () => {
  const source = css('shell/shell.css');
  expect(declarations(source, '.twr-app.target-frame[data-inspector="expanded"]'))
    .toContain('grid-template-columns: 192px minmax(0, 1fr) 480px 48px');
  const pane = 1280 - 2 - 192 - 480 - 48;
  expect(pane).toBe(558);
  expect(declarations(source, '.twr-app .ws-head')).toContain('flex-wrap: wrap');
  expect(declarations(source, '.twr-app .strip')).toContain('flex: 0 0 auto');
  expect(declarations(source, '.twr-app .strip')).toContain('overflow: visible');
  expect(declarations(source, '.twr-app .insp-body, .twr-app .wbody')).toContain('min-height: 0');
  const reports = css('draftreports/reports.css');
  expect(declarations(reports, '.draft-report .draft-table-scroll table'))
    .toContain('table-layout: fixed');
  expect(declarations(reports, '.draft-report .draft-table-scroll :is(th, td)'))
    .toContain('min-width: 0');
  expect(declarations(reports, '.draft-report .draft-table-scroll :is(th, td)'))
    .toContain('overflow-wrap: anywhere');
  expect(declarations(reports, '.draft-report .draft-table-scroll th .act-button'))
    .toContain('max-width: 100%');
  expect(declarations(reports, '.draft-report .draft-table-scroll table.report-dense :is(th, td)'))
    .toContain('min-width: 0; padding: var(--density-table-padding, 8px 2px)');
});
it('a rendered Now player name dispatches inspect and the franchise header is inspectable', () => {
  const snapshot = parseSnapshot(fixture);
  commands.loadSnapshot(snapshot);
  const roster = snapshot.rosters.value[0], player = roster.players[0];
  const name = snapshot.players.value.find(p => p.id === player.id)!.name!;
  const entity = { text: name, kind: 'player' as const, id: player.id };
  const text = ReportText({ text: `${name} · HOU`, entities: [entity] });
  const act = (text.props.children as ReactElement[])[1];
  expect(act.type).toBe(Act);
  const button = Act(act.props);
  button.props.onClick();
  expect(commands.read().subject?.id).toBe(player.id);
  const franchise = snapshot.franchises.value.find(f => f.id === roster.franchiseId)!;
  const report: Report = { id: 'X-03', title: 'My matchup', status: 'ok', note: '',
    source: 'MFL', asOf: '', how: '', columns: ['Slot', franchise.name!, 'Points'], summary: [],
    rows: [{ cells: ['QB', `${name} · HOU`, '22.00'], range: null }], link: null,
    entities: [entity, { text: franchise.name!, kind: 'franchise', id: franchise.id }] };
  const html = renderToStaticMarkup(createElement(ReportView, { report, howLabel: 'How' }));
  expect(html).toContain(`aria-label="Inspect ${name}"`);
  expect(html).toContain(`aria-label="Inspect ${franchise.name}"`);
  expect(html).toContain('act-text');
});
it('standalone box reports and the asset board use the same pane-fit rules', () => {
  const reports = css('draftreports/reports.css');
  expect(declarations(reports, '.draft-report .draft-table-scroll table')).toContain('width: 100%');
  expect(declarations(css('shell/shell.css'), '.twr-app .report-table')).toContain('table-layout: fixed');
});
it('every held play-spine player is an inline inspect Act', () => {
  const view = parseLive(liveWire);
  const html = renderToStaticMarkup(createElement(PlaySpine, {
    entries: view.spine, title: 'Our players', empty: 'No plays',
  }));
  for (const entry of view.spine) expect(html).toContain(`aria-label="Inspect ${entry.player.name}"`);
  expect((html.match(/class="act act-text"/g) ?? []).length).toBe(view.spine.length);
});
it('every-matchup franchise group headings are inspectable, not just player cells', () => {
  const report = parseNow(nowWire.now).board.reports.find(r => r.id === 'M-005')!;
  const html = renderToStaticMarkup(createElement(ReportView, { report, howLabel: 'How' }));
  const groups = [...html.matchAll(/class="report-group"><th[^>]*>(.*?)<\/th>/g)];
  expect(groups.length).toBeGreaterThan(0);
  for (const group of groups) expect(group[1]).toContain('aria-label="Inspect ');
});

it.each([1280, 1920])('Comms owns 280px without covering work at a %spx window', width => {
  const shell = css('shell/shell.css'), comms = css('comms/comms.css');
  expect(comms).not.toMatch(/width:\s*(?:min\(480px|480px)/);
  expect(declarations(shell, '.twr-app .comms-panel')).toContain('width: 280px');
  expect(declarations(shell, '.twr-app .comms-panel')).toContain('box-sizing: border-box');
  expect(declarations(shell, '.twr-app[data-comms="true"] .insp')).toContain('right: 328px');
  const app = css('shell/TargetApp.tsx');
  expect(app).toContain('data-comms={s.comms}');
  for (const [selector, inspector] of [
    ['.twr-app.target-frame[data-comms="true"]', 320],
    ['.twr-app.target-frame[data-inspector="expanded"][data-comms="true"]', 480],
    ['.twr-app.target-frame[data-inspector="closed"][data-comms="true"]', 40],
  ] as const) {
    const tracks = declarations(shell, selector).match(/192px minmax\(0, 1fr\) (\d+)px 48px/);
    expect(tracks).not.toBeNull();
    const reserve = Number(tracks![1]);
    expect(reserve).toBe(inspector + 280);
    const workRight = width - 2 - 48 - reserve;
    const commsLeft = width - 2 - 48 - 280;
    expect(workRight).toBeLessThanOrEqual(commsLeft);
    expect(workRight - 192).toBeGreaterThan(250);
  }
});

it('command-bar empty copy separates endpoints from players', () => {
  expect(css('commands/CommandBar.tsx'))
    .toContain('No matching commands, places, endpoints, players or franchises.');
});
