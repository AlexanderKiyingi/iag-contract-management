-- Zone-works milestone governance: the per-milestone governance profiles
-- (checklist / scope / deliverables / comment thread) and the per-contract
-- execution trackers, migrated out of the frontend blob into Postgres.

-- Milestone profiles: rich governance detail keyed by the milestone id it
-- annotates. Nested value objects are stored as JSONB so the UI shape
-- round-trips without a table per sub-collection.
CREATE TABLE IF NOT EXISTS zw_milestone_profiles (
    milestone_id TEXT PRIMARY KEY,
    contract_no  TEXT NOT NULL,
    value        BIGINT NOT NULL DEFAULT 0,
    checklist    JSONB NOT NULL DEFAULT '[]'::jsonb,
    scope        JSONB NOT NULL DEFAULT '[]'::jsonb,
    deliverables JSONB NOT NULL DEFAULT '[]'::jsonb,
    comments     JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_zw_milestone_profiles_contract ON zw_milestone_profiles (contract_no);

-- Execution trackers: the execution stage a zone-works contract has reached,
-- the steps completed so far, and an optional execution date. Keyed by
-- contract number.
CREATE TABLE IF NOT EXISTS zw_execution_trackers (
    contract_no    TEXT PRIMARY KEY,
    stage          INT NOT NULL DEFAULT 0,
    steps          JSONB NOT NULL DEFAULT '[]'::jsonb,
    execution_date TEXT,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
