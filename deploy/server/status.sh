#!/bin/sh
# CT-only console dashboard (CHECKLIST.md). It only reads, and contacts nothing outside
# this container: the server's private status socket and two loopback addresses.
set -eu
[ "$(id -u)" = 0 ] || { echo 'Run this as root in the CT console.' >&2; exit 1; }
exec su -s /bin/sh twr -c '/usr/local/bin/twr-server status -socket /var/lib/twr/status.sock -origin 127.0.0.1:8443 -origin-cert /etc/twr/origin.pem -tunnel 127.0.0.1:20241 -state /var/lib/twr'
