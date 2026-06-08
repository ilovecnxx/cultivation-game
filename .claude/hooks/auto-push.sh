#!/bin/bash
# Auto-save: commits changes locally but does NOT push to any branch.
cd /root/projects/cultivation-game
git add -A 2>/dev/null
git diff --cached --quiet && exit 0
git commit -m "auto: $(date +%m-%d\ %H:%M) — $(git diff --cached --name-only | head -3 | paste -sd ',')" 2>/dev/null
