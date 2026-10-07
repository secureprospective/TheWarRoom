// Headless pass over every screen: page errors, console errors and screenshots.
// Usage: node tests/screens.mjs [baseURL] [outDir] [view ...]
import {createRequire} from 'node:module';
import {execSync} from 'node:child_process';
const require = createRequire(import.meta.url);
// playwright-core is not a project dependency; use $PLAYWRIGHT_CORE or the copy pnpm cached on this machine.
function playwright() {
  for (const p of [process.env.PLAYWRIGHT_CORE, 'playwright-core']) {
    try { if (p) return require(p); } catch { /* try the next one */ }
  }
  const found = execSync('find "$HOME/.cache/pnpm" -type d -path "*node_modules/playwright-core" 2>/dev/null | head -1').toString().trim();
  if (!found) throw new Error('playwright-core not found; set PLAYWRIGHT_CORE');
  return require(found);
}
const {chromium} = playwright();
const base = process.argv[2] || 'http://127.0.0.1:8766/';
const out = process.argv[3] || '/tmp';
const only = process.argv.slice(4);
const VIEWS = ['market', 'draft', 'tradelog', 'method'];
const browser = await chromium.launch({executablePath: process.env.BROWSER_PATH || '/usr/bin/brave-browser', headless: true});
const page = await browser.newPage({viewport: {width: 1500, height: 1000}});
const problems = [];
page.on('pageerror', e => problems.push(`pageerror: ${e.message}`));
page.on('console', m => { if (m.type() === 'error') problems.push(`console: ${m.text()}`); });
for (const v of (only.length ? only : VIEWS)) {
  const before = problems.length;
  await page.goto(`${base}#/${v}`, {waitUntil: 'networkidle'});
  await page.waitForSelector('.view-head h1', {timeout: 15000});
  await page.waitForTimeout(600);
  const err = await page.$$eval('.error', e => e.map(x => x.textContent));
  for (const e of err) problems.push(`${v}: ${e}`);
  await page.screenshot({path: `${out}/${v}.png`, fullPage: true});
  console.log(`${v}: ${problems.length - before ? 'PROBLEMS' : 'ok'}`);
}
if (problems.length) console.log(problems.join('\n'));
await browser.close();
process.exit(problems.length ? 1 : 0);
