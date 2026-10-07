import { describe, it, expect } from 'vitest';
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
    expect(Object.keys(keyHandlerAllowlist)).toEqual(['shell/MFLConnection.tsx', 'commands/CommandBar.tsx']);
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
