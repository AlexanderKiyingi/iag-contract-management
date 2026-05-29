-- IAG contract-management — PostgreSQL schema.
-- Post-platform-cutover: identity, passwords, and group membership live in
-- the authentication service. The only user-related data this service holds
-- is the contractor-portal binding (which contractor email is scoped to
-- which supervisor's contracts).
--
-- Legacy-safety pattern: Railway deployments carry pre-cutover tables whose
-- shape predates this schema. CREATE TABLE IF NOT EXISTS is a no-op on those
-- tables, so the CREATE INDEX statements that follow fail with "column X
-- does not exist" (SQLSTATE 42703). Each table block therefore follows the
-- shape:
--   CREATE TABLE IF NOT EXISTS … (strict shape for fresh DBs)
--   ALTER TABLE … ADD COLUMN IF NOT EXISTS … (idempotently backfills shape
--     on legacy DBs; FK / UNIQUE constraints are intentionally omitted on
--     the ALTER path because pre-existing rows may not be backfillable)
--   CREATE INDEX IF NOT EXISTS …
-- The whole file is sent under the simple query protocol so multi-statement
-- bodies execute as one Exec — see internal/persistence/migrate.go.

CREATE TABLE IF NOT EXISTS schema_migrations (
    version TEXT PRIMARY KEY,
    applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
ALTER TABLE schema_migrations ADD COLUMN IF NOT EXISTS applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

CREATE TABLE IF NOT EXISTS contractor_supervisors (
    email TEXT PRIMARY KEY,
    supervisor TEXT NOT NULL
);
ALTER TABLE contractor_supervisors ADD COLUMN IF NOT EXISTS supervisor TEXT NOT NULL DEFAULT '';
CREATE INDEX IF NOT EXISTS idx_contractor_supervisors_sup ON contractor_supervisors (supervisor);

CREATE TABLE IF NOT EXISTS zones (
    code TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    supervisor TEXT NOT NULL DEFAULT '',
    contract_sum BIGINT NOT NULL DEFAULT 0,
    paid BIGINT NOT NULL DEFAULT 0,
    balance BIGINT NOT NULL DEFAULT 0,
    color TEXT NOT NULL DEFAULT '',
    contract_count INT NOT NULL DEFAULT 0
);
ALTER TABLE zones ADD COLUMN IF NOT EXISTS name TEXT NOT NULL DEFAULT '';
ALTER TABLE zones ADD COLUMN IF NOT EXISTS description TEXT NOT NULL DEFAULT '';
ALTER TABLE zones ADD COLUMN IF NOT EXISTS supervisor TEXT NOT NULL DEFAULT '';
ALTER TABLE zones ADD COLUMN IF NOT EXISTS contract_sum BIGINT NOT NULL DEFAULT 0;
ALTER TABLE zones ADD COLUMN IF NOT EXISTS paid BIGINT NOT NULL DEFAULT 0;
ALTER TABLE zones ADD COLUMN IF NOT EXISTS balance BIGINT NOT NULL DEFAULT 0;
ALTER TABLE zones ADD COLUMN IF NOT EXISTS color TEXT NOT NULL DEFAULT '';
ALTER TABLE zones ADD COLUMN IF NOT EXISTS contract_count INT NOT NULL DEFAULT 0;
CREATE INDEX IF NOT EXISTS idx_zones_supervisor ON zones (supervisor);

CREATE TABLE IF NOT EXISTS contracts (
    contract_no TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    zone_code TEXT NOT NULL REFERENCES zones (code) ON UPDATE CASCADE,
    contract_sum BIGINT NOT NULL DEFAULT 0,
    paid BIGINT NOT NULL DEFAULT 0,
    balance BIGINT NOT NULL DEFAULT 0,
    progress INT NOT NULL DEFAULT 0,
    status TEXT NOT NULL,
    priority TEXT NOT NULL,
    workers INT NOT NULL DEFAULT 0,
    supervisor TEXT NOT NULL DEFAULT '',
    remarks TEXT NOT NULL DEFAULT '',
    created_on TEXT NOT NULL DEFAULT ''
);
ALTER TABLE contracts ADD COLUMN IF NOT EXISTS name TEXT NOT NULL DEFAULT '';
ALTER TABLE contracts ADD COLUMN IF NOT EXISTS zone_code TEXT NOT NULL DEFAULT '';
ALTER TABLE contracts ADD COLUMN IF NOT EXISTS contract_sum BIGINT NOT NULL DEFAULT 0;
ALTER TABLE contracts ADD COLUMN IF NOT EXISTS paid BIGINT NOT NULL DEFAULT 0;
ALTER TABLE contracts ADD COLUMN IF NOT EXISTS balance BIGINT NOT NULL DEFAULT 0;
ALTER TABLE contracts ADD COLUMN IF NOT EXISTS progress INT NOT NULL DEFAULT 0;
ALTER TABLE contracts ADD COLUMN IF NOT EXISTS status TEXT NOT NULL DEFAULT '';
ALTER TABLE contracts ADD COLUMN IF NOT EXISTS priority TEXT NOT NULL DEFAULT '';
ALTER TABLE contracts ADD COLUMN IF NOT EXISTS workers INT NOT NULL DEFAULT 0;
ALTER TABLE contracts ADD COLUMN IF NOT EXISTS supervisor TEXT NOT NULL DEFAULT '';
ALTER TABLE contracts ADD COLUMN IF NOT EXISTS remarks TEXT NOT NULL DEFAULT '';
ALTER TABLE contracts ADD COLUMN IF NOT EXISTS created_on TEXT NOT NULL DEFAULT '';
CREATE INDEX IF NOT EXISTS idx_contracts_zone ON contracts (zone_code);
CREATE INDEX IF NOT EXISTS idx_contracts_status ON contracts (status);
CREATE INDEX IF NOT EXISTS idx_contracts_priority ON contracts (priority);
CREATE INDEX IF NOT EXISTS idx_contracts_supervisor ON contracts (supervisor);
CREATE INDEX IF NOT EXISTS idx_contracts_created ON contracts (created_on);
CREATE INDEX IF NOT EXISTS idx_contracts_zone_status ON contracts (zone_code, status);

CREATE TABLE IF NOT EXISTS engineers (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    role TEXT NOT NULL,
    zone_code TEXT NOT NULL,
    phone TEXT NOT NULL DEFAULT '',
    email TEXT NOT NULL DEFAULT '',
    active TEXT NOT NULL DEFAULT 'Yes'
);
ALTER TABLE engineers ADD COLUMN IF NOT EXISTS name TEXT NOT NULL DEFAULT '';
ALTER TABLE engineers ADD COLUMN IF NOT EXISTS role TEXT NOT NULL DEFAULT '';
ALTER TABLE engineers ADD COLUMN IF NOT EXISTS zone_code TEXT NOT NULL DEFAULT '';
ALTER TABLE engineers ADD COLUMN IF NOT EXISTS phone TEXT NOT NULL DEFAULT '';
ALTER TABLE engineers ADD COLUMN IF NOT EXISTS email TEXT NOT NULL DEFAULT '';
ALTER TABLE engineers ADD COLUMN IF NOT EXISTS active TEXT NOT NULL DEFAULT 'Yes';
CREATE INDEX IF NOT EXISTS idx_engineers_zone ON engineers (zone_code);
CREATE INDEX IF NOT EXISTS idx_engineers_email ON engineers (email);
CREATE INDEX IF NOT EXISTS idx_engineers_active ON engineers (active);

CREATE TABLE IF NOT EXISTS contractors (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL
);
ALTER TABLE contractors ADD COLUMN IF NOT EXISTS name TEXT NOT NULL DEFAULT '';
CREATE INDEX IF NOT EXISTS idx_contractors_name ON contractors (name);

CREATE TABLE IF NOT EXISTS workspace_users (
    id TEXT PRIMARY KEY,
    email TEXT NOT NULL UNIQUE,
    display_name TEXT NOT NULL,
    role TEXT NOT NULL,
    status TEXT NOT NULL,
    custom_role_id TEXT
);
ALTER TABLE workspace_users ADD COLUMN IF NOT EXISTS email TEXT NOT NULL DEFAULT '';
ALTER TABLE workspace_users ADD COLUMN IF NOT EXISTS display_name TEXT NOT NULL DEFAULT '';
ALTER TABLE workspace_users ADD COLUMN IF NOT EXISTS role TEXT NOT NULL DEFAULT '';
ALTER TABLE workspace_users ADD COLUMN IF NOT EXISTS status TEXT NOT NULL DEFAULT '';
ALTER TABLE workspace_users ADD COLUMN IF NOT EXISTS custom_role_id TEXT;
CREATE INDEX IF NOT EXISTS idx_workspace_users_role ON workspace_users (role);
CREATE INDEX IF NOT EXISTS idx_workspace_users_status ON workspace_users (status);
CREATE INDEX IF NOT EXISTS idx_workspace_users_custom_role ON workspace_users (custom_role_id);

CREATE TABLE IF NOT EXISTS custom_roles (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    permissions JSONB NOT NULL DEFAULT '[]'::jsonb,
    template TEXT
);
ALTER TABLE custom_roles ADD COLUMN IF NOT EXISTS name TEXT NOT NULL DEFAULT '';
ALTER TABLE custom_roles ADD COLUMN IF NOT EXISTS description TEXT NOT NULL DEFAULT '';
ALTER TABLE custom_roles ADD COLUMN IF NOT EXISTS permissions JSONB NOT NULL DEFAULT '[]'::jsonb;
ALTER TABLE custom_roles ADD COLUMN IF NOT EXISTS template TEXT;
CREATE INDEX IF NOT EXISTS idx_custom_roles_name ON custom_roles (name);

CREATE TABLE IF NOT EXISTS milestones (
    id TEXT PRIMARY KEY,
    title TEXT NOT NULL,
    due_date TEXT NOT NULL,
    zone_code TEXT NOT NULL,
    status TEXT NOT NULL,
    owner TEXT NOT NULL DEFAULT ''
);
ALTER TABLE milestones ADD COLUMN IF NOT EXISTS title TEXT NOT NULL DEFAULT '';
ALTER TABLE milestones ADD COLUMN IF NOT EXISTS due_date TEXT NOT NULL DEFAULT '';
ALTER TABLE milestones ADD COLUMN IF NOT EXISTS zone_code TEXT NOT NULL DEFAULT '';
ALTER TABLE milestones ADD COLUMN IF NOT EXISTS status TEXT NOT NULL DEFAULT '';
ALTER TABLE milestones ADD COLUMN IF NOT EXISTS owner TEXT NOT NULL DEFAULT '';
CREATE INDEX IF NOT EXISTS idx_milestones_zone ON milestones (zone_code);
CREATE INDEX IF NOT EXISTS idx_milestones_status ON milestones (status);
CREATE INDEX IF NOT EXISTS idx_milestones_due ON milestones (due_date);

CREATE TABLE IF NOT EXISTS materials (
    id TEXT PRIMARY KEY,
    item TEXT NOT NULL,
    zone_code TEXT NOT NULL,
    quantity INT NOT NULL DEFAULT 0,
    unit TEXT NOT NULL DEFAULT '',
    entry_date TEXT NOT NULL,
    supplier TEXT NOT NULL DEFAULT ''
);
ALTER TABLE materials ADD COLUMN IF NOT EXISTS item TEXT NOT NULL DEFAULT '';
ALTER TABLE materials ADD COLUMN IF NOT EXISTS zone_code TEXT NOT NULL DEFAULT '';
ALTER TABLE materials ADD COLUMN IF NOT EXISTS quantity INT NOT NULL DEFAULT 0;
ALTER TABLE materials ADD COLUMN IF NOT EXISTS unit TEXT NOT NULL DEFAULT '';
ALTER TABLE materials ADD COLUMN IF NOT EXISTS entry_date TEXT NOT NULL DEFAULT '';
ALTER TABLE materials ADD COLUMN IF NOT EXISTS supplier TEXT NOT NULL DEFAULT '';
CREATE INDEX IF NOT EXISTS idx_materials_zone ON materials (zone_code);
CREATE INDEX IF NOT EXISTS idx_materials_date ON materials (entry_date);
CREATE INDEX IF NOT EXISTS idx_materials_item ON materials (item);

CREATE TABLE IF NOT EXISTS task_projects (
    id SERIAL PRIMARY KEY,
    sort_order INT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    sections JSONB NOT NULL DEFAULT '[]'::jsonb
);
ALTER TABLE task_projects ADD COLUMN IF NOT EXISTS sort_order INT NOT NULL DEFAULT 0;
ALTER TABLE task_projects ADD COLUMN IF NOT EXISTS name TEXT NOT NULL DEFAULT '';
ALTER TABLE task_projects ADD COLUMN IF NOT EXISTS sections JSONB NOT NULL DEFAULT '[]'::jsonb;
CREATE INDEX IF NOT EXISTS idx_task_projects_sort ON task_projects (sort_order);

CREATE TABLE IF NOT EXISTS task_items (
    id TEXT PRIMARY KEY,
    project_id INT NOT NULL REFERENCES task_projects (id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    column_key TEXT NOT NULL,
    assignee TEXT NOT NULL DEFAULT ''
);
ALTER TABLE task_items ADD COLUMN IF NOT EXISTS project_id INT;
ALTER TABLE task_items ADD COLUMN IF NOT EXISTS title TEXT NOT NULL DEFAULT '';
ALTER TABLE task_items ADD COLUMN IF NOT EXISTS column_key TEXT NOT NULL DEFAULT '';
ALTER TABLE task_items ADD COLUMN IF NOT EXISTS assignee TEXT NOT NULL DEFAULT '';
CREATE INDEX IF NOT EXISTS idx_task_items_project ON task_items (project_id);
CREATE INDEX IF NOT EXISTS idx_task_items_column ON task_items (column_key);

CREATE TABLE IF NOT EXISTS audit_entries (
    id TEXT PRIMARY KEY,
    logged_at TEXT NOT NULL,
    logged_at_ts TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    user_name TEXT NOT NULL,
    action TEXT NOT NULL,
    detail TEXT NOT NULL DEFAULT ''
);
ALTER TABLE audit_entries ADD COLUMN IF NOT EXISTS logged_at TEXT NOT NULL DEFAULT '';
ALTER TABLE audit_entries ADD COLUMN IF NOT EXISTS logged_at_ts TIMESTAMPTZ NOT NULL DEFAULT NOW();
ALTER TABLE audit_entries ADD COLUMN IF NOT EXISTS user_name TEXT NOT NULL DEFAULT '';
ALTER TABLE audit_entries ADD COLUMN IF NOT EXISTS action TEXT NOT NULL DEFAULT '';
ALTER TABLE audit_entries ADD COLUMN IF NOT EXISTS detail TEXT NOT NULL DEFAULT '';
CREATE INDEX IF NOT EXISTS idx_audit_logged_at_ts ON audit_entries (logged_at_ts DESC);
CREATE INDEX IF NOT EXISTS idx_audit_user ON audit_entries (user_name);
CREATE INDEX IF NOT EXISTS idx_audit_action ON audit_entries (action);

CREATE TABLE IF NOT EXISTS assistance_messages (
    id SERIAL PRIMARY KEY,
    sender TEXT NOT NULL,
    body TEXT NOT NULL,
    sent_at TEXT NOT NULL
);
ALTER TABLE assistance_messages ADD COLUMN IF NOT EXISTS sender TEXT NOT NULL DEFAULT '';
ALTER TABLE assistance_messages ADD COLUMN IF NOT EXISTS body TEXT NOT NULL DEFAULT '';
ALTER TABLE assistance_messages ADD COLUMN IF NOT EXISTS sent_at TEXT NOT NULL DEFAULT '';
CREATE INDEX IF NOT EXISTS idx_assistance_sent_at ON assistance_messages (sent_at DESC);

CREATE TABLE IF NOT EXISTS profile_photos (
    email TEXT PRIMARY KEY,
    data_url TEXT NOT NULL
);
ALTER TABLE profile_photos ADD COLUMN IF NOT EXISTS data_url TEXT NOT NULL DEFAULT '';

CREATE TABLE IF NOT EXISTS app_meta (
    key TEXT PRIMARY KEY,
    value JSONB NOT NULL DEFAULT 'null'::jsonb
);
ALTER TABLE app_meta ADD COLUMN IF NOT EXISTS value JSONB NOT NULL DEFAULT 'null'::jsonb;
