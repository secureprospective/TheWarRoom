#!/usr/bin/env bash
# Release assembly; no credentials or league state are included.
set -euo pipefail
cd "$(dirname "$0")/../.."
[[ $# == 1 ]]
out=$(realpath -m "$1")
[[ $out == "$PWD/"* ]] || { echo 'Output must stay inside this worktree.' >&2; exit 1; }
work=$(mktemp -d "$PWD/.scratch/server-bundle.XXXXXX")
trap 'rm -rf "$work"' EXIT
export GOTOOLCHAIN=go1.26.10
# GoReleaser OSS, not the separately licensed Pro distribution.
curl --fail --location --silent --show-error --connect-timeout 10 --max-time 180 \
  https://github.com/goreleaser/goreleaser/releases/download/v2.19.0/goreleaser_Linux_x86_64.tar.gz \
  -o "$work/goreleaser.tar.gz"
echo 'ebc3194ad1d1b1787614405285a0039665cd4c5bd3305215a1b201600f90630b  goreleaser.tar.gz' | (cd "$work" && sha256sum -c -)
tar -xzf "$work/goreleaser.tar.gz" -C "$work" goreleaser
"$work/goreleaser" check --config deploy/server/goreleaser.yaml
GOOS=linux GOARCH=amd64 "$work/goreleaser" build --snapshot --clean --single-target \
  --config deploy/server/goreleaser.yaml --output "$work/twr-server"
# Official Cloudflare release asset, independently pinned to its GitHub release digest.
curl --fail --location --silent --show-error --connect-timeout 10 --max-time 180 \
  https://github.com/cloudflare/cloudflared/releases/download/2026.10.3/cloudflared-linux-amd64 \
  -o "$work/cloudflared"
echo '053b80b0a23031a038fdbf3c073f59f82b8f1f152aa67971d7383ffafc0dbab0  cloudflared' | (cd "$work" && sha256sum -c -)
cp deploy/server/{install.sh,backup-setup.sh,restore.sh,reset-sign-ins.sh,status.sh,cloudflare.sh,twr-sandbox,twr-server.initd,cloudflared.initd,twr-logrotate,proxmox.sh} "$work/"
cp docs/ui/endpoint-registry.csv "$work/"
mkdir -p "$(dirname "$out")"
tar -czf "$out" -C "$work" .
sha256sum "$out"
