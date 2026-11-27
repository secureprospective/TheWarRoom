#!/bin/sh
# First-install script for the bare-metal Alpine Season server. Read deploy/season/README.md first.
# Runs from the Beelink over SSH/SCP; pure POSIX sh. Target must be typed explicitly (never defaulted).
# Usage:
#   season-install.sh --target <user@host> --version <name> --bundle <tar.gz>
#     [--sha256 <64-hex>] [--public-host <dns-name>] [--allow-lab-artifact]
#     [--dry-run] [--christopher-ok]
# Exit: 0 done (or dry run), 1 a step failed, 2 bad usage or artifact refusal, 3 refused by guard.
set -eu

APP_ROOT=${SEASON_APP_ROOT:-/opt/twr}
DATA_DIR=${SEASON_DATA_DIR:-/var/lib/twr}
BACKUP_DIR=${SEASON_BACKUP_DIR:-/var/backups/twr}
ETC_DIR=${SEASON_ETC_DIR:-/etc/twr}
INIT_DIR=${SEASON_INIT_DIR:-/etc/init.d}
LOGROTATE_DIR=${SEASON_LOGROTATE_DIR:-/etc/logrotate.d}
PERIODIC_DIR=${SEASON_PERIODIC_DIR:-/etc/periodic/hourly}
STAGE_DIR=${SEASON_STAGE_DIR:-/root/.twr-install-stage}
USR_LOCAL_DIR=${SEASON_USR_LOCAL_DIR:-/usr/local}
OS_RELEASE=${SEASON_OS_RELEASE:-/etc/os-release}
PORT=${SEASON_PORT:-5199}
SERVICE="twr"

usage() {
  cat >&2 <<EOF
usage: $0 --target <user@host> --version <name> --bundle <path.tar.gz> \\
         [--sha256 <64-hex>] [--public-host <dns-name>] [--allow-lab-artifact] \\
         [--dry-run] [--christopher-ok]
EOF
  exit 2
}

TARGET=""
VERSION=""
BUNDLE=""
EXPECTED_SHA=""
PUBLIC_HOST=""
ALLOW_LAB=0
DRY_RUN=0
CHRISTOPHER_OK=0

while [ "$#" -gt 0 ]; do
  case "$1" in
    --target)
      [ "$#" -ge 2 ] || usage
      TARGET=$2
      shift 2
      ;;
    --version)
      [ "$#" -ge 2 ] || usage
      VERSION=$2
      shift 2
      ;;
    --bundle)
      [ "$#" -ge 2 ] || usage
      BUNDLE=$2
      shift 2
      ;;
    --sha256)
      [ "$#" -ge 2 ] || usage
      EXPECTED_SHA=$2
      shift 2
      ;;
    --public-host)
      [ "$#" -ge 2 ] || usage
      PUBLIC_HOST=$2
      shift 2
      ;;
    --allow-lab-artifact)
      ALLOW_LAB=1
      shift
      ;;
    --dry-run)
      DRY_RUN=1
      shift
      ;;
    --christopher-ok)
      CHRISTOPHER_OK=1
      shift
      ;;
    -h | --help)
      usage
      ;;
    *)
      echo "unknown option: $1" >&2
      usage
      ;;
  esac
done

[ -n "$TARGET" ] || { echo "missing required --target <user@host> (never defaulted)" >&2; usage; }
[ -n "$VERSION" ] || { echo "missing required --version <version-dir-name>" >&2; usage; }
[ -n "$BUNDLE" ] || { echo "missing required --bundle <path.tar.gz>" >&2; usage; }

valid_target() {
  case "$1" in
    '' | -* | *@*@* | *[!A-Za-z0-9._@-]*) return 1 ;;
    ?*@?*) ;;
    *) return 1 ;;
  esac
  t_user=${1%@*}
  t_host=${1#*@}
  case "$t_user" in '' | -* | *[!A-Za-z0-9._-]*) return 1 ;; esac
  case "$t_host" in '' | -* | *[!A-Za-z0-9._-]*) return 1 ;; esac
  return 0
}

valid_version() {
  case "$1" in
    '' | [!A-Za-z0-9]* | *..* | *[!A-Za-z0-9._-]*) return 1 ;;
  esac
  return 0
}

valid_abs_path() {
  case "$1" in
    '' | / | [!\/]* | *..* | *//* | *[!A-Za-z0-9/._-]*) return 1 ;;
  esac
  return 0
}

valid_port() {
  case "$1" in
    '' | *[!0-9]*) return 1 ;;
  esac
  [ "$1" -ge 1 ] && [ "$1" -le 65535 ]
}

valid_sha256() {
  [ "${#1}" -eq 64 ] || return 1
  case "$1" in *[!0-9a-f]*) return 1 ;; esac
  return 0
}

valid_public_host() {
  case "$1" in '' | .* | *. | *..* | *[!a-z0-9.-]*) return 1 ;; esac
  [ "${#1}" -le 253 ] || return 1
  printf '%s\n' "$1" | awk -F. '
    NF < 2 { exit 1 }
    {
      for (i = 1; i <= NF; i++) {
        if (length($i) == 0 || length($i) > 63 || substr($i, 1, 1) == "-" || substr($i, length($i), 1) == "-") exit 1
      }
    }
    NF == 4 {
      ip = 1
      for (i = 1; i <= NF; i++) if ($i !~ /^[0-9]+$/ || $i + 0 > 255) ip = 0
      if (ip) exit 1
    }
  '
}

valid_target "$TARGET" || { echo "invalid --target (expected user@host): $TARGET" >&2; exit 2; }
valid_version "$VERSION" || { echo "version name must be a plain folder name (letters, digits, . _ -): $VERSION" >&2; exit 2; }
valid_port "$PORT" || { echo "invalid port: $PORT" >&2; exit 2; }

for p in "$APP_ROOT" "$DATA_DIR" "$BACKUP_DIR" "$ETC_DIR" "$INIT_DIR" "$LOGROTATE_DIR" "$PERIODIC_DIR" "$STAGE_DIR" "$USR_LOCAL_DIR" "$OS_RELEASE"; do
  valid_abs_path "$p" || { echo "paths must be plain absolute paths, not /: $p" >&2; exit 2; }
done

if [ -n "$PUBLIC_HOST" ]; then
  valid_public_host "$PUBLIC_HOST" || { echo "invalid --public-host (bare lowercase DNS hostname required): $PUBLIC_HOST" >&2; exit 2; }
fi

if [ -n "$EXPECTED_SHA" ]; then
  valid_sha256 "$EXPECTED_SHA" || { echo "invalid --sha256 (64 lowercase hex characters required)" >&2; exit 2; }
fi

# Sets NOW_DOW (1=Mon .. 7=Sun), NOW_HOUR (0-23) and NOW_TEXT in America/Chicago time.
read_clock() {
  if [ -n "${SEASON_NOW:-}" ]; then
    case "$SEASON_NOW" in
      [0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9]T[0-9][0-9]:[0-9][0-9]) ;;
      *) echo "SEASON_NOW must look like 2026-11-28T19:00 (Chicago time)" >&2; exit 2 ;;
    esac
    now_date=${SEASON_NOW%%T*}
    now_time=${SEASON_NOW#*T}
    now_h=${now_time%%:*}
    now_m=${now_time#*:}
    now_h=${now_h#0}; NOW_HOUR=${now_h:-0}
    now_m=${now_m#0}; now_min=${now_m:-0}
    if [ "$NOW_HOUR" -gt 23 ] || [ "$now_min" -gt 59 ]; then
      echo "SEASON_NOW time out of range: $SEASON_NOW" >&2
      exit 2
    fi
    NOW_DOW=$(date -d "$now_date" +%u) || { echo "invalid SEASON_NOW date: $now_date" >&2; exit 2; }
    NOW_TEXT="$SEASON_NOW (SEASON_NOW test override, Chicago time)"
  else
    NOW_TEXT=$(TZ=America/Chicago date '+%a %F %H:%M %Z')
    case "$NOW_TEXT" in
      *CST* | *CDT*) ;;
      *) echo "cannot read America/Chicago time (is tzdata installed?)" >&2; exit 1 ;;
    esac
    NOW_DOW=$(TZ=America/Chicago date '+%u')
    now_h=$(TZ=America/Chicago date '+%H')
    now_h=${now_h#0}; NOW_HOUR=${now_h:-0}
  fi
}

print_approval() {
  cmd="$0 --target $TARGET --version $VERSION --bundle $BUNDLE"
  [ -z "$EXPECTED_SHA" ] || cmd="$cmd --sha256 $EXPECTED_SHA"
  [ -z "$PUBLIC_HOST" ] || cmd="$cmd --public-host $PUBLIC_HOST"
  [ "$ALLOW_LAB" -eq 0 ] || cmd="$cmd --allow-lab-artifact"
  echo "Exact command for Christopher to approve (run only inside Saturday 18:00-23:59 Chicago after he says yes):"
  echo "  $cmd --christopher-ok"
}

# Guard runs before inspecting or contacting any remote host.
# Real installs require BOTH --christopher-ok AND Saturday 18:00-23:59 America/Chicago.
# Dry runs inside the window need no flag; outside the window a dry run needs --christopher-ok.
read_clock
IN_WINDOW=0
if [ "$NOW_DOW" = 6 ] && [ "$NOW_HOUR" -ge 18 ] && [ "$NOW_HOUR" -le 23 ]; then
  IN_WINDOW=1
fi

if [ "$DRY_RUN" -eq 0 ]; then
  if [ "$CHRISTOPHER_OK" -eq 0 ] || [ "$IN_WINDOW" -eq 0 ]; then
    print_approval
    echo "Season server installs require BOTH --christopher-ok and the Saturday 18:00-23:59" >&2
    echo "America/Chicago window; it is now $NOW_TEXT. Nothing was sent to any host." >&2
    exit 3
  fi
else
  if [ "$CHRISTOPHER_OK" -eq 0 ] && [ "$IN_WINDOW" -eq 0 ]; then
    print_approval
    echo "Season server dry runs outside Saturday 18:00-23:59 America/Chicago require" >&2
    echo "--christopher-ok; it is now $NOW_TEXT. Nothing was sent to any host." >&2
    exit 3
  fi
fi

# Artifact checks run locally before any remote command.
[ -f "$BUNDLE" ] && [ -r "$BUNDLE" ] || { echo "bundle file is missing or unreadable: $BUNDLE" >&2; exit 2; }
BUNDLE_BASE=$(basename "$BUNDLE")
case "$BUNDLE_BASE" in
  *LAB-ONLY*)
    if [ "$ALLOW_LAB" -eq 0 ]; then
      echo "Refused: $BUNDLE_BASE is a LAB-ONLY bundle (carries a throwaway Lab TUF root)." >&2
      echo "Approve promoting its verified twr-server binary with --allow-lab-artifact, or supply a Season bundle." >&2
      exit 2
    fi
    ;;
esac

BUNDLE_SHA=$(sha256sum "$BUNDLE" | cut -d ' ' -f 1)
if [ -n "$EXPECTED_SHA" ] && [ "$BUNDLE_SHA" != "$EXPECTED_SHA" ]; then
  echo "Refused: local bundle SHA-256 ($BUNDLE_SHA) does not match --sha256 ($EXPECTED_SHA)." >&2
  exit 2
fi
EXPECTED_SHA=$BUNDLE_SHA

# Verify locally that the tarball holds the required fixed members before sending anything.
tar_list=$(tar -tzf "$BUNDLE" 2>/dev/null) || { echo "Refused: bundle is not a readable gzip tar archive: $BUNDLE" >&2; exit 2; }
printf '%s\n' "$tar_list" | grep -qx './twr-server' || { echo "Refused: bundle lacks ./twr-server" >&2; exit 2; }
printf '%s\n' "$tar_list" | grep -qx './endpoint-registry.csv' || { echo "Refused: bundle lacks ./endpoint-registry.csv" >&2; exit 2; }

print_approval

step() {
  echo ""
  echo "== $*"
}

# Print every remote ssh command before executing it; skip execution on --dry-run.
remote() {
  printf '   ssh -o BatchMode=yes -o ConnectTimeout=10 %s %s\n' "$TARGET" "'$1'"
  if [ "$DRY_RUN" -eq 0 ]; then
    # shellcheck disable=SC2029  # intentional: payload is expanded locally and sent as one remote command
    ssh -o BatchMode=yes -o ConnectTimeout=10 "$TARGET" "$1"
  fi
}

remote_copy() {
  src=$1
  dst=$2
  printf '   scp -q -o BatchMode=yes -o ConnectTimeout=10 %s %s:%s\n' "$src" "$TARGET" "$dst"
  if [ "$DRY_RUN" -eq 0 ]; then
    scp -q -o BatchMode=yes -o ConnectTimeout=10 "$src" "$TARGET:$dst"
  fi
}

die() {
  echo "FAILED: $1" >&2
  echo "Inspect the target before retrying. To remove partial install files, use:" >&2
  echo "  $(dirname "$0")/season-uninstall.sh --target $TARGET --christopher-ok" >&2
  exit 1
}

REMOTE_BUNDLE="$STAGE_DIR/bundle.tar.gz"
INSTALL_STAMP=$(date -u +%Y-%m-%dT%H:%M:%SZ)

step "1/7 Pre-flight: check Alpine 3 x86_64 and refuse if anything TheWarRoom already exists"
PREFLIGHT="set -eu
[ \"\$(id -u)\" = 0 ] || { echo 'Refused: must run as root on target.' >&2; exit 1; }
[ \"\$(uname -m)\" = x86_64 ] || { echo 'Refused: x86_64 host required.' >&2; exit 1; }
[ -r '$OS_RELEASE' ] || { echo 'Refused: cannot read $OS_RELEASE.' >&2; exit 1; }
. '$OS_RELEASE'
[ \"\$ID\" = alpine ] && [ \"\${VERSION_ID%%.*}\" = 3 ] || { echo 'Refused: Alpine 3.x required.' >&2; exit 1; }
for existing in '$APP_ROOT' '$DATA_DIR' '$BACKUP_DIR' '$ETC_DIR' '$INIT_DIR/$SERVICE' '$INIT_DIR/twr-server' '$LOGROTATE_DIR/twr' '$PERIODIC_DIR/twr-logrotate' '$STAGE_DIR' '$USR_LOCAL_DIR/bin/twr-server' '$USR_LOCAL_DIR/libexec/twr-sandbox'; do
  if [ -e \"\$existing\" ] || [ -L \"\$existing\" ]; then
    echo \"Refused: existing TheWarRoom path found: \$existing\" >&2
    exit 1
  fi
done
if id twr >/dev/null 2>&1; then
  echo 'Refused: system user twr already exists.' >&2
  exit 1
fi
echo 'Pre-flight OK: clean Alpine 3 x86_64 target.'"
remote "$PREFLIGHT" || die "pre-flight refused or failed"

step "2/7 Stage and verify release bundle on target"
remote "set -eu; umask 077; install -d -m 0700 -o root -g root '$STAGE_DIR'" || die "could not create staging directory"
remote_copy "$BUNDLE" "$REMOTE_BUNDLE" || die "could not copy bundle to target"
VERIFY_REMOTE="set -eu
actual=\$(sha256sum '$REMOTE_BUNDLE' | cut -d ' ' -f 1)
[ \"\$actual\" = '$EXPECTED_SHA' ] || { echo \"Refused: remote SHA-256 mismatch (\$actual != $EXPECTED_SHA)\" >&2; rm -rf '$STAGE_DIR'; exit 1; }
echo 'Bundle SHA-256 verified on target.'"
remote "$VERIFY_REMOTE" || die "remote bundle verification failed"

step "3/7 Install Alpine packages (keeping openssh-server for Beelink admin)"
remote "set -eu; apk add --no-cache openrc bubblewrap openssl ca-certificates curl jq logrotate busybox-openrc" \
  || die "package installation failed"

step "4/7 Create service user, Season directory layout, and extract version $VERSION"
LAYOUT_CMD="set -eu
umask 077
addgroup -S twr
adduser -S -D -H -s /sbin/nologin -G twr twr
install -d -m 0755 -o root -g root '$APP_ROOT' '$APP_ROOT/versions' '$APP_ROOT/versions/$VERSION' '$APP_ROOT/libexec'
install -d -m 0700 -o twr -g twr '$DATA_DIR'
install -d -m 0700 -o root -g root '$BACKUP_DIR'
install -d -m 0750 -o root -g twr '$ETC_DIR'
tar -xzOf '$REMOTE_BUNDLE' ./twr-server > '$APP_ROOT/versions/$VERSION/twr-server'
chown root:root '$APP_ROOT/versions/$VERSION/twr-server'
chmod 0755 '$APP_ROOT/versions/$VERSION/twr-server'
tar -xzOf '$REMOTE_BUNDLE' ./endpoint-registry.csv > '$APP_ROOT/versions/$VERSION/endpoint-registry.csv'
chown root:twr '$APP_ROOT/versions/$VERSION/endpoint-registry.csv'
chmod 0640 '$APP_ROOT/versions/$VERSION/endpoint-registry.csv'
install -m 0640 -o root -g twr '$APP_ROOT/versions/$VERSION/endpoint-registry.csv' '$ETC_DIR/endpoint-registry.csv'
ln -sfn '$APP_ROOT/versions/$VERSION' '$APP_ROOT/current'
install -m 0600 -o twr -g twr /dev/null '$DATA_DIR/server.db'
rm -rf '$STAGE_DIR'"
remote "$LAYOUT_CMD" || die "layout or extraction failed"

step "5/7 Mint private TLS and authenticator keys, sandbox, helpers, and log rotation"
PUBLIC_HOST_STEP=""
if [ -n "$PUBLIC_HOST" ]; then
  PUBLIC_HOST_STEP="printf '%s\n' '$PUBLIC_HOST' > '$ETC_DIR/public-host'
chown root:root '$ETC_DIR/public-host'
chmod 0644 '$ETC_DIR/public-host'"
fi

HARDEN_CMD="set -eu
umask 077
openssl req -x509 -newkey ec -pkeyopt ec_paramgen_curve:P-256 -nodes \\
  -keyout '$ETC_DIR/origin.key' -out '$ETC_DIR/origin.pem' -days 36500 \\
  -subj /CN=localhost -addext 'subjectAltName=DNS:localhost,IP:127.0.0.1'
openssl rand -out '$ETC_DIR/authenticator.key' 32
chown twr:twr '$ETC_DIR/origin.key' '$ETC_DIR/authenticator.key'
chmod 0600 '$ETC_DIR/origin.key' '$ETC_DIR/authenticator.key'
chown root:twr '$ETC_DIR/origin.pem'
chmod 0640 '$ETC_DIR/origin.pem'
$PUBLIC_HOST_STEP
cat > '$APP_ROOT/libexec/twr-sandbox' <<'SANDBOX'
#!/bin/sh
set -eu
umask 077
if [ \"\$#\" = 0 ]; then
  set -- $APP_ROOT/current/twr-server hosted -listen 127.0.0.1:$PORT \\
    -db $DATA_DIR/server.db -tls-cert $ETC_DIR/origin.pem \\
    -tls-key $ETC_DIR/origin.key -totp-key-file $ETC_DIR/authenticator.key \\
    -backup-config $ETC_DIR/backup.json -preview-registry $ETC_DIR/endpoint-registry.csv \\
    -status-socket $DATA_DIR/status.sock
  if [ -s $ETC_DIR/public-host ]; then
    set -- \"\$@\" -public-host \"\$(cat $ETC_DIR/public-host)\"
  fi
  if [ -e $ETC_DIR/trusted-proxy ]; then
    set -- \"\$@\" -trusted-proxy 127.0.0.1
  fi
  if [ -s $ETC_DIR/football.sha256 ]; then
    set -- \"\$@\" -football \"\$(head -c 64 $ETC_DIR/football.sha256)\"
  fi
  if [ -e $ETC_DIR/official-stats ]; then
    set -- \"\$@\" -official-stats
  fi
  if [ -e $ETC_DIR/nfl-schedule-refresh ]; then
    set -- \"\$@\" -nfl-schedule-refresh
  fi
  if [ -e $ETC_DIR/sleeper-projections ]; then
    set -- \"\$@\" -sleeper-projections
  fi
fi
exec bwrap --unshare-all --share-net --die-with-parent --new-session --cap-drop ALL \\
  --ro-bind /usr /usr --ro-bind /lib /lib --ro-bind /bin /bin \\
  --ro-bind $APP_ROOT $APP_ROOT --ro-bind $ETC_DIR $ETC_DIR \\
  --ro-bind /etc/ssl/certs /etc/ssl/certs --ro-bind /etc/resolv.conf /etc/resolv.conf \\
  --proc /proc --dev /dev --tmpfs /tmp --tmpfs /run \\
  --bind $DATA_DIR $DATA_DIR --chdir $DATA_DIR --clearenv \\
  --setenv PATH /usr/bin:/bin --setenv HOME $DATA_DIR --setenv TMPDIR $DATA_DIR \\
  --setenv GOMEMLIMIT 768MiB --setenv GOMAXPROCS 2 \\
  \"\$@\"
SANDBOX
chown root:root '$APP_ROOT/libexec/twr-sandbox'
chmod 0755 '$APP_ROOT/libexec/twr-sandbox'
cat > '$APP_ROOT/libexec/twr-backup-setup' <<'BACKUPSETUP'
#!/bin/sh
# Private R2 backup setup for the Season server. Generate the age key offline, never here.
set -eu
umask 077
[ \"\$(id -u)\" = 0 ]
[ -d $ETC_DIR ] && [ -d $DATA_DIR ]
input=\$(mktemp /tmp/twr-backup-input.XXXXXX)
cleanup() { stty echo 2>/dev/null || true; rm -f \"\$input\"; }
trap cleanup EXIT
trap 'exit 1' HUP INT TERM
printf 'R2 S3 endpoint (hostname only): '; IFS= read -r endpoint
printf 'Backup bucket name: '; IFS= read -r bucket
printf 'R2 Access Key ID: '; IFS= read -r access
printf 'R2 Secret Access Key (hidden): '
stty -echo; IFS= read -r secret; stty echo; printf '\\n'
printf 'Public age recipient (age1..., NEVER AGE-SECRET-KEY): '; IFS= read -r recipient
printf '%s\\n' \"\$endpoint\" \"\$bucket\" \"\$access\" \"\$secret\" \"\$recipient\" |
  jq -Rn '[inputs] | {endpoint:.[0],bucket:.[1],access_key:.[2],secret_key:.[3],recipient:.[4]}' > \"\$input\"
$APP_ROOT/current/twr-server backup-check -config \"\$input\"
rc-service $SERVICE --ifstarted stop
install -m 0600 -o twr -g twr \"\$input\" $ETC_DIR/backup.json
rm -f \"\$input\"
rc-service $SERVICE start
printf 'Backup settings saved privately. Changes stay paused until an encrypted R2 upload passes read-back verification.\\n'
BACKUPSETUP
chown root:root '$APP_ROOT/libexec/twr-backup-setup'
chmod 0755 '$APP_ROOT/libexec/twr-backup-setup'
cat > '$APP_ROOT/libexec/twr-status' <<'STATUS'
#!/bin/sh
set -eu
umask 077
[ \"\$(id -u)\" = 0 ] || { echo 'Run this as root.' >&2; exit 1; }
$APP_ROOT/current/twr-server version | sed -n 's/^Server build /Installed server build /p'
exec su -s /bin/sh twr -c '$APP_ROOT/current/twr-server status -socket $DATA_DIR/status.sock -origin 127.0.0.1:$PORT -origin-cert $ETC_DIR/origin.pem -state $DATA_DIR'
STATUS
chown root:root '$APP_ROOT/libexec/twr-status'
chmod 0755 '$APP_ROOT/libexec/twr-status'
install -d -m 0755 -o root -g root '$LOGROTATE_DIR' '$PERIODIC_DIR'
cat > '$LOGROTATE_DIR/twr' <<'LOGROTATE'
$DATA_DIR/server.log {
    size 1M
    rotate 3
    missingok
    notifempty
    copytruncate
    su twr twr
    create 0600 twr twr
}
LOGROTATE
chown root:root '$LOGROTATE_DIR/twr'
chmod 0600 '$LOGROTATE_DIR/twr'
ln -sf /etc/periodic/daily/logrotate '$PERIODIC_DIR/twr-logrotate'
rc-update add crond default
rc-service crond --ifstopped start
cat > '$APP_ROOT/install-manifest.txt' <<'MANIFEST'
installed_at=$INSTALL_STAMP
version=$VERSION
bundle=$BUNDLE_BASE
sha256=$EXPECTED_SHA
port=$PORT
service=$SERVICE
app_root=$APP_ROOT
data_dir=$DATA_DIR
backup_dir=$BACKUP_DIR
etc_dir=$ETC_DIR
init_script=$INIT_DIR/$SERVICE
logrotate=$LOGROTATE_DIR/twr
periodic=$PERIODIC_DIR/twr-logrotate
MANIFEST
chown root:root '$APP_ROOT/install-manifest.txt'
chmod 0644 '$APP_ROOT/install-manifest.txt'"
remote "$HARDEN_CMD" || die "hardening or wrapper setup failed"

step "6/7 Install and start OpenRC service $SERVICE"
INIT_CMD="set -eu
umask 077
install -d -m 0755 -o root -g root '$INIT_DIR'
cat > '$INIT_DIR/$SERVICE' <<'INITD'
#!/sbin/openrc-run
name=\"TheWarRoom\"
description=\"Loopback-only Season league service\"
supervisor=\"supervise-daemon\"
command=\"$APP_ROOT/libexec/twr-sandbox\"
command_user=\"twr:twr\"
respawn_delay=5
respawn_max=5
respawn_period=60
umask=0077
output_log=\"$DATA_DIR/server.log\"
error_log=\"$DATA_DIR/server.log\"

depend() {
	need net
}

start_pre() {
	checkpath --directory --mode 0700 --owner twr:twr $DATA_DIR
	checkpath --file --mode 0600 --owner twr:twr $DATA_DIR/server.log
}
INITD
chown root:root '$INIT_DIR/$SERVICE'
chmod 0755 '$INIT_DIR/$SERVICE'
rc-update add '$SERVICE' default
rc-service '$SERVICE' start"
remote "$INIT_CMD" || die "OpenRC service setup or start failed"

step "7/7 Health check on 127.0.0.1:$PORT/app/"
if [ -n "$PUBLIC_HOST" ]; then
  HEALTH_CMD="set -eu
attempt=0
while [ \"\$attempt\" -lt 30 ]; do
  code=\$(curl --silent --cacert '$ETC_DIR/origin.pem' --max-time 2 \\
    -H 'Host: $PUBLIC_HOST' --output /dev/null --write-out '%{http_code}' \\
    'https://127.0.0.1:$PORT/app/') || code=000
  if [ \"\$code\" = 200 ]; then
    echo 'Health OK: https://127.0.0.1:$PORT/app/ answered 200 for $PUBLIC_HOST.'
    exit 0
  fi
  attempt=\$((attempt + 1))
  sleep 1
done
echo 'Health check failed on https://127.0.0.1:$PORT/app/ (expected 200).' >&2
exit 1"
else
  # Local-only install without --public-host: twr-server answers 404 on /app/ over verified TLS,
  # proving the hosted origin is listening and serving TLS on 127.0.0.1:$PORT/app/.
  HEALTH_CMD="set -eu
attempt=0
while [ \"\$attempt\" -lt 30 ]; do
  code=\$(curl --silent --cacert '$ETC_DIR/origin.pem' --max-time 2 \\
    --output /dev/null --write-out '%{http_code}' \\
    'https://127.0.0.1:$PORT/app/') || code=000
  if [ \"\$code\" = 404 ]; then
    echo 'Health OK: https://127.0.0.1:$PORT/app/ answered 404 over verified TLS (browser app paused until --public-host is configured).'
    exit 0
  fi
  attempt=\$((attempt + 1))
  sleep 1
done
echo 'Health check failed on https://127.0.0.1:$PORT/app/ (expected 404 over TLS).' >&2
exit 1"
fi
remote "$HEALTH_CMD" || die "health check on 127.0.0.1:$PORT/app/ failed"

echo ""
echo "Season first-install complete ($VERSION -> $APP_ROOT/current)."
echo "Next steps before admitting any owner:"
echo "  1. Configure off-site R2 backups on the target: sh $APP_ROOT/libexec/twr-backup-setup"
echo "     (Changes stay paused until the first encrypted R2 upload verifies.)"
echo "  2. Verify NTP sync and dashboard: sh $APP_ROOT/libexec/twr-status"
echo "Undo command (removes installed program/config, preserves data unless --purge-data is added):"
echo "  $(dirname "$0")/season-uninstall.sh --target $TARGET --christopher-ok"
