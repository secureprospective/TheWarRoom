#!/usr/bin/env bash
# Proxmox console recovery. Never changes a league account or prints a key.
set -euo pipefail
umask 077
[[ $EUID == 0 ]]
read -rp 'League container number: ' id
[[ $id =~ ^[1-9][0-9]{2,8}$ ]]
image=/var/lib/twr-storage/$id.luks
[[ -f $image && ! -L $image ]]
[[ $(findmnt -n -o FSTYPE /run) == tmpfs ]]
systemctl restart twr-no-swap.service
[[ $(wc -l < /proc/swaps) == 1 ]]
echo 'Use the backup disk passphrase from your sealed paper copy.'
read -rsp 'Backup disk passphrase: ' pass; echo
keydir=$(mktemp -d /run/twr-recover.XXXXXX)
cleanup() { rm -f "$keydir/backup" "$keydir/auto" "$keydir/sealed"; rmdir "$keydir"; }
trap cleanup EXIT
printf %s "$pass" > "$keydir/backup"
unset pass
# Wrong passphrase changes nothing, including the running container.
cryptsetup open --test-passphrase --key-slot 1 --key-file "$keydir/backup" "$image"
systemctl stop "twr-container@$id.service" "twr-storage@$id.service"
# Also handle a retained failed installation that was not yet enabled as a unit.
if pct status "$id" | grep -q 'status: running'; then pct shutdown "$id" --timeout 60; fi
/usr/local/libexec/twr-storage stop "$id"
cryptsetup open --key-file "$keydir/backup" "$image" "twr-$id"
head -c 32 /dev/urandom > "$keydir/auto"
# Repair the automatic binding on a replacement TPM if available. Slot 1 is never removed.
if /usr/local/bin/twr-unlock seal < "$keydir/auto" > "$keydir/sealed" &&
   /usr/local/bin/twr-unlock unseal < "$keydir/sealed" | cmp - "$keydir/auto"; then
  if cryptsetup luksDump --dump-json-metadata "$image" | jq -e '.keyslots | has("0")' >/dev/null; then
    cryptsetup luksKillSlot --key-file "$keydir/backup" "$image" 0
  fi
  cryptsetup luksAddKey --key-slot 0 --key-file "$keydir/backup" "$image" "$keydir/auto"
  cryptsetup open --test-passphrase --key-slot 0 --key-file "$keydir/auto" "$image"
  install -m 600 "$keydir/sealed" "/etc/twr-storage/$id.jwe.new"
  mv "/etc/twr-storage/$id.jwe.new" "/etc/twr-storage/$id.jwe"
  echo 'Automatic disk unlock has been repaired on this computer.'
else
  echo 'The security chip is unavailable. Use your backup again after the next reboot.' >&2
fi
/usr/local/libexec/twr-storage recover "$id"
systemctl reset-failed "twr-storage@$id.service" "twr-container@$id.service"
systemctl start "twr-container@$id.service"
echo 'The league disk is unlocked and the container has started. Keep both paper copies safe.'
