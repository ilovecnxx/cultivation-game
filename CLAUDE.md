# CLAUDE.md — 九转修仙 (Cultivation Game)

<!--
  HUMAN NOTES (token-free — stripped from Claude's context at runtime):
  - This file is the ROOT context. Keep it under 180 lines.
  - Subdirectory CLAUDE.md files load ON-DEMAND when Claude touches files there.
  - Review this file every 3-6 months, and after major model releases.
  - If Claude gets something wrong twice, add a gotcha below.
  - If a convention changes, update this file IMMEDIATELY.
  - DRI: the developer who last edited this file owns keeping it current.
-->

## Project (3 sentences)

九转修仙 is an MMORPG cultivation-themed web game. Players progress from mortal through 9 realms via meditation, combat (PVE/PVP), exploration, and social systems. Backend is Go microservices (Gateway → 8 domain services), frontend is Vue 3 + TypeScript + Pinia.

## Quick Commands

```bash
# Backend: always run Go commands from backend/ (go.work lives there)
cd backend
go test ./... -race -count=1 -timeout=60s          # All tests
go test ./services/combat/... -v                    # Single service tests
go vet ./...                                         # Lint
gofmt -l -s ./services/...                           # Format check

# Frontend
cd frontend
npm run dev                    # http://localhost:3000
npm run test                   # Vitest
npx vue-tsc --noEmit           # Type-check

# Docker
make -f deploy/Makefile dev    # Start all services
make -f deploy/Makefile logs   # Tail all logs
```

## Codebase Map

<!-- Each top-level dir with a one-liner. Subdirectory CLAUDE.md files provide detail. -->

| Directory | What's there | CLAUDE.md? |
|-----------|-------------|------------|
| `services/` | 9 Go microservices (gateway, auth, player, combat, cultivation, social, world, trade, ranking) | ✅ `services/CLAUDE.md` |
| `frontend/` | Vue 3 SPA — WebSocket client, ~35 routes, Pinia stores, SCSS | ✅ `frontend/CLAUDE.md` |
| `database/` | MySQL migrations (~20 files), Redis config, MongoDB schema | ✅ `database/CLAUDE.md` |
| `shared/` | Go shared library: config, eventbus, models, codec, proto, plugin | — |
| `deploy/` | Docker Compose, K8s manifests, Nginx, deploy scripts | — |
| `docs/` | ARCHITECTURE, API, DESIGN, BALANCE, GAMEPLAY, DEV_GUIDE, DEPLOY | — |
| `backend/` | go.work only — maps all modules | — |

## Architecture (condensed)

```
Client(Vue3) ──WebSocket──▶ Gateway(:8080) ──NATS──▶ Auth(:8082)
                                                   ▶ Player(:8083)
                                                   ▶ Cultivation(:8084)
                                                   ▶ Combat(:8085)
                                                   ▶ Social(:8086)
                                                   ▶ World(:8087)
                                                   ▶ Trade(:8088)
                                                   ▶ Ranking(:8089)
```

- **Gateway**: WebSocket + JWT + rate limiting. Single node = 50k connections.
- **Service pattern**: Handler → Service → Repository (see `services/CLAUDE.md` for full template)
- **DB**: MySQL 8.0 (structured data), Redis 7 (cache/leaderboards), MongoDB 7 (chat/logs)
- **Inter-service**: Gateway routes by msg_id. Some services call each other (e.g., Combat → Player for rewards).

## Hard Constraints

<!-- Rules Claude MUST follow. Update when violations occur. -->

- **All Go commands from `backend/`** — cross-module resolution depends on go.work.
- **Handler → Service → Repository** — never put business logic in handlers or main.go.
- **Use `%w` for error wrapping**, never ignore errors silently.
- **MySQL migrations must have**: `ENGINE=InnoDB`, `CHARSET=utf8mb4`, `COMMENT` on every column, indexes on FKs.
- **Vue components use `<script setup lang="ts">`** — no Options API.
- **Pinia for state** — no direct fetch() in components, no props drilling > 2 levels.
- **Protobuf regeneration**: edit `.proto` files in `services/gateway/api/proto/`, then run `protoc`.
- **Commit format**: `type(scope): description` (Conventional Commits). Co-author line required.

## Key Gotchas

<!-- Things Claude consistently gets wrong. Add to this list. -->

1. **go.work is in `backend/`, not repo root** — `cd backend` before any `go` command that spans modules.
2. **go.work uses Go 1.24** — some services' go.mod declare >= 1.24. If `go work use` complains, check individual go.mod files.
3. **Some services use `database/sql`, others GORM** — check the existing repo pattern before adding new queries.
4. **Game data lives in JSON files** (`internal/data/*.json`), not always in MySQL — check `internal/data/` before adding config tables.
5. **Frontend `@/` alias = `src/`** — imports like `@/core/api` resolve to `frontend/src/core/api.ts`.
6. **Vite proxies API calls to individual backends in dev** — Gateway is bypassed during development.

## Git Conventions

- Branch: `feature/issue-xxx-desc` or `fix/issue-xxx-desc` from `main`
- Commit: `type(scope): description` — types: feat/fix/refactor/perf/test/docs/style/chore/ci — scopes: gateway/player/combat/cultivation/social/world/trade/ranking/frontend/shared/db
- Merge: Squash merge after review
- End commit bodies with: `Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>`

## Docs Index

| Doc | When to read |
|-----|-------------|
| `docs/ARCHITECTURE.md` | Understanding data flows, DB design, security model |
| `docs/API.md` | API endpoint reference |
| `docs/GAMEPLAY.md` | Game mechanics — what each system actually does |
| `docs/BALANCE.md` | Numerical design — monster stats, skill damage, economy |
| `docs/DESIGN.md` | Why things were built this way |
| `docs/DEV_GUIDE.md` | Setting up dev environment, debugging |
| `docs/DEPLOY.md` | Production deployment (Docker/K8s) |
| `CONTRIBUTING.md` | PR process, code standards |

## Session Management

<!-- Claude: read primer.md at session start for current state. Rewrite it at session end. -->

At session start: read `primer.md` for current project state, active work, and blockers.
At session end: rewrite `primer.md` with: (1) what was accomplished, (2) what's in progress, (3) next steps, (4) blockers/decisions made, (5) files changed summary.
If `primer.md` doesn't exist, create it at the end of the first session.
