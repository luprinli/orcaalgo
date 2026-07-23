-- UP: Settings, audit log, data source extensions
CREATE TABLE settings (
    key         VARCHAR(128) PRIMARY KEY,
    value       JSONB NOT NULL DEFAULT '{}',
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE audit_log (
    id          BIGSERIAL PRIMARY KEY,
    timestamp   TIMESTAMPTZ NOT NULL DEFAULT now(),
    level       VARCHAR(16) NOT NULL DEFAULT 'info',
    component   VARCHAR(64) NOT NULL,
    message     TEXT NOT NULL,
    metadata    JSONB
);
CREATE INDEX ON audit_log (timestamp DESC);
CREATE INDEX ON audit_log (component, timestamp DESC);

CREATE TABLE kill_switch_history (
    id          BIGSERIAL PRIMARY KEY,
    triggered_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    reason      VARCHAR(256) NOT NULL,
    source      VARCHAR(64),
    resolved_at  TIMESTAMPTZ
);
CREATE INDEX ON kill_switch_history (triggered_at DESC);
