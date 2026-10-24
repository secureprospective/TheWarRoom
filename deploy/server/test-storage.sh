#!/usr/bin/env bash
# Real Clevis TPM binding and LUKS slots, confined to a disposable container.
set -euo pipefail
cd "$(dirname "$0")/../.."
export GOTOOLCHAIN=go1.26.10
CGO_ENABLED=0 go build -o .scratch/twr-unlock ./cmd/twr-unlock
name="twr-storage-test-$$"
trap 'docker rm -f "$name" >/dev/null 2>&1 || true' EXIT
image=alpine:3.23@sha256:ff7f0c727dd796aa36787bfcd9a64bf35a6ce72a73451b3359fde849b6703d63
# No real TPM, host devices, privileges, host network or writable host mount.
docker run -d --name "$name" -v "$PWD:/src:ro" "$image" sleep 600 >/dev/null
docker exec "$name" sh -ec '
  apk add --no-cache swtpm tpm2-tools tpm2-tss-tcti-swtpm cryptsetup
  umask 077
  mkdir /tmp/tpm
  swtpm socket --tpm2 --tpmstate dir=/tmp/tpm --server type=unixio,path=/dev/tpmrm0 --ctrl type=unixio,path=/dev/tpmrm0.ctrl --flags not-need-init,startup-clear --daemon --pid file=/tmp/tpm.pid
  head -c 32 /dev/urandom > /tmp/key
  /src/.scratch/twr-unlock seal < /tmp/key > /tmp/key.jwe
  /src/.scratch/twr-unlock unseal < /tmp/key.jwe > /tmp/recovered
  cmp /tmp/key /tmp/recovered
  truncate -s 64M /tmp/disk.luks
  # No device mapper access: real LUKS header/keyslot and passphrase tests need none.
  cryptsetup luksFormat --batch-mode --type luks2 --pbkdf pbkdf2 --key-file /tmp/key /tmp/disk.luks
  printf %s "deputy backup passphrase" > /tmp/backup
  cryptsetup luksAddKey --pbkdf pbkdf2 --key-file /tmp/key /tmp/disk.luks /tmp/backup
  cryptsetup open --test-passphrase --key-file /tmp/recovered /tmp/disk.luks
  cryptsetup open --test-passphrase --key-file /tmp/backup /tmp/disk.luks
  before=$(sha256sum /tmp/disk.luks)
  printf wrong > /tmp/wrong
  if cryptsetup open --test-passphrase --key-file /tmp/wrong /tmp/disk.luks; then exit 1; fi
  # A measured-boot change refuses automatic unlock, but backup still works.
  TPM2TOOLS_TCTI=swtpm:path=/dev/tpmrm0 tpm2_pcrextend 7:sha256=aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
  if /src/.scratch/twr-unlock unseal < /tmp/key.jwe > /tmp/refused; then exit 1; fi
  test ! -s /tmp/refused
  cryptsetup open --test-passphrase --key-file /tmp/backup /tmp/disk.luks
  test "$before" = "$(sha256sum /tmp/disk.luks)"
  # Replacing the TPM also refuses unlock without changing the disk.
  kill "$(cat /tmp/tpm.pid)"
  mkdir /tmp/other-tpm
  swtpm socket --tpm2 --tpmstate dir=/tmp/other-tpm --server type=unixio,path=/dev/tpmrm0 --ctrl type=unixio,path=/dev/tpmrm0.ctrl --flags not-need-init,startup-clear --daemon
  if /src/.scratch/twr-unlock unseal < /tmp/key.jwe > /tmp/refused; then exit 1; fi
  test ! -s /tmp/refused
  cryptsetup open --test-passphrase --key-file /tmp/backup /tmp/disk.luks
  test "$before" = "$(sha256sum /tmp/disk.luks)"
  # Recovery keeps slot 1 usable across an interrupted automatic-slot replacement.
  cryptsetup luksKillSlot --key-file /tmp/backup /tmp/disk.luks 0
  cryptsetup open --test-passphrase --key-slot 1 --key-file /tmp/backup /tmp/disk.luks
  head -c 32 /dev/urandom > /tmp/replacement
  cryptsetup luksAddKey --pbkdf pbkdf2 --key-slot 0 --key-file /tmp/backup /tmp/disk.luks /tmp/replacement
  cryptsetup open --test-passphrase --key-slot 0 --key-file /tmp/replacement /tmp/disk.luks
  cryptsetup open --test-passphrase --key-slot 1 --key-file /tmp/backup /tmp/disk.luks
  if cryptsetup open --test-passphrase --key-file /tmp/key /tmp/disk.luks; then exit 1; fi
'
echo 'PASS: TPM roundtrip, changed-PCR/wrong-TPM refusal, LUKS automatic/backup slots, unchanged disk on refusal.'
echo 'UNVERIFIED: Proxmox cold reboot, locked mounted data, systemd credentials, deputy console and host swap.'
