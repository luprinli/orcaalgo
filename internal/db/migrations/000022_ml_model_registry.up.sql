-- 000022: ML model registry
-- Content-addressable model store. Each trained ML model (meta-labeler,
-- regime classifier, exit orchestrator) is stored with its SHA-256 hash
-- as the primary key. The hash is computed from model JSON + training
-- dataset metadata + hyperparameters (see D1 model_hash).

CREATE TABLE IF NOT EXISTS ml_models (
    model_hash  TEXT PRIMARY KEY,
    model_type  TEXT NOT NULL,
    model_name  TEXT NOT NULL,
    brier_score DOUBLE PRECISION,
    roc_auc     DOUBLE PRECISION,
    metadata_json JSONB DEFAULT '{}',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_ml_models_type ON ml_models(model_type);
CREATE INDEX IF NOT EXISTS idx_ml_models_created ON ml_models(created_at DESC);

COMMENT ON TABLE ml_models IS 'Content-addressable ML model registry. Hash = SHA-256(model JSON + training metadata + hyperparameters).';
