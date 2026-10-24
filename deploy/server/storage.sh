#!/usr/bin/env bash
# Proxmox host only. No existing disk, container or storage is reformatted.
set -euo pipefail
umask 077
[[ $EUID == 0 && $# == 2 && $2 =~ ^[1-9][0-9]{2,8}$ ]]
action=$1 id=$2
image=/var/lib/twr-storage/$id.luks
mountpoint=/mnt/twr-$id
mapper=twr-$id
case "$action" in
  start|recover)
    # CT swap=0 does not protect host helpers or host swap. Refuse any active swap.
    [[ $(wc -l < /proc/swaps) == 1 ]] || { echo 'Swap is active; encrypted storage remains locked.' >&2; exit 1; }
    [[ -f $image && ! -L $image && $(stat -c %a "$image") == 600 ]]
    if ! cryptsetup status "$mapper" >/dev/null 2>&1; then
      if [[ $action == recover ]]; then
        echo 'Enter the backup disk passphrase. This does not change any league account.' >&2
        cryptsetup open --type luks2 "$image" "$mapper"
      else
        /usr/local/bin/twr-unlock unseal < "${CREDENTIALS_DIRECTORY:?}/unlock.jwe" |
          cryptsetup open --type luks2 --key-file - "$image" "$mapper"
      fi
    fi
    # Never mount an unrelated mapper or cover an existing mount.
    device=$(cryptsetup status "$mapper" | awk '$1 == "device:" {print $2}')
    losetup -j "$image" --output NAME --noheadings | grep -Fxq "$device"
    if mountpoint -q "$mountpoint"; then
      [[ $(findmnt -n -o SOURCE --target "$mountpoint") == /dev/mapper/"$mapper" ]]
    else
      mount -o nodev,nosuid /dev/mapper/"$mapper" "$mountpoint"
    fi
    ;;
  stop)
    if mountpoint -q "$mountpoint"; then umount "$mountpoint"; fi
    if cryptsetup status "$mapper" >/dev/null 2>&1; then cryptsetup close "$mapper"; fi
    ;;
  *) exit 1 ;;
esac
