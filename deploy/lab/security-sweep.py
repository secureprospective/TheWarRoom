"""Lab-host security checks. Never print subprocess errors, response bodies or STATE values."""
import argparse
import ipaddress
import json
import re
import secrets
import socket
import subprocess
import sys
import time


def run(argv, data=None, timeout=90):
    return subprocess.run(argv, input=data, stdout=subprocess.PIPE, stderr=subprocess.PIPE,
                          text=True, timeout=timeout, check=False)


def validate_target(ct, host):
    if not re.fullmatch(r"2[0-9]{2}", ct) or ct in ("110", "256", "257"):
        raise ValueError("refused: protected or non-lab CT (110, 256 and 257 are forbidden)")
    if not re.fullmatch(r"twr-lab-" + ct + r"\.[a-zA-Z0-9]+(?:[.-][a-zA-Z0-9]+)*", host):
        raise ValueError("refused: public host must be twr-lab-<ctid>.<domain>, with no URL, port or path")


def peer_address(value):
    """Accept DNS/IPv4 host:port or bracketed IPv6, never command options."""
    match = re.fullmatch(r"(?:\[([0-9a-fA-F:]+)\]|([a-zA-Z0-9][a-zA-Z0-9.-]*)):(\d+)", value)
    if not match or not 1 <= int(match[3]) <= 65535:
        raise argparse.ArgumentTypeError("peer must be HOST:PORT or [IPv6]:PORT, port 1..65535")
    host = match[1] or match[2]
    if match[1]:
        try:
            ipaddress.IPv6Address(host)
        except ValueError as exc:
            raise argparse.ArgumentTypeError("invalid IPv6 peer") from exc
    return host, str(int(match[3]))


def quote_config(value):
    return '"' + value.replace('\\', '\\\\').replace('"', '\\"').replace('\r', '\\r').replace('\n', '\\n') + '"'


def known_secrets(value, key=""):
    """STATE can include once-shown codes/keys beyond onboard's passwords and bearers."""
    if isinstance(value, dict):
        return [s for k, v in value.items() for s in known_secrets(v, k)]
    if isinstance(value, list):
        return [s for v in value for s in known_secrets(v, key)]
    if isinstance(value, str) and re.search(r"password|bearer|token|secret|code|key", key, re.I):
        return [line for line in value.splitlines() if line]
    return []


class Sweep:
    def __init__(self, args, state):
        self.args, self.state, self.failed = args, state, False
        self.emitted = set()

    def emit(self, number, verdict, evidence):
        self.failed |= verdict == "FAIL"
        self.emitted.add(number)
        print(f"{number:02d} {verdict}  {evidence}", flush=True)

    def x(self, *argv, data=None, timeout=90):
        return run(["pct", "exec", self.args.ct, "--", *argv], data, timeout)

    def curl(self, path, body=None, bearer=None, extra=(), scheme="https", method=None, recent=None):
        # curl's config arrives over stdin: neither body nor bearer is a process argument.
        fields = [("url", scheme + "://" + self.args.host + path),
                  ("header", "X-TWR-Protocol: 1.1"), ("header", "X-TWR-Commands: 1")]
        if body is not None:
            fields += [("header", "Content-Type: application/json"),
                       ("data-binary", body if isinstance(body, str) else json.dumps(body))]
        if bearer:
            fields.append(("header", "Authorization: Bearer " + bearer))
        if recent:
            fields.append(("header", "X-TWR-Recent-Auth: " + recent))
        if method:
            fields.append(("request", method))
        config = '\n'.join(k + " = " + quote_config(v) for k, v in fields) + '\n'
        p = run(["curl", "--disable", "--silent", "--noproxy", "*", "--connect-timeout", "5",
                 "--max-time", "60", "--proto", "=http,https", "--dump-header", "-",
                 "--write-out", "\nSWEEP_STATUS:%{http_code}", *extra, "--config", "-"], config, 65)
        text, _, code = p.stdout.rpartition("\nSWEEP_STATUS:")
        headers, _, response = text.rpartition("\r\n\r\n") if '\r\n' in text else text.rpartition("\n\n")
        status = int(code) if code.isdigit() else 0
        return status, headers, response, p.returncode

    def console(self):
        p = self.x("sh", "-c", '''found=0; for d in /proc/[0-9]*; do
          [ "$(readlink "$d/exe" 2>/dev/null)" = /usr/local/bin/twr-server ] || continue
          [ "$(tr '\\000' ' ' < "$d/cmdline" | cut -d' ' -f2)" = hosted ] || continue
          found=1; [ "$(awk '/^Uid:/{print $2,$3,$4,$5}' "$d/status")" = "$(id -u twr) $(id -u twr) $(id -u twr) $(id -u twr)" ] || exit 1
        done; [ "$found" = 1 ]''')
        mode = self.x("stat", "-c", "%n %U %a", "/var/lib/twr")
        ok = p.returncode == mode.returncode == 0 and mode.stdout.strip() == "/var/lib/twr twr 700"
        self.emit(1, "PASS" if ok else "FAIL", "hosted process UID=twr and /var/lib/twr owner=twr mode=0700" if ok else "process UID or directory stat did not match")
        expected = {"/var/lib/twr/server.db": ("twr", "600"), "/etc/twr/origin.key": ("twr", "600"),
                    "/etc/twr/authenticator.key": ("twr", "600"), "/etc/cloudflared/credentials.json": ("root", "640")}
        rows, ok = [], True
        for path, (owner, mode) in expected.items():
            p = self.x("stat", "-c", "%n %U %a", path)
            row = p.stdout.strip()
            # Only allow the stat grammar, even on unexpected command output.
            if p.returncode or not re.fullmatch(re.escape(path) + r" [a-zA-Z0-9_-]+ [0-7]{3,4}", row):
                rows.append(path + " stat failed")
                ok = False
            else:
                rows.append(row)
                ok &= row == f"{path} {owner} {mode}"
        self.emit(2, "PASS" if ok else "FAIL", "; ".join(rows))
        p = self.x("ss", "-H", "-lntup")
        endpoints, invalid = set(), False
        for row in p.stdout.splitlines():
            fields = row.split()
            if len(fields) < 5 or fields[0] != "tcp" or fields[1] != "LISTEN":
                invalid = True
            else:
                endpoints.add(fields[4])
        ok = not p.returncode and not invalid and endpoints == {"127.0.0.1:8443", "127.0.0.1:20241"}
        self.emit(3, "PASS" if ok else "FAIL", "ss -H -lntup: " + ("only loopback 8443,20241" if ok else "missing or unexpected listeners (inspect with pct exec CT -- ss -lntup)"))
        self.egress()
        if not self.args.with_reboot:
            self.emit(5, "MANUAL", f"rerun with --with-reboot; or pct reboot {self.args.ct}; "
                      f"pct exec {self.args.ct} -- rc-service twr-server status; pct exec {self.args.ct} -- rc-service twr-tunnel status; "
                      f"pct exec {self.args.ct} -- /usr/local/libexec/twr-status (zero PROBLEM)")
        else:
            p = run(["pct", "reboot", self.args.ct], timeout=120)
            ok = False
            if p.returncode == 0:
                for _ in range(24):
                    time.sleep(5)
                    a = self.x("rc-service", "twr-server", "status")
                    b = self.x("rc-service", "twr-tunnel", "status")
                    c = self.x("/usr/local/libexec/twr-status")
                    if a.returncode == b.returncode == c.returncode == 0 and "PROBLEM" not in c.stdout and "OK" in c.stdout:
                        ok = True
                        break
            self.emit(5, "PASS" if ok else "FAIL", "pct reboot; both rc-service statuses and twr-status zero problems" if ok else "reboot/service/status deadline failed")
        secrets = sorted(set(known_secrets(self.state)))
        if not secrets:
            self.emit(6, "MANUAL", "--state STATE with test passwords, bearers and once-shown enrollment/recovery codes/key values required; "
                      f"feed known values privately to pct exec {self.args.ct} -- grep -c -F -f - /var/lib/twr/server.log (counts only)")
        else:
            p = self.x("grep", "-c", "-F", "-f", "-", "/var/lib/twr/server.log", data='\n'.join(secrets) + '\n')
            count = p.stdout.strip()
            ok = p.returncode in (0, 1) and count == "0"
            # Passwords/bearers do not prove absence of key material or codes absent from STATE.
            complete = self.state.get("security_secrets_complete") is True
            verdict = "FAIL" if not ok else ("PASS" if complete else "MANUAL")
            self.emit(6, verdict, f"known-secret matching lines={count if count.isdigit() else 'unavailable'}; "
                      + ("operator declared STATE secret inventory complete" if complete else
                         "coverage incomplete: supply all test passwords, bearers, enrollment/recovery codes and key material privately in STATE; set security_secrets_complete=true only after verifying inventory, then rerun"))

    def egress(self):
        if not self.args.lan_peer:
            self.emit(4, "MANUAL", "rerun with --lan-peer HOST:PORT (home-LAN peer with an open TCP port); "
                      "optionally --v6-peer '[IPv6]:PORT'; without v6 peer verify no global IPv6 route")
            return
        outcomes = []
        evidence = []
        for family, peer in (("-4", self.args.lan_peer), ("-6", self.args.v6_peer)):
            if peer is None:
                route = self.x("ip", "-6", "route", "show", "table", "all", "scope", "global")
                ok = route.returncode == 0 and not route.stdout.strip()
                outcomes.append(ok)
                evidence.append("no --v6-peer: global IPv6 route absent=" + str(ok))
                continue
            host, port = peer
            # A dead peer cannot establish lateral isolation. Require a lab-host control first.
            control = run(["nc", family, "-z", "-w", "3", host, port], timeout=5)
            blocked = self.x("nc", family, "-z", "-w", "3", host, port, timeout=5)
            outcomes.append(control.returncode == 0 and blocked.returncode == 1)
            evidence.append(f"nc {family} {host}:{port} host_control={control.returncode} container={blocked.returncode} (expected 0,1)")
        dns = self.x("nslookup", "example.com", "1.1.1.1", timeout=20)
        https = self.x("curl", "--disable", "--noproxy", "*", "--fail", "--silent", "--head",
                       "--connect-timeout", "5", "--max-time", "15", "https://example.com", timeout=20)
        outcomes.extend((dns.returncode == 0, https.returncode == 0))
        evidence.append(f"nslookup example.com 1.1.1.1 exit={dns.returncode}; HTTPS HEAD https://example.com exit={https.returncode} (expected 0,0)")
        self.emit(4, "PASS" if all(outcomes) else "FAIL", "; ".join(evidence))

    def lan(self):
        p = self.x("ip", "-4", "-o", "addr", "show", "scope", "global")
        addresses = re.findall(r"\binet ([0-9.]+)/", p.stdout)
        addresses = [a for a in addresses if ipaddress.ip_address(a).is_private]
        if p.returncode or len(addresses) != 1:
            self.emit(7, "MANUAL", f"find CT address: pct exec {self.args.ct} -- ip -4 -o addr show scope global; then nc -z -w 3 <CT-address> PORT for 22,8443,20241 (all fail)")
            return
        results = [run(["nc", "-z", "-w", "3", addresses[0], str(port)], timeout=5).returncode for port in (22, 8443, 20241)]
        self.emit(7, "PASS" if all(r == 1 for r in results) else "FAIL", f"nc -z -w 3 {addresses[0]} ports 22,8443,20241; exit codes=" + ','.join(map(str, results)))

    def public(self):
        paths = ("/v1/security-sweep-not-a-route", "/admin", "/metrics", "/status", "/healthz")
        codes = [self.curl(path)[0] for path in paths]
        self.emit(8, "PASS" if all(s == 404 for s in codes) else "FAIL", ', '.join(f"{p}={s}" for p, s in zip(paths, codes)))
        plain = self.curl("/", scheme="http")
        tls = self.curl("/", extra=("--tlsv1.2", "--tls-max", "1.2"))
        # Redirects are not plaintext application content; never follow them.
        plain_ok = (plain[0] in (301, 302, 307, 308) and re.search(r"(?im)^location: https://" + re.escape(self.args.host) + r"(?:/|\s|$)", plain[1])) or (plain[0] == 0 and plain[3] == 7)
        self.emit(9, "PASS" if plain_ok and tls[0] == 0 and tls[3] == 35 else "FAIL",
                  f"HTTP /={plain[0]} (HTTPS-only redirect permitted); curl --tlsv1.2 --tls-max 1.2 /={tls[0]} curl_exit={tls[3]} (must be handshake refusal)")
        owner = next((o for o in self.state.get("owners", []) if o.get("bearers")), None)
        self.throttle(owner)
        self.limits(owner)
        self.roles(owner)
        self.unauthenticated()
        s = self.curl("/join", "league=security-sweep&code=invalid", extra=("--header", "Origin: https://cross-site.invalid", "--header", "Content-Type: application/x-www-form-urlencoded"))[0]
        self.emit(14, "PASS" if s == 400 else "FAIL", f"POST /join Origin:https://cross-site.invalid form-urlencoded={s} (expected 400)")

    def throttle(self, owner):
        if not owner or not owner.get("user") or not owner.get("password"):
            self.emit(10, "MANUAL", "--state STATE account user/password required: POST /v1/auth/login 12 wrong passwords, require 429 + Retry-After, wait, then POST correct credentials (200)")
            return
        body = {"protocol": 1, "username": owner["user"], "password": "security-sweep-wrong-password", "device": "security sweep"}
        replies = [self.curl("/v1/auth/login", body) for _ in range(12)]
        retries = [re.search(r"(?im)^retry-after:\s*(\d+)\s*$", r[1]) for r in replies if r[0] == 429]
        waits = [int(m[1]) for m in retries if m]
        if not waits or len(waits) != len(retries) or max(waits) > 120:
            self.emit(10, "FAIL", "POST /v1/auth/login x12: missing/invalid 429 Retry-After (wait capped at 120s)")
            return
        time.sleep(max(waits) + 1)
        body["password"] = owner["password"]
        s = self.curl("/v1/auth/login", body)[0]
        ok = all(r[0] in (401, 429) for r in replies) and all(r[0] == 429 for r in replies[10:]) and s == 200
        self.emit(10, "PASS" if ok else "FAIL", f"POST /v1/auth/login x12 statuses=" + ','.join(str(r[0]) for r in replies) + f"; waited {max(waits)+1}s; correct sign-in={s}")

    def limits(self, owner):
        if not owner:
            self.emit(11, "MANUAL", "--state STATE bearer required: POST /v1/native/ledger with 300 KiB JSON body, 16 KiB X-Sweep header, then curl --limit-rate 1 --max-time 60 with incomplete JSON; require refusal and observed peer close, not client timeout")
            return
        token = owner["bearers"][0]
        path = "/v1/native/ledger"
        normal = {"protocol": 1, "league": self.state["league"]}
        if self.curl(path, normal, token)[0] != 200:
            self.emit(11, "MANUAL", "signed-in control POST /v1/native/ledger did not answer 200; restore healthy admission before limit checks")
            return
        body = self.curl(path, '{"protocol":1,"league":"' + 'x' * (300 * 1024) + '"}', token)
        header = self.curl(path, normal, token, extra=("--header", "X-Sweep: " + 'x' * (16 * 1024)))
        start = time.monotonic()
        slow = self.curl(path, json.dumps(normal) + ' ' * 1024, token, extra=("--limit-rate", "1"))
        elapsed = time.monotonic() - start
        ok = body[0] in (400, 413) and header[0] in (400, 431) and (slow[0] in (400, 408) or slow[3] in (52, 56)) and elapsed < 59
        verdict = "MANUAL" if slow[3] == 28 else ("PASS" if ok else "FAIL")
        self.emit(11, verdict, f"POST {path}: 300KiB={body[0]}, 16KiB header={header[0]}, slow={slow[0]} curl_exit={slow[3]} elapsed={elapsed:.1f}s; client timeout is not proof of server drop")

    def roles(self, owner):
        deputy = self.state.get("co_commissioner")
        target = owner.get("principal", "") if owner else ""
        if not owner or not deputy or not deputy.get("bearers") or not target.startswith("acct."):
            self.emit(12, "MANUAL", "missing STATE co_commissioner.bearers and/or owners[0].principal (known acct.<id> target); "
                      "also provide co_commissioner.recent_auth if TOTP-enabled. After supplying them rerun this script. "
                      "Commands: curl --config <private-role-check.curl> for POST /v1/native/people and /v1/native/ledger before/after; "
                      "POST /v1/native/preview and /v1/native/execute with command=commish.role.grant, "
                      "args.people={principal:<known-acct.id>,role:co-commissioner}, current expected revision, unique request, "
                      "reason, effective=now, visibility=private. Expect 403 preview / 409 execute and unchanged reads.")
            return
        outcomes = []
        for actor, expected_role in ((owner, "Owner"), (deputy, "Co-commissioner")):
            token, recent = actor["bearers"][0], actor.get("recent_auth")
            body = {"protocol": 1, "league": self.state["league"]}
            before = [self.curl(path, body, token, recent=recent) for path in ("/v1/native/people", "/v1/native/ledger")]
            if any(r[0] != 200 for r in before):
                self.emit(12, "MANUAL", "signed-in people/ledger controls failed; verify deputy recent_auth (TOTP step-up proof) and both bearers; rerun")
                return
            people, ledger = (json.loads(r[2]) for r in before)
            if people.get("yourRole") != expected_role or people.get("mayGrantRoles") is not False:
                self.emit(12, "FAIL", "POST /v1/native/people: STATE identity is not the required non-commissioner role")
                return
            revision = ledger["revision"]
            command = {"league": self.state["league"], "command": "commish.role.grant",
                       "args": {"people": {"principal": target, "role": "co-commissioner"}},
                       "expected": revision, "request": "security-sweep." + secrets.token_hex(12),
                       "reason": "Lab permission refusal check", "effective": "now", "visibility": "private"}
            raw = {"protocol": 1, "command": command}
            preview = self.curl("/v1/native/preview", raw, token, recent=recent)[0]
            if preview != 403:
                self.emit(12, "FAIL", f"{expected_role} role.grant preview={preview}; execute NOT sent (unsafe or inconclusive preview)")
                return
            execute = self.curl("/v1/native/execute", raw, token, recent=recent)[0]
            after = [self.curl(path, body, token, recent=recent) for path in ("/v1/native/people", "/v1/native/ledger")]
            unchanged = all(a[0] == 200 and json.loads(a[2]) == json.loads(b[2]) for a, b in zip(after, before))
            outcomes.append(execute == 409 and unchanged and people["revision"] == revision)
        self.emit(12, "PASS" if all(outcomes) else "FAIL", "owner + co-commissioner POST /v1/native/preview=403, /execute=409 for commish.role.grant; /people + /ledger unchanged" if all(outcomes) else "refusal or unchanged-state proof failed; inspect private controls")

    def unauthenticated(self):
        league = self.state.get("league", "native.security-sweep")
        raw = {"protocol": 1, "league": league}
        reads = ("/v1/native/ledger", "/v1/native/people")
        outcomes = [(p, self.curl(p, raw)[0]) for p in reads]
        workflow = {"protocol": 1, "workflow": {"command": "native.assets.read", "version": "1", "mode": "read", "arguments": {"league": league}}}
        outcomes.append(("/v1/native/workflow", self.curl("/v1/native/workflow", workflow)[0]))
        outcomes.append(("/v1/native/invalidations", self.curl("/v1/native/invalidations", extra=("--header", "X-TWR-League: " + league))[0]))
        self.emit(13, "PASS" if all(s == 401 for _, s in outcomes) else "FAIL", "unsigned reads: " + ', '.join(f"{p}={s}" for p, s in outcomes) + "; workflow commands share authenticateNative")

    def notes(self):
        print("NOTE throttle restart: internal/serverhttp/limits.go stores windows in Handler memory; new process clears them. Live restart behaviour not measured.")
        print("NOTE token lifetimes (internal/accounts): device idle 7d / absolute 30d; agent read 30d / execute 24h; enrollment 48h; TOTP setup 10m; recent auth 5m; join links have no time expiry (revocable); recovery codes are single-use, no time expiry; human command consent 5m (internal/serverhttp/confirmation.go).")
        print("NOTE outside hosts: Cloudflare API/Tunnel; DNS 1.1.1.1; ESPN/Sleeper live feeds; github.com nflverse official stats; configured R2 endpoint and release metadata/targets hosts. Exact runtime inventory MANUAL: audit deployed command/config and DNS/egress logs privately; never print credential/config contents.")


def main(argv):
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument("ct")
    p.add_argument("host")
    p.add_argument("--state")
    p.add_argument("--lan-peer", type=peer_address, help="home-LAN host:port with an open TCP port")
    p.add_argument("--v6-peer", type=peer_address, help="reachable IPv6 host:port; otherwise require no global IPv6 route")
    p.add_argument("--with-reboot", action="store_true")
    args = p.parse_args(argv)
    try:
        validate_target(args.ct, args.host)
        if socket.gethostname() != "twr-lab":
            raise ValueError("refused: run only on lab host twr-lab")
        conf = run(["pct", "config", args.ct])
        if conf.returncode or not re.search(r"(?m)^hostname:\s*twr-lab-" + args.ct + r"\s*$", conf.stdout):
            raise ValueError("refused: CT hostname must be twr-lab-<ctid>")
        state = {}
        if args.state:
            with open(args.state) as f:
                state = json.load(f)
            if not isinstance(state, dict):
                raise ValueError("refused: STATE must be an object")
    except (ValueError, OSError):
        # Fixed diagnostics cannot echo credentials from a malformed STATE or URL.
        reason = "refused: protected/non-lab CT, mismatched lab hostname, invalid STATE, or not on host twr-lab"
        print(reason, file=sys.stderr)
        return 2
    sweep = Sweep(args, state)
    # Keep one line per check even if a utility is absent; never emit captured stderr.
    try:
        sweep.console()
        sweep.lan()
        sweep.public()
    except (OSError, subprocess.TimeoutExpired, ValueError, KeyError, TypeError):
        missing = [n for n in range(1, 15) if n not in sweep.emitted]
        for i, number in enumerate(missing):
            sweep.emit(number, "FAIL" if i == 0 else "MANUAL",
                       "required utility, valid STATE, or bounded command unavailable; rerun security-sweep.sh after fixing prerequisites")
    sweep.notes()
    return int(sweep.failed)


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
