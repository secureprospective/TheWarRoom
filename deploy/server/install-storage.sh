#!/usr/bin/env bash
# Run by proxmox.sh before allocating the CT. Never format an existing disk.
set -euo pipefail
umask 077
[[ $EUID == 0 && $# == 2 && $1 =~ ^[1-9][0-9]{2,8}$ ]]
id=$1
helper=$2
base=$(cd "$(dirname "$0")" && pwd)
[[ -x $helper && -c /dev/tpmrm0 ]] || {
  echo 'This host needs a TPM 2.0 and the release disk-unlock helper. Nothing was formatted.' >&2; exit 1;
}
[[ ! -e /var/lib/twr-storage/$id.luks && ! -e /etc/twr-storage/$id.jwe && ! -e /mnt/twr-$id ]]
if pvesm status --storage "twr-$id" >/dev/null 2>&1; then
  echo 'League storage already exists; refusing to replace it.' >&2; exit 1;
fi
[[ $(findmnt -n -o FSTYPE /run) == tmpfs ]]
echo 'Your league disk will unlock automatically on this computer.'
echo 'If the computer changes or its security settings change, use your backup disk passphrase.'
echo 'Write it on two paper copies: keep one safe and give the sealed second copy to your deputy.'
echo 'It unlocks the league disk, not your league login. Losing both copies can lose the league.'
echo 'For safety this installer also disables swap on the Proxmox host (not just this container).'
read -rp 'Allow this host-wide swap change? Type yes: ' answer
[[ $answer == yes ]] || exit 1
apt-get update
apt-get install -y cryptsetup e2fsprogs
# Mask configured swap units, including inactive entries. swapoff failure stops install.
while read -r swapunit _; do
  [[ $swapunit != *.swap ]] || systemctl mask "$swapunit"
done < <(systemctl list-units --all --type=swap --no-legend --plain)
while read -r swapunit _; do
  [[ $swapunit != *.swap ]] || systemctl mask "$swapunit"
done < <(systemctl list-unit-files --type=swap --no-legend)
systemctl mask swap.target
install -m 644 "$base/twr-no-swap.service" "$base/twr-storage@.service" "$base/twr-container@.service" /etc/systemd/system/
install -d -m 755 /usr/local/libexec
install -m 755 "$base/storage.sh" /usr/local/libexec/twr-storage
install -m 755 "$base/recover-storage.sh" /usr/local/bin/twr-unlock-league
install -m 755 "$helper" /usr/local/bin/twr-unlock
systemctl daemon-reload
systemctl enable --now twr-no-swap.service
[[ $(wc -l < /proc/swaps) == 1 ]]
# Collect secrets only after host swap is disabled.
read -rsp 'Choose a backup disk passphrase (at least 16 characters): ' pass; echo
read -rsp 'Enter it again: ' confirm; echo
[[ ${#pass} -ge 16 && $pass == "$confirm" ]] || { echo 'Passphrases did not match or were too short.' >&2; exit 1; }
read -rp 'Have you secured both paper copies? Type yes: ' answer
[[ $answer == yes ]] || exit 1
install -d -m 700 /var/lib/twr-storage /etc/twr-storage
install -d -m 700 "/mnt/twr-$id"
keydir=$(mktemp -d /run/twr-storage.XXXXXX)
cleanup() { rm -f "$keydir/auto" "$keydir/backup"; rmdir "$keydir"; }
trap cleanup EXIT
# Plain keys/passphrase exist only on /run tmpfs briefly; only the Clevis JWE persists.
head -c 32 /dev/urandom > "$keydir/auto"
printf %s "$pass" > "$keydir/backup"
unset pass confirm
/usr/local/bin/twr-unlock seal < "$keydir/auto" > "/etc/twr-storage/$id.jwe"
/usr/local/bin/twr-unlock unseal < "/etc/twr-storage/$id.jwe" | cmp - "$keydir/auto"
(set -o noclobber; : > "/var/lib/twr-storage/$id.luks")
fallocate -l 8G "/var/lib/twr-storage/$id.luks"
cryptsetup luksFormat --batch-mode --type luks2 --key-slot 0 --key-file "$keydir/auto" "/var/lib/twr-storage/$id.luks"
cryptsetup luksAddKey --key-slot 1 --key-file "$keydir/auto" "/var/lib/twr-storage/$id.luks" "$keydir/backup"
cryptsetup open --test-passphrase --key-file "$keydir/backup" "/var/lib/twr-storage/$id.luks"
cryptsetup open --key-file "$keydir/auto" "/var/lib/twr-storage/$id.luks" "twr-$id"
mkfs.ext4 -q "/dev/mapper/twr-$id"
cryptsetup close "twr-$id"
cleanup
trap - EXIT
systemctl start "twr-storage@$id.service"
pvesm add dir "twr-$id" --path "/mnt/twr-$id" --content rootdir --is_mountpoint 1 --nodes "$(hostname)"
