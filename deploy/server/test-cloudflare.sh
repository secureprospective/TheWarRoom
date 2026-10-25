#!/bin/sh
# Runs in the disposable Alpine test container after install.sh.
set -eu
umask 077
mkdir -p /tmp/fake-api
cat > /tmp/fake-api/curl <<'CURL'
#!/bin/sh
output=
previous=
method=GET
url=
for arg do
  [ "$previous" != --output ] || output=$arg
  [ "$previous" != -X ] || method=$arg
  case "$arg" in https://*) url=$arg;; esac
  case "$previous:$arg" in --data-binary:@*) cp "${arg#@}" /tmp/fake-api/last-payload;; esac
  previous=$arg
done
printf '%s %s\n' "$method" "$url" >> /tmp/fake-api/calls
[ ! -e /tmp/fake-api/outage ] || exit 7
for arg do
  case "$arg" in
    *'/cfd_tunnel/11111111-1111-4111-8111-111111111111')
      [ ! -e /tmp/fake-api/fail-tunnel ] || exit 7
      if [ -e /tmp/fake-api/missing-tunnel ]; then
        printf '%s\n' '{"success":false}' > "$output"
        printf 404
        exit 22
      fi;;
  esac
done
respond() {
for arg do
  case "$arg" in
    *'/zones?name='*) printf '%s\n' '{"success":true,"result":[{"id":"zone-id","account":{"id":"account-id"}}]}'; exit;;
    *'/dns_records?name='*)
      if [ -e /tmp/fake-api/occupied-other ]; then
        printf '%s\n' '{"success":true,"result":[{"id":"existing","type":"A","content":"192.0.2.1"}]}'
      elif [ -e /tmp/fake-api/occupied ]; then
        printf '%s\n' '{"success":true,"result":[{"id":"existing","type":"CNAME","content":"22222222-2222-4222-8222-222222222222.cfargotunnel.com"}]}'
      else
        printf '%s\n' '{"success":true,"result":[]}'
      fi
      exit;;
    *'/cfd_tunnel/22222222-2222-4222-8222-222222222222')
      if [ "$method" = DELETE ]; then
        printf '%s\n' "$arg" >> /tmp/fake-api/deleted
        printf '%s\n' '{"success":true,"result":null}'
      elif [ -e /tmp/fake-api/foreign-tunnel ]; then
        printf '%s\n' '{"success":true,"result":{"name":"homelab","status":"down","connections":[],"deleted_at":null}}'
      elif [ -e /tmp/fake-api/old-connected ]; then
        printf '%s\n' '{"success":true,"result":{"name":"twr-105","status":"healthy","connections":[{"id":"c"}],"deleted_at":null}}'
      else
        printf '%s\n' '{"success":true,"result":{"name":"twr-105","status":"down","connections":[],"deleted_at":null}}'
      fi
      exit;;
    *'/dns_records/existing')
      [ "$method" != DELETE ] || printf '%s\n' "$arg" >> /tmp/fake-api/deleted
      printf '%s\n' '{"success":true,"result":{"id":"existing"}}'; exit;;
    *'/cfd_tunnel') printf '%s\n' '{"success":true,"result":{"id":"11111111-1111-4111-8111-111111111111"}}'; exit;;
    *'/dns_records')
      [ ! -e /tmp/fake-api/fail-dns ] || exit 22
      printf '%s\n' '{"success":true,"result":{"id":"dns-id"}}'; exit;;
    *'/dns_records/dns-id'|*'/cfd_tunnel/11111111-1111-4111-8111-111111111111')
      printf '%s\n' "$arg" >> /tmp/fake-api/deleted
      printf '%s\n' '{"success":true,"result":null}'; exit;;
  esac
done
exit 1
}
respond "$@" > "$output" || exit $?
printf 200
CURL
chmod 755 /tmp/fake-api/curl
printf '%s\n' '{"zone":"example.org","hostname":"league.example.org","api_token":"test-only"}' > /tmp/cloudflare-input
PATH=/tmp/fake-api:$PATH sh /root/twr/cloudflare.sh /tmp/cloudflare-input
/usr/local/bin/cloudflared --config /etc/cloudflared/config.yml tunnel ingress validate
for path in /v1/native/ledger /v1/auth/login /v1/native/invalidations; do
  /usr/local/bin/cloudflared --config /etc/cloudflared/config.yml tunnel ingress rule "https://league.example.org$path" | grep -q 'Matched rule #0'
done
for url in https://league.example.org/admin https://league.example.org/metrics https://other.example.org/v1/native/ledger; do
  /usr/local/bin/cloudflared --config /etc/cloudflared/config.yml tunnel ingress rule "$url" | grep -q 'Matched rule #1'
done
jq -e '.ingress[0].originRequest | .caPool == "/etc/cloudflared/origin.pem" and .originServerName == "localhost" and (.noTLSVerify != true)' /etc/cloudflared/config.yml
# Existing DNS is never silently replaced; credentials/config stay unchanged.
[ "$(stat -c %a /etc/cloudflared/credentials.json)" = 640 ]
if su -s /bin/sh twr -c 'cat /etc/cloudflared/credentials.json'; then exit 1; fi
PATH=/tmp/fake-api:$PATH sh /root/twr/cloudflare.sh --rollback /tmp/cloudflare-input
before=$(sha256sum /etc/cloudflared/*)
touch /tmp/fake-api/occupied
if PATH=/tmp/fake-api:$PATH sh /root/twr/cloudflare.sh /tmp/cloudflare-input; then exit 1; fi
[ "$before" = "$(sha256sum /etc/cloudflared/*)" ]
rm /tmp/fake-api/occupied
PATH=/tmp/fake-api:$PATH sh /root/twr/cloudflare.sh --rollback /tmp/cloudflare-input
[ ! -e /etc/twr-cloudflare.ids ]
[ ! -e /etc/cloudflared/credentials.json ]
[ "$(wc -l < /tmp/fake-api/deleted)" = 2 ]
touch /tmp/fake-api/fail-dns
if PATH=/tmp/fake-api:$PATH sh /root/twr/cloudflare.sh /tmp/cloudflare-input; then exit 1; fi
[ ! -e /etc/twr-cloudflare.ids ]
[ "$(wc -l < /tmp/fake-api/deleted)" = 3 ]
rm /tmp/fake-api/fail-dns
PATH=/tmp/fake-api:$PATH sh /root/twr/cloudflare.sh /tmp/cloudflare-input
cp /etc/twr-cloudflare.ids /tmp/expected-ids
touch /tmp/fake-api/outage /tmp/fake-api/occupied
if PATH=/tmp/fake-api:$PATH sh /root/twr/cloudflare.sh --rollback /tmp/cloudflare-input; then exit 1; fi
cmp /etc/twr-cloudflare.ids /tmp/expected-ids
[ "$(wc -l < /tmp/fake-api/deleted)" = 3 ]
rm /tmp/fake-api/outage
# Normal entry retries recorded cleanup first, then refuses unrelated occupied DNS.
if PATH=/tmp/fake-api:$PATH sh /root/twr/cloudflare.sh /tmp/cloudflare-input; then exit 1; fi
[ ! -e /etc/twr-cloudflare.ids ]
[ ! -e /etc/cloudflared/credentials.json ]
[ "$(wc -l < /tmp/fake-api/deleted)" = 5 ]
[ "$(grep -c '/dns_records/dns-id$' /tmp/fake-api/deleted)" = 2 ]
[ "$(grep -c '/cfd_tunnel/11111111-1111-4111-8111-111111111111$' /tmp/fake-api/deleted)" = 3 ]
rm /tmp/fake-api/occupied
PATH=/tmp/fake-api:$PATH sh /root/twr/cloudflare.sh /tmp/cloudflare-input
touch /tmp/fake-api/fail-tunnel
if PATH=/tmp/fake-api:$PATH sh /root/twr/cloudflare.sh --rollback /tmp/cloudflare-input; then exit 1; fi
jq -e 'has("dns") == false and has("tunnel")' /etc/twr-cloudflare.ids
[ "$(wc -l < /tmp/fake-api/deleted)" = 6 ]
rm /tmp/fake-api/fail-tunnel
# A resource deleted remotely before the local ledger update is safe to retry.
touch /tmp/fake-api/missing-tunnel
PATH=/tmp/fake-api:$PATH sh /root/twr/cloudflare.sh --rollback /tmp/cloudflare-input
[ ! -e /etc/twr-cloudflare.ids ]
[ "$(wc -l < /tmp/fake-api/deleted)" = 6 ]
rm /tmp/fake-api/missing-tunnel
# Spare-host takeover: only a disconnected TheWarRoom Tunnel's record is repointed.
printf '%s\n' '{"zone":"example.org","hostname":"league.example.org","api_token":"test-only","replace":true}' > /tmp/replace-input
touch /tmp/fake-api/occupied
for refusal in old-connected foreign-tunnel occupied-other; do
  touch "/tmp/fake-api/$refusal"
  : > /tmp/fake-api/calls
  if PATH=/tmp/fake-api:$PATH sh /root/twr/cloudflare.sh /tmp/replace-input; then exit 1; fi
  rm "/tmp/fake-api/$refusal"
  [ ! -e /etc/twr-cloudflare.ids ]
  if grep -E '^(POST|PATCH|DELETE) ' /tmp/fake-api/calls; then exit 1; fi
done
PATH=/tmp/fake-api:$PATH sh /root/twr/cloudflare.sh /tmp/replace-input
grep -qx 'PATCH https://api.cloudflare.com/client/v4/zones/zone-id/dns_records/existing' /tmp/fake-api/calls
jq -e '.content == "11111111-1111-4111-8111-111111111111.cfargotunnel.com"' /tmp/fake-api/last-payload
grep -q '/cfd_tunnel/22222222-2222-4222-8222-222222222222$' /tmp/fake-api/deleted
jq -e '.dns == "existing" and .tunnel == "11111111-1111-4111-8111-111111111111"' /etc/twr-cloudflare.ids
/usr/local/bin/cloudflared --config /etc/cloudflared/config.yml tunnel ingress validate
PATH=/tmp/fake-api:$PATH sh /root/twr/cloudflare.sh --rollback /tmp/replace-input
[ ! -e /etc/twr-cloudflare.ids ]
rm /tmp/expected-ids /tmp/replace-input
rm -rf /tmp/fake-api /tmp/cloudflare-input
printf '%s\n' 'Cloudflare config, CA verification, route denial, occupied DNS preservation and lost-server hostname takeover: PASS (API simulated)'
