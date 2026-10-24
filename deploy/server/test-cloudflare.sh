#!/bin/sh
# Runs in the disposable Alpine test container after install.sh.
set -eu
umask 077
mkdir -p /tmp/fake-api
cat > /tmp/fake-api/curl <<'CURL'
#!/bin/sh
for arg do
  case "$arg" in
    *'/zones?name='*) printf '%s\n' '{"success":true,"result":[{"id":"zone-id","account":{"id":"account-id"}}]}'; exit;;
    *'/dns_records?name='*)
      if [ -e /tmp/fake-api/occupied ]; then
        printf '%s\n' '{"success":true,"result":[{"id":"existing"}]}'
      else
        printf '%s\n' '{"success":true,"result":[]}'
      fi
      exit;;
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
before=$(sha256sum /etc/cloudflared/*)
touch /tmp/fake-api/occupied
if PATH=/tmp/fake-api:$PATH sh /root/twr/cloudflare.sh /tmp/cloudflare-input; then exit 1; fi
[ "$before" = "$(sha256sum /etc/cloudflared/*)" ]
[ "$(stat -c %a /etc/cloudflared/credentials.json)" = 640 ]
if su -s /bin/sh twr -c 'cat /etc/cloudflared/credentials.json'; then exit 1; fi
rm /tmp/fake-api/occupied
PATH=/tmp/fake-api:$PATH sh /root/twr/cloudflare.sh --rollback /tmp/cloudflare-input
[ ! -e /etc/twr-cloudflare.ids ]
[ ! -e /etc/cloudflared/credentials.json ]
[ "$(wc -l < /tmp/fake-api/deleted)" = 2 ]
touch /tmp/fake-api/fail-dns
if PATH=/tmp/fake-api:$PATH sh /root/twr/cloudflare.sh /tmp/cloudflare-input; then exit 1; fi
[ ! -e /etc/twr-cloudflare.ids ]
[ "$(wc -l < /tmp/fake-api/deleted)" = 3 ]
rm -rf /tmp/fake-api /tmp/cloudflare-input
printf '%s\n' 'Cloudflare config, CA verification, route denial and occupied DNS preservation: PASS (API simulated)'
