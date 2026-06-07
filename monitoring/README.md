# Cultivation Game - Monitoring System

## Architecture

```
Game Services (auth, player, cultivation, combat, social, world)
    |
    ├──> Prometheus  ──>  Grafana (Dashboards)
    ├──> Loki        <──  Promtail (Log collection)
    └──> Jaeger      (Distributed tracing)

Node Exporter ──> Prometheus (Host metrics)
Redis Exporter ──> Prometheus
MySQL Exporter ──> Prometheus
```

## Quick Start

```bash
# Ensure the game-backend network exists
docker network create game-backend

# Start the monitoring stack
docker compose -f docker-compose.monitor.yml up -d

# Check status
docker compose -f docker-compose.monitor.yml ps
```

## Access Points

| Service     | URL                        | Default Credentials      |
|-------------|----------------------------|--------------------------|
| Grafana     | http://<host>:3000         | admin / admin            |
| Prometheus  | http://<host>:9090         | -                        |
| Jaeger UI   | http://<host>:16686        | -                        |
| Loki API    | http://<host>:3100         | -                        |

## Dashboards

### 1. Game Overview (`game_overview`)
- Real-time online players (Gauge)
- Message throughput by type (QPS, Graph)
- Message latency P50 / P90 / P99 (Graph)
- Per-service CPU and memory usage (Stat)
- Error rate (Graph + Stat)
- Active players over 24h (Graph)
- Redis cache hit rate (Gauge)

### 2. Business Metrics (`business_metrics`)
- DAU / MAU (Stat + sparkline)
- New registrations (Stat)
- Revenue and ARPU (Graph)
- Realm distribution (Pie chart)
- Online duration distribution (Histogram)

## Alert Rules (Prometheus)

| Severity | Rule                        | Threshold              | Duration   |
|----------|-----------------------------|------------------------|------------|
| P1       | OnlinePlayerDrop            | Players drop > 30%     | 1 min      |
| P1       | ServiceDown                 | Service unreachable    | 30 sec     |
| P2       | HighMessageLatency          | P99 > 500ms            | 5 min      |
| P2       | HighErrorRate               | Error rate > 1%        | 5 min      |
| P2       | RedisMemoryHigh             | Memory > 80%           | 2 min      |
| P3       | HighCPUUsage                | CPU > 80%              | 5 min      |
| P3       | HighDiskUsage               | Disk > 80%             | 5 min      |

## Logging

Loki receives logs from two sources:
1. Docker containers (via Promtail scraping `/var/lib/docker/containers/`)
2. Game application logs (`/var/log/game/*.log`)

Search logs in Grafana Explore with label selectors:
- `{job="game-logs"}` - Docker container logs
- `{job="game-app",service="auth"}` - Auth service logs

## Tracing

Services should export OpenTelemetry (OTLP) traces to `jaeger:4317` (gRPC) or `jaeger:4318` (HTTP).

## Maintenance

### Stop the stack
```bash
docker compose -f docker-compose.monitor.yml down
```

### Wipe all data
```bash
docker compose -f docker-compose.monitor.yml down -v
```

### Check Prometheus targets
Visit http://<host>:9090/targets to verify all scrape targets are up.

### Backup Grafana dashboards
Dashboards are provisioned from `grafana/dashboards/`. Dashboard JSON can also be exported from the Grafana UI and committed to version control.
