// csp-check.mjs — the built index.html carries the Content-Security-Policy (vite.config.ts), the
// policy allows no remote origin and no eval, and the page has no inline script it would block.
import { readFileSync } from 'node:fs';
import { resolve, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';

const dist = resolve(dirname(fileURLToPath(import.meta.url)), '..', 'dist');
const html = readFileSync(resolve(dist, 'index.html'), 'utf8');
const meta = html.match(/<meta http-equiv="Content-Security-Policy" content="([^"]+)"/);
if (!meta) throw new Error('dist/index.html has no Content-Security-Policy meta');
// Vite 8 HTML-escapes apostrophes in injected attributes; browsers decode them.
const policy = meta[1].replace(/&#39;/g, "'");
for (const bad of ["'unsafe-eval'", 'http:', 'https:', 'ws:', 'wss:', '*']) {
  if (policy.includes(bad)) throw new Error(`CSP allows ${bad}: ${policy}`);
}
const scriptSrc = policy.split(';').map((d) => d.trim()).find((d) => d.startsWith('script-src '));
if (scriptSrc !== "script-src 'self'") throw new Error(`CSP script-src is not same-origin only: ${scriptSrc}`);
if (html.indexOf(meta[0]) > html.search(/<script\b/)) throw new Error('CSP meta comes after a script');
for (const [tag, attrs] of html.matchAll(/<script\b([^>]*)>/g).map((m) => [m[0], m[1]])) {
  if (!/\bsrc="\/[^/]/.test(attrs)) throw new Error(`script the CSP would block or allows remote: ${tag}`);
}
if (/\son[a-z]+=/i.test(html)) throw new Error('inline event handler the CSP would block');
console.log(`csp-check: ok (${policy})`);
