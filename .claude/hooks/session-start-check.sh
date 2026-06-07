#!/usr/bin/env bash
# Check Docker container status on session start
# Run as a Stop hook so it fires periodically

check_containers() {
  echo "=== Container Status ===" >&2
  if command -v docker &>/dev/null && docker info &>/dev/null 2>&1; then
    containers=$(docker ps --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}" 2>/dev/null | head -12)
    if [ -n "$containers" ]; then
      echo "$containers" >&2
    else
      echo "No running containers." >&2
    fi
  else
    echo "Docker not available." >&2
  fi
  echo "=======================" >&2
}

check_containers
exit 0
