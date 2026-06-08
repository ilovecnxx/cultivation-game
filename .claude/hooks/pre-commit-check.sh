#!/usr/bin/env bash
# Pre-commit quality check: runs go vet + tests before git commit
# Blocks commit if critical issues found

cmd=$(echo "${CLAUDE_TOOL_INPUT:-}" | python3 -c "import json,sys; d=json.load(sys.stdin); print(d.get('command',''))" 2>/dev/null)

# Only trigger on git commit
if ! echo "$cmd" | grep -qE 'git commit'; then
  exit 0
fi

echo "[Hook] Running pre-commit checks..." >&2

# Go vet check
if [ -d "/root/projects/cultivation-game/backend" ]; then
  cd /root/projects/cultivation-game/backend
  if ! go vet ./... 2>&1; then
    echo "[Hook] ❌ go vet found issues. Please fix before committing." >&2
    exit 2  # Block commit
  fi
  echo "[Hook] ✅ go vet passed" >&2
fi

# Quick unit test (timeout 60s to avoid blocking too long)
if ! go test ./... -count=1 -timeout=60s 2>&1; then
  echo "[Hook] ❌ Tests failed. Please fix before committing." >&2
  exit 2  # Block commit
fi
echo "[Hook] ✅ All tests passed" >&2

exit 0
