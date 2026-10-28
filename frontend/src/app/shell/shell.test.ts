import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { createCommands } from '../commands/registry';
import { nodes, nodeKeys, homeRoute, routeFor } from './nodes';
import { parseRoute, formatRoute } from './routes';
import { FixtureProvider } from '../data/provider';

describe('Shell routes and data', () => {
  it('matches all six names and workspace names in spec §4, with the Trade Floor history exception', () => {
    const spec = readFileSync('../docs/ui/Target_UI_Spec_2026-10.md', 'utf8');
    expect(nodeKeys).toEqual(['home', 'war', 'hq', 'trade', 'pulse', 'control']);
    for (const n of nodeKeys) {
      const row = spec
        .split('\n')
        .find((line) => line.startsWith(`| ${nodes[n].label} |`))!;
      expect(row).toBeDefined();
      const names = row.split('|')[2].trim().replace(' (owns the node)', '').split(' · ');
      // Tab labels are sentence case; the spec's table writes some workspaces in running text.
      expect(nodes[n].workspaces.map((w) => w.label.toLowerCase())).toEqual(
        names.map((name) => name.toLowerCase()).filter(name =>
          !['market', 'offers history', 'counterparties', 'archive', 'data and sources'].includes(name)),
      );
      expect(nodes[n].workspaces.length).toBeLessThanOrEqual(n === 'trade' ? 4 : 3);
    }
  });
  it('round-trips every workspace and falls back safely', () => {
    for (const n of nodeKeys)
      for (const w of nodes[n].workspaces) {
        const route = routeFor(n, w.slug)!;
        expect(parseRoute(formatRoute(route))).toEqual(route);
      }
    for (const hash of [
      '',
      '#/unknown/foo',
      '#/home/market',
      '#/home',
      '#/home/seasonal-card/extra',
      '#/%ZZ/%AA',
      '#/constructor/foo',
    ]) {
      expect(parseRoute(hash)).toEqual(homeRoute);
    }
  });
  it('leaves freshness to the status strip, without a Home summary card', () => {
    const shell = readFileSync(new URL('./TargetApp.tsx', import.meta.url), 'utf8');
    expect(shell).toContain('From MFL (reference only)');
    expect(shell).not.toContain('Snapshot summary');
    expect(shell).not.toContain('snapshotSummary');
  });
});
describe('Shell transitions through dispatch only', () => {
  it('remembers workspaces per node and isolates instances', () => {
    const c = createCommands();
    c.dispatch('nav.open', { node: 'war', workspace: 'research' });
    c.dispatch('nav.open', { node: 'hq', workspace: 'contracts-and-cap' });
    c.dispatch('nav.open', { node: 'war' });
    expect(c.read().node).toBe('war');
    expect(c.read().workspace.war).toBe('research');
    expect(c.read().workspace.hq).toBe('contracts-and-cap');
    expect(createCommands().read().node).toBe('home');
  });
  it('sets density and inspector rest → expanded → closed → rest', () => {
    const c = createCommands();
    expect(c.read().inspector).toBe('rest');
    c.dispatch('density.set', { density: 'matrix' });
    expect(c.read().density).toBe('matrix');
    c.dispatch('inspector.expand', {});
    expect(c.read().inspector).toBe('expanded');
    c.dispatch('inspector.close', {});
    expect(c.read().inspector).toBe('closed');
    c.dispatch('inspector.toggle', {});
    expect(c.read().inspector).toBe('rest');
    expect(c.read().subject).toBe(null);
  });
  it('toggles summons, comms, notice and closes overlays', () => {
    const c = createCommands();
    c.dispatch('calendar.summon', {});
    expect(c.read().summoned).toBe('calendar');
    c.dispatch('calendar.summon', {});
    expect(c.read().summoned).toBe(null);
    c.dispatch('comms.toggle', {});
    expect(c.read().comms).toBe(true);
    c.dispatch('comms.toggle', {});
    expect(c.read().comms).toBe(false);
    c.dispatch('commandbar.open', {});
    expect(c.read().commandbar).toBe(true);
    c.dispatch('calendar.summon', {});
    c.dispatch('comms.toggle', {});
    c.dispatch('overlays.close', {});
    expect(c.read()).toMatchObject({
      inspector: 'closed',
      summoned: null,
      comms: false,
      notice: null,
    });
  });
  it('uses the registry keyboard bindings', () => {
    const c = createCommands();
    for (const [key, density] of [
      ['1', 'narrative'],
      ['2', 'tactical'],
      ['3', 'matrix'],
    ]) {
      expect(c.key(key)).toBe(true);
      expect(c.read().density).toBe(density);
    }
    c.key('i');
    expect(c.read().inspector).toBe('closed');
    c.key('I');
    expect(c.read().inspector).toBe('rest');
    c.key('Escape');
    expect(c.read().inspector).toBe('closed');
    expect(c.key('x')).toBe(false);
  });
});
