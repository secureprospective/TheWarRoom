#!/usr/bin/env python3
"""cdp_app.py <phone-json> <port> <shot-name> [signin] (target: lab). Uses the front Chrome tab: opens /app/, signs in
when asked, waits, screenshots. Prints page text only (no secrets)."""
import base64, json, os, sys, time, urllib.request
from websockets.sync.client import connect
D = os.environ.get("TWR_LAB_KIT") or sys.exit("set TWR_LAB_KIT to the Lab kit folder (secrets live there, never in git)")
credf, port, shot = sys.argv[1:4]; signin = len(sys.argv) > 4
cred = json.load(open(credf)); base = open(f"{D}/server.url").read().strip().rstrip("/")
op = urllib.request.build_opener(urllib.request.ProxyHandler({}))
t = next(t for t in json.load(op.open(f"http://127.0.0.1:{port}/json", timeout=5)) if t["type"] == "page" and "/app/" in t["url"])
op.open(f"http://127.0.0.1:{port}/json/activate/{t['id']}", timeout=5); time.sleep(3)
ws = connect(t["webSocketDebuggerUrl"], max_size=16 << 20, proxy=None); n = [0]
def cmd(m, p=None):
    n[0] += 1; ws.send(json.dumps({"id": n[0], "method": m, "params": p or {}}))
    while True:
        r = json.loads(ws.recv(timeout=40))
        if r.get("id") == n[0]: return r.get("result", {})
def ev(x): return cmd("Runtime.evaluate", {"expression": x, "returnByValue": True, "awaitPromise": True}).get("result", {}).get("value")
time.sleep(1)
print("before:", (ev("document.body.innerText") or "")[:200].replace("\n", " / "))
if signin:
    print(ev("(()=>{const ins=[...document.querySelectorAll('input')];const u=ins.find(i=>/user|name/i.test(i.autocomplete+i.name+i.id)&&i.type!=='password');const p=ins.find(i=>i.type==='password');"
             "if(!u||!p)return 'no form: '+ins.map(i=>i.type+':'+i.id).join(',');const set=(e,v)=>{const s=Object.getOwnPropertyDescriptor(HTMLInputElement.prototype,'value').set;s.call(e,v);e.dispatchEvent(new Event('input',{bubbles:true}))};"
             f"set(u,{json.dumps(cred['username'])});set(p,{json.dumps(cred['password'])});"
             "const b=[...document.querySelectorAll('button')].filter(b=>/sign in/i.test(b.innerText)).pop();if(!b)return 'no button';b.click();return 'submitted'})()"))
    time.sleep(10)
print("after:", (ev("document.body.innerText") or "")[:300].replace("\n", " / "))
png = cmd("Page.captureScreenshot", {"format": "png"}).get("data")
open(f"{D}/shots/{shot}.png", "wb").write(base64.b64decode(png)); print("screenshot", shot)
