#!/usr/bin/env bash
# TheWarRoom coverage soak (11-04). Read-only walk of the 5 locked screens + Trade Floor
# on a coverage-instrumented HQ-1 build, against a throwaway copy of ~/f2click.
# Clicks ONLY the nav rail and screen tabs (coordinates measured 11-04 at 1920x1080).
# Usage: coverage-soak.sh [HOURS]   (PREWALK=1 -> one loop, every shot kept)
# Output: ~/scratch/soak/run-<ts>/ ; DONE file holds the exit status.
set -uo pipefail
HOURS=${1:-3}
SRC=$HOME/work/TheWarRoom/.worktrees/soak          # detached at c196a523, built with GOFLAGS=-cover
BIN=$SRC/build/bin/thewarroom
RUN=$HOME/scratch/soak/run-$(date +%m%d-%H%M)
PROF=$RUN/profile COV=$RUN/cov SHOTS=$RUN/shots LOG=$RUN/walk.log
export DISPLAY=:5 PATH=/usr/local/go/bin:$HOME/go/bin:$PATH GOTOOLCHAIN=go1.26.10
mkdir -p "$PROF/config/TheWarRoom" "$PROF/data" "$COV" "$SHOTS"
status=1; pid=''
log(){ printf '%s %s\n' "$(TZ=America/Chicago date +%H:%M:%S)" "$*" >>"$LOG"; }
finish(){
  if [[ -n "$pid" ]] && kill -0 "$pid" 2>/dev/null; then
    kill -TERM "$pid"; for _ in $(seq 60); do kill -0 "$pid" 2>/dev/null || break; sleep 1; done
  fi
  ( cd "$SRC" && go tool covdata percent -i="$COV" >"$RUN/percent.txt" 2>&1 \
    && go tool covdata textfmt -i="$COV" -o "$RUN/cover.out" \
    && go tool cover -func="$RUN/cover.out" >"$RUN/func.txt" 2>&1 \
    && python3 -I "$HOME/scratch/soak/soak-report.py" "$RUN" >"$RUN/report.log" 2>&1 ) || status=3
  rm -rf "$PROF"                                    # copy of his data; never keep it
  log "finish status=$status"; echo "$status" >"$RUN/DONE"
}
trap finish EXIT

[[ $(xrandr | head -1) == *'current 1920 x 1080'* ]] || { log 'DISPLAY :5 not 1920x1080'; status=1; exit; }
go version -m "$BIN" | grep -q -- '-cover=true' || { log 'binary not coverage-built'; status=1; exit; }
[[ $(df --output=avail -k "$HOME" | tail -1) -gt 4000000 ]] || { log 'under 4G free'; status=1; exit; }

rsync -a --exclude backups "$HOME/f2click/config/TheWarRoom/" "$PROF/config/TheWarRoom/"
cp -a "$HOME/f2click/data/." "$PROF/data/"
XDG_CONFIG_HOME=$PROF/config XDG_DATA_HOME=$PROF/data "$BIN" \
  --relink-identity-from "$HOME/f2click/config/TheWarRoom" >"$RUN/relink.log" 2>&1
# F8: the old-schema copy migrates on launch (full integrity check x2 + backup) inside a 2-min
# deadline and fails about half the time. A failed migration rolls back; retry on the same profile.
for attempt in 1 2 3; do
  GOCOVERDIR=$COV XDG_CONFIG_HOME=$PROF/config XDG_DATA_HOME=$PROF/data "$BIN" >"$RUN/app.log" 2>&1 &
  pid=$!
  t0=$(date +%s)   # nav is dead until the launch refresh lands (measured ~3 min on 11-04)
  for _ in $(seq 300); do
    grep -qE 'launch refresh:|startup failed' "$RUN/app.log" && break
    kill -0 $pid || break; sleep 2
  done
  grep -q 'launch refresh:' "$RUN/app.log" && break
  log "attempt=$attempt failed after $(( $(date +%s)-t0 ))s: $(grep -m1 -oE 'startup failed.*' "$RUN/app.log")"
  cp "$RUN/app.log" "$RUN/app-attempt$attempt.log"
  kill -TERM $pid 2>/dev/null; wait $pid 2>/dev/null; pid=''
done
[[ -n "$pid" ]] || { log 'startup failed 3 times'; status=2; exit; }
log "usable after $(( $(date +%s)-t0 ))s"; sleep 10
win=$(xdotool search --onlyvisible --pid $pid 2>/dev/null | head -1)
[[ -n "$win" ]] || { log 'no window'; status=2; exit; }
xdotool windowmove "$win" 0 0 windowsize "$win" 1920 1080
log "start pid=$pid bin=$(sha256sum "$BIN" | cut -c1-12) src=$(git -C "$SRC" rev-parse --short HEAD) hours=$HOURS"
printf 'ts\tloop\trss_kb\n' >"$RUN/rss.tsv"
# DIAG=1: per-process CPU/RSS every 5 s (Go app vs WebKit children); first stalled loop -> thread
# snapshot + Go goroutine dump (SIGQUIT ends the run; coverage is not flushed in that case).
if [[ ${DIAG:-0} == 1 ]]; then
  ( while kill -0 $pid 2>/dev/null; do
      kids=$(pgrep -P $pid | tr '\n' ,)
      ps -o pid=,comm=,pcpu=,rss= -p "$pid,${kids%,}" 2>/dev/null | awk -v t="$(date +%H:%M:%S)" '{print t"\t"$1"\t"$2"\t"$3"\t"$4}' >>"$RUN/procs.tsv"
      sleep 5
    done ) &
fi

# name:x:y:dwell  (R = nav rail x=80; HQ has its own rail, Home there is y=221)
STEPS=(
 home:80:175:8 home-brief:228:79:6 home-depth:330:79:6 home-nflstats:448:79:6 home-brief:228:79:3
 home-alerts:414:21:6 home-digests:491:21:6 home-seasonal:318:21:6
 war:80:212:8 war-market:437:22:8 war-draft:533:22:8 war-plans:624:22:8 war-trade:344:22:8
 trade:80:286:6 trade-draft:434:22:6 trade-desk:344:22:4
 pulse:80:322:8 pulse-race:392:22:8 pulse-now:338:22:8
 control:80:359:8 cr-situations:350:78:5 cr-rules:442:78:5 cr-season:540:78:5 cr-fix:652:78:5
 cr-picks:708:78:5 cr-people:780:78:5 cr-league:856:78:5 cr-server:936:78:5 cr-jobs:224:78:5
 hq:80:249:10 hq-cap:479:18:10 hq-lineup:309:18:8 hq-home:80:221:8
)
end=$(( $(date +%s) + HOURS*3600 )); loop=0
while :; do
  loop=$((loop+1))
  for s in "${STEPS[@]}"; do
    IFS=: read -r name x y dwell <<<"$s"
    kill -0 $pid 2>/dev/null || { log "APP EXITED loop=$loop before $name"; status=2; exit; }
    xdotool mousemove "$x" "$y" click 1; sleep "$dwell"
    f=$SHOTS/L$loop-$name.png; scrot -o "$f"
    log "loop=$loop step=$name hash=$(md5sum "$f" | cut -c1-8)"
    # keep shots of loop 1 and every 10th loop only (disk); prewalk keeps all
    [[ ${PREWALK:-0} == 1 || $loop == 1 || $((loop % 10)) == 0 ]] || rm -f "$f"
  done
  printf '%s\t%s\t%s\n' "$(date +%s)" "$loop" "$(ps -o rss= -p $pid)" >>"$RUN/rss.tsv"
  # UI keeps up when most of the ${#STEPS[@]} steps give distinct frames (11-04 prewalk: 9 of 33, clicks queued minutes)
  log "loop=$loop distinct=$(grep "loop=$loop step=" "$LOG" | awk '{print $4}' | sort -u | wc -l)/${#STEPS[@]}"
  if [[ ${DIAG:-0} == 1 ]] && (( $(grep "loop=$loop step=" "$LOG" | awk '{print $4}' | sort -u | wc -l) < 15 )) && (( loop > 1 )); then
    log "STALL loop=$loop: capturing threads + goroutines"
    top -b -n 1 -H -p "$pid,$(pgrep -P $pid | paste -sd,)" | head -40 >"$RUN/stall-threads.txt"
    kill -QUIT "$pid"; sleep 5; pid=''; status=5; exit
  fi
  [[ $(df --output=avail -k "$HOME" | tail -1) -gt 3000000 ]] || { log 'disk under 3G, stopping'; status=4; exit; }
  [[ ${PREWALK:-0} == 1 || $(date +%s) -ge $end ]] && break
done
log "walk complete loops=$loop"; status=0
