#!/usr/bin/env python3
"""seat_phone.py <s8|iphone> <join-link-file> <franchise> (target: lab). Claims <franchise> on the phone's join page
with phone-<phone>.json, saves recovery codes to codes-<phone>.txt, then signs the phone's app in. Prints no secrets."""
import json, os, subprocess, sys, time
D = os.environ.get("TWR_LAB_KIT") or sys.exit("set TWR_LAB_KIT to the Lab kit folder (secrets live there, never in git)"); os.umask(0o077)
phone, linkfile, franchise = sys.argv[1:4]
cred = json.load(open(f"{D}/phone-{phone}.json")); base = open(f"{D}/server.url").read().strip().rstrip("/")
link = open(linkfile).read().strip()
if link.startswith("/"): link = base + link
link += "&franchise=" + franchise
PC = os.path.expanduser("~/fleet/bin/phone-chrome.py")
def pc(*a, quiet=False):
    r = subprocess.run(["python3", PC, phone, *a], capture_output=True, text=True, timeout=120)
    if not quiet: print(f"[{a[0]}] rc={r.returncode}")
    return r.stdout.strip()
def js(expr, quiet=False):
    out = pc("eval", expr, quiet=quiet)
    try: return str((json.loads(out.splitlines()[-1]).get("result") or {}).get("value") or "")
    except Exception: return ""
def state(): return js("JSON.stringify(['v0','v1','v2','v3'].filter(i=>{const e=document.getElementById(i);return e&&!e.hidden}))", quiet=True)
pc("goto", link, quiet=True); time.sleep(2); js("location.reload(),1", quiet=True); time.sleep(5); print("view", state(), js("document.getElementById('welcomeHead').innerText", quiet=True))
js("(document.getElementById('start')&&!document.getElementById('v0').hidden)?(document.getElementById('start').click(),'start'):'no start'")
for _ in range(15):
    time.sleep(2)
    if 'v2' in state() or 'v1' in state(): break
print("view", state())
if 'v2' not in state():
    sys.exit("not on the password step; stopping without claiming: " + state() + " " + js("document.getElementById('welcomeHead').innerText", quiet=True))
if False:  # pick the team from the grid by its id text
    js(f"(()=>{{const b=[...document.querySelectorAll('#grid button')].find(b=>(b.dataset.franchise||b.value||'')=== '{franchise}');return b?(b.click(),'picked'):'grid:'+document.getElementById('grid').innerText.slice(0,120)}})()")
    time.sleep(2); print("view", state())
fill = ("(()=>{const set=(id,v)=>{const e=document.getElementById(id);e.value=v;e.dispatchEvent(new Event('input',{bubbles:true}))};"
        f"set('username',{json.dumps(cred['username'])});set('password',{json.dumps(cred['password'])});"
        "document.getElementById('claim').click();return 'claimed'})()")
js(fill)
for _ in range(15):
    time.sleep(2)
    if 'v3' in state(): break
print("view", state(), "error:", js("(document.getElementById('claimError')||{}).innerText||''", quiet=True)[:200])
codes = js("[...document.querySelectorAll('#codes li')].map(l=>l.innerText).join('\\n')", quiet=True)
if codes.strip(): open(f"{D}/codes-{phone}.txt", "w").write(codes + "\n"); print("recovery codes saved:", len(codes.split()), "lines")
pc("shot", f"{D}/shots/{phone}-claimed.png", quiet=True)
