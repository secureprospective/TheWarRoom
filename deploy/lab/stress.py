#!/usr/bin/env python3
"""Lab-only league stress client for a twr-server CT (run on the lab host, never the Beelink).

  stress.py onboard BASE JOINFILE STATE   claim every open team through the join link, sign each owner
                                          in on 2-3 devices, paced under the 10/min/IP auth limit
  stress.py probe   BASE STATE            one owner: try every screen read once; print status per screen
  stress.py run     BASE STATE OUTDIR     every device: screen reads, confirmed lineup clicks, and an
                                          invalidation stream; phase from OUTDIR/phase (steady|burst|pause|stop)
  stress.py hammer  BASE STATE SCREEN SECS  one device, back-to-back reads of one screen (for a profile)
  stress.py digest  BASE STATE OUTFILE    sha256 + size of every screen for owners 1 and 2 (lab parity diff;
                                          waivers at a fixed instant); bodies are never written
  stress.py reconcile OUTDIR CHECKFILE    every confirmed action is in the ledger once, at its receipt's
                                          revision; CHECKFILE is stresscheck's output

STATE (0600, keep in /dev/shm) holds usernames, passwords and bearers. Logs hold ids, statuses,
counts and timings only; never codes, bearers, passwords, names or row contents.
"""
import hashlib, http.client, json, os, random, re, secrets, ssl, sys, threading, time, urllib.parse

HDR = {"Content-Type": "application/json", "X-TWR-Protocol": "1.1", "X-TWR-Commands": "1", "User-Agent": "twr-lab-stress"}
SCREENS = [  # every owner read the hosted server serves (WorkflowCommands + /v1/native/ledger); f = franchise, s = season
    ("ledger", None),
    ("native.assets.read", lambda L, f, s: {"league": L}),
    ("lineup.read", lambda L, f, s: {"league": L, "franchise": f, "week": 13}),
    ("bids.read", lambda L, f, s: {"league": L, "franchise": f}),
    ("moves.read", lambda L, f, s: {"league": L, "franchise": f}),
    ("trades.read", lambda L, f, s: {"league": L, "viewer": f}),
    ("waivers.read", lambda L, f, s: {"league": L, "franchise": f, "at": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime())}),
    ("draft.read", lambda L, f, s: {"league": L, "franchise": f}),
    ("standings.read", lambda L, f, s: {"league": L, "season": s}),
    ("results.read", lambda L, f, s: {"league": L, "season": s, "week": 9}),
    ("value.read", lambda L, f, s: {"league": L, "franchise": f, "season": s}),
    ("odds.read", lambda L, f, s: {"league": L, "season": s}),
    ("native.roster.check", lambda L, f, s: {"league": L}),
    ("native.reserve.check", lambda L, f, s: {"league": L}),
    ("commish.rules.read", lambda L, f, s: {"league": L}),
    ("commish.corrections.read", lambda L, f, s: {"league": L}),
]


class Client:
    """One keep-alive HTTPS connection per device, as a phone or desktop app holds one."""

    def __init__(self, base):
        u = urllib.parse.urlsplit(base)
        self.host, self.conn = u.hostname, None

    def post(self, path, body, bearer=None, timeout=30):
        h = dict(HDR)
        if bearer:
            h["Authorization"] = "Bearer " + bearer
        data = json.dumps(body).encode()
        for attempt in (0, 1):
            try:
                if self.conn is None:
                    self.conn = http.client.HTTPSConnection(self.host, timeout=timeout, context=ssl.create_default_context())
                t = time.perf_counter()
                self.conn.request("POST", path, body=data, headers=h)
                r = self.conn.getresponse()
                out = r.read()
                return r.status, out, (time.perf_counter() - t) * 1000, r.getheader("Retry-After")
            except (OSError, http.client.HTTPException) as e:
                self.close()
                if attempt == 1:
                    return 0, str(type(e).__name__).encode(), 0.0, None
        return 0, b"", 0.0, None

    def close(self):
        if self.conn is not None:
            try:
                self.conn.close()
            except OSError:
                pass
        self.conn = None


def wf(c, bearer, command, mode, args, football=""):
    w = {"command": command, "version": "1", "mode": mode, "arguments": args}
    if football:
        w["football"] = football
    return c.post("/v1/native/workflow", {"protocol": 1, "workflow": w}, bearer)


def save(path, state):
    tmp = path + ".tmp"
    fd = os.open(tmp, os.O_WRONLY | os.O_CREAT | os.O_TRUNC, 0o600)
    with os.fdopen(fd, "w") as f:
        json.dump(state, f)
    os.replace(tmp, path)


class Pacer:
    """At most `per_min` auth calls in any rolling minute from this client address."""

    def __init__(self, per_min=9):
        self.per_min, self.sent = per_min, []

    def wait(self):
        while True:
            now = time.time()
            self.sent = [t for t in self.sent if now - t < 61]
            if len(self.sent) < self.per_min:
                self.sent.append(now)
                return
            time.sleep(61 - (now - self.sent[0]))


def auth(c, pacer, path, body):
    while True:
        pacer.wait()
        s, b, ms, retry = c.post(path, body, timeout=60)
        if s in (429, 503) or s == 0:
            print(f"  {path} -> {s}, retry after {retry or 60}s", flush=True)
            time.sleep(int(retry or 60))
            continue
        return s, b, ms


def onboard(base, joinfile, statefile):
    text = open(joinfile).read()
    league = re.search(r"league (\S+)", text).group(1)
    join = {"league": league, "code": re.search(r"code +([A-Za-z0-9_-]{43})", text).group(1)}
    state = json.load(open(statefile)) if os.path.exists(statefile) else {"league": league, "owners": []}
    c, pacer, start = Client(base), Pacer(), time.time()
    s, b, _ = auth(c, pacer, "/v1/auth/join/franchises", {"protocol": 1, "join": join})
    franchises = json.loads(b)["franchises"]
    have = {o["franchise"] for o in state["owners"]}
    todo = [f["id"] for f in franchises if not f["taken"] and f["id"] not in have]
    print(f"franchises {s}: {len(franchises)} listed, {len(todo)} to claim", flush=True)
    for fid in todo:
        n = len(state["owners"]) + 1
        o = {"user": f"stressowner{n:02d}", "password": secrets.token_urlsafe(18), "franchise": fid,
             "devices": 3 if n <= 16 else 2, "bearers": []}
        s, b, ms = auth(c, pacer, "/v1/auth/join/claim", {"protocol": 1, "claim": dict(join, franchise=fid, username=o["user"], password=o["password"])})
        print(f"claim {fid} {o['user']}: {s} {ms:.0f} ms (t+{time.time() - start:.0f}s)", flush=True)
        if s != 201:
            continue
        state["owners"].append(o)
        save(statefile, state)
    for o in state["owners"]:
        while len(o["bearers"]) < o["devices"]:
            d = len(o["bearers"]) + 1
            s, b, ms = auth(c, pacer, "/v1/auth/login", {"protocol": 1, "username": o["user"], "password": o["password"], "device": f"stress device {d}"})
            if s != 200:
                print(f"login {o['user']} device {d}: {s}", flush=True)
                break
            o["bearers"].append(json.loads(b)["bearer"])
            save(statefile, state)
        print(f"login {o['user']}: {len(o['bearers'])} devices (t+{time.time() - start:.0f}s)", flush=True)
    devices = sum(len(o["bearers"]) for o in state["owners"])
    print(f"onboarded {len(state['owners'])} owners, {devices} devices in {time.time() - start:.0f} s", flush=True)


def season_of(c, bearer, league):
    s, b, _, _ = wf(c, bearer, "native.assets.read", "read", {"league": league})
    resp = json.loads(b)
    return resp["data"]["snapshot"]["season"], resp


def probe(base, statefile):
    state = json.load(open(statefile))
    league, o = state["league"], state["owners"][0]
    c, bearer = Client(base), o["bearers"][0]
    season, _ = season_of(c, bearer, league)
    for name, args in SCREENS:
        if args is None:
            s, b, ms, _ = c.post("/v1/native/ledger", {"protocol": 1, "league": league}, bearer)
        else:
            s, b, ms, _ = wf(c, bearer, name, "read", args(league, o["franchise"], season))
        why = "" if s == 200 else b.decode(errors="replace").strip()[:90]
        print(f"{name:24s} {s} {len(b):8d} B {ms:7.0f} ms {why}", flush=True)


def read_screen(c, bearer, league, fid, season, name, args):
    if args is None:
        return c.post("/v1/native/ledger", {"protocol": 1, "league": league}, bearer)
    return wf(c, bearer, name, "read", args(league, fid, season))


def hammer(base, statefile, screen, secs):
    state = json.load(open(statefile))
    league, o = state["league"], state["owners"][0]
    c, bearer = Client(base), o["bearers"][0]
    season, _ = season_of(c, bearer, league)
    args = dict(SCREENS)[screen]
    lat, size, codes, end = [], 0, {}, time.time() + secs
    while time.time() < end:
        s, b, ms, _ = read_screen(c, bearer, league, o["franchise"], season, screen, args)
        lat.append(ms)
        size = len(b)
        codes[s] = codes.get(s, 0) + 1
    lat.sort()
    print(json.dumps({"screen": screen, "n": len(lat), "p50": round(lat[len(lat) // 2]), "p95": round(lat[int(len(lat) * .95)]),
                      "bytes": size, "codes": codes}), flush=True)


FIXED_AT = "2026-11-08T12:00:00Z"


def digest(base, statefile, outfile):
    state = json.load(open(statefile))
    league, out = state["league"], {}
    for o in state["owners"][:2]:
        c, bearer = Client(base), o["bearers"][0]
        season, _ = season_of(c, bearer, league)
        for name, args in SCREENS:
            if name == "waivers.read":
                args = lambda L, f, s: {"league": L, "franchise": f, "at": FIXED_AT}
            s, b, _, _ = read_screen(c, bearer, league, o["franchise"], season, name, args)
            out[f"{o['franchise']}/{name}"] = {"status": s, "bytes": len(b), "sha256": hashlib.sha256(b).hexdigest()}
    json.dump(out, open(outfile, "w"), indent=1, sort_keys=True)


class Run:
    def __init__(self, base, state, outdir):
        self.base, self.state, self.outdir = base, state, outdir
        self.league = state["league"]
        self.lock = threading.Lock()
        self.lat = open(os.path.join(outdir, "latency.jsonl"), "a", buffering=1)
        self.confirmed = open(os.path.join(outdir, "confirmed.jsonl"), "a", buffering=1)
        self.events = open(os.path.join(outdir, "events.jsonl"), "a", buffering=1)
        self.screens = [s for s in SCREENS if s[0] in set(state.get("screens", [n for n, _ in SCREENS]))]
        self.seq = 0

    def phase(self):
        try:
            return open(os.path.join(self.outdir, "phase")).read().strip() or "steady"
        except OSError:
            return "steady"

    def log(self, f, **kw):
        kw["t"] = round(time.time(), 3)
        with self.lock:
            f.write(json.dumps(kw) + "\n")

    def next_request(self, fid):
        with self.lock:
            self.seq += 1
            return f"stress.{fid}.{os.getpid()}.{self.seq}.{secrets.token_hex(3)}"

    def device(self, owner, d, bearer):
        c, fid, season = Client(self.base), owner["franchise"], None
        lineups, football, toggle = {}, "", 0
        while True:
            ph = self.phase()
            if ph == "stop":
                return
            if ph == "pause":
                time.sleep(1)
                continue
            try:
                if season is None:
                    s, b, ms, _ = wf(c, bearer, "native.assets.read", "read", {"league": self.league})
                    self.log(self.lat, ph=ph, kind="screen", op="native.assets.read", ms=round(ms, 1), s=s, dev=f"{fid}/{d}")
                    if s != 200:
                        time.sleep(5)
                        continue
                    resp = json.loads(b)
                    season, football = resp["data"]["snapshot"]["season"], resp.get("football", "")
                if random.random() < (0.15 if ph == "steady" else 0.35):
                    toggle += 1
                    self.click(c, owner, d, bearer, season, lineups, toggle, ph)
                else:
                    name, args = random.choice(self.screens)
                    if args is None:
                        s, b, ms, _ = c.post("/v1/native/ledger", {"protocol": 1, "league": self.league}, bearer)
                    else:
                        s, b, ms, _ = wf(c, bearer, name, "read", args(self.league, fid, season))
                    self.log(self.lat, ph=ph, kind="screen", op=name, ms=round(ms, 1), s=s, dev=f"{fid}/{d}",
                             err=None if s == 200 else b.decode(errors="replace")[:80])
            except Exception as e:  # keep the device alive across server restarts; record why
                self.log(self.events, ev="device-error", dev=f"{fid}/{d}", err=type(e).__name__ + ":" + str(e)[:80])
                c.close()
                time.sleep(2)
            time.sleep(random.uniform(4, 20) if ph == "steady" else random.uniform(0, 0.5))

    def lineup_for(self, c, bearer, fid, week):
        s, b, ms, _ = wf(c, bearer, "lineup.read", "read", {"league": self.league, "franchise": fid, "week": week})
        if s != 200:
            return None
        lineup = json.loads(b)["data"]
        pool = lineup["bench"] + lineup["starters"]
        if any(p["locked"] for p in pool):
            return None
        by = {}
        for p in pool:
            by.setdefault(p["position"], []).append(p["player"])
        pos_of = {p["player"]: p["position"] for p in pool}
        pick, used = [], set()

        def take(pos, n):
            for pid in by.get(pos, []):
                if n == 0:
                    return
                if pid not in used:
                    pick.append(pid)
                    used.add(pid)
                    n -= 1
        for pos, n in [("QB", 1), ("RB", 1), ("WR", 2), ("TE", 1), ("PK", 1), ("DT", 2), ("DE", 2), ("LB", 2), ("CB", 2), ("S", 2)]:
            take(pos, n)
        off = lambda: sum(1 for x in used if pos_of[x] in ("QB", "RB", "WR", "TE", "OL"))
        for pos in ("OL", "WR", "RB", "TE", "QB"):
            while off() < 8 and any(x not in used for x in by.get(pos, [])):
                take(pos, 1)
        dfn = lambda: sum(1 for x in used if pos_of[x] in ("DT", "DE", "LB", "CB", "S"))
        for pos in ("LB", "S", "CB", "DE", "DT"):
            while dfn() < 12 and any(x not in used for x in by.get(pos, [])):
                take(pos, 1)
        # A second lineup differs by one starter, so alternating writes always change something.
        alt = list(pick)
        for i, pid in enumerate(pick):
            spare = [x for x in by.get(pos_of[pid], []) if x not in used]
            if spare:
                alt[i] = spare[0]
                break
        return [pick, alt]

    def click(self, c, owner, d, bearer, season, lineups, toggle, ph):
        fid = owner["franchise"]
        week = random.choice([13, 14, 15, 16, 17])
        if week not in lineups:
            lineups[week] = self.lineup_for(c, bearer, fid, week)
        if not lineups[week]:
            return
        s, b, ms, _ = wf(c, bearer, "native.assets.read", "read", {"league": self.league})
        if s != 200:
            return
        resp = json.loads(b)
        revision, football = resp["data"]["revision"], resp.get("football", "")
        req = self.next_request(fid)
        cmd = {"league": self.league, "command": "commish.lineup.set",
               "args": {"lineup": {"season": season, "franchise": fid, "week": week, "starters": lineups[week][toggle % 2]}},
               "expected": revision, "request": req, "reason": "", "effective": "now", "visibility": "private"}
        s, b, ms, _ = wf(c, bearer, "commish.lineup.set", "preview", cmd, football)
        self.log(self.lat, ph=ph, kind="preview", op="lineup.preview", ms=round(ms, 1), s=s, dev=f"{fid}/{d}",
                 err=None if s == 200 else b.decode(errors="replace")[:80])
        if s != 200:
            return
        self.log(self.events, ev="sent", req=req, dev=f"{fid}/{d}", expected=revision)
        first, tries = time.perf_counter(), 0
        while True:
            tries += 1
            s, b, ms, _ = wf(c, bearer, "commish.lineup.set", "execute", cmd, football)
            if s == 200:
                data = json.loads(b).get("data", {})
                self.log(self.lat, ph=ph, kind="click", op="lineup.execute", ms=round(ms, 1), s=s, dev=f"{fid}/{d}", tries=tries,
                         total_ms=round((time.perf_counter() - first) * 1000, 1))
                self.log(self.confirmed, req=req, revision=data.get("revision"), franchise=fid, week=week, tries=tries)
                return
            self.log(self.lat, ph=ph, kind="click", op="lineup.execute", ms=round(ms, 1), s=s, dev=f"{fid}/{d}", tries=tries,
                     err=b.decode(errors="replace")[:80])
            if s in (0, 502, 503, 504, 530) and tries < 200:
                time.sleep(3)  # server down or busy: the same request id, retried, must land exactly once
                continue
            self.log(self.events, ev="refused", req=req, s=s, err=b.decode(errors="replace")[:80])
            return

    def stream(self, owner, d, bearer):
        fid = owner["franchise"]
        while self.phase() != "stop":
            conn = None
            try:
                conn = http.client.HTTPSConnection(Client(self.base).host, timeout=40, context=ssl.create_default_context())
                h = {"Authorization": "Bearer " + bearer, "X-TWR-League": self.league, "User-Agent": "twr-lab-stress"}
                conn.request("GET", "/v1/native/invalidations", headers=h)
                r = conn.getresponse()
                self.log(self.events, ev="stream-open", dev=f"{fid}/{d}", s=r.status)
                if r.status != 200:
                    r.read()
                    time.sleep(30)
                    continue
                while self.phase() != "stop":
                    line = r.fp.readline()
                    if not line:
                        break
                    if line.startswith(b"event: invalidate"):
                        self.log(self.events, ev="invalidate", dev=f"{fid}/{d}")
                self.log(self.events, ev="stream-closed", dev=f"{fid}/{d}")
            except Exception as e:
                self.log(self.events, ev="stream-error", dev=f"{fid}/{d}", err=type(e).__name__)
            finally:
                if conn is not None:
                    conn.close()
            time.sleep(random.uniform(2, 6))

    def go(self):
        threads, limit = [], int(os.environ.get("STRESS_DEVICES", "0")) or None
        devices = [(o, d, b) for o in self.state["owners"] for d, b in enumerate(o["bearers"], 1)]
        devices.sort(key=lambda x: x[1])  # first device of every owner before anyone's second
        for o, d, bearer in devices[:limit]:
            if True:
                threads.append(threading.Thread(target=self.device, args=(o, d, bearer), daemon=True))
                threads.append(threading.Thread(target=self.stream, args=(o, d, bearer), daemon=True))
        for t in threads:
            t.start()
            time.sleep(0.05)
        print(f"started {len(threads) // 2} devices with streams", flush=True)
        for t in threads:
            t.join()


def sent_ids(outdir):
    return {json.loads(x)["req"] for x in open(os.path.join(outdir, "events.jsonl")) if '"ev": "sent"' in x}


def reconcile(outdir, checkfile):
    ledger, integrity, heads, stored, actors = {}, None, {}, {}, set()
    for line in open(checkfile):
        f = line.split(" ")
        if f and f[0] == "integrity":
            integrity = " ".join(f[1:]).strip()
        elif f and f[0] == "head":
            heads[(f[1], f[2])] = int(f[3].strip())
        elif f and f[0] == "req":
            actor, cid, rev = json.loads(f[1]), json.loads(f[2]), int(f[3])
            actors.add(actor)
            stored.setdefault(cid, []).append(rev)
    confirmed = [json.loads(x) for x in open(os.path.join(outdir, "confirmed.jsonl"))]
    # Hosted requests are stored as recipient.sha256(principal NUL request); map each client id back.
    for req in {c["req"] for c in confirmed} | sent_ids(outdir):
        for actor in actors:
            cid = "recipient." + hashlib.sha256((actor + "\0" + req).encode()).hexdigest()
            if cid in stored:
                ledger[req] = stored[cid]
    sent = sent_ids(outdir)
    ids = [c["req"] for c in confirmed]
    lost = [c["req"] for c in confirmed if c["req"] not in ledger]
    dup_ledger = [r for r, revs in ledger.items() if len(revs) > 1]
    dup_confirm = sorted({r for r in ids if ids.count(r) > 1})
    unconfirmed_landed = [r for r in ledger if r not in set(ids)]
    dup_ledger += [c for c, revs in stored.items() if len(revs) > 1 and c not in dup_ledger]
    retried = sum(1 for c in confirmed if c.get("tries", 1) > 1)
    print(json.dumps({"integrity": integrity, "heads": {f"{k[0]}/{k[1]}": v for k, v in heads.items()},
                      "confirmed": len(confirmed), "confirmed_after_retry": retried, "sent": len(sent),
                      "in_ledger": len(ledger), "lost": len(lost),
                      "dup_in_ledger": len(dup_ledger), "dup_confirmations": len(dup_confirm),
                      "landed_not_yet_confirmed": len(unconfirmed_landed),
                      "max_ledger_revision_of_confirmed": max([max(ledger[c["req"]]) for c in confirmed if c["req"] in ledger], default=0),
                      "examples": {"lost": lost[:3], "dup": dup_ledger[:3]}}))


if __name__ == "__main__":
    cmd = sys.argv[1]
    if cmd == "onboard":
        onboard(sys.argv[2], sys.argv[3], sys.argv[4])
    elif cmd == "probe":
        probe(sys.argv[2], sys.argv[3])
    elif cmd == "hammer":
        hammer(sys.argv[2], sys.argv[3], sys.argv[4], float(sys.argv[5]))
    elif cmd == "digest":
        digest(sys.argv[2], sys.argv[3], sys.argv[4])
    elif cmd == "reconcile":
        reconcile(sys.argv[2], sys.argv[3])
    elif cmd == "run":
        Run(sys.argv[2], json.load(open(sys.argv[3])), sys.argv[4]).go()
    else:
        sys.exit(__doc__)
