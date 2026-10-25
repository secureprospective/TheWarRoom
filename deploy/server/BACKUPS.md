# Private off-site backups (Cloudflare R2)

The server encrypts each recovery set **before** sending it to R2. Cloudflare sees
only encrypted bytes. Your private restore key stays on your computer and with
your deputy, never in the server container (CT). The public `age1...` recipient
cannot decrypt backups.

## 1. Keep the restore key on your own computer

Install [age](https://age-encryption.org/) using its instructions for your computer.
On **your computer, not the Proxmox host or CT**, run:

```sh
age-keygen -o warroom-restore-key.txt
```

Keep that file somewhere private and offline. Print a sealed copy for your wife,
the deputy key holder. Keep the displayed **public key**, beginning `age1`, ready
for setup. Never paste a line beginning `AGE-SECRET-KEY` into the server. Losing
all copies of the private key makes the backups unrecoverable. Do not generate a
replacement recipient casually: old sets still need the old private key.

## 2. Create the backup bucket in Cloudflare

1. Sign in to the Cloudflare dashboard with your own account.
2. Open **Storage & databases → R2 object storage → Overview**. If prompted,
   activate R2. Cloudflare may require billing details even for its free allowance.
3. Choose **Create bucket**. Name it, for example, `warroom-private-backups`.
4. Choose **Standard** storage, not Infrequent Access. Leave public access disabled.
   You do not need a custom domain, public URL, Worker or website.
5. Open the new bucket's **Settings**. Copy its **S3 API endpoint**. For setup,
   use only the hostname: remove `https://` and any trailing slash or bucket name.
   Use the jurisdiction-specific endpoint shown there if you selected a jurisdiction.
6. Return to R2 Overview and choose **Manage R2 API tokens** (sometimes shown under
   **API**). Choose **Create Account API token**; a user token also works if that
   is the option your account offers.
7. Name it `WarRoom backups`. Select **Object Read & Write**. Under bucket scope,
   choose **Apply to specific buckets only** and select **only your backup bucket**.
   Do not choose administrator permissions or all buckets. Leave expiry unset for
   unattended backups; if you choose an expiry, replace it before that date.
8. Create the token. Copy the **Access Key ID** and **Secret Access Key** shown
   on the success page. The secret is shown only once. The generic API token is
   not the S3 secret. Do not send these values to owners or put them in the app.

R2 currently offers no dashboard object-write-only permission. Read access is
needed to check uploaded bytes; object deletion is needed for retention. Bucket-
scoped Object Read & Write is the least sufficient dashboard option.

## 3. Enter the five settings inside the CT

Open this server's **Proxmox CT console**, then run:

```sh
sh /usr/local/libexec/twr-backup-setup
```

Paste the endpoint hostname, bucket name, Access Key ID, Secret Access Key and
**public** age recipient when asked. The secret prompt is hidden. The script
checks the settings, saves `/etc/twr/backup.json` as mode 0600 owned by `twr`,
removes the temporary input and restarts the service. It does not change Proxmox,
host storage, firewall rules or any other container. No hand-edited cron jobs,
service files or host packages are needed. Run the same setup to replace an
expired/revoked token. It stops the service before changing immutable settings.

Until the first complete encrypted upload has been downloaded and checked,
changes and sign-ins stay paused with a plain explanation. Missing settings
never enable unprotected writes. Setup success means the settings were accepted,
not that R2 connectivity or recoverability was proved. The VPN dashboard in
slice 38 will display `offsite.Manager.Status` (capture time, verification time,
writable flag and plain failure); no public backup-status endpoint is added.

## What is saved and when

Every five minutes while healthy (one-minute retries on failure), the server:

1. Reserves SQLite's one writer and makes a `VACUUM INTO` snapshot through the
   existing DB backup boundary. History, evidence blobs/provenance, auth and the
   security audit share `server.db`; WAL is never copied as a live file.
2. Copies immutable `/etc/twr` configuration behind that same writer barrier.
   TLS key/certificate, authenticator key, preview registry and private R2 settings
   must live together there. Stop the server before changing these files.
3. Verifies both audit chains against the captured database and includes a saved
   checkpoint, a versioned manifest, member sizes and SHA-256 digests.
4. Streams a compressed tar archive through age X25519 encryption to a private
   temporary file. No restore private key is required or accepted.
5. Uploads one atomic object under `sets/`, downloads it over HTTPS, and compares
   its size and full ciphertext SHA-256 with the local encrypted file. An ETag,
   successful PUT or upload completion timestamp alone is not coverage.
6. Renews coverage using the **snapshot start**, then prunes retention. Temporary
   snapshots are removed on ordinary success/failure. Files are 0600, dirs 0700.

Each attempt has a two-minute deadline. New league/auth/evidence writes stop at
**14 minutes**, before the 15-minute RPO is breached. Admission is checked again
**after waiting for the writer**. Hosted transactions have a 30-second context
limit, so work admitted just before the cutoff cannot run past the safety margin.
Wall-clock rollback cannot extend coverage: the manager also tracks monotonic
elapsed time. Restart begins paused and must verify a fresh set, not trust a
local receipt. A stopped process or failed connection cannot extend coverage.
Read/auth operations that require session writes also pause; no stale sign-in
state is silently accepted. The due worker uses the same DB admission gate.

## Retention and cost limits

Keep the newest twelve objects, plus the newest object per UTC day for the last
30 days. The encrypted object limit is 128 MiB; the total uncompressed member
limit is 512 MiB. At most 43 retained full sets plus the next in-flight upload
are about 5.9 GB, below R2's current 10 GB Standard free allowance, assuming a
dedicated bucket with no foreign/future-dated objects. Typical smaller leagues
use less. PUT/GET/LIST traffic at a five-minute cadence is well below R2's current
monthly free operation allowances. Check Cloudflare's billing page: quotas are
account-wide, and other buckets count too. Failed pruning is visible as a status
failure; do not ignore it indefinitely. Never enable public access.

Oversized sets are not uploaded or counted as coverage. Changes pause rather
than silently omitting evidence or billing unbounded data. Foreign object names
are not deleted. Retention runs only after a new set has passed full read-back.
This is recovery against physical host loss, not immutability against someone
who steals the running server's bucket read/write credential. Keep desktop
checkpoints too; R2 read/write credentials can delete objects.

## Restoring

Follow [RESTORE.md](RESTORE.md): install a fresh CT on the spare, then run
`twr-restore` inside it. It downloads the newest set from R2 into the CT, unlocks
it with the restore key typed at the console (held in memory, never saved),
compares the saved checkpoint, signs every owner out and only then starts the
service. Do not copy a decrypted `server.db` into a server by hand: sessions and
used recovery codes from before the snapshot would come back. The fresh CT keeps
its own TLS key and Tunnel; the old R2 token is replaced with a new one.

## Research and evidence

Checked upstream identity and maintained APIs before choosing age v1.3.2
(`filippo.io/age`, upstream FiloSottile/age) and MinIO Go SDK v7.4.1
(`github.com/minio/minio-go/v7`). MinIO provides the focused S3 client without
assembling the modular AWS SDK's config/credential stack; no custom signing or
crypto. Transitive module changes are their required minimum versions, not an
unrelated blanket upgrade. TLS uses the CT's read-only CA bundle. SDK unsigned-
payload mode over verified TLS avoids S3 streaming-signature compatibility issues;
the object is independently verified by full read-back, not by a signature shim.

- https://github.com/FiloSottile/age
- https://github.com/minio/minio-go
- https://developers.cloudflare.com/r2/api/tokens/
- https://developers.cloudflare.com/r2/platform/pricing/

`internal/offsite` race tests exercise the real SDK against an in-process TLS S3
server: encrypted upload/read-back, off-host-only restore, missing/tampered objects,
wrong member digest, truncated ciphertext, retention and coverage boundaries.
`internal/db` tests exercise snapshot/config writer exclusion and a queued write
that loses coverage without changing stored state. Real Cloudflare credentials,
Proxmox host loss and the four-hour spare-host drill are not exercised by these
unit/integration tests; they remain live acceptance, not a claimed pass.
