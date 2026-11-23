#!/usr/bin/env python3
"""Lab-only commissioner client (target: lab). Secrets stay in this 0700 folder; nothing secret is printed.
  enroll                 set up the authenticator for the account in lab-commish.json (seed -> totp.seed)
  helper <league>        sign in, step up, issue a record-level helper for franchise.01 -> helper-<league>.txt (twr1 line)
  confirm <twrc1-file>   sign in, step up, POST /v1/native/confirm; prints only the confirmation code
  revoke <league>        disconnect the helper made for <league>"""
import base64, hashlib, hmac, json, os, struct, sys, time, urllib.request, urllib.error
D = os.environ.get("TWR_LAB_KIT") or sys.exit("set TWR_LAB_KIT to the Lab kit folder (secrets live there, never in git)"); os.umask(0o077)
CRED = json.load(open(f"{D}/lab-commish.json")); BASE = open(f"{D}/server.url").read().strip().rstrip("/")

def post(path, body, headers=None):
    req = urllib.request.Request(BASE + path, data=json.dumps(body).encode(), method="POST",
                                 headers={"Content-Type": "application/json", "User-Agent": "twr-lab-script/1", **(headers or {})})
    try:
        with urllib.request.urlopen(req, timeout=30) as r: return json.load(r)
    except urllib.error.HTTPError as e:
        sys.exit(f"{path}: HTTP {e.code}: {e.read().decode()[:200]}")

def code():
    seed = open(f"{D}/totp.seed").read().strip(); key = base64.b32decode(seed + "=" * (-len(seed) % 8))
    mac = hmac.new(key, struct.pack(">Q", int(time.time()) // 30), hashlib.sha1).digest()
    o = mac[-1] & 15; return "%06d" % ((struct.unpack(">I", mac[o:o + 4])[0] & 0x7FFFFFFF) % 1000000)

def fresh_code():  # each code is single-use: wait for the next 30 s window
    time.sleep(31 - time.time() % 30); return code()

def creds(**extra): return {"protocol": 1, "username": CRED["username"], "password": CRED["password"], **extra}

def session():
    bearer = post("/v1/auth/login", {"protocol": 1, "username": CRED["username"], "password": CRED["password"],
                                     "device": "tom-lab-script"})["bearer"]
    proof = post("/v1/auth/totp/step-up", creds(code=fresh_code()))["recent_auth"]
    return {"Authorization": "Bearer " + bearer, "X-TWR-Recent-Auth": proof}

cmd = sys.argv[1:] or ["help"]
if cmd[0] == "enroll":
    seed = post("/v1/auth/totp/begin", creds())["seed"]
    open(f"{D}/totp.seed", "w").write(seed); post("/v1/auth/totp/confirm", creds(code=code())); print("authenticator set up")
elif cmd[0] == "helper":
    league = cmd[1]; h = session()
    commands = json.load(open(f"{D}/helper-commands.json"))
    r = post("/v1/auth/agent-token", {"protocol": 1, "scope": {"league": league, "franchise": "franchise.01",
             "commands": commands, "modes": ["read", "preview", "execute"]}}, h)
    open(f"{D}/helper-{league}.txt", "w").write(f"twr1.{r['id']}.{r['bearer']}\n")
    open(f"{D}/helper-{league}.id", "w").write(r["id"]); print(f"helper line saved for {league}, expires {r['expiresAt']}")
elif cmd[0] == "reader":  # read-only token for the Tuesday report card (30-day life) -> reader-<league>.txt
    league = cmd[1]; h = session()
    r = post("/v1/auth/agent-token", {"protocol": 1, "scope": {"league": league, "franchise": "franchise.01",
             "commands": ["native.directory.read", "results.read", "standings.read", "nfl.games.read", "lineup.read"], "modes": ["read"]}}, h)
    open(f"{D}/reader-{league}.txt", "w").write(r["bearer"] + "\n")  # twr-check takes the bare bearer
    print(f"read-only token saved for {league}, expires {r['expiresAt']}")
elif cmd[0] == "confirm":
    raw = open(cmd[1]).read().strip(); assert raw.startswith("twrc1.")
    req = json.loads(base64.urlsafe_b64decode(raw[6:] + "=" * (-len(raw[6:]) % 4)))
    r = post("/v1/native/confirm", {"protocol": 1, "command": req["change"], "tokenId": req["tokenId"]}, session())
    print(r["confirmation"])
elif cmd[0] == "revoke":
    tid = open(f"{D}/helper-{cmd[1]}.id").read().strip()
    post("/v1/auth/agent-token/revoke", {"protocol": 1, "id": tid}, session()); print("helper disconnected")
else:
    print(__doc__)
