# database/CLAUDE.md — Database Conventions

<!-- This file loads ON-DEMAND when Claude touches files under database/. -->

## Storage Summary

| Database | Purpose | Config |
|----------|---------|--------|
| **MySQL 8.0** | Primary: players, inventory, equipment, realm_config, sects, trade, etc. | `database/mysql/` |
| **Redis 7.2 Cluster** | Sessions, online status, leaderboards (Sorted Set), rate limiting, caching | `database/redis/` |
| **MongoDB 7.0** | Chat history, combat replays, event logs (flexible schema) | `database/mongodb/` |

## MySQL Migrations

Files in `database/mysql/` are numbered sequentially. Current highest: ~020.

### Migration conventions

Every migration file MUST:
- Start with a header comment: `-- 功能描述 - 数据表`
- Use `ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`
- Have `COMMENT` on every table and column
- Include indexes on foreign keys and query columns
- Include `created_at` and `updated_at` timestamps with defaults

### Table naming
- Snake_case: `player_achievements`, `sect_members`, `dungeon_records`
- Foreign keys: `player_id` referencing `players.id`
- Config tables: `*_config` suffix (e.g., `realm_config`, `technique_templates`)

### Example migration

```sql
-- ===================================================================
-- 灵兽孵化系统 - 数据表
-- ===================================================================

CREATE TABLE pet_eggs (
    id              BIGINT AUTO_INCREMENT PRIMARY KEY COMMENT '宠物蛋ID',
    player_id       BIGINT NOT NULL                COMMENT '所属玩家ID',
    pet_template_id INT NOT NULL                   COMMENT '宠物模板ID',
    quality         TINYINT DEFAULT 0              COMMENT '品质(0=普通,1=稀有,2=传说)',
    hatch_progress  INT DEFAULT 0                  COMMENT '孵化进度',
    created_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

    INDEX idx_player_id (player_id),
    INDEX idx_quality (quality)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='宠物蛋表';
```

### Creating a new migration

1. Find the highest number in `database/mysql/`
2. Create `database/mysql/<NNN>_<feature_name>.sql` with next number
3. Use the `add-gameplay-system` skill for full-stack features that include migrations

## Redis Usage Patterns

- **Sessions**: `session:<token>` → player data (TTL: 24h)
- **Online status**: `online:<player_id>` → timestamp (TTL: 5min heartbeat)
- **Leaderboards**: ZSET `rank:<type>` scored by value
- **Rate limiting**: `ratelimit:<ip>:<action>` with TTL

## MongoDB Usage Patterns

- **Chat history**: `chat_messages` collection (TLL index on `created_at`)
- **Combat replays**: `combat_replays` collection
- **Event logs**: `event_logs` collection (capped or TTL)

## Infrastructure Startup

```bash
docker compose -f database/redis/docker-compose.redis.yml up -d
docker compose -f database/mysql/docker-compose.yml up -d
docker compose -f database/mongodb/docker-compose.yml up -d
```
