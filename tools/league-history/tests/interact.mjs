// Clicks through the Lab the way a person would and checks each thing responds.
// Usage: node tests/interact.mjs [baseURL] [outDir]
import {createRequire} from 'node:module';
import {execSync} from 'node:child_process';
import assert from 'node:assert/strict';
const require = createRequire(import.meta.url);
function playwright() {
  for (const p of [process.env.PLAYWRIGHT_CORE, 'playwright-core']) { try { if (p) return require(p); } catch { /* next */ } }
  return require(execSync('find "$HOME/.cache/pnpm" -type d -path "*node_modules/playwright-core" 2>/dev/null | head -1').toString().trim());
}
const {chromium} = playwright();
const base = process.argv[2] || 'http://127.0.0.1:8765/';
const out = process.argv[3] || '/tmp';
const browser = await chromium.launch({executablePath: process.env.BROWSER_PATH || '/usr/bin/brave-browser', headless: true});
const context = await browser.newContext({viewport: {width: 1500, height: 1000}});
const page = await context.newPage();
const errors = [];
page.on('pageerror', e => errors.push(e.message));
page.on('console', m => { if (m.type() === 'error') errors.push(m.text()); });
const step = async (name, fn) => { await fn(); console.log(`ok  ${name}`); };
const closeDrawer = async () => { await page.keyboard.press('Escape'); await page.waitForSelector('#drawer[hidden]', {state: 'attached'}); };

await page.goto(base, {waitUntil: 'networkidle'});
await page.waitForSelector('h1:has-text("What the league misprices")');

await step('the market opens on a buy list and a sell list, with no team lens', async () => {
  assert.ok((await page.textContent('.mk-headline')).length > 20);
  assert.ok(await page.locator('.mk-list.buy .mk-line').count() >= 1);
  assert.ok(await page.locator('.mk-list.sell .mk-line').count() >= 1);
  assert.equal(await page.locator('select[aria-label="Your team"]').count(), 0);
  assert.ok(!(await page.textContent('main')).includes('Cardinals'));
  assert.equal(await page.locator('.yr-phase').count(), 8);
});

await step('the time of year and the assets narrow the lists, and survive a reload', async () => {
  const trades = async () => +(await page.textContent('.mk-unit-line')).match(/([\d,]+) trades/)[1].replace(/,/g, '');
  const before = await trades();
  await page.click('.yr-window:has-text("In season")');
  await page.waitForFunction(() => /deadline/.test(decodeURIComponent(location.hash)));
  assert.ok(await trades() < before);
  assert.equal(await page.locator('.yr-phase.on').count(), 3);
  assert.match(await page.textContent('.mk-unit-line'), /in season/);
  await page.screenshot({path: `${out}/market-in-season.png`, fullPage: true});
  await page.click('.chip:has-text("LB")');
  await page.waitForFunction(() => /LB/.test(decodeURIComponent(location.hash)));
  // A chosen asset gets its own market: why, when, against everything else, what came back.
  assert.deepEqual(await page.$$eval('.ap-head h2', e => e.map(x => x.textContent)), ['Linebackers']);
  assert.equal(await page.locator('.ap-when-cell').count(), 8);
  assert.ok(await page.locator('.ap-against .ap-row').count() >= 8);
  assert.ok(await page.locator('.ap-back-row').count() >= 3);
  assert.match(await page.textContent('.flt-note'), /moved linebackers/);
  assert.equal(await page.locator('.mk-line.focus').count(), 1);
  await page.locator('.ap-when-cell:not([disabled])').first().click();
  await page.waitForSelector('#drawer:not([hidden]) .trade');
  await closeDrawer();
  await page.click('.segmented button:has-text("Split by age")');
  await page.waitForFunction(() => /byAge":true/.test(decodeURIComponent(location.hash)));
  assert.ok((await page.$$eval('.ap-head h2', e => e.map(x => x.textContent))).some(n => /25 to 27/.test(n)));
  await page.reload({waitUntil: 'networkidle'});
  await page.waitForSelector('.mk-list');
  assert.equal(await page.locator('.yr-phase.on').count(), 3);
  assert.equal(await page.locator('.chip.on').count(), 1);
  await page.click('.flt-all:has-text("All year")');
  await page.click('.flt-all:has-text("Everything")');
  await page.click('.segmented button:has-text("Together")');
  await page.waitForFunction(() => !/phases":\[|assets":\[|byAge":true/.test(decodeURIComponent(location.hash)));
  assert.equal(await trades(), before);
});

await step('a line opens the trades behind it, with the asset outlined', async () => {
  await page.locator('.mk-list.buy .mk-line').first().click();
  await page.waitForSelector('#drawer:not([hidden]) .trade');
  assert.ok(await page.locator('#drawer .assets li.hit').count() > 0);
  await page.screenshot({path: `${out}/drawer-kind.png`});
  await closeDrawer();
});

await step('the seasons range narrows every screen and survives a reload', async () => {
  await page.selectOption('select[aria-label="First season"]', '2021');
  await page.waitForFunction(() => /from=2021/.test(location.hash));
  assert.match(await page.textContent('.mk-how'), /2021 to 2026/);
  await page.reload({waitUntil: 'networkidle'});
  await page.waitForSelector('.mk-list');
  assert.equal(await page.inputValue('select[aria-label="First season"]'), '2021');
  await page.selectOption('select[aria-label="First season"]', '2017');
});

await step('draft timing: maps, a position’s own draft, and the filters', async () => {
  await page.click('#nav a:has-text("Draft timing")');
  await page.waitForSelector('h1:has-text("When to take each position")');
  assert.equal(await page.locator('.dm-table').count(), 2);
  const line = await page.textContent('.mk-unit-line');
  await page.click('.segmented button:has-text("First 2 seasons")');
  await page.waitForFunction(n => document.querySelector('.mk-unit-line').textContent !== n, line);
  assert.match(await page.textContent('.mk-unit-line'), /first 2 seasons/);
  await page.locator('.dm-cell.click').first().click();
  await page.waitForSelector('#drawer:not([hidden]) .pick-row');
  await page.screenshot({path: `${out}/drawer-picks.png`});
  await closeDrawer();
  // A position's own draft: where to take it, when it pays, what decides it, draft or buy.
  await page.click('.dm-pos:has-text("Defensive lineman")');
  await page.waitForSelector('.ap-head h2:has-text("Defensive linemen")');
  assert.equal(await page.locator('.ap .ap-when-grid').count(), 4);
  assert.ok(await page.locator('.ap-buy-row').count() >= 3);
  // Narrowing the draft and the NFL round narrows the picks judged.
  const n = async () => +(await page.textContent('.mk-unit-line')).match(/([\d,]+) picks/)[1].replace(/,/g, '');
  const before = await n();
  await page.click('.yr-phase:has-text("Round 3")');
  await page.waitForFunction(() => /segments":\[/.test(decodeURIComponent(location.hash)));
  assert.ok(await n() < before);
  await page.click('.chip:has-text("4th–7th")');
  await page.waitForFunction(() => /nfl":\[/.test(decodeURIComponent(location.hash)));
  await page.reload({waitUntil: 'networkidle'});
  await page.waitForSelector('.flt');
  assert.equal(await page.locator('.yr-phase.on').count(), 1);
  assert.equal(await page.locator('.chip.on').count(), 2);
  await page.click('.flt-all:has-text("Every pick") >> nth=0');
  await page.click('.flt-all:has-text("Every pick") >> nth=1');
  await page.waitForFunction(() => !/segments":\[|groups":\[|nfl":\[/.test(decodeURIComponent(location.hash)));
  assert.equal(await page.locator('#mine').count(), 0);
});

await step('how the teams trade: tiers, trends, teams, one team, and the log', async () => {
  await page.click('#nav a:has-text("Trade log")');
  await page.waitForSelector('h1:has-text("How the teams trade")');
  assert.equal(await page.locator('.tier-card').count(), 3);
  assert.equal(await page.locator('.team-table tbody tr').count(), 32);
  assert.ok(!(await page.textContent('main')).includes('(you)'));
  // Offers against acceptance: eight stretches, sent and received, narrowed by the time of year.
  assert.equal(await page.locator('.of-col').count(), 8);
  await page.click('.of-head .segmented button:has-text("Sent")');
  await page.waitForFunction(() => /offerDir":"sent"/.test(decodeURIComponent(location.hash)));
  assert.match(await page.textContent('.dm[aria-label="Offers against acceptance"] .ap-say'), /sent by the/);
  await page.click('.of-head .segmented button:has-text("Every offer")');
  // Sorting the teams.
  await page.click('.th-sort:has-text("Trades")');
  await page.waitForFunction(() => /sort":"n"/.test(decodeURIComponent(location.hash)));
  const counts = await page.$$eval('.team-table tbody tr td:nth-of-type(1)', e => e.map(x => +x.textContent.replace(/,/g, '')));
  assert.deepEqual(counts, [...counts].sort((a, b) => b - a));
  // One team in depth, from its row.
  await page.locator('.team-table tbody tr').first().click();
  await page.waitForSelector('.ap .season-cell');
  assert.ok(await page.locator('.partner').count() >= 3);
  await page.locator('.season-cell:not([disabled])').first().click();
  await page.waitForSelector('#drawer:not([hidden]) .trade');
  await closeDrawer();
  // Narrowing to rebuilding teams in season narrows the sides and the log.
  const log = async () => +(await page.textContent('.log-tools .muted')).match(/[\d,]+/)[0].replace(/,/g, '');
  const before = await log();
  await page.click('.yr-phase:has-text("Rebuilding")');
  await page.click('.chip:has-text("In season")');
  await page.waitForFunction(() => /windows":\[/.test(decodeURIComponent(location.hash)));
  assert.ok(await log() < before);
  assert.equal(await page.locator('.of-col').count(), 3);
  // The log still searches and orders.
  const count = await page.textContent('.log-tools .muted');
  await page.fill('input[type=search]', 'zzzz-nobody');
  await page.waitForSelector('.empty');
  await page.fill('input[type=search]', '');
  await page.waitForFunction(c => document.querySelector('.log-tools .muted').textContent === c, count);
  await page.click('.segmented button:has-text("Most lopsided since")');
  await page.waitForSelector('.trade-verdict');
  assert.match(await page.locator('.trade-verdict').first().textContent(), /Since then/);
  await page.click('.flt-all:has-text("Every team")');
  await page.click('.flt-all:has-text("Every trade")');
});

await step('how it’s measured shows every check', async () => {
  await page.click('#nav a:has-text("How it’s measured")');
  await page.waitForSelector('h1:has-text("How it’s measured")');
  for (const h of ['One currency', 'The forecast', 'The league’s own prices', 'Would it have worked?', 'A complete record']) {
    assert.ok(await page.locator(`.panel h2:has-text("${h}")`).count(), h);
  }
});

await step('works offline once loaded', async () => {
  await page.waitForFunction(() => document.getElementById('offline')?.textContent === 'Works offline', null, {timeout: 20000});
  await page.reload({waitUntil: 'networkidle'});
  await context.setOffline(true);
  await page.reload();
  await page.waitForSelector('h1:has-text("How it’s measured")', {timeout: 15000});
  await page.click('#nav a:has-text("The market")');
  await page.waitForSelector('.mk-list');
  await context.setOffline(false);
});

await browser.close();
if (errors.length) { console.log('page errors:\n' + errors.join('\n')); process.exit(1); }
console.log('all interactions passed');
