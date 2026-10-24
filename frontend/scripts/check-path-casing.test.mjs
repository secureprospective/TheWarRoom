import { test } from 'vitest';
import assert from 'node:assert/strict';
import { caseConflicts } from './check-path-casing.mjs';

test('rejects case-only files and directories, not distinct paths', () => {
  assert.deepEqual(caseConflicts(['a/Thing.ts', 'a/thing.ts', 'Cards', 'cards', 'b/Panel.tsx', 'b/panel.ts']),
    [['a/Thing.ts', 'a/thing.ts'], ['Cards', 'cards'], ['b/Panel.tsx', 'b/panel.ts']]);
  assert.deepEqual(caseConflicts(['a/Thing.ts', 'b/thing.ts', 'a/thing.css']), []);
});
