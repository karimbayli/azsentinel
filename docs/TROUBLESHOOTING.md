# Sentinel V2 — Troubleshooting

## Build Issues

### `CGO_ENABLED=1` required
SQLite dependency requires CGO. Ensure `gcc` is installed:
```bash
# macOS: included with Xcode CLI tools
# Ubuntu: sudo apt install gcc
# Alpine: apk add gcc musl-dev sqlite-dev
```

### Module download failures
```bash
go clean -modcache
go mod tidy
go mod download
```

## Runtime Issues

### Central server won't start
1. Check PostgreSQL is running: `docker compose logs timescaledb`
2. Verify `.env` matches `configs/central.yaml`
3. Check migrations ran: connect to DB and check `\dt`

### Probe agent can't reach central
1. Test connectivity: `curl -v https://sentinel.example.com/healthz`
2. Check HMAC secret matches central config
3. Check buffer depth growing (expected if offline)
4. Review probe logs: `journalctl -u sentinel-probe -n 50`

### BGP monitor not connecting
1. Check network to RIPE: `curl -v wss://ris-live.ripe.net/v1/ws/`
2. Monitor reconnection in logs (exponential backoff: 5s→10s→30s→60s→120s)
3. RIPE RIS may have maintenance — reconnection is automatic

### No data in Grafana
1. Verify datasource configuration in Grafana → Data Sources
2. Test query: `SELECT count(*) FROM probe_results WHERE time > now() - interval '1 hour'`
3. Check central is ingesting data: search logs for "received probe batch"

### High memory usage
1. Check buffer depth, flush may be stuck
2. Review connection pool settings
3. TimescaleDB may need chunk compression

### Telegram alerts not working
1. Verify bot token: `curl https://api.telegram.org/bot<token>/getMe`
2. Verify chat ID: send `/start` to bot, check `getUpdates`
3. Ensure `alert.enabled: true` in config
4. Check calibration mode is OFF

## Database Issues

### TimescaleDB extension not found
```sql
CREATE EXTENSION IF NOT EXISTS timescaledb;
```

### Retention not working
```sql
-- Check scheduled jobs
SELECT * FROM timescaledb_information.jobs WHERE proc_name = 'policy_retention';
-- Manual run
CALL run_job(<job_id>);
```

### Disk full
```bash
# Check chunk sizes
docker exec sentinel-db psql -U sentinel -d sentinel -c "SELECT * FROM chunks_detailed_size('probe_results') ORDER BY total_bytes DESC LIMIT 10;"
# Force compression
docker exec sentinel-db psql -U sentinel -d sentinel -c "SELECT compress_chunk(c) FROM show_chunks('probe_results', older_than => '3 days') c;"
```
