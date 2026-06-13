-- Contract-governance domain (Phase 1): governance contracts + rich milestones.
--
-- Modeled separately from the legacy zone-works `contracts` table so it can
-- carry the full governance shape (contractor, type, dates, value, retention,
-- documents, activity) and the 8-state lifecycle the Contract Governance UI
-- expects. Nested value objects (scope, deliverables, checklist, docs,
-- comments, inspection, completion report) are JSONB.

CREATE TABLE IF NOT EXISTS gov_contracts (
    id                 TEXT PRIMARY KEY,
    number             TEXT NOT NULL UNIQUE,
    name               TEXT NOT NULL,
    contractor         TEXT NOT NULL DEFAULT '',
    contractor_contact TEXT NOT NULL DEFAULT '',
    type               TEXT NOT NULL DEFAULT '',
    start_date         TEXT,
    end_date           TEXT,
    location           TEXT NOT NULL DEFAULT '',
    pm                 TEXT NOT NULL DEFAULT '',
    department         TEXT NOT NULL DEFAULT '',
    value              BIGINT NOT NULL DEFAULT 0,
    retention          INT NOT NULL DEFAULT 0,
    status             TEXT NOT NULL DEFAULT 'Draft',
    documents          JSONB NOT NULL DEFAULT '[]'::jsonb,
    activity           JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS gov_milestones (
    id                TEXT PRIMARY KEY,
    contract_id       TEXT NOT NULL REFERENCES gov_contracts (id) ON DELETE CASCADE,
    name              TEXT NOT NULL,
    value             BIGINT NOT NULL DEFAULT 0,
    target_date       TEXT,
    status            TEXT NOT NULL DEFAULT 'Pending',
    scope             JSONB NOT NULL DEFAULT '[]'::jsonb,
    deliverables      JSONB NOT NULL DEFAULT '[]'::jsonb,
    checklist         JSONB NOT NULL DEFAULT '[]'::jsonb,
    docs              JSONB NOT NULL DEFAULT '[]'::jsonb,
    comments          JSONB NOT NULL DEFAULT '[]'::jsonb,
    inspection        JSONB,
    completion_report JSONB,
    sort_order        INT NOT NULL DEFAULT 0,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_gov_milestones_contract ON gov_milestones (contract_id, sort_order);
