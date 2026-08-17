-- Per-contract currency (value/milestones/payments are in this currency).
-- Empty defaults to UGX in the app; the column default keeps existing rows UGX.
ALTER TABLE gov_contracts ADD COLUMN IF NOT EXISTS currency TEXT NOT NULL DEFAULT 'UGX';

-- Governed, reusable project managers — the source for the contract form's
-- "Project manager" dropdown (select an existing PM or add a new one).
CREATE TABLE IF NOT EXISTS gov_project_managers (
    id         TEXT PRIMARY KEY,
    name       TEXT NOT NULL,
    email      TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Case-insensitive unique name so "add new" is idempotent (ON CONFLICT).
CREATE UNIQUE INDEX IF NOT EXISTS idx_gov_project_managers_lower_name
    ON gov_project_managers (lower(name));
