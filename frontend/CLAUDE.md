# frontend/CLAUDE.md — Vue 3 Frontend Conventions

<!-- This file loads ON-DEMAND when Claude touches files under frontend/. -->

## Quick Commands

```bash
cd frontend
npm run dev          # http://localhost:3000 (HMR, proxies to backends directly)
npm run build        # vue-tsc + vite build
npm run test         # Vitest
npx vue-tsc --noEmit # Type-check only
```

## Architecture

```
frontend/src/
├── main.ts                # Pinia + Router + WebSocket init
├── App.vue                # Root
├── router/index.ts        # Hash-mode, auth guard, ~35 routes
├── core/
│   ├── api.ts             # fetch wrapper (auto token, 401 → login)
│   ├── network/
│   │   ├── WebSocketClient.ts  # Singleton WS (auto-reconnect, heartbeat, queue)
│   │   ├── MessageCodec.ts     # Protobuf/JSON codec
│   │   └── HttpFallback.ts     # HTTP polling when WS fails
│   ├── event/EventBus.ts
│   └── store/             # Pinia stores: player, combat, cultivation, world, social, trade, inventory
├── modules/               # Feature modules — one dir per game domain
│   ├── player/            # ArtifactView, DongFuView, PetView, FormationView, RebirthView, etc.
│   ├── combat/            # PveView, DungeonView, TowerView, WorldBossView
│   ├── cultivation/       # CultivationView, HeartDemonView
│   ├── social/            # ChatPanel, MasterView, DaolvView, FactionWarView, SectTechView
│   ├── world/             # WorldView, DivinationView, TreasureView, VeinContestView
│   ├── trade/             # MarketView, BlackMarketView
│   ├── inventory/         # InventoryView
│   ├── ranking/           # RankingView
│   └── alchemy/           # Alchemy views
├── views/                 # Top-level pages (Home, Login, Register, GameLayout)
├── components/            # Shared: NavBar, CombatLog, VirtualList, ThemeSwitcher
├── composables/           # Shared: useVirtualScroll, useNumberScroll
├── styles/                # SCSS: variables, mixins, themes, pc/mobile responsive
└── types/                 # TypeScript type definitions
```

## Component Conventions

- **Always `<script setup lang="ts">`** — never Options API.
- File names: **PascalCase** for components, **camelCase** for utilities.
- Component file top: JSDoc comment describing purpose.
- SCSS: use `@use '@/styles/variables' as *` — variables auto-injected via Vite `additionalData`.
- Responsive: both PC and mobile layouts considered (check `styles/` for breakpoints).

## State Management

- **Pinia stores** in `core/store/` — one store per domain.
- **Never call fetch() directly in components** — go through the store.
- **No props drilling > 2 levels** — use store or provide/inject for deeper component trees.
- Store pattern:
  ```ts
  export const useXxxStore = defineStore('xxx', () => {
    const data = ref<DataType | null>(null)
    const loading = ref(false)
    async function fetchData(id: number) { ... }
    return { data, loading, fetchData }
  })
  ```

## Network

- **WebSocket is primary** — `WebSocketClient.ts` handles auto-reconnect and heartbeat.
- **HTTP fallback** — `HttpFallback.ts` activates when WS connection fails.
- **API calls use `core/api.ts`** wrapper — auto-attaches auth token, redirects to login on 401.

## Routing

- Hash-mode routing (`createWebHashHistory`).
- All game views are children of `/game` route under `GameLayout.vue` (includes NavBar).
- Auth guard on routes requiring login (`meta: { requiresAuth: true }`).

## Path Aliases

- `@/` = `frontend/src/`
- Example: `import { api } from '@/core/api'`

## Dev Proxy

- Vite dev server proxies API calls to individual backend services directly — Gateway is bypassed in development.
- Check `vite.config.ts` for proxy rules.

## Testing

```bash
npm run test              # All Vitest tests
npm run test:watch        # Watch mode
npm run test -- --coverage   # With coverage
```

## Adding a New Module

Use the `add-gameplay-system` skill. Manual steps:
1. Create `src/modules/<name>/<Name>View.vue` (PascalCase)
2. Create `src/core/store/<name>.ts` (Pinia store)
3. Add types to `src/types/<name>.ts`
4. Register route in `src/router/index.ts`
5. Add store to `src/core/store/index.ts`
