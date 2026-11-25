# On-host payload for season-swap.sh; POSIX sh for Alpine.
set -eu
umask 077
refuse() { echo "Refused: $*" >&2; exit 1; }
current="$APP_ROOT/current"
backup="$BACKUP_DIR/twr-data-$STAMP.tgz"
record="$backup.record"
if [ "$ROLLBACK" = 1 ]; then
  [ -f "$backup" ] && [ -r "$backup" ] && [ -s "$backup" ] || refuse "backup is missing or unreadable."
  [ -r "$record" ] || refuse "backup has no recorded swap."
  previous=$(sed -n '1p' "$record")
  candidate=$(sed -n '2p' "$record")
  digest=$(sed -n '3p' "$record")
  case "$previous" in "$APP_ROOT"/versions/*) ;; *) refuse "recorded previous version is invalid." ;; esac
  name=${previous##*/}
  case "$name" in *..*|*[!A-Za-z0-9._-]*|'') refuse "previous version name is invalid." ;; esac
  [ "$previous" = "$APP_ROOT/versions/$name" ] || refuse "previous version path is invalid."
  [ -d "$previous" ] || refuse "previous version folder is gone."
  [ "$(readlink "$current")" = "$candidate" ] || refuse "backup does not match the current swap."
  [ "$(sha256sum "$backup" | cut -d ' ' -f1)" = "$digest" ] || refuse "backup does not match the recorded swap."
  tar -tzf "$backup" >/dev/null || refuse "backup is unreadable."
  stage="$BACKUP_DIR/restore-$STAMP"
  aside="$DATA_DIR.before-rollback-$STAMP"
  [ ! -e "$stage" ] && [ ! -e "$aside" ] || refuse "recovery paths already exist; preserve them and inspect."
  # Extract before stopping or moving the only live copy. Only root-created archives are accepted.
  mkdir "$stage"
  tar -xzf "$backup" -C "$stage" || refuse "backup could not be staged."
  [ -d "$stage/$(basename "$DATA_DIR")" ] || refuse "backup lacks the data folder."
  rc-service twr stop
  mv "$DATA_DIR" "$aside"
  mv "$stage/$(basename "$DATA_DIR")" "$DATA_DIR"
  ln -sfn "$previous" "$current"
  echo "Current data preserved at $aside"
else
  candidate="$APP_ROOT/versions/$VERSION"
  [ -d "$candidate" ] || refuse "new version folder is missing."
  previous=$(readlink "$current") || refuse "current version is not a link."
  case "$previous" in "$APP_ROOT"/versions/*) ;; *) refuse "previous version is outside versions." ;; esac
  name=${previous##*/}
  case "$name" in *..*|*[!A-Za-z0-9._-]*|'') refuse "previous version name is invalid." ;; esac
  [ "$previous" = "$APP_ROOT/versions/$name" ] || refuse "previous version path is invalid."
  [ -d "$previous" ] || refuse "previous version folder is gone."
  [ "$previous" != "$candidate" ] || refuse "candidate is already current."
  mkdir -p "$BACKUP_DIR"
  [ ! -e "$backup" ] && [ ! -e "$record" ] || refuse "swap backup already exists."
  rc-service twr stop
  tar -C "$(dirname "$DATA_DIR")" -czf "$backup" "$(basename "$DATA_DIR")"
  digest=$(sha256sum "$backup" | cut -d ' ' -f1)
  printf '%s\n' "$previous" "$candidate" "$digest" > "$record"
  ln -sfn "$candidate" "$current"
fi
rc-service twr start
i=0
until curl -fsS http://127.0.0.1:5199/app/ >/dev/null; do
  i=$((i+1)); [ "$i" -lt 15 ] || exit 1; sleep 2
done
