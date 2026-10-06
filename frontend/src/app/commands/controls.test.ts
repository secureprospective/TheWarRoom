import { describe, it, expect } from 'vitest';
import { readdirSync, readFileSync, mkdtempSync, writeFileSync, rmSync } from 'node:fs';
import { join, resolve, relative } from 'node:path';

function scan(root: string): string[] {
  const violations: string[] = [];
  function visit(dir: string) {
    for (const entry of readdirSync(dir, { withFileTypes: true })) {
      const file = join(dir, entry.name);
      if (entry.isDirectory()) { if (entry.name !== 'specimen') visit(file); continue; }
      if (!file.endsWith('.tsx') || relative(root, file) === 'commands/Act.tsx') continue;
      if (/<button\b|<a\s|\bonClick\s*=|\bonKeyDown\s*=|\brole\s*=\s*["']button["']/.test(readFileSync(file, 'utf8'))) violations.push(file);
    }
  }
  visit(root);
  return violations;
}
describe('Controls are verbs (§M2/§M3)', () => {
  it('rejects raw controls throughout app except Act and specimen', () => {
    expect(scan(resolve('src/app'))).toEqual([]);
  });
  it('detects each deliberate violation in a temporary file', () => {
    const dir = mkdtempSync(resolve('src/app/commands/.control-proof-'));
    try {
      const file = join(dir, 'Violation.tsx');
      for (const violation of ['<button />', '<a href="#" />', '<div onClick={() => {}} />', '<div onKeyDown={() => {}} />', '<div role="button" />']) {
        writeFileSync(file, violation);
        expect(scan(dir)).toEqual([file]);
      }
    } finally { rmSync(dir, { recursive: true, force: true }); }
  });
});
