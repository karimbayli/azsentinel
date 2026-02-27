# Sentinel V2 — Operations Runbook

## Daily Operations

### Health Check
```bash
curl -s https://sentinel.example.com/healthz | jq
curl -s https://sentinel.example.com/api/v1/nodes | jq
```

### Check Logs
```bash
# Central
docker compose -f deployments/docker-compose.central.yml logs -f central --tail=100

# Probe node
sudo journalctl -u sentinel-probe -f
```

## Common Scenarios

### Probe Node Not Reporting
1. Check node connectivity: `curl -s https://sentinel.example.com/healthz`
2. Check probe service: `sudo systemctl status sentinel-probe`
3. Check buffer depth: look for `sentinel_probe_buffer_depth` metric
4. Check logs: `sudo journalctl -u sentinel-probe --since "1 hour ago"`
5. If buffer is growing, central may be unreachable (expected during AZ outage)

### False Positive Alert
1. Check if the alert target is actually reachable from your location
2. Review which nodes reported failure at `/api/v1/status/<target>`
3. Check if any nodes are marked unreliable
4. Review BGP events at `/api/v1/bgp/events?hours=1`
5. If a single node issue, check that node's health

### Database Growing Too Large
1. Verify retention policies: `SELECT * FROM timescaledb_information.jobs;`
2. Manually compress: `SELECT compress_chunk(c) FROM show_chunks('probe_results') c;`
3. Check disk usage per table in Grafana

### BGP Monitor Disconnecting Frequently
1. Check logs for WebSocket errors
2. Verify network connectivity to `wss://ris-live.ripe.net`
3. RIPE RIS may have maintenance windows — reconnection is automatic

### After Central Server Restart
1. Verify all services are running: `docker compose ps`
2. Check TimescaleDB health: test query in Grafana
3. Probe nodes will automatically flush buffered data
4. Correlation engine will resume on its 30s cycle

## Maintenance Windows

### Database Maintenance
```bash
# Create backup first
make db-backup

# Connect to DB
docker exec -it sentinel-db psql -U sentinel -d sentinel

# Check chunk status
SELECT * FROM timescaledb_information.chunks ORDER BY range_start DESC LIMIT 20;

# Force compression
SELECT compress_chunk(c) FROM show_chunks('probe_results', older_than => '7 days') c;
```

### Rotating HMAC Secret
1. Generate new secret: `bash scripts/generate-certs.sh`
2. Update central `.env` and restart central
3. Update each probe node config and restart
4. During rotation, probes from nodes with old secret will be rejected

## Grafana Access
- URL: `https://sentinel.example.com/grafana/`
- Default credentials: See `GRAFANA_USER`/`GRAFANA_PASSWORD` in `.env`
- Main dashboard: "Sentinel V2 — Azerbaijan Internet Monitor"
