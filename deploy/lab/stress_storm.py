"""Bounded lab request storm; credentials stay in memory, aggregates on disk."""
import argparse
import collections
import concurrent.futures
import json
import math
import os
import re
import statistics
import threading
import time
import urllib.parse

from stress import Client, wf

STEPS = (16, 32, 64, 128, 256, 512)


def lab_base(base):
    u = urllib.parse.urlsplit(base)
    if (u.scheme != "https" or not re.fullmatch(r"twr-lab-[0-9]+\.[a-zA-Z0-9.-]+", u.hostname or "")
            or u.username or u.password or u.port not in (None, 443)
            or u.path not in ("", "/") or u.query or u.fragment):
        raise ValueError("storm refused: target must be https://twr-lab-<digits>.<domain> (port 443)")
    return base


class Metrics:
    """Fixed-width histogram bounds memory during sustained refusals (milliseconds)."""
    def __init__(self):
        self.codes = collections.Counter()
        self.errors = collections.Counter()
        self.hist = collections.Counter()
        self.worst = 0

    def add(self, status, ms, error):
        if status:
            self.codes[str(status)] += 1
        else:
            self.errors[error or "unknown"] += 1
        self.hist[math.ceil(ms)] += 1
        self.worst = max(self.worst, ms)

    def summary(self, elapsed):
        n = sum(self.hist.values())
        def percentile(q):
            cumulative = 0
            for ms, count in sorted(self.hist.items()):
                cumulative += count
                if cumulative >= math.ceil(n * q):
                    return ms
            return None
        answers = sum(self.codes.values())
        return {"requests_sent": n, "statuses": dict(self.codes), "connection_errors": dict(self.errors),
                "seconds": elapsed, "served_rps": sum(v for k, v in self.codes.items() if 200 <= int(k) < 300) / elapsed,
                "refusal_rate": (self.codes["429"] + self.codes["503"]) / answers if answers else 0,
                "p50_ms": percentile(.5), "p95_ms": percentile(.95), "worst_ms": self.worst}


PULSE_COMMANDS = ("native.directory.read", "week.read", "standings.read", "bracket.read", "odds.read",
                  "results.read", "lineup.read", "projections.read", "nfl.games.read", "nfl.game.read")


def pulse(client, owner, league, origin):
    """Time the hosted Now screen's reads sequentially, without UI rendering or SSE."""
    start = time.monotonic()
    requests = []
    status, error = 200, None

    def read(command, **arguments):
        nonlocal status, error
        code, body, ms, _ = wf(client, owner["bearers"][0], command, "read", dict(league=league, **arguments))
        requests.append({"command": command, "status": code, "ms": ms,
                         "error": client.error if not code else None})
        if code != 200:
            status, error = code, client.error if not code else None
            raise ValueError("Pulse request refused")
        data = json.loads(body)["data"]
        if not isinstance(data, dict):
            raise ValueError("unreadable Pulse data")
        return data

    try:
        directory = read("native.directory.read")
        season = directory["rosterSeason"]
        if directory["league"] != league or type(season) is not int or season < 1920:
            raise ValueError("unreadable Pulse directory")
        mine = owner["franchise"]
        week_data = read("week.read", franchise=mine, season=season, localZone="UTC", at="")
        schedule = week_data["schedule"]
        if (schedule["league"], schedule["franchise"], schedule["season"]) != (league, mine, season):
            raise ValueError("unreadable Pulse week")
        week = (week_data.get("current") or {}).get("week") or week_data.get("nextWeek")
        for command in ("standings.read", "bracket.read", "odds.read"):
            read(command, season=season)
        if week is not None:
            if type(week) is not int or week < 1:
                raise ValueError("unreadable Pulse week number")
            results = read("results.read", season=season, week=week)
            games = results["games"]
            selected = next((g for g in games if mine in (g["away"], g["home"])), games[0] if games else None)
            if selected:
                sides = (selected["away"], selected["home"])
                if mine in sides:
                    read("lineup.read", franchise=mine, week=week, at="")
                players = list(dict.fromkeys(p["player"] for score in results["scores"]
                    if score["franchise"] in sides for p in score["starters"]))
                if not all(isinstance(p, str) and p and "," not in p for p in players):
                    raise ValueError("unreadable Pulse players")
                for offset in range(0, len(players), 120):
                    read("projections.read", players=",".join(players[offset:offset + 120]), season=season, week=week)
        nfl = read("nfl.games.read")
        selected = next((g for g in nfl["board"]["games"] if g["status"] == "in"), None)
        if selected:
            if not isinstance(selected["id"], str) or not selected["id"]:
                raise ValueError("unreadable NFL game")
            read("nfl.game.read", game=selected["id"])
    except (ValueError, KeyError, TypeError, AttributeError):
        if status == 200:
            status, error = 0, "InvalidPulseResponse"
    return {"at": time.time(), "seconds": start - origin, "completed_seconds": time.monotonic() - origin,
            "status": status, "ms": (time.monotonic() - start) * 1000, "error": error, "requests": requests}


def sample_pulse(client, owner, league, duration, interval):
    start, rows, tick = time.monotonic(), [], 0
    while tick < duration:
        time.sleep(max(0, start + tick - time.monotonic()))
        if time.monotonic() - start >= duration:
            break
        rows.append(pulse(client, owner, league, start))
        tick += interval
        # Skip missed ticks rather than emitting a catch-up burst.
        tick = max(tick, math.ceil((time.monotonic() - start) / interval) * interval)
    time.sleep(max(0, start + duration - time.monotonic()))
    return rows


def flood(base, state, clients, duration, factory, observer=None):
    ready = threading.Event()
    end = [0.0]
    # A different account, not just a different socket, is reserved for Pulse.
    devices = [(o, b) for o in state["owners"][1:] for b in o["bearers"]]
    def worker(index):
        c, stats = factory(base), Metrics()
        c.retry = False  # Retries would hide resets and inflate request counts.
        _, bearer = devices[index % len(devices)]
        try:
            ready.wait()
            while time.monotonic() < end[0]:
                status, _, ms, _ = c.post("/v1/native/ledger", {"protocol": 1, "league": state["league"]}, bearer)
                stats.add(status, ms, c.error)
        finally:
            c.close()
        return stats
    with concurrent.futures.ThreadPoolExecutor(max_workers=clients) as pool:
        try:
            futures = [pool.submit(worker, i) for i in range(clients)]
        finally:
            # Release already-created workers even if the client thread limit is hit.
            start = time.monotonic()
            end[0] = start + duration
            ready.set()
        opens = observer() if observer else []
        combined = Metrics()
        for future in futures:
            part = future.result()
            combined.codes.update(part.codes)
            combined.errors.update(part.errors)
            combined.hist.update(part.hist)
            combined.worst = max(combined.worst, part.worst)
    elapsed = time.monotonic() - start
    return dict(combined.summary(elapsed), clients=clients), opens


def storm(base, state, out, max_step=512, step_seconds=60, hold_seconds=600, *, factory=Client,
          baseline_interval=2, hold_interval=15, recovery_seconds=180, recovery_interval=10):
    lab_base(base)  # Before state access, output creation, or connections.
    if len(state.get("owners", [])) < 2 or any(not o.get("bearers") or not o.get("franchise") for o in state["owners"]):
        raise ValueError("storm requires two signed-in owners with franchises; first owner is reserved for Pulse")
    if max_step not in STEPS or any(not math.isfinite(n) or n <= 0 for n in (step_seconds, hold_seconds, recovery_seconds)):
        raise ValueError("storm durations must be positive; max-step must be a ramp step")
    os.makedirs(out, exist_ok=True)
    c, owner = factory(base), state["owners"][0]
    c.retry = False
    result = {"load_read": "POST /v1/native/ledger", "pulse_endpoint": "POST /v1/native/workflow",
              "pulse_commands": list(PULSE_COMMANDS),
              "pulse_scope": "Hosted Now reads in sequence; conditional results, matchup and NFL detail; no rendering/SSE/drawers",
              "steps": [], "started_at": time.time()}
    try:
        start = time.monotonic()
        baseline = []
        for i in range(10):
            time.sleep(max(0, start + i * baseline_interval - time.monotonic()))
            baseline.append(pulse(c, owner, state["league"], start))
        result["baseline"] = baseline
        if any(row["status"] != 200 for row in baseline):
            raise ValueError("pre-storm baseline failed; all 10 opens must answer 200")
        median = statistics.median(row["ms"] for row in baseline)
        result["baseline_median_ms"] = median
        for step in STEPS:
            row, _ = flood(base, state, step, step_seconds, factory)
            result["steps"].append(row)
            if row["refusal_rate"] >= .1 or step >= max_step:
                break
        result["stop_reason"] = "refusals" if row["refusal_rate"] >= .1 else "max_step"
        result["hold"], result["hold_opens"] = flood(base, state, step, hold_seconds, factory,
            lambda: sample_pulse(c, owner, state["league"], hold_seconds, hold_interval))
        # All workers and their outstanding requests are drained before recovery starts.
        result["stopped_at"] = time.time()
        result["recovery_opens"] = sample_pulse(c, owner, state["league"], recovery_seconds, recovery_interval)
        phases = result["steps"] + [result["hold"]]
        bad5xx = sum(n for p in phases for code, n in p["statuses"].items() if 500 <= int(code) <= 599 and code != "503")
        errors = collections.Counter()
        for p in phases:
            errors.update(p["connection_errors"])
        for opened in baseline + result["hold_opens"] + result["recovery_opens"]:
            for row in opened["requests"]:
                bad5xx += int(500 <= row["status"] <= 599 and row["status"] != 503)
                if row["error"]:
                    errors[row["error"]] += 1
        recovered = next((r["completed_seconds"] for r in result["recovery_opens"]
                          if r["status"] == 200 and r["ms"] <= 1.5 * median), None)
        opens = result["hold_opens"]
        result["verdicts"] = {"gate9": {"unexpected_5xx": bad5xx, "connection_errors": dict(errors),
            "connection_resets": errors["ConnectionResetError"], "crash_oom": "MANUAL: inspect container log"},
            "gate10": {"pulse_200_share": sum(r["status"] == 200 for r in opens) / len(opens) if opens else None},
            "gate11": {"recovered_seconds": recovered, "within_120_seconds": recovered is not None and recovered <= 120}}
        result["max_served_rps_before_refusals"] = max((r["served_rps"] for r in result["steps"] if r["refusal_rate"] == 0), default=0)
    finally:
        c.close()
        with open(os.path.join(out, "storm.json"), "w") as f:
            json.dump(result, f, indent=2)
    print("phase clients requests 2xx/s refusal p50ms p95ms worstms")
    for phase, row in [("ramp", r) for r in result["steps"]] + [("hold", result["hold"])]:
        print(f"{phase:5} {row['clients']:7} {row['requests_sent']:8} {row['served_rps']:5.1f} {row['refusal_rate']:7.1%} "
              f"{row['p50_ms']} {row['p95_ms']} {row['worst_ms']:.1f}")
    for gate, verdict in result["verdicts"].items():
        print(gate + " " + json.dumps(verdict, sort_keys=True))
    return result


def main(argv):
    p = argparse.ArgumentParser(prog="stress.py storm", description="Lab-only storm; time the signed-in hosted League Pulse Now read sequence.")
    p.add_argument("base")
    p.add_argument("state")
    p.add_argument("--out", required=True)
    p.add_argument("--max-step", type=int, choices=STEPS, default=512)
    p.add_argument("--step-seconds", type=float, default=60)
    p.add_argument("--hold-seconds", type=float, default=600)
    args = p.parse_args(argv)
    try:
        lab_base(args.base)
        with open(args.state) as f:
            state = json.load(f)
        storm(args.base, state, args.out, args.max_step, args.step_seconds, args.hold_seconds)
    except (ValueError, OSError, KeyError):
        p.exit(2, "storm refused or failed: require lab HTTPS hostname, valid STATE with two signed-in owners/franchises, positive durations and writable --out\n")
