# services/CLAUDE.md — Go Microservice Conventions

<!-- This file loads ON-DEMAND when Claude touches files under services/. -->

## Service Structure (standard layout)

```
services/<name>/
├── cmd/<name>/main.go          # Entry: config load, DB connect, Gin router, graceful shutdown
├── internal/
│   ├── config/config.go        # Config struct with Load() using env vars
│   ├── handler/<name>_handler.go   # Gin handlers (request binding + validation only)
│   ├── service/<name>_service.go   # Business logic (orchestration, validation, transactions)
│   ├── model/<name>.go             # Domain types / request-response structs
│   ├── repository/
│   │   ├── mysql/<name>_repo.go    # MySQL access (database/sql preferred, some use GORM)
│   │   └── redis/<name>_redis.go   # Redis access (optional)
│   └── data/*.json                 # Static game data (monsters, skills, realm configs)
├── go.mod
└── Dockerfile                     # Multi-stage: golang:1.22-alpine → alpine:3.19
```

## Running services

All Go commands from `backend/`:

```bash
# Single service
go run ../services/player/cmd/player/main.go

# Per-service tests (preferred — scoped, fast)
go test ../services/player/... -v
go test ../services/combat/... -v -run TestBattleStart

# All services tests
go test ../services/... -count=1 -timeout=120s
```

## Code Conventions

### Handler pattern
- One handler struct per domain concept, injected with its service and logger
- Route: `GET /api/v1/<resource>/:id`, `POST /api/v1/<resource>`
- Errors: `{"code": <int>, "msg": "<description>"}`
- Success: `{"code": 0, "msg": "success", "data": <result>}`
- Never put business logic in handlers — they parse requests, call services, write responses.

### Service pattern
- Error wrapping: `fmt.Errorf("操作描述: %w", err)` — always wrap, never swallow.
- Transactions controlled at service layer.
- Static config data loaded from `internal/data/*.json` at startup, with mutex-guarded hot-reload.

### Repository pattern
- MySQL: prefer `database/sql` (raw SQL) — check existing code to confirm per service.
- Redis: sessions, leaderboards (ZSET), rate limiting, online status.
- Use prepared statements / parameterized queries.

### Config pattern
- All config from env vars with defaults in `config.go`.
- Never hardcode hostnames, ports, or credentials.

## Service Dependencies

```
Gateway ──▶ all services (routes by msg_id, not by service name)
Combat  ──▶ Player (read stats, grant rewards), Cultivation (recalc after breakthrough)
Social  ──▶ Player (read nickname/realm)
World   ──▶ Player (read quest progress)
Trade   ──▶ Player (read inventory, update money)
Ranking ──▶ Player (read rank data), Combat (read PVP score)
```

## Port Assignments

| Service | Port |
|---------|------|
| Gateway | 8080 (WS), 8081 (HTTP) |
| Auth | 8082 |
| Player | 8083 |
| Cultivation | 8084 |
| Combat | 8085 |
| Social | 8086 |
| World | 8087 |
| Trade | 8088 |
| Ranking | 8089 |

New services start from 8090.

## Adding a New Service

Use the `add-microservice` skill (`/add-microservice`) which generates the complete skeleton. Then:
1. Add to `backend/go.work` use block
2. Add K8s manifests to `deploy/k8s/`
3. Update the service topology in root `CLAUDE.md`
4. Run `cd backend && go mod tidy` for the new module
