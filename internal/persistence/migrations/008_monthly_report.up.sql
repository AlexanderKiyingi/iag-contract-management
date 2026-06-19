-- Monthly Report (MR) parity: the data behind the Inspire Africa Construction
-- Department monthly report workbook.
--
-- Adds a normalized contractor parent, per-(contract,period) progress snapshots,
-- and contractor-level IPC valuations; and extends gov_contracts with the
-- operational fields the Tracker sheet carries (execution status, progress %,
-- received-to-date, variation total, planned completion).
--
-- Follows the legacy-safe pattern from 001_schema.up.sql: CREATE TABLE IF NOT
-- EXISTS for fresh DBs, ALTER TABLE ADD COLUMN IF NOT EXISTS to backfill shape
-- on pre-existing tables, then CREATE INDEX IF NOT EXISTS. The whole file runs
-- under the simple query protocol as one Exec (see migrate.go).

-- ----- Contractors (normalized parent: one contractor -> many work-orders) -----
CREATE TABLE IF NOT EXISTS gov_contractors (
    id         TEXT PRIMARY KEY,
    name       TEXT NOT NULL UNIQUE,
    contact    TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
ALTER TABLE gov_contractors ADD COLUMN IF NOT EXISTS contact TEXT NOT NULL DEFAULT '';
CREATE INDEX IF NOT EXISTS idx_gov_contractors_name ON gov_contractors (name);

-- ----- Extend gov_contracts with the Tracker operational columns -----
ALTER TABLE gov_contracts ADD COLUMN IF NOT EXISTS contractor_id      TEXT;
ALTER TABLE gov_contracts ADD COLUMN IF NOT EXISTS execution_status   TEXT NOT NULL DEFAULT 'Not Started';
ALTER TABLE gov_contracts ADD COLUMN IF NOT EXISTS progress           INT NOT NULL DEFAULT 0;
ALTER TABLE gov_contracts ADD COLUMN IF NOT EXISTS received           BIGINT NOT NULL DEFAULT 0;
ALTER TABLE gov_contracts ADD COLUMN IF NOT EXISTS variation_total    BIGINT NOT NULL DEFAULT 0;
ALTER TABLE gov_contracts ADD COLUMN IF NOT EXISTS planned_completion TEXT;
CREATE INDEX IF NOT EXISTS idx_gov_contracts_contractor ON gov_contracts (contractor_id);
CREATE INDEX IF NOT EXISTS idx_gov_contracts_exec_status ON gov_contracts (execution_status);

-- ----- Per-(contract, period) progress snapshots -----
-- Absorbs the Tracker narrative (current activity, accomplishments, challenges,
-- interventions) AND the Work Programme forward projection (proposed dates,
-- duration, responsible, target) — they are the same rows, period-scoped.
CREATE TABLE IF NOT EXISTS gov_progress_reports (
    id                  TEXT PRIMARY KEY,
    contract_id         TEXT NOT NULL REFERENCES gov_contracts (id) ON DELETE CASCADE,
    period              TEXT NOT NULL,                       -- e.g. '2026-05'
    progress            INT NOT NULL DEFAULT 0,
    execution_status    TEXT NOT NULL DEFAULT '',
    current_activity    TEXT NOT NULL DEFAULT '',
    accomplishments     TEXT NOT NULL DEFAULT '',
    challenges          TEXT NOT NULL DEFAULT '',
    interventions       TEXT NOT NULL DEFAULT '',
    responsible         TEXT NOT NULL DEFAULT '',
    target_date         TEXT NOT NULL DEFAULT '',
    proposed_start      TEXT NOT NULL DEFAULT '',
    proposed_completion TEXT NOT NULL DEFAULT '',
    duration            TEXT NOT NULL DEFAULT '',
    planned_next        TEXT NOT NULL DEFAULT '',
    planned_progress    INT NOT NULL DEFAULT 0,               -- consultancy: planned vs actual
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (contract_id, period)
);
CREATE INDEX IF NOT EXISTS idx_gov_progress_reports_period ON gov_progress_reports (period);
CREATE INDEX IF NOT EXISTS idx_gov_progress_reports_contract ON gov_progress_reports (contract_id);

-- ----- Contractor-level IPC valuations (the "Contractors verified" sheet) -----
CREATE TABLE IF NOT EXISTS gov_valuations (
    id                        TEXT PRIMARY KEY,
    contractor_id             TEXT,
    contractor_name           TEXT NOT NULL DEFAULT '',
    period                    TEXT NOT NULL DEFAULT '',
    contract_sum              BIGINT NOT NULL DEFAULT 0,
    amount_paid               BIGINT NOT NULL DEFAULT 0,
    verified_value_owed       BIGINT NOT NULL DEFAULT 0,
    consultant_recommendation BIGINT NOT NULL DEFAULT 0,
    ceo_approval              BIGINT NOT NULL DEFAULT 0,
    remarks                   TEXT NOT NULL DEFAULT '',
    verified_date             TEXT NOT NULL DEFAULT '',
    created_at                TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at                TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_gov_valuations_contractor ON gov_valuations (contractor_id);
CREATE INDEX IF NOT EXISTS idx_gov_valuations_period ON gov_valuations (period);
