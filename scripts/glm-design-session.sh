#!/usr/bin/env bash
# glm-design-session.sh — scripted GLM design run for the UI wireframe sessions
# (docs/ui/Wireframe_Session_Plan.md §5). Ships the brief to the Beelink, runs the
# z.ai coding-endpoint call there detached (the API key stays on the Beelink), and
# blocks locally until the remote EXIT= sentinel lands. Launch via a background
# task and let the harness notify on completion — never hand-poll.
#
# Usage: glm-design-session.sh <brief.md> <out.md> [model]
#   model defaults to glm-5.2; pass glm-4.7 as the hang fallback.
set -euo pipefail

BRIEF="${1:?usage: glm-design-session.sh <brief.md> <out.md> [model]}"
OUT="${2:?usage: glm-design-session.sh <brief.md> <out.md> [model]}"
MODEL="${3:-glm-5.2}"
BEELINK="chris@192.168.1.190"
SSHKEY="/root/.ssh/beelink"
RUNID="glmds-$(date +%s)-$$"
RDIR="/tmp/glm-design/${RUNID}"
TIMEOUT_S="${GLM_TIMEOUT_S:-900}"   # generous: the Beelink is shared

ssh -i "$SSHKEY" "$BEELINK" "mkdir -p '$RDIR'"
scp -q -i "$SSHKEY" "$BRIEF" "$BEELINK:$RDIR/brief.md"

# Remote job: detached (setsid) so an SSH channel drop can't kill it; writes
# incrementally and appends the EXIT= sentinel last.
ssh -i "$SSHKEY" "$BEELINK" "setsid bash -c '
  set -a; source ~/.config/opencode/zai.env; set +a
  cd \"$RDIR\"
  python3 - \"$MODEL\" <<\"PYEOF\" > out.md 2> err.log; echo \"EXIT=\$?\" >> out.md
import json, os, sys, urllib.request
model = sys.argv[1]
brief = open(\"brief.md\").read()
req = urllib.request.Request(
    \"https://api.z.ai/api/coding/paas/v4/chat/completions\",
    data=json.dumps({
        \"model\": model,
        \"messages\": [{\"role\": \"user\", \"content\": brief}],
        \"max_tokens\": 8000,
        \"temperature\": 0.8,
    }).encode(),
    headers={\"Content-Type\": \"application/json\",
             \"Authorization\": \"Bearer \" + os.environ[\"ZAI_API_KEY\"]})
with urllib.request.urlopen(req, timeout=840) as r:
    body = json.load(r)
print(body[\"choices\"][0][\"message\"][\"content\"])
PYEOF
' >/dev/null 2>&1 &"

# Local wait: poll the remote sentinel (inside this one process — the caller
# runs us in the background and gets a single completion notification).
elapsed=0
while (( elapsed < TIMEOUT_S )); do
  if ssh -i "$SSHKEY" "$BEELINK" "grep -q '^EXIT=' '$RDIR/out.md' 2>/dev/null"; then
    scp -q -i "$SSHKEY" "$BEELINK:$RDIR/out.md" "$OUT"
    code=$(grep '^EXIT=' "$OUT" | tail -1 | cut -d= -f2)
    sed -i '/^EXIT=/d' "$OUT"
    if [[ "$code" != "0" ]]; then
      echo "GLM run FAILED (remote exit $code); remote err.log:" >&2
      ssh -i "$SSHKEY" "$BEELINK" "cat '$RDIR/err.log'" >&2 || true
      exit 1
    fi
    echo "GLM run OK: $(wc -l < "$OUT") lines -> $OUT (model $MODEL, ${elapsed}s)"
    exit 0
  fi
  sleep 20; elapsed=$((elapsed + 20))
done
echo "GLM run TIMED OUT after ${TIMEOUT_S}s (run dir $RDIR on the Beelink survives — inspect or re-collect)" >&2
exit 2
