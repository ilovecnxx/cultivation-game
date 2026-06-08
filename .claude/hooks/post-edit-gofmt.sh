#!/usr/bin/env bash
# Auto-format Go files after Write/Edit
# Uses CLAUDE_FILE_PATH env var from Claude Code

file="${CLAUDE_FILE_PATH:-${CLAUDE_TOOL_INPUT_FILE_PATH}}"
if [ -z "$file" ]; then
  # Try to extract from tool input JSON
  file=$(echo "${CLAUDE_TOOL_INPUT:-}" | python3 -c "import json,sys; d=json.load(sys.stdin); print(d.get('file_path',''))" 2>/dev/null)
fi

# Only format Go files
if [ -n "$file" ] && echo "$file" | grep -qE '\.go$'; then
  cd /root/projects/cultivation-game/backend 2>/dev/null || cd /root/projects/cultivation-game 2>/dev/null || true
  gofmt -w "$file" 2>/dev/null && echo "[Hook] gofmt formatted: $file" >&2 || true
fi
exit 0
