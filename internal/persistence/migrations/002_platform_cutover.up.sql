-- Platform cutover migration: only does work on in-place upgrades from a
-- pre-cutover deployment that already has 001_schema recorded as applied.
-- Fresh installs see this as a near no-op because 001_schema (current
-- contents) already creates contractor_supervisors and never created
-- auth_accounts.

-- 1. Ensure contractor_supervisors exists for in-place upgraders whose 001
--    pre-dated the cutover. CREATE TABLE IF NOT EXISTS is idempotent.
CREATE TABLE IF NOT EXISTS contractor_supervisors (
    email TEXT PRIMARY KEY,
    supervisor TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_contractor_supervisors_sup ON contractor_supervisors (supervisor);

-- 2. Drop the orphan auth_accounts table from pre-cutover deployments.
--    Password hashes, role assignments, and contractor_sup mappings should
--    have been migrated to iag-authentication and contractor_supervisors
--    BEFORE applying this migration. Backups are the operator's responsibility.
DROP TABLE IF EXISTS auth_accounts;
