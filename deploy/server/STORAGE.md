# Private league files inside the container (slice 35)

TheWarRoom is a recreational game. The installer protects its own files and
container without reconfiguring the Proxmox host's security or storage.

## What is protected

- The service runs as its own non-login user inside an unprivileged container.
  The container has swap set to zero; host swap settings are left alone.
- League state lives in `/var/lib/twr`, owned by the service user with mode 0700.
  Other ordinary container users cannot browse it.
- The service starts with umask 077. The database, SQLite WAL and shared-memory
  files, and service log have mode 0600.
- Log rotation runs hourly, checks a 1 MiB size threshold and keeps three private
  copies. This bounds retained log history, not instantaneous log size.
- App sign-in and TOTP protect account access. The audit chain makes changes
  detectable when checked against a trusted record; it is not disk encryption.
- Encrypted off-host backups come in slice 36. This slice does not provide them.

## What is not protected

There is no disk encryption supplied by TheWarRoom. Proxmox host root can read
the container's files. Container root and a compromised running service can also
read league data. File permissions do not protect a stolen host disk, host swap,
or an unencrypted copy or snapshot. The host operator remains responsible for
host security, storage, swap and any existing backups.

The installer selects existing storage and creates/configures only its own
container, including its firewall rules and normal Proxmox autostart. It does not
install host packages, add host systemd services, change host swap, create LUKS
images, register storage or install an unlock helper. Required host tools,
including jq, must already be present. There is no disk passphrase or TPM setup.

## Evidence and limits

`deploy/server/test-alpine.sh` exercises installation and restart in disposable
Alpine/OpenRC, checks 0600 database/WAL/SHM/log permissions, forces log rotation
and checks the active and rotated logs remain private. It also checks sandbox,
TLS and loopback-only access, plus refusal to overwrite an existing installation.

A real Proxmox install, isolation and reboot still need acceptance testing on a
disposable node. The Alpine test does not establish those host-level outcomes.
