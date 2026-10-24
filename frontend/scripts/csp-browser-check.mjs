// Exercise the built policy in a real browser without loading the app or using app credentials.
import {readFileSync, mkdtempSync, rmSync} from 'node:fs';
import {createServer} from 'node:http';
import {spawn} from 'node:child_process';
import {once} from 'node:events';
import {resolve, dirname} from 'node:path';
import {fileURLToPath} from 'node:url';
import assert from 'node:assert/strict';

const frontend = resolve(dirname(fileURLToPath(import.meta.url)), '..');
const html = readFileSync(resolve(frontend, 'dist/index.html'), 'utf8');
const meta = html.match(/<meta http-equiv="Content-Security-Policy" content="[^"]+"/);
assert.ok(meta, 'build the frontend first');
const profile = mkdtempSync(resolve(frontend, '.csp-browser-'));
let remoteRequests = 0;
const remote = createServer((_request, response) => {
  remoteRequests++;
  response.end('window.compromised = true');
});
let finish;
const result = new Promise((resolveResult) => { finish = resolveResult; });
const server = createServer((request, response) => {
  response.setHeader('Cache-Control', 'no-store');
  if (request.url.startsWith('/result?')) {
    finish(JSON.parse(new URL(request.url, 'http://localhost').searchParams.get('value')));
    response.end('ok');
  } else if (request.url === '/probe.js') {
    response.setHeader('Content-Type', 'text/javascript');
    response.end(`
      window.violations = [];
      document.addEventListener('securitypolicyviolation', e => window.violations.push(e.violatedDirective));
      window.addEventListener('load', () => {
        let evalBlocked = false;
        try { eval('window.compromised = true'); } catch (error) { evalBlocked = error instanceof EvalError; }
        setTimeout(() => fetch('/result?value=' + encodeURIComponent(JSON.stringify({
          compromised: Boolean(window.compromised), evalBlocked, violations: window.violations
        }))), 300);
      });
    `);
  } else if (request.url === '/') {
    response.setHeader('Content-Type', 'text/html');
    response.end(`<!doctype html><html><head>${meta[0]}><script src="/probe.js"></script></head><body>
      <script>window.compromised = true</script>
      <img src="/missing" onerror="window.compromised = true">
      <script src="http://127.0.0.1:${remote.address().port}/evil.js"></script>
      </body></html>`);
  } else {
    response.writeHead(404).end();
  }
});
let browser;
let timer;
try {
  remote.listen(0, '127.0.0.1');
  server.listen(0, '127.0.0.1');
  await Promise.all([once(remote, 'listening'), once(server, 'listening')]);
  browser = spawn(process.env.FIREFOX ?? 'firefox', ['--headless', '--no-remote', '--profile', profile, `http://127.0.0.1:${server.address().port}/`], {stdio: 'ignore'});
  const failure = new Promise((_, reject) => {
    browser.once('error', reject);
    browser.once('exit', (code) => reject(new Error(`browser exited before proof (${code})`)));
    timer = setTimeout(() => reject(new Error('browser CSP proof timed out')), 30000);
  });
  const proof = await Promise.race([result, failure]);
  assert.equal(proof.compromised, false, 'injected JavaScript ran');
  assert.equal(proof.evalBlocked, true, 'eval was permitted');
  assert.ok(proof.violations.length >= 4, `missing browser rejection evidence: ${JSON.stringify(proof)}`);
  assert.equal(remoteRequests, 0, 'browser fetched a foreign script');
  console.log('csp-browser-check: inline script, event handler, foreign script and eval blocked');
} finally {
  clearTimeout(timer);
  if (browser?.pid && browser.exitCode === null && browser.signalCode === null) {
    const stopped = once(browser, 'exit');
    browser.kill('SIGKILL');
    await stopped;
  }
  server.closeAllConnections();
  remote.closeAllConnections();
  await Promise.all([new Promise(r => server.close(r)), new Promise(r => remote.close(r))]);
  rmSync(profile, {recursive: true, force: true});
}
