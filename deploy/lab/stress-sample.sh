#!/bin/sh
# Lab only: run inside a server CT. Every 10 s, one CSV line: time, twr-server pid, RSS anon/file (kB),
# CT cgroup memory current/peak (B), server.db and WAL size (B), open fds, established TCP conns, server CPU
# (utime+stime clock ticks since start) and CT cgroup CPU (usage_usec): CPU% is the delta between lines.
# Survives server restarts (re-finds the pid). Usage: stress-sample.sh > /root/sample.csv &
db=${1:-/var/lib/twr/server.db}
echo t,pid,rss_anon_kb,rss_file_kb,cg_current,cg_peak,db_b,wal_b,fds,tcp_est,cpu_ticks,cg_cpu_usec
# The server runs inside bwrap: find it by /proc exe + argv[1], not pgrep (whose -f pattern and PATH
# differ between busybox and the pct exec environment, which left the pid column empty on CT 253).
server_pid() {
  for d in /proc/[0-9]*; do
    [ "$(readlink "$d/exe" 2>/dev/null)" = /usr/local/bin/twr-server ] || continue
    [ "$(tr '\0' ' ' < "$d/cmdline" 2>/dev/null | cut -d' ' -f2)" = hosted ] && { echo "${d#/proc/}"; return; }
  done
}
while :; do
  pid=$(server_pid)
  anon=; file=; fds=; ticks=
  if [ -n "$pid" ]; then
    anon=$(awk '/^RssAnon/{print $2}' /proc/$pid/status 2>/dev/null)
    file=$(awk '/^RssFile/{print $2}' /proc/$pid/status 2>/dev/null)
    fds=$(ls /proc/$pid/fd 2>/dev/null | wc -l)
    # Fields 14 and 15 of /proc/pid/stat, counted after the parenthesised command name.
    ticks=$(sed 's/^.*) //' /proc/$pid/stat 2>/dev/null | awk '{print $12 + $13}')
  fi
  est=$(awk 'NR>1 && $4=="01"' /proc/net/tcp /proc/net/tcp6 2>/dev/null | wc -l)
  echo "$(date +%s),$pid,$anon,$file,$(cat /sys/fs/cgroup/memory.current),$(cat /sys/fs/cgroup/memory.peak 2>/dev/null),$(stat -c %s "$db" 2>/dev/null),$(stat -c %s "$db-wal" 2>/dev/null || echo 0),$fds,$est,$ticks,$(awk '/^usage_usec/{print $2}' /sys/fs/cgroup/cpu.stat 2>/dev/null)"
  sleep 10
done
