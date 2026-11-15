// Plan step 1, phone: real-tunnel RTT from Brave on the S25 over CDP (Beelink localhost:9334).
// Opens ONE tab on the server origin (/cdn-cgi/trace: the app 404 CSP rightly forbids fetch), measures, closes that tab. Prints "phone-warm <ms>" / "phone-cold <ms>" lines.
const CDP = 'http://localhost:9334';
const origin = process.argv[2];
const n = Number(process.argv[3] || 300);
const path = process.argv[4] || '/v1/not-a-route';
const kind = process.argv[5] || 'phone';

async function main() {
  const before = (await (await fetch(`${CDP}/json`)).json()).filter(p => p.type === 'page').length;
  // Brave on Android refuses /json/new; create the tab with the browser-level Target domain instead.
  const bws = new WebSocket(`ws://localhost:9334/devtools/browser`);
  await new Promise((ok, bad) => { bws.onopen = ok; bws.onerror = bad; });
  const targetId = await new Promise(r => { bws.onmessage = m => { const d = JSON.parse(m.data); if (d.id === 1) r(d.result && d.result.targetId); };
    bws.send(JSON.stringify({ id: 1, method: 'Target.createTarget', params: { url: `${origin}/cdn-cgi/trace` } })); });
  bws.close();
  if (!targetId) throw new Error('Target.createTarget failed');
  const tab = { id: targetId };
  const ws = new WebSocket(`ws://localhost:9334/devtools/page/${targetId}`);
  await new Promise((ok, bad) => { ws.onopen = ok; ws.onerror = bad; });
  let id = 0; const wait = new Map();
  ws.onmessage = m => { const d = JSON.parse(m.data); if (wait.has(d.id)) { wait.get(d.id)(d); wait.delete(d.id); } };
  const call = (method, params = {}) => new Promise(r => { const i = ++id; wait.set(i, r); ws.send(JSON.stringify({ id: i, method, params })); });
  try {
    await new Promise(r => setTimeout(r, 4000)); // let the first navigation finish
    const expr = `(async () => {
      const nav = performance.getEntriesByType('navigation')[0];
      const cold = nav ? nav.responseEnd - nav.startTime : -1;
      const out = []; const url = ${JSON.stringify(path)};
      await fetch(url, {cache: 'no-store'}); // connection already warm from the navigation
      for (let i = 0; i < ${n}; i++) {
        const t = performance.now(); const r = await fetch(url, {cache: 'no-store'}); await r.arrayBuffer();
        out.push([performance.now() - t, r.status]);
      }
      return JSON.stringify({cold, ua: navigator.userAgent, conn: navigator.connection && navigator.connection.effectiveType, out});
    })()`;
    const res = await call('Runtime.evaluate', { expression: expr, awaitPromise: true, returnByValue: true });
    if (!res.result || res.result.exceptionDetails || typeof res.result.result.value !== "string") { console.error("evaluate reply:", JSON.stringify(res).slice(0, 600)); throw new Error("evaluate failed"); }
    const v = JSON.parse(res.result.result.value);
    console.error(`ua=${v.ua} effectiveType=${v.conn}`);
    const codes = new Set(v.out.map(x => x[1])); console.error(`status codes: ${[...codes].join(',')}`);
    console.log(`${kind}-cold ${v.cold.toFixed(2)}`);
    for (const [ms] of v.out) console.log(`${kind}-warm ${ms.toFixed(2)}`);
  } finally {
    ws.close();
    await fetch(`${CDP}/json/close/${tab.id}`);
    await new Promise(r => setTimeout(r, 1000));
    const after = (await (await fetch(`${CDP}/json`)).json()).filter(p => p.type === 'page').length;
    console.error(`tabs before=${before} after=${after}`);
  }
}
main().catch(e => { console.error(e); process.exit(1); });
