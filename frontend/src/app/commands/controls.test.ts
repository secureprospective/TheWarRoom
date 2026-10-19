import { describe, it, expect } from 'vitest';
import { createElement } from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import { Act } from './Act';
import {
  readdirSync,
  readFileSync,
  mkdtempSync,
  mkdirSync,
  writeFileSync,
  rmSync,
} from 'node:fs';
import { join, resolve, relative } from 'node:path';

const keyHandlerAllowlist = {
  'shell/MFLConnection.tsx': 'Password field Enter dispatches Connect without copying the key.',
  'commands/CommandBar.tsx':
    'Search input needs arrows, Enter and Escape; execution remains dispatch.',
  'commissioner/SideTray.tsx':
    'The tray dialog traps Tab focus and closes on Escape; its Close control is an Act.',
  'commissioner/HoldToFire.tsx':
    'Hold-to-fire times a held Space or Enter; the held button then dispatches commissioner.execute.',
} as const;

function scan(root: string): string[] {
  const violations: string[] = [];
  function visit(dir: string) {
    for (const entry of readdirSync(dir, { withFileTypes: true })) {
      const file = join(dir, entry.name);
      if (entry.isDirectory()) {
        if (entry.name !== 'specimen') visit(file);
        continue;
      }
      if (!file.endsWith('.tsx') || relative(root, file) === 'commands/Act.tsx') continue;
      const source = readFileSync(file, 'utf8');
      const allowed = Object.hasOwn(keyHandlerAllowlist, relative(root, file));
      if (
        /<button\b|<a\s|\bonClick\s*=|\brole\s*=\s*["']button["']/.test(source) ||
        (!allowed && /\bonKeyDown\s*=/.test(source))
      )
        violations.push(file);
    }
  }
  visit(root);
  return violations;
}
describe('Controls are verbs (§M2/§M3)', () => {
  it('has exactly the reviewed keyboard exceptions', () => {
    expect(Object.keys(keyHandlerAllowlist)).toEqual([
      'shell/MFLConnection.tsx',
      'commands/CommandBar.tsx',
      'commissioner/SideTray.tsx',
      'commissioner/HoldToFire.tsx',
    ]);
    expect(keyHandlerAllowlist['commands/CommandBar.tsx']).toContain('dispatch');
  });
  it('rejects raw controls throughout app except Act and specimen', () => {
    expect(scan(resolve('src/app'))).toEqual([]);
  });
  it('allows only key handling, never raw controls, in the command bar', () => {
    const dir = mkdtempSync(resolve('src/app/commands/.allowlist-proof-'));
    try {
      const commandsDir = join(dir, 'commands');
      mkdirSync(commandsDir);
      const file = join(commandsDir, 'CommandBar.tsx');
      writeFileSync(file, '<input onKeyDown={handle} />');
      expect(scan(dir)).toEqual([]);
      writeFileSync(file, '<button onKeyDown={handle} />');
      expect(scan(dir)).toEqual([file]);
    } finally {
      rmSync(dir, { recursive: true, force: true });
    }
  });
  it('detects each deliberate violation in a temporary file', () => {
    const dir = mkdtempSync(resolve('src/app/commands/.control-proof-'));
    try {
      const file = join(dir, 'Violation.tsx');
      for (const violation of [
        '<button />',
        '<a href="#" />',
        '<div onClick={() => {}} />',
        '<div onKeyDown={() => {}} />',
        '<div role="button" />',
      ]) {
        writeFileSync(file, violation);
        expect(scan(dir)).toEqual([file]);
      }
    } finally {
      rmSync(dir, { recursive: true, force: true });
    }
  });
});

function longLines(root: string): string[] {
  return readdirSync(root, { withFileTypes: true }).flatMap((entry) => {
    const file = join(root, entry.name);
    if (entry.isDirectory()) return longLines(file);
    if (!/\.tsx?$/.test(file)) return [];
    return readFileSync(file, 'utf8')
      .split('\n')
      .flatMap((line, index) => (line.length > 110 ? [`${file}:${index + 1}`] : []));
  });
}

describe('Readable app source (§M2/§M3)', () => {
  it('forbids lines longer than 110 characters', () => {
    expect(longLines(resolve('src/app'))).toEqual([]);
  });
  it('rejects a deliberate long line', () => {
    const dir = mkdtempSync(resolve('src/app/commands/.length-proof-'));
    try {
      const file = join(dir, 'Violation.ts');
      writeFileSync(file, '// ' + 'x'.repeat(110));
      expect(longLines(dir)).toEqual([`${file}:1`]);
    } finally {
      rmSync(dir, { recursive: true, force: true });
    }
  });
});

describe('Act tooltips', () => {
  it('omits a redundant title on plain visible text, including nested text', () => {
    for (const children of [undefined, 'Edit lineup', createElement('span', null, 'Edit lineup')]) {
      const html = renderToStaticMarkup(createElement(Act, {
        verb: 'lineup.edit', variant: 'button', args: { franchiseId: '0025' }, label: 'Edit lineup', children,
      }));
      expect(html).toContain('Edit lineup</');
      expect(html).toContain('aria-label="Edit lineup"');
      expect(html).not.toContain('title=');
    }
  });
  it('gives a self-describing card no title, so none lingers over the next card', () => {
    const Card = () => createElement('div', null, 'Perine, Samaje · RB · CIN');
    const html = renderToStaticMarkup(createElement(Act, {
      verb: 'lineup.edit', variant: 'button', args: { franchiseId: '0025' }, label: 'Bench Perine, Samaje',
      children: createElement(Card),
    }));
    expect(html).toContain('Perine, Samaje · RB · CIN');
    expect(html).toContain('aria-label="Bench Perine, Samaje"');
    expect(html).not.toContain('title=');
  });
  it('keeps a disabled reason when the visible text does not explain it', () => {
    const html = renderToStaticMarkup(createElement(Act, {
      verb: 'lineup.draft', variant: 'button', args: {}, disabled: true,
      label: 'No changes from saved lineup', children: 'Check and save plan',
    }));
    expect(html).toContain('>Check and save plan</button>');
    expect(html).toContain('title="No changes from saved lineup"');
    expect(html).toContain('aria-label="No changes from saved lineup"');
  });
  it.each(['icon'] as const)('keeps the title on a textless %s Act', (variant) => {
    const html = renderToStaticMarkup(createElement(Act, {
      verb: 'lineup.edit', args: { franchiseId: '0025' }, variant,
      label: 'Edit lineup', children: createElement('svg', { 'aria-hidden': true }),
    }));
    expect(html).toContain('title="Edit lineup"');
    expect(html).toContain('aria-label="Edit lineup"');
  });
});

it('replacement command buttons never carry native titles, even shortened or disabled', () => {
  for (const disabled of [true, false]) {
    const html = renderToStaticMarkup(createElement(Act, {
      verb: 'contracts.set', args: { field: 'release-confirm', value: 'release-1' }, variant: 'command',
      label: 'Make this a plan: Cut Banks, Deonte', children: 'Make this a plan', disabled,
    }));
    expect(html).toContain('aria-label="Make this a plan: Cut Banks, Deonte"');
    expect(html).not.toContain('title=');
  }
});

it.each(['war', 'hq', 'trade'] as const)('F: rail %s keeps its name without a native tooltip', node => {
  const html = renderToStaticMarkup(createElement(Act, {
    verb: 'nav.open', args: { node }, variant: 'rail', children: node,
  }));
  expect(html).toContain('aria-label="Open workspace"');
  expect(html).not.toContain('title=');
});

it('F: the replacing Admin Reset control has an accessible name, not a tooltip', () => {
  const source = readFileSync('src/app/datasources/DataSources.tsx', 'utf8');
  expect(source).not.toContain('title="Clear the Admin value');
  expect(source).toContain('label={w.clear}');
});
