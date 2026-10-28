import { readdirSync, readFileSync } from 'node:fs';
import { join, resolve } from 'node:path';
import { describe, expect, it } from 'vitest';

// The app records notices and shows them inside itself. Nothing on these screens can carry words out.
const forbidden = [/mailto:/i, /\bfetch\s*\(/, /XMLHttpRequest/, /sendBeacon/, /WebSocket/, /window\.open/,
  /location\.(href|assign|replace)/, /<a[\s>]/, /\bsmtp\b/i, /\bsendmail\b/i, /nodemailer/i];
const roots = [resolve('src/app/commissioner'), resolve('src/app/governed')];
const extra = [resolve('src/app/datasources/cacheState.ts')];

function sources(): string[] {
  const files = roots.flatMap(root => readdirSync(root).map(name => join(root, name)))
    .filter(file => /\.tsx?$/.test(file) && !/\.test\.tsx?$/.test(file));
  return [...files, ...extra];
}
const violations = (files: string[], patterns = forbidden) => files.flatMap(file => {
  const text = readFileSync(file, 'utf8');
  return patterns.filter(p => p.test(text)).map(p => `${file}: ${p}`);
});

describe('No path on the commissioner screens sends a message out of the app', () => {
  it('has no mail link, network call, socket or navigation in any commissioner source', () => {
    const files = sources();
    expect(files.length).toBeGreaterThan(40);
    expect(files.some(f => f.endsWith('NoteForm.tsx'))).toBe(true);
    expect(violations(files)).toEqual([]);
  });
  it('would catch each way of sending, so the scan is not decoration', () => {
    for (const sample of ['href="mailto:a@b.c"', "fetch('/send')", 'new XMLHttpRequest()',
      'navigator.sendBeacon(u)', 'new WebSocket(u)', 'window.open(u)', 'location.href = u', '<a href="x">']) {
      expect(forbidden.some(p => p.test(sample)), sample).toBe(true);
    }
  });
  it('shows the posting forms only inside the app: they say nothing is emailed', () => {
    const form = readFileSync(resolve('src/app/commissioner/NoteForm.tsx'), 'utf8');
    expect(form.match(/sends no email and no messages/g)).toHaveLength(2);
  });
});
