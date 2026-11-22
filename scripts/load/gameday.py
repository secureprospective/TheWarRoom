#!/usr/bin/env python3
"""32-team game-day load script for twr-server hosted (Python 3 stdlib only).

Simulates 1..32 owners signing in through a join link, opening the phone Live tab
reads (board, game detail, matchup, standings) on the app's cadence, holding an
SSE invalidation stream, and bursting when a score changes or an invalidation arrives.

Credentials live in /dev/shm with mode 0600 and are deleted at the end.
Logs hold ids, statuses, counts and timings only: never passwords, bearers, cookies,
codes or names.
"""
import argparse
import gzip
import http.client
import json
import os
import random
import re
import secrets
import ssl
import sys
import threading
import time
import urllib.parse

DEFAULT_TARGET = "https://127.0.0.1:8443"
HDR = {
    "Content-Type": "application/json",
    "X-TWR-Protocol": "1.1",
    "X-TWR-Commands": "1",
    "User-Agent": "twr-gameday-load",
}


def make_ssl_context(ca_path):
    if ca_path:
        return ssl.create_default_context(cafile=ca_path)
    return ssl.create_default_context()


class Client:
    """One keep-alive HTTPS connection per device."""

    def __init__(self, base, ssl_ctx, use_gzip=True):
        u = urllib.parse.urlsplit(base)
        self.host = u.hostname
        self.port = u.port or 443
        self.ssl_ctx = ssl_ctx
        self.use_gzip = use_gzip
        self.conn = None

    def post(self, path, body, bearer=None, timeout=30):
        h = dict(HDR)
        if bearer:
            h["Authorization"] = "Bearer " + bearer
        if self.use_gzip:
            h["Accept-Encoding"] = "gzip"
        data = json.dumps(body).encode()
        for attempt in (0, 1):
            try:
                if self.conn is None:
                    self.conn = http.client.HTTPSConnection(
                        self.host, self.port, timeout=timeout, context=self.ssl_ctx
                    )
                t0 = time.perf_counter()
                self.conn.request("POST", path, body=data, headers=h)
                resp = self.conn.getresponse()
                raw = resp.read()
                ms = (time.perf_counter() - t0) * 1000.0
                if resp.getheader("Content-Encoding") == "gzip" and raw:
                    raw = gzip.decompress(raw)
                return resp.status, raw, ms, resp.getheader("Retry-After")
            except (OSError, http.client.HTTPException) as exc:
                self.close()
                if attempt == 1:
                    return 0, type(exc).__name__.encode(), 0.0, None
        return 0, b"", 0.0, None

    def close(self):
        if self.conn is not None:
            try:
                self.conn.close()
            except OSError:
                pass
            self.conn = None


def wf(client, bearer, command, args):
    body = {
        "protocol": 1,
        "workflow": {
            "command": command,
            "version": "1",
            "mode": "read",
            "arguments": args,
        },
    }
    return client.post("/v1/native/workflow", body, bearer)


def save_state(path, state):
    tmp = f"{path}.{os.getpid()}.tmp"
    fd = os.open(tmp, os.O_WRONLY | os.O_CREAT | os.O_TRUNC, 0o600)
    with os.fdopen(fd, "w") as f:
        json.dump(state, f)
    os.chmod(tmp, 0o600)
    os.replace(tmp, path)


class Pacer:
    """Pace auth calls under the 10/min/IP server limit."""

    def __init__(self, per_min=9):
        self.per_min = per_min
        self.sent = []

    def wait(self):
        while True:
            now = time.time()
            self.sent = [t for t in self.sent if now - t < 61.0]
            if len(self.sent) < self.per_min:
                self.sent.append(now)
                return
            time.sleep(max(0.1, 61.0 - (now - self.sent[0])))


def auth_post(client, pacer, path, body):
    while True:
        pacer.wait()
        status, raw, ms, retry = client.post(path, body, timeout=60)
        if status in (429, 503) or status == 0:
            wait_s = int(retry) if retry and retry.isdigit() else 60
            print(f"auth {path} status={status} retry={wait_s}s", flush=True)
            time.sleep(wait_s)
            continue
        return status, raw, ms


def parse_join_file(join_file):
    with open(join_file, "r") as f:
        text = f.read()
    m_league = re.search(r"league\s+(\S+)", text)
    m_code = re.search(r"code\s+([A-Za-z0-9_-]{20,})", text)
    if not m_league or not m_code:
        raise ValueError("join file must contain 'league <id>' and 'code <token>'")
    return m_league.group(1), m_code.group(1)


def onboard(base, ssl_ctx, join_file, state_file, teams, devices_per_team, use_gzip):
    league, code = parse_join_file(join_file)
    join = {"league": league, "code": code}
    state = {"league": league, "owners": []}
    client = Client(base, ssl_ctx, use_gzip=use_gzip)
    pacer = Pacer()
    t0 = time.time()
    try:
        s, raw, _ = auth_post(
            client, pacer, "/v1/auth/join/franchises", {"protocol": 1, "join": join}
        )
        if s != 200:
            raise RuntimeError(f"join/franchises failed status={s}")
        franchises = json.loads(raw)["franchises"]
        open_ids = [f["id"] for f in franchises if not f.get("taken")][:teams]
        print(
            f"onboard: league={league} open={len(open_ids)} target_teams={teams}",
            flush=True,
        )
        for idx, fid in enumerate(open_ids, 1):
            user = f"loadowner{idx:02d}_{secrets.token_hex(2)}"
            pw = secrets.token_urlsafe(18)
            claim_body = {
                "protocol": 1,
                "claim": {
                    "league": league,
                    "code": code,
                    "franchise": fid,
                    "username": user,
                    "password": pw,
                },
            }
            s, _, ms = auth_post(client, pacer, "/v1/auth/join/claim", claim_body)
            print(f"claim franchise={fid} status={s} ms={ms:.0f}", flush=True)
            if s != 201:
                continue
            state["owners"].append(
                {
                    "user": user,
                    "password": pw,
                    "franchise": fid,
                    "bearers": [],
                }
            )
            save_state(state_file, state)

        for owner in state["owners"]:
            fid = owner["franchise"]
            for d in range(1, devices_per_team + 1):
                login_body = {
                    "protocol": 1,
                    "username": owner["user"],
                    "password": owner["password"],
                    "device": f"load-dev-{d}",
                }
                s, raw, ms = auth_post(client, pacer, "/v1/auth/login", login_body)
                if s != 200:
                    print(f"login franchise={fid} dev={d} status={s}", flush=True)
                    break
                owner["bearers"].append(json.loads(raw)["bearer"])
                save_state(state_file, state)
            print(
                f"login franchise={fid} devices={len(owner['bearers'])} "
                f"elapsed={time.time() - t0:.0f}s",
                flush=True,
            )
    finally:
        client.close()
    return state


def sample_rss_mib(pid):
    if not pid:
        return None
    try:
        with open(f"/proc/{pid}/status", "r") as f:
            for line in f:
                if line.startswith("VmRSS:"):
                    kb = int(line.split()[1])
                    return kb / 1024.0
    except OSError:
        return None
    return None


class LoadRunner:
    def __init__(self, args, state, ssl_ctx):
        self.base = args.target
        self.state = state
        self.ssl_ctx = ssl_ctx
        self.league = state["league"]
        self.cadence = args.cadence_sec
        self.burst_sec = args.burst_sec
        self.use_gzip = args.gzip
        self.server_pid = args.server_pid
        self.stop_ev = threading.Event()
        self.lock = threading.Lock()
        self.latencies = []
        self.statuses = {}
        self.max_rss_mib = 0.0
        self.invalidation_count = 0
        self.log_fp = None
        if args.log_dir:
            os.makedirs(args.log_dir, exist_ok=True)
            self.log_fp = open(
                os.path.join(args.log_dir, "latency.jsonl"), "a", buffering=1
            )

    def record(self, fid, dev, op, status, ms):
        with self.lock:
            self.statuses[status] = self.statuses.get(status, 0) + 1
            if status == 200:
                self.latencies.append(ms)
            if self.log_fp is not None:
                rec = {
                    "t": round(time.time(), 3),
                    "fid": fid,
                    "dev": dev,
                    "op": op,
                    "s": status,
                    "ms": round(ms, 1),
                }
                self.log_fp.write(json.dumps(rec) + "\n")

    def read_op(self, client, bearer, fid, dev, op, args):
        s, raw, ms, _ = wf(client, bearer, op, args)
        self.record(fid, dev, op, s, ms)
        if s == 200 and raw:
            try:
                return json.loads(raw).get("data")
            except json.JSONDecodeError:
                return None
        return None

    def live_tab_cycle(self, client, bearer, fid, dev, ctx):
        board = self.read_op(client, bearer, fid, dev, "nfl.games.read", {"league": self.league})
        game_id = None
        game_status = None
        if isinstance(board, dict):
            games = (board.get("board") or {}).get("games") or []
            picked = next((g for g in games if g.get("status") == "in"), None)
            if picked is None and games:
                picked = games[0]
            if picked:
                game_id = picked.get("id")
                game_status = picked.get("status")

        if game_id:
            self.read_op(
                client,
                bearer,
                fid,
                dev,
                "nfl.game.read",
                {"league": self.league, "game": game_id},
            )
            if game_status == "pre":
                self.read_op(
                    client,
                    bearer,
                    fid,
                    dev,
                    "nfl.game.pregame.read",
                    {"league": self.league, "game": game_id},
                )

        if ctx["season"] and ctx["week"]:
            self.read_op(
                client,
                bearer,
                fid,
                dev,
                "results.read",
                {"league": self.league, "season": ctx["season"], "week": ctx["week"]},
            )
            self.read_op(
                client,
                bearer,
                fid,
                dev,
                "lineup.read",
                {"league": self.league, "franchise": fid, "week": ctx["week"], "at": ""},
            )
            self.read_op(
                client,
                bearer,
                fid,
                dev,
                "standings.read",
                {"league": self.league, "season": ctx["season"]},
            )

    def discover_context(self, client, bearer, fid, dev):
        ctx = {"season": 2026, "week": 1}
        d = self.read_op(
            client, bearer, fid, dev, "native.directory.read", {"league": self.league}
        )
        if isinstance(d, dict) and d.get("season"):
            ctx["season"] = d["season"]
        w = self.read_op(
            client,
            bearer,
            fid,
            dev,
            "week.read",
            {
                "league": self.league,
                "franchise": fid,
                "season": ctx["season"],
                "localZone": "America/Chicago",
                "at": "",
            },
        )
        if isinstance(w, dict):
            wk = w.get("week") or w.get("nextWeek")
            if wk:
                ctx["week"] = wk
        return ctx

    def device_worker(self, fid, dev, bearer, wake_ev):
        client = Client(self.base, self.ssl_ctx, use_gzip=self.use_gzip)
        try:
            ctx = self.discover_context(client, bearer, fid, dev)
            # Stagger initial load slightly across devices.
            time.sleep(random.uniform(0.0, 1.0))
            while not self.stop_ev.is_set():
                self.live_tab_cycle(client, bearer, fid, dev, ctx)
                wait_s = random.uniform(self.cadence * 0.75, self.cadence * 1.25)
                if wake_ev.wait(timeout=wait_s):
                    wake_ev.clear()
                    # Score-change burst: brief jitter then re-read.
                    time.sleep(random.uniform(0.05, 0.5))
        finally:
            client.close()

    def stream_worker(self, fid, dev, bearer, wake_ev):
        u = urllib.parse.urlsplit(self.base)
        host, port = u.hostname, u.port or 443
        while not self.stop_ev.is_set():
            conn = None
            try:
                conn = http.client.HTTPSConnection(
                    host, port, timeout=40, context=self.ssl_ctx
                )
                hdr = {
                    "Authorization": "Bearer " + bearer,
                    "X-TWR-League": self.league,
                    "User-Agent": "twr-gameday-load",
                }
                conn.request("GET", "/v1/native/invalidations", headers=hdr)
                resp = conn.getresponse()
                if resp.status != 200:
                    resp.read()
                    self.stop_ev.wait(timeout=15)
                    continue
                while not self.stop_ev.is_set():
                    line = resp.fp.readline()
                    if not line:
                        break
                    if line.startswith(b"event: invalidate"):
                        with self.lock:
                            self.invalidation_count += 1
                        wake_ev.set()
            except (OSError, http.client.HTTPException):
                pass
            finally:
                if conn is not None:
                    try:
                        conn.close()
                    except OSError:
                        pass
            self.stop_ev.wait(timeout=random.uniform(1.0, 3.0))

    def rss_sampler(self):
        while not self.stop_ev.is_set():
            rss = sample_rss_mib(self.server_pid)
            if rss is not None:
                with self.lock:
                    if rss > self.max_rss_mib:
                        self.max_rss_mib = rss
            self.stop_ev.wait(timeout=2.0)

    def burst_ticker(self, wake_events):
        if self.burst_sec <= 0:
            return
        while not self.stop_ev.wait(timeout=self.burst_sec):
            for ev in wake_events:
                ev.set()

    def run(self, minutes):
        threads = []
        wake_events = []
        for owner in self.state["owners"]:
            fid = owner["franchise"]
            for dev, bearer in enumerate(owner["bearers"], 1):
                ev = threading.Event()
                wake_events.append(ev)
                t_dev = threading.Thread(
                    target=self.device_worker, args=(fid, dev, bearer, ev), daemon=True
                )
                t_sse = threading.Thread(
                    target=self.stream_worker, args=(fid, dev, bearer, ev), daemon=True
                )
                threads.extend([t_dev, t_sse])

        if self.server_pid:
            threads.append(threading.Thread(target=self.rss_sampler, daemon=True))
        threads.append(
            threading.Thread(target=self.burst_ticker, args=(wake_events,), daemon=True)
        )

        for t in threads:
            t.start()
            time.sleep(0.02)

        print(
            f"running: teams={len(self.state['owners'])} devices={len(wake_events)} "
            f"minutes={minutes}",
            flush=True,
        )
        time.sleep(minutes * 60.0)
        self.stop_ev.set()
        for ev in wake_events:
            ev.set()
        for t in threads:
            t.join(timeout=5.0)
        if self.log_fp is not None:
            self.log_fp.close()

    def summary_line(self):
        with self.lock:
            n502 = self.statuses.get(502, 0)
            n5xx = sum(cnt for s, cnt in self.statuses.items() if 500 <= s < 600 and s != 502)
            n0 = self.statuses.get(0, 0)
            lats = sorted(self.latencies)
            p95 = lats[min(len(lats) - 1, int(0.95 * len(lats)))] if lats else 0.0
            rss = self.max_rss_mib
            total = sum(self.statuses.values())
            print(
                f"summary: total={total} statuses={dict(sorted(self.statuses.items()))} "
                f"invalidations={self.invalidation_count}",
                flush=True,
            )
            rss_str = f"{rss:.1f} MiB" if self.server_pid else "n/a MiB"
            ok = n502 == 0 and n5xx == 0 and n0 == 0 and total > 0
            tag = "PASS" if ok else "FAIL"
            return f"{tag}: {n502} x 502, {n5xx + n0} x 5xx, p95 {p95:.0f} ms, max RSS {rss_str}"


def main():
    p = argparse.ArgumentParser(description="32-team game-day load script")
    p.add_argument("--target", default=DEFAULT_TARGET, help="Target origin (default local)")
    p.add_argument("--ca", default="", help="CA certificate PEM for TLS verification")
    p.add_argument("--join-file", required=True, help="File with 'league <id>' and 'code <tok>'")
    p.add_argument("--teams", type=int, default=32, help="Number of teams (1..32)")
    p.add_argument("--devices-per-team", type=int, default=2, help="Devices per team")
    p.add_argument("--minutes", type=float, default=3.0, help="Load duration in minutes")
    p.add_argument("--cadence-sec", type=float, default=15.0, help="Steady Live tab interval")
    p.add_argument("--burst-sec", type=float, default=60.0, help="Simulated score-change burst")
    p.add_argument("--no-gzip", dest="gzip", action="store_false", help="Disable gzip")
    p.add_argument("--server-pid", type=int, default=0, help="Local twr-server PID for RSS")
    p.add_argument("--log-dir", default="", help="Optional directory for latency.jsonl")
    p.add_argument(
        "--state-file",
        default=f"/dev/shm/twr-load-{os.getpid()}.json",
        help="Credential state file in /dev/shm (mode 0600, deleted at end)",
    )
    args = p.parse_args()

    if not 1 <= args.teams <= 32:
        p.error("--teams must be 1..32")
    if args.devices_per_team < 1:
        p.error("--devices-per-team must be >= 1")
    if not args.state_file.startswith("/dev/shm/"):
        p.error("--state-file must live in /dev/shm")

    ssl_ctx = make_ssl_context(args.ca)
    try:
        state = onboard(
            args.target,
            ssl_ctx,
            args.join_file,
            args.state_file,
            args.teams,
            args.devices_per_team,
            args.gzip,
        )
        if not state["owners"]:
            print("FAIL: 0 x 502, 0 x 5xx, p95 0 ms, max RSS 0.0 MiB", flush=True)
            sys.exit(1)
        runner = LoadRunner(args, state, ssl_ctx)
        runner.run(args.minutes)
        line = runner.summary_line()
        print(line, flush=True)
        sys.exit(0 if line.startswith("PASS:") else 1)
    finally:
        if os.path.exists(args.state_file):
            os.remove(args.state_file)


if __name__ == "__main__":
    main()
