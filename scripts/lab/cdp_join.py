#!/usr/bin/env python3
"""cdp_join.py <phone-json> <franchise> <devtools-port> (target: lab, S8 via adb forward). Opens the join link in a new
Chrome tab, claims <franchise>, saves recovery codes, screenshot. Prints no secrets."""
import base64, json, os, sys, time, urllib.parse, urllib.request
from websockets.sync.client import connect
D = os.environ.get("TWR_LAB_KIT") or sys.exit("set TWR_LAB_KIT to the Lab kit folder (secrets live there, never in git)"); os.umask(0o077)
credf, franchise, port = sys.argv[1:4]; phone = os.path.basename(credf)[6:-5]
cred = json.load(open(credf)); base = open(f"{D}/server.url").read().strip().rstrip("/")
link = base + open(f"{D}/join-link.txt").read().strip() + "&franchise=" + franchise
op = urllib.request.build_opener(urllib.request.ProxyHandler({}))
t = json.load(op.open(urllib.request.Request(f"http://127.0.0.1:{port}/json/new?" + urllib.parse.quote(link, safe=""), method="PUT"), timeout=15))
ws = connect(t["webSocketDebuggerUrl"], max_size=16 << 20, proxy=None); n = [0]
def cmd(m, p=None):
    n[0] += 1; ws.send(json.dumps({"id": n[0], "method": m, "params": p or {}}))
    while True:
        r = json.loads(ws.recv(timeout=30))
        if r.get("id") == n[0]: return r.get("result", {})
def ev(x): return cmd("Runtime.evaluate", {"expression": x, "returnByValue": True, "awaitPromise": True}).get("result", {}).get("value")
VIEW = "JSON.stringify(['v0','v1','v2','v3'].filter(i=>{const e=document.getElementById(i);return e&&!e.hidden}))"
for _ in range(30):
    time.sleep(1)
    try:
        if ev("document.readyState") == "complete" and ev(VIEW) not in (None, "[]"): break
    except Exception: pass
print("loaded:", ev("location.pathname"), ev(VIEW), "|", (ev("(document.getElementById('welcomeError')||{}).innerText") or "")[:120])
ev("document.getElementById('start').click()"); time.sleep(3); print("after start:", ev(VIEW))
ev("(()=>{const s=(i,v)=>{const e=document.getElementById(i);e.value=v;e.dispatchEvent(new Event('input',{bubbles:true}))};"
   f"s('username',{json.dumps(cred['username'])});s('password',{json.dumps(cred['password'])});document.getElementById('claim').click()}})()")
for _ in range(20):
    time.sleep(1)
    if '"v3"' in (ev(VIEW) or ""): break
print("after claim:", ev(VIEW), "|", (ev("(document.getElementById('claimError')||{}).innerText") or "")[:150])
codes = ev("[...document.querySelectorAll('#codes li')].map(l=>l.innerText).join('\\n')") or ""
if codes.strip(): open(f"{D}/codes-{phone}.txt", "w").write(codes + "\n"); print("recovery codes saved:", len(codes.splitlines()))
png = cmd("Page.captureScreenshot", {"format": "png"}).get("data")
if png: open(f"{D}/shots/{phone}-claimed.png", "wb").write(base64.b64decode(png)); print("screenshot saved")
