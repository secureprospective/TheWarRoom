#!/bin/sh
# CT-only console dashboard and its root-only OpenRC browser-config preparation modes.
set -eu
umask 077

mode=${1:-status}
case "$mode" in
  --prepare-browser | --prepare-ingress)
    [ "$#" -le 3 ] || { echo 'Unexpected browser preparation argument.' >&2; exit 1; }
    shift
    config=${1:-/etc/cloudflared/config.yml}
    host_file=${2:-/etc/twr/public-host}
    if [ "$(id -u)" != 0 ] && { [ "$config" = /etc/cloudflared/config.yml ] || [ "$host_file" = /etc/twr/public-host ]; }; then
      echo 'Browser config preparation for installed paths must run as root.' >&2
      exit 1
    fi
    ;;
  status)
    [ "$#" = 0 ] || { echo 'Unexpected status argument.' >&2; exit 1; }
    [ "$(id -u)" = 0 ] || { echo 'Run this as root in the CT console.' >&2; exit 1; }
    ;;
  *) echo 'Unknown status command.' >&2; exit 1 ;;
esac

log() {
  printf '%s\n' "$1"
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

prepare_browser() {
  if [ ! -f "$config" ] || [ -L "$config" ] || ! jq -e \
    '.ingress | type == "array" and length > 0 and (.[0] | type == "object" and (.path | type == "string"))' \
    "$config" >/dev/null 2>&1; then
    if [ "$mode" = --prepare-browser ] && [ ! -s "$host_file" ]; then
      log 'Browser app disabled: no usable public hostname; Tunnel config is missing or unreadable.'
    else
      log 'Browser app config was not reconciled: Tunnel config is missing or unreadable.'
    fi
    return 0
  fi

  path=$(jq -er '.ingress[0].path' "$config" 2>/dev/null) || path=
  changed=no
  if ! printf '%s' "$path" | grep -Fq 'app(/.*)?'; then
    config_tmp=$(mktemp "$config.XXXXXX" 2>/dev/null) || {
      log 'Browser app config was not reconciled: could not create a temporary Tunnel config.'
      return 0
    }
    if ! jq -e '
      .ingress[0].path as $path |
      if ($path | startswith("^/(") and endswith(")?$")) then
        .ingress[0].path = (($path | rtrimstr(")?$")) + "|app(/.*)?)?$")
      else error("unsupported first ingress path") end
    ' "$config" > "$config_tmp" 2>/dev/null; then
      rm -f "$config_tmp"
      log 'Browser app config was not reconciled: the first Tunnel ingress path is not a supported route pattern.'
      return 0
    fi
    config_mode=$(stat -c '%a' "$config" 2>/dev/null) || {
      rm -f "$config_tmp"
      log 'Browser app config was not reconciled: could not read Tunnel config permissions.'
      return 0
    }
    config_owner=$(stat -c '%u:%g' "$config" 2>/dev/null) || {
      rm -f "$config_tmp"
      log 'Browser app config was not reconciled: could not read Tunnel config ownership.'
      return 0
    }
    if [ "$(id -u)" = 0 ] && ! chown "$config_owner" "$config_tmp"; then
      rm -f "$config_tmp"
      log 'Browser app config was not reconciled: could not preserve Tunnel config ownership.'
      return 0
    fi
    if ! chmod "$config_mode" "$config_tmp"; then
      rm -f "$config_tmp"
      log 'Browser app config was not reconciled: could not preserve Tunnel config permissions.'
      return 0
    fi
    if ! mv -f "$config_tmp" "$config"; then
      rm -f "$config_tmp"
      log 'Browser app config was not reconciled: could not install the updated Tunnel config.'
      return 0
    fi
    changed=yes
  fi

  if [ "$mode" = --prepare-ingress ]; then
    [ "$changed" = yes ] && log 'Browser app Tunnel ingress updated for /app.'
    return 0
  fi

  if [ ! -s "$host_file" ]; then
    public_host=$(jq -er '.ingress[0].hostname | select(type == "string" and length > 0)' "$config" 2>/dev/null) || public_host=
    if valid_public_host "$public_host"; then
      host_tmp=$(mktemp "$host_file.XXXXXX" 2>/dev/null) || {
        log 'Browser app disabled: public hostname could not be written; check /etc/twr permissions.'
        return 0
      }
      if ! printf '%s\n' "$public_host" > "$host_tmp"; then
        rm -f "$host_tmp"
        log 'Browser app disabled: public hostname could not be written; check /etc/twr permissions.'
        return 0
      fi
      if [ "$(id -u)" = 0 ] && ! chown root:root "$host_tmp"; then
        rm -f "$host_tmp"
        log 'Browser app disabled: public hostname could not be installed with root ownership.'
        return 0
      fi
      if ! chmod 644 "$host_tmp"; then
        rm -f "$host_tmp"
        log 'Browser app disabled: public hostname could not be installed with mode 0644.'
        return 0
      fi
      if ! mv -f "$host_tmp" "$host_file"; then
        rm -f "$host_tmp"
        log 'Browser app disabled: public hostname could not be installed; check /etc/twr permissions.'
        return 0
      fi
      if [ "$changed" = yes ]; then
        log 'Browser app config reconciled: added /app ingress and recovered the public hostname.'
      else
        log 'Browser app config reconciled: recovered the public hostname.'
      fi
      return 0
    fi
    log 'Browser app disabled: no valid public hostname in the first Tunnel ingress.'
    return 0
  fi

  [ "$changed" = yes ] && log 'Browser app Tunnel ingress updated for /app.'
  return 0
}

if [ "$mode" != status ]; then
  prepare_browser
  exit 0
fi

/usr/local/bin/twr-server version | sed -n 's/^Server build /Installed server build /p'
if [ ! -s /etc/twr/public-host ]; then
  printf '%s\n' 'PROBLEM  Browser app: Disabled; no public hostname was recovered from /etc/cloudflared/config.yml. Restart twr-server after repairing the Tunnel config.'
fi
exec su -s /bin/sh twr -c '/usr/local/bin/twr-server status -socket /var/lib/twr/status.sock -origin 127.0.0.1:8443 -origin-cert /etc/twr/origin.pem -tunnel 127.0.0.1:20241 -state /var/lib/twr'
