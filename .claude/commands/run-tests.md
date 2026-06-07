---
name: run-tests
description: Run tests for a specific service or all services
args:
  - name: service
    description: Service name (e.g., player, combat) or 'all'
    required: false
---
Run Go tests:
- If service is 'all' or empty: cd backend && go test ../services/... -count=1 -timeout=120s
- Otherwise: cd backend && go test ../services/$ARGUMENTS.service/... -v -count=1 -timeout=60s
Report pass/fail and any failures.
