"""Offline tests for season-install.sh and season-uninstall.sh.

Run: python3 -m unittest discover -s deploy/season -v
Executes under strict POSIX `sh` with local stubs; zero network or remote host contact.
"""
import hashlib
import io
import os
from pathlib import Path
import shutil
import stat
import subprocess
import tarfile
import tempfile
import unittest

HERE = Path(__file__).resolve().parent
INSTALL = HERE / "season-install.sh"
UNINSTALL = HERE / "season-uninstall.sh"
SWAP = HERE / "season-swap.sh"

STUB_SSH_LOG = """#!/bin/sh
printf "ssh %s\\n" "$*" >> "$FAKE_NET_LOG"
exit 0
"""

STUB_SCP_LOG = """#!/bin/sh
printf "scp %s\\n" "$*" >> "$FAKE_NET_LOG"
exit 0
"""


def make_bundle(path: Path, *, include_binary=True, include_registry=True) -> str:
    with tarfile.open(path, "w:gz") as tf:
        if include_binary:
            data = b"#!/bin/sh\necho 'Server build v107'\n"
            info = tarfile.TarInfo("./twr-server")
            info.size = len(data)
            info.mode = 0o755
            tf.addfile(info, io.BytesIO(data))
        if include_registry:
            reg = b"endpoint,method\n/v1/native/ledger,POST\n"
            info = tarfile.TarInfo("./endpoint-registry.csv")
            info.size = len(reg)
            info.mode = 0o644
            tf.addfile(info, io.BytesIO(reg))
    return hashlib.sha256(path.read_bytes()).hexdigest()


def write_stub(directory: Path, name: str, body: str) -> Path:
    p = directory / name
    p.write_text("#!/bin/sh\n" + body + "\n", encoding="utf-8")
    p.chmod(0o755)
    return p


class SeasonInstallGuardTest(unittest.TestCase):
    def setUp(self):
        self.tmp = Path(tempfile.mkdtemp())
        self.bin = self.tmp / "bin"
        self.bin.mkdir()
        self.net_log = self.tmp / "net.log"
        write_stub(self.bin, "ssh", STUB_SSH_LOG)
        write_stub(self.bin, "scp", STUB_SCP_LOG)
        self.bundle = self.tmp / "twr-server-season-r11-a43521f7.tar.gz"
        self.bundle_sha = make_bundle(self.bundle)
        self.lab_bundle = self.tmp / "twr-server-LAB-ONLY-r11-a43521f7.tar.gz"
        self.lab_sha = make_bundle(self.lab_bundle)

    def tearDown(self):
        shutil.rmtree(self.tmp, ignore_errors=True)

    def net_calls(self):
        if not self.net_log.exists():
            return []
        return self.net_log.read_text(encoding="utf-8").splitlines()

    def run_script(self, script: Path, now: str, *args):
        env = dict(os.environ)
        env["PATH"] = str(self.bin) + os.pathsep + env.get("PATH", "")
        env["FAKE_NET_LOG"] = str(self.net_log)
        env["SEASON_NOW"] = now
        return subprocess.run(
            ["sh", str(script), *args],
            env=env, capture_output=True, text=True, timeout=30, check=False,
        )

    def base_args(self, bundle=None):
        return [
            "--target", "root@192.0.2.10",
            "--version", "v107",
            "--bundle", str(bundle or self.bundle),
        ]

    def test_a_missing_target_refused_with_no_default(self):
        proc = self.run_script(
            INSTALL, "2026-11-28T19:00",
            "--version", "v107", "--bundle", str(self.bundle), "--dry-run",
        )
        self.assertEqual(proc.returncode, 2, proc.stdout + proc.stderr)
        self.assertIn("--target", proc.stderr)
        self.assertEqual(self.net_calls(), [])

    def test_b_bad_target_and_bad_version_refused(self):
        for bad_target in ("192.0.2.10", "-oProxyCommand=id@host", "root@a b"):
            proc = self.run_script(
                INSTALL, "2026-11-28T19:00",
                "--target", bad_target, "--version", "v107",
                "--bundle", str(self.bundle), "--dry-run",
            )
            self.assertEqual(proc.returncode, 2, f"target={bad_target}: {proc.stderr}")
        proc = self.run_script(
            INSTALL, "2026-11-28T19:00",
            "--target", "root@192.0.2.10", "--version", "../etc",
            "--bundle", str(self.bundle), "--dry-run",
        )
        self.assertEqual(proc.returncode, 2, proc.stderr)
        self.assertEqual(self.net_calls(), [])

    def test_c_guard_refusals_outside_and_inside_saturday_window(self):
        # Sunday without OK -> exit 3
        p = self.run_script(INSTALL, "2026-11-29T14:00", *self.base_args())
        self.assertEqual(p.returncode, 3, p.stdout + p.stderr)
        self.assertIn("--christopher-ok", p.stdout)
        self.assertNotIn("== 1/7", p.stdout)

        # Sunday WITH --christopher-ok (real run) -> STILL REFUSED (exit 3)!
        p = self.run_script(INSTALL, "2026-11-29T14:00", *self.base_args(), "--christopher-ok")
        self.assertEqual(p.returncode, 3, p.stdout + p.stderr)

        # Saturday 17:59 WITH --christopher-ok (real run) -> REFUSED (exit 3)!
        p = self.run_script(INSTALL, "2026-11-28T17:59", *self.base_args(), "--christopher-ok")
        self.assertEqual(p.returncode, 3, p.stdout + p.stderr)

        # Saturday 17:59 dry-run without OK -> REFUSED (exit 3), 18:00 dry-run -> ALLOWED (exit 0)
        self.assertEqual(
            self.run_script(INSTALL, "2026-11-28T17:59", *self.base_args(), "--dry-run").returncode,
            3,
        )
        self.assertEqual(
            self.run_script(INSTALL, "2026-11-28T18:00", *self.base_args(), "--dry-run").returncode,
            0,
        )

        # Saturday 19:00 real run WITHOUT --christopher-ok -> REFUSED (exit 3)
        p = self.run_script(INSTALL, "2026-11-28T19:00", *self.base_args())
        self.assertEqual(p.returncode, 3, p.stdout + p.stderr)

        # Tuesday dry-run WITH --christopher-ok -> ALLOWED (exit 0, zero SSH)
        p = self.run_script(
            INSTALL, "2026-11-24T10:00", *self.base_args(), "--christopher-ok", "--dry-run"
        )
        self.assertEqual(p.returncode, 0, p.stdout + p.stderr)

        self.assertEqual(self.net_calls(), [], "no ssh/scp call may run on refused or dry runs")

    def test_d_lab_only_bundle_refused_unless_explicitly_allowed(self):
        p = self.run_script(
            INSTALL, "2026-11-28T19:00", *self.base_args(self.lab_bundle), "--dry-run"
        )
        self.assertEqual(p.returncode, 2, p.stdout + p.stderr)
        self.assertIn("LAB-ONLY", p.stderr)
        self.assertIn("--allow-lab-artifact", p.stderr)

        p2 = self.run_script(
            INSTALL, "2026-11-28T19:00",
            *self.base_args(self.lab_bundle), "--allow-lab-artifact", "--dry-run",
        )
        self.assertEqual(p2.returncode, 0, p2.stdout + p2.stderr)
        self.assertIn("--allow-lab-artifact --christopher-ok", p2.stdout)
        self.assertEqual(self.net_calls(), [])

    def test_e_sha256_mismatch_refused_before_any_remote_call(self):
        bad_sha = "0" * 64
        p = self.run_script(
            INSTALL, "2026-11-28T19:00",
            *self.base_args(), "--sha256", bad_sha, "--dry-run",
        )
        self.assertEqual(p.returncode, 2, p.stdout + p.stderr)
        self.assertIn("does not match --sha256", p.stderr)
        self.assertEqual(self.net_calls(), [])

    def test_f_dry_run_prints_every_step_and_remote_command_without_sending(self):
        p = self.run_script(
            INSTALL, "2026-11-28T19:00",
            *self.base_args(), "--sha256", self.bundle_sha, "--dry-run",
        )
        self.assertEqual(p.returncode, 0, p.stdout + p.stderr)
        for n in range(1, 8):
            self.assertIn(f"== {n}/7", p.stdout)
        self.assertIn("ssh -o BatchMode=yes -o ConnectTimeout=10 root@192.0.2.10", p.stdout)
        self.assertIn("scp -q -o BatchMode=yes -o ConnectTimeout=10", p.stdout)
        self.assertIn("https://127.0.0.1:5199/app/", p.stdout)
        self.assertIn("season-uninstall.sh --target root@192.0.2.10 --christopher-ok", p.stdout)
        self.assertEqual(self.net_calls(), [])

    def test_g_uninstall_guard_and_missing_target(self):
        p = self.run_script(UNINSTALL, "2026-11-28T19:00", "--dry-run")
        self.assertEqual(p.returncode, 2)
        p = self.run_script(
            UNINSTALL, "2026-11-29T14:00", "--target", "root@192.0.2.10", "--christopher-ok"
        )
        self.assertEqual(p.returncode, 3)
        p = self.run_script(
            UNINSTALL, "2026-11-28T19:00", "--target", "root@192.0.2.10", "--dry-run"
        )
        self.assertEqual(p.returncode, 0, p.stdout + p.stderr)
        self.assertIn("== 1/4", p.stdout)
        self.assertIn("== 4/4", p.stdout)
        self.assertEqual(self.net_calls(), [])


class SeasonInstallOfflineExecutionTest(unittest.TestCase):
    """Execute the real remote install/uninstall payloads locally via stub commands."""

    def setUp(self):
        self.tmp = Path(tempfile.mkdtemp())
        self.bin = self.tmp / "bin"
        self.bin.mkdir()
        self.fs = self.tmp / "host"
        self.fs.mkdir()
        self.app = self.fs / "opt_twr"
        self.data = self.fs / "var_lib_twr"
        self.backups = self.fs / "var_backups_twr"
        self.etc = self.fs / "etc_twr"
        self.initd = self.fs / "etc_initd"
        self.logrotate = self.fs / "etc_logrotate"
        self.periodic = self.fs / "etc_periodic"
        self.stage = self.fs / "root_stage"
        self.usr_local = self.fs / "usr_local"
        self.os_release = self.fs / "os-release"
        self.os_release.write_text('ID=alpine\nVERSION_ID=3.24.4\n', encoding="utf-8")
        # Pre-existing laptop file (must NOT cause refusal and must survive uninstall).
        self.sysctl = self.fs / "10-twr.conf"
        self.sysctl.write_text("# pre-existing sysctl\n", encoding="utf-8")

        self.bundle = self.tmp / "twr-server-season-r11-a43521f7.tar.gz"
        self.bundle_sha = make_bundle(self.bundle)
        self.state_user = self.tmp / "twr_user_exists"
        self.service_log = self.tmp / "service.log"
        self.curl_log = self.tmp / "curl.log"
        self.apk_log = self.tmp / "apk.log"

        self._install_stubs()

    def tearDown(self):
        shutil.rmtree(self.tmp, ignore_errors=True)

    def _install_stubs(self):
        # Local ssh stub runs the remote command string directly with POSIX sh.
        write_stub(self.bin, "ssh", 'for arg do cmd="$arg"; done\nexec sh -c "$cmd"')
        # Local scp stub copies local file to destination path after ':'.
        write_stub(
            self.bin, "scp",
            'prev=""; for arg do src="$prev"; prev="$arg"; done\n'
            'dst=${prev#*:}\n'
            'cp "$src" "$dst"',
        )
        # Stub id: -u -> 0 (root); id twr -> succeeds only if state_user marker exists.
        write_stub(
            self.bin, "id",
            'if [ "${1:-}" = "-u" ]; then echo 0; exit 0; fi\n'
            'if [ "${1:-}" = "twr" ]; then [ -f "$STUB_TWR_USER" ]; exit $?; fi\n'
            'exit 1',
        )
        write_stub(self.bin, "uname", 'if [ "${1:-}" = "-m" ]; then echo x86_64; else echo Linux; fi')
        write_stub(self.bin, "apk", 'echo "$*" >> "$STUB_APK_LOG"')
        write_stub(self.bin, "addgroup", 'exit 0')
        write_stub(self.bin, "adduser", ': > "$STUB_TWR_USER"')
        write_stub(self.bin, "deluser", 'rm -f "$STUB_TWR_USER"')
        write_stub(self.bin, "delgroup", 'exit 0')
        write_stub(self.bin, "chown", 'exit 0')
        # Stub install: honors -d and -m, ignores -o/-g so non-root tests succeed.
        write_stub(
            self.bin, "install",
            'dir_mode=0; mode=""; args=""\n'
            'while [ "$#" -gt 0 ]; do\n'
            '  case "$1" in\n'
            '    -d) dir_mode=1; shift ;;\n'
            '    -m) mode=$2; shift 2 ;;\n'
            '    -o|-g) shift 2 ;;\n'
            '    *) break ;;\n'
            '  esac\n'
            'done\n'
            'if [ "$dir_mode" = 1 ]; then\n'
            '  for d in "$@"; do mkdir -p "$d"; [ -z "$mode" ] || chmod "$mode" "$d"; done\n'
            'else\n'
            '  cp "$1" "$2"; [ -z "$mode" ] || chmod "$mode" "$2"\n'
            'fi',
        )
        write_stub(
            self.bin, "openssl",
            'if [ "${1:-}" = "req" ]; then\n'
            '  keyout=""; out=""\n'
            '  while [ "$#" -gt 0 ]; do\n'
            '    case "$1" in -keyout) keyout=$2; shift 2 ;; -out) out=$2; shift 2 ;; *) shift ;; esac\n'
            '  done\n'
            '  printf "FAKE-KEY\\n" > "$keyout"\n'
            '  printf "FAKE-CERT\\n" > "$out"\n'
            'elif [ "${1:-}" = "rand" ]; then\n'
            '  out=""\n'
            '  while [ "$#" -gt 0 ]; do\n'
            '    case "$1" in -out) out=$2; shift 2 ;; *) shift ;; esac\n'
            '  done\n'
            '  printf "0123456789abcdef0123456789abcdef" > "$out"\n'
            'fi',
        )
        write_stub(self.bin, "rc-update", 'echo "rc-update $*" >> "$STUB_SVC_LOG"')
        write_stub(self.bin, "rc-service", 'echo "rc-service $*" >> "$STUB_SVC_LOG"')
        write_stub(
            self.bin, "curl",
            'echo "$*" >> "$STUB_CURL_LOG"\n'
            'case "$*" in\n'
            '  *Host:*) printf "%s" "${STUB_CURL_CODE:-200}" ;;\n'
            '  *) printf "%s" "${STUB_CURL_CODE:-404}" ;;\n'
            'esac',
        )

    def make_env(self, now="2026-11-28T19:00", curl_code=None):
        env = dict(
            os.environ,
            PATH=str(self.bin) + os.pathsep + os.environ["PATH"],
            SEASON_NOW=now,
            SEASON_APP_ROOT=str(self.app),
            SEASON_DATA_DIR=str(self.data),
            SEASON_BACKUP_DIR=str(self.backups),
            SEASON_ETC_DIR=str(self.etc),
            SEASON_INIT_DIR=str(self.initd),
            SEASON_LOGROTATE_DIR=str(self.logrotate),
            SEASON_PERIODIC_DIR=str(self.periodic),
            SEASON_STAGE_DIR=str(self.stage),
            SEASON_USR_LOCAL_DIR=str(self.usr_local),
            SEASON_OS_RELEASE=str(self.os_release),
            STUB_TWR_USER=str(self.state_user),
            STUB_SVC_LOG=str(self.service_log),
            STUB_CURL_LOG=str(self.curl_log),
            STUB_APK_LOG=str(self.apk_log),
        )
        if curl_code is not None:
            env["STUB_CURL_CODE"] = curl_code
        return env

    def run_install(self, *extra_args, env=None):
        return subprocess.run(
            [
                "sh", str(INSTALL),
                "--target", "root@192.0.2.10",
                "--version", "v107",
                "--bundle", str(self.bundle),
                "--sha256", self.bundle_sha,
                *extra_args,
            ],
            env=env or self.make_env(),
            capture_output=True, text=True, timeout=30, check=False,
        )

    def test_full_install_layout_matches_season_swap_and_uninstall(self):
        proc = self.run_install("--christopher-ok")
        self.assertEqual(proc.returncode, 0, proc.stdout + proc.stderr)

        # 1. Exact layout expected by season-swap.sh
        self.assertEqual(os.readlink(self.app / "current"), str(self.app / "versions/v107"))
        self.assertTrue((self.app / "versions/v107/twr-server").is_file())
        self.assertTrue((self.app / "versions/v107/endpoint-registry.csv").is_file())
        self.assertTrue(self.data.is_dir())
        self.assertTrue(self.backups.is_dir())
        self.assertTrue((self.data / "server.db").is_file())
        self.assertFalse(self.stage.exists(), "staging directory must be cleaned up")

        # 2. Private permissions & colocated /etc/twr files (backup.go invariant)
        self.assertEqual(stat.S_IMODE(self.data.stat().st_mode), 0o700)
        self.assertEqual(stat.S_IMODE(self.backups.stat().st_mode), 0o700)
        self.assertEqual(stat.S_IMODE(self.etc.stat().st_mode), 0o750)
        self.assertEqual(stat.S_IMODE((self.data / "server.db").stat().st_mode), 0o600)
        for keyfile in ("origin.key", "authenticator.key"):
            p = self.etc / keyfile
            self.assertTrue(p.is_file(), keyfile)
            self.assertEqual(stat.S_IMODE(p.stat().st_mode), 0o600)
        self.assertEqual(len((self.etc / "authenticator.key").read_bytes()), 32)
        for cfg in ("origin.pem", "endpoint-registry.csv"):
            self.assertTrue((self.etc / cfg).is_file(), cfg)
            self.assertEqual(stat.S_IMODE((self.etc / cfg).stat().st_mode), 0o640)

        # 3. Hardened sandbox, backup helper, status helper, OpenRC service named twr
        sandbox = self.app / "libexec/twr-sandbox"
        backup_setup = self.app / "libexec/twr-backup-setup"
        status_helper = self.app / "libexec/twr-status"
        init_script = self.initd / "twr"
        for gen in (sandbox, backup_setup, status_helper):
            self.assertTrue(gen.is_file(), str(gen))
            chk = subprocess.run(["sh", "-n", str(gen)], capture_output=True, text=True, check=False)
            self.assertEqual(chk.returncode, 0, f"syntax error in {gen}: {chk.stderr}")
        sandbox_text = sandbox.read_text(encoding="utf-8")
        self.assertIn("bwrap --unshare-all --share-net --die-with-parent --new-session --cap-drop ALL", sandbox_text)
        self.assertIn(f"{self.app}/current/twr-server hosted -listen 127.0.0.1:5199", sandbox_text)
        self.assertIn(f"--ro-bind {self.app} {self.app}", sandbox_text)
        self.assertIn(f"--bind {self.data} {self.data}", sandbox_text)
        backup_text = backup_setup.read_text(encoding="utf-8")
        self.assertIn(f"{self.app}/current/twr-server backup-check", backup_text)
        self.assertIn("rc-service twr", backup_text)
        init_text = init_script.read_text(encoding="utf-8")
        self.assertIn('command_user="twr:twr"', init_text)
        self.assertIn(f'command="{self.app}/libexec/twr-sandbox"', init_text)

        # 4. Packages installed without removing openssh-server; service started; health probed
        apk_calls = self.apk_log.read_text(encoding="utf-8")
        self.assertIn("curl", apk_calls)
        self.assertIn("bubblewrap", apk_calls)
        self.assertNotIn("del openssh", apk_calls)
        svc_lines = self.service_log.read_text(encoding="utf-8").splitlines()
        self.assertIn("rc-update add twr default", svc_lines)
        self.assertIn("rc-service twr start", svc_lines)
        curl_lines = self.curl_log.read_text(encoding="utf-8")
        self.assertIn("https://127.0.0.1:5199/app/", curl_lines)
        self.assertIn(f"--cacert {self.etc}/origin.pem", curl_lines)

        # 5. Refuse-if-present: re-running install must fail at step 1 without touching keys
        orig_key = (self.etc / "origin.key").read_bytes()
        again = self.run_install("--christopher-ok")
        self.assertEqual(again.returncode, 1, again.stdout + again.stderr)
        self.assertIn("existing TheWarRoom path found", again.stderr)
        self.assertEqual((self.etc / "origin.key").read_bytes(), orig_key)

        # 6. Direct compatibility check: season-swap.sh can swap from this installed layout!
        (self.app / "versions/v108").mkdir()
        swap_proc = subprocess.run(
            ["bash", str(SWAP), "v108", "--christopher-ok"],
            env=self.make_env(), capture_output=True, text=True, timeout=30, check=False,
        )
        self.assertEqual(swap_proc.returncode, 0, swap_proc.stdout + swap_proc.stderr)
        self.assertEqual(os.readlink(self.app / "current"), str(self.app / "versions/v108"))
        self.assertTrue(list(self.backups.glob("twr-data-*.tgz")))

        # 7. Default uninstall preserves data/backups, removes program/config/service
        un_keep = subprocess.run(
            ["sh", str(UNINSTALL), "--target", "root@192.0.2.10", "--christopher-ok"],
            env=self.make_env(), capture_output=True, text=True, timeout=30, check=False,
        )
        self.assertEqual(un_keep.returncode, 0, un_keep.stdout + un_keep.stderr)
        self.assertFalse(self.app.exists())
        self.assertFalse(self.etc.exists())
        self.assertFalse((self.initd / "twr").exists())
        self.assertTrue(self.data.exists(), "default uninstall must preserve data dir")
        self.assertTrue(self.backups.exists(), "default uninstall must preserve backup dir")
        self.assertTrue(self.sysctl.exists(), "pre-existing host files must stay untouched")

        # 8. --purge-data removes remaining state & user, leaving host clean for reinstall
        un_purge = subprocess.run(
            ["sh", str(UNINSTALL), "--target", "root@192.0.2.10", "--purge-data", "--christopher-ok"],
            env=self.make_env(), capture_output=True, text=True, timeout=30, check=False,
        )
        self.assertEqual(un_purge.returncode, 0, un_purge.stdout + un_purge.stderr)
        self.assertFalse(self.data.exists())
        self.assertFalse(self.backups.exists())
        self.assertFalse(self.state_user.exists())
        self.assertTrue(self.sysctl.exists())

    def test_refuse_if_any_individual_path_already_exists(self):
        for label, create in (
            ("data_dir", lambda: self.data.mkdir()),
            ("etc_dir", lambda: self.etc.mkdir()),
            ("twr_server_init", lambda: (self.initd.mkdir(parents=True, exist_ok=True), (self.initd / "twr-server").write_text("x"))),
            ("usr_local_bin", lambda: ((self.usr_local / "bin").mkdir(parents=True, exist_ok=True), (self.usr_local / "bin/twr-server").write_text("x"))),
            ("twr_user", lambda: self.state_user.write_text("")),
        ):
            with self.subTest(existing=label):
                # Clean host dir per subtest
                shutil.rmtree(self.fs)
                self.fs.mkdir()
                self.os_release.write_text('ID=alpine\nVERSION_ID=3.24.4\n', encoding="utf-8")
                self.state_user.unlink(missing_ok=True)
                create()
                proc = self.run_install("--christopher-ok")
                self.assertEqual(proc.returncode, 1, proc.stdout + proc.stderr)
                self.assertIn("Refused:", proc.stderr)
                self.assertFalse(self.app.exists(), f"{label}: must refuse before creating app root")

    def test_public_host_install_checks_http_200_with_host_header(self):
        proc = self.run_install("--public-host", "season.example.org", "--christopher-ok")
        self.assertEqual(proc.returncode, 0, proc.stdout + proc.stderr)
        self.assertEqual((self.etc / "public-host").read_text(encoding="utf-8").strip(), "season.example.org")
        curl_lines = self.curl_log.read_text(encoding="utf-8")
        self.assertIn("-H Host: season.example.org", curl_lines)


if __name__ == "__main__":
    unittest.main()
