# Database Migrations

## Applying Migrations
- Migrations are in internal/db/migrations/
- 36 migrations (000001 through 000036)
- Schema: .up.sql for forward, .down.sql for rollback

## Key Migrations
- 000001: Initial schema (hypertables, compression, retention)
- 000030: Parameter versions table
- 000035: Token revocations table
- 000036: Candles compression and retention policies

## Checking Pending Migrations
- API: GET /api/v1/admin/migrations/pending
- The server lists pending .up.sql files on startup

## Backup
- pg_dump -U orca -h localhost -p 5433 orca_core > backup.sql
