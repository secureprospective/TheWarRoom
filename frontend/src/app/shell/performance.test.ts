import { describe, it, expect } from 'vitest';
import { readdirSync, readFileSync } from 'node:fs';
import { join, resolve } from 'node:path';
import ts from 'typescript';
import { shallow } from 'zustand/shallow';
import { renderState } from './state';
import { connectDensity } from './density';
import { createCommands } from '../commands/registry';
import { assertShortList } from './shortList';

function densityRead(source: string): boolean {
  const tree = ts.createSourceFile('view.tsx', source, ts.ScriptTarget.Latest, true, ts.ScriptKind.TSX);
  let violation = false;
  function visit(node: ts.Node) {
    if (ts.isPropertyAccessExpression(node) && node.name.text === 'density') violation = true;
    if (ts.isElementAccessExpression(node) && node.argumentExpression &&
      ts.isStringLiteral(node.argumentExpression) && node.argumentExpression.text === 'density') {
      violation = true;
    }
    if (ts.isBindingElement(node) && (node.propertyName ?? node.name).getText(tree) === 'density') {
      violation = true;
    }
    // Views get only the render projection, never a store, read or subscription capability.
    if (ts.isCallExpression(node)) {
      const call = node.expression.getText(tree);
      if (/useStore|useShellStore|\.(read|getState|subscribe)$/.test(call)) violation = true;
      if (call.endsWith('.use') && node.arguments.length) violation = true;
    }
    ts.forEachChild(node, visit);
  }
  visit(tree);
  return violation;
}

function views(root: string): string[] {
  return readdirSync(root, { withFileTypes: true }).flatMap((entry) => {
    const path = join(root, entry.name);
    return entry.isDirectory() ? views(path) : path.endsWith('.tsx') ? [path] : [];
  });
}

const read = (path: string) => readFileSync(resolve('src/app', path), 'utf8');

describe('Performance is law (§18, §M3)', () => {
  it('rejects density reads and raw store access in every view, including specimens', () => {
    expect(views(resolve('src/app')).filter((path) => densityRead(readFileSync(path, 'utf8'))))
      .toEqual([]);
    for (const mutant of [
      'const density = commands.use(s => s.density);',
      'const { density: d } = commands.use();',
      'const d = commands.use()[\'density\'];',
      'useStore(shell, s => s);',
      'commands.read().density;',
    ]) expect(densityRead(mutant)).toBe(true);
  });
  it('changes the CSS attribute, but not the React render snapshot, for N/T/M', () => {
    const c = createCommands(() => undefined);
    const writes: string[] = [];
    const root = {
      setAttribute: (_key: string, value: string) => writes.push(value),
      querySelectorAll: () => [] as unknown as NodeListOf<Element>,
    };
    const stop = connectDensity(c, root);
    const before = renderState(c.read());
    for (const key of ['N', 'T', 'M']) {
      expect(c.key(key)).toBe(true);
      expect(shallow(before, renderState(c.read()))).toBe(true);
    }
    expect(writes).toEqual(['tactical', 'narrative', 'tactical', 'matrix']);
    const mutant = { ...renderState(c.read()), density: c.read().density };
    expect(shallow(before, mutant)).toBe(false);
    stop();
    c.key('N');
    expect(writes).toHaveLength(4);
  });
  it('requires a dynamic fixture boundary and rejects a static import', () => {
    const valid = (source: string) => source.includes("await import('./fixtures/snapshot.json')") &&
      !/import\s+\w+\s+from\s+['"].*snapshot\.json/.test(source);
    expect(valid(read('data/provider.ts'))).toBe(true);
    expect(valid("import fixture from './fixtures/snapshot.json';")).toBe(false);
  });
  it('keeps the harness behind a lazy boundary and rejects a static import', () => {
    const source = readFileSync(resolve('src/TargetMount.tsx'), 'utf8');
    const valid = (text: string) => text.includes("lazy(() => import('./App'))") &&
      text.includes('Opening harness…') && !/import App from/.test(text);
    expect(valid(source)).toBe(true);
    expect(valid(source.replace("lazy(() => import('./App'))", 'App'))).toBe(false);
    expect(valid(source + "\nimport App from './App';")).toBe(false);
  });
  it('requires bounded dev lists and rejects a long list', () => {
    expect(() => assertShortList(80)).not.toThrow();
    expect(() => assertShortList(81)).toThrow('virtualization required');
    expect(() => assertShortList(9, 8)).toThrow('virtualization required');
    const source = read('shell/FranchiseHQ.tsx');
    expect(source).toContain('import.meta.env.DEV');
    expect(source.match(/assertShortList\(/g)).toHaveLength(2);
  });
  it('requires snapshot-keyed roster and search memoization and rejects missing memos', () => {
    const roster = (source: string) => /useMemo\([\s\S]*?franchiseRoster[\s\S]*?\[snapshot, s.franchiseId\]/
      .test(source);
    const index = (source: string) => /useMemo\([\s\S]*?searchCandidates[\s\S]*?\[snapshot\]/.test(source);
    expect(roster(read('shell/FranchiseHQ.tsx'))).toBe(true);
    expect(index(read('commands/CommandBar.tsx'))).toBe(true);
    expect(roster(read('shell/FranchiseHQ.tsx').replace('useMemo(', 'compute('))).toBe(false);
    expect(index(read('commands/CommandBar.tsx').replace('useMemo(', 'compute('))).toBe(false);
  });
  it('requires overlay-only 120ms CSS motion and reduced motion, rejecting mutations', () => {
    const css = read('shell/shell.css');
    const valid = (source: string) => source.includes('width 120ms') &&
      source.includes('transform 120ms') && source.includes('opacity 120ms') &&
      source.includes('prefers-reduced-motion: reduce') && source.includes('transition: none') &&
      /\.insp\s*\{\s*position: absolute/.test(source) && source.includes('grid-column: auto');
    expect(valid(css)).toBe(true);
    const tokens = ['120ms', 'prefers-reduced-motion: reduce', 'position: absolute', 'grid-column: auto'];
    for (const token of tokens) {
      expect(valid(css.split(token).join('broken'))).toBe(false);
    }
    const jsMotion = /requestAnimationFrame|setInterval|\.animate\(/;
    for (const path of views(resolve('src/app'))) {
      expect(readFileSync(path, 'utf8')).not.toMatch(jsMotion);
    }
    for (const mutant of ['requestAnimationFrame(step)', 'setInterval(step)', 'panel.animate([])']) {
      expect(jsMotion.test(mutant)).toBe(true);
    }
  });
});
