# Startup / Shutdown Procedures

## Prerequisites
- Docker and Docker Compose installed
- .env file configured with ORCA_JWT_SECRET, ORCA_ADMIN_PASSWORD, ORCA_DB_* vars

## Startup
1. `docker compose up -d postgres` — start database
2. Wait for PostgreSQL health check: `docker compose ps postgres`
3. `docker compose --profile prod up -d app` — start the application
4. Verify: `curl http://localhost:8080/api/v1/system/health`

## Shutdown
1. `docker compose --profile prod stop app` — graceful stop (30s drain timeout)
2. `docker compose stop postgres` — stop database

## Environment Validation
- Server refuses to start without ORCA_JWT_SECRET (≥32 chars)
- Server validates DB, Alpaca keys on startup
- Check logs: `docker compose logs app`

## Health Checks
- Liveness: GET /healthz → 200
- Readiness: GET /readyz → 200 (checks DB + broker)
