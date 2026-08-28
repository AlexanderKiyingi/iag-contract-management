-- Project structure, requests and the shared attachment store
--
-- Created during the ERP monolith clone, when the tail of the migration turned
-- out to have no platform tables to land in. Recorded here so a fresh
-- environment reproduces them; IF NOT EXISTS makes this a no-op where they
-- already stand.
--
-- Cross-service references (project_id -> finance.projects) are plain columns
-- rather than foreign keys: the services deploy independently and a constraint
-- would couple their migration order. The accompanying *_name column carries
-- the source's own text, so the link survives even where the id does not.

CREATE TABLE IF NOT EXISTS public.gov_project_phases (
    id               UUID PRIMARY KEY,
    project_id       UUID,  -- references finance.projects; not an FK, see header
    project_name     TEXT NOT NULL DEFAULT '',
    code             TEXT NOT NULL DEFAULT '',
    name             TEXT NOT NULL,
    progress_percent NUMERIC(5,2),
    sort_order       INTEGER NOT NULL DEFAULT 0,
    status           TEXT NOT NULL DEFAULT '',
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_gov_project_phases_project ON public.gov_project_phases (project_id, sort_order);

CREATE TABLE IF NOT EXISTS public.gov_project_activities (
    id               UUID PRIMARY KEY,
    project_id       UUID,  -- references finance.projects; not an FK, see header
    project_name     TEXT NOT NULL DEFAULT '',
    phase            TEXT NOT NULL DEFAULT '',
    code             TEXT NOT NULL DEFAULT '',
    name             TEXT NOT NULL,
    progress_percent NUMERIC(5,2),
    status           TEXT NOT NULL DEFAULT '',
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_gov_project_activities_project ON public.gov_project_activities (project_id);

CREATE TABLE IF NOT EXISTS public.gov_project_updates (
    id               UUID PRIMARY KEY,
    project_id       UUID,  -- references finance.projects; not an FK, see header
    project_name     TEXT NOT NULL DEFAULT '',
    phase            TEXT NOT NULL DEFAULT '',
    activity         TEXT NOT NULL DEFAULT '',
    description      TEXT NOT NULL DEFAULT '',
    progress_percent NUMERIC(5,2),
    status           TEXT NOT NULL DEFAULT '',
    update_date      DATE,
    updated_by       TEXT NOT NULL DEFAULT '',
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_gov_project_updates_project ON public.gov_project_updates (project_id, update_date DESC);

-- Distinct from erp_employees: a project manager here may be a contractor
-- rather than staff, which is why contractor is a column and this is not a
-- view over employees.
CREATE TABLE IF NOT EXISTS public.gov_project_managers (
    id          UUID PRIMARY KEY,
    code        TEXT NOT NULL DEFAULT '',
    name        TEXT NOT NULL,
    title       TEXT NOT NULL DEFAULT '',
    department  TEXT NOT NULL DEFAULT '',
    contractor  TEXT NOT NULL DEFAULT '',
    email       TEXT NOT NULL DEFAULT '',
    phone       TEXT NOT NULL DEFAULT '',
    username    TEXT NOT NULL DEFAULT '',
    status      TEXT NOT NULL DEFAULT 'Active',
    notes       TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Carries the two-stage approval the source records: PM, then quantity surveyor.
CREATE TABLE IF NOT EXISTS public.gov_project_requisitions (
    id                UUID PRIMARY KEY,
    reference         TEXT NOT NULL UNIQUE,
    project_id        UUID,  -- references finance.projects; not an FK, see header
    project_name      TEXT NOT NULL DEFAULT '',
    phase             TEXT NOT NULL DEFAULT '',
    activity          TEXT NOT NULL DEFAULT '',
    milestone         TEXT NOT NULL DEFAULT '',
    contractor        TEXT NOT NULL DEFAULT '',
    description       TEXT NOT NULL DEFAULT '',
    quantity          NUMERIC(18,4),
    unit              TEXT NOT NULL DEFAULT '',
    amount            NUMERIC(18,4) NOT NULL DEFAULT 0,
    currency          TEXT NOT NULL DEFAULT 'UGX',
    status            TEXT NOT NULL DEFAULT 'Draft',
    fulfillment_path  TEXT NOT NULL DEFAULT '',
    needed_by         DATE,
    fulfilled_on      DATE,
    requisition_date  DATE,
    approved_by_pm    TEXT, approved_by_pm_at TIMESTAMPTZ,
    approved_by_qs    TEXT, approved_by_qs_at TIMESTAMPTZ,
    issued_by         TEXT, issued_by_at      TIMESTAMPTZ,
    -- the source carries line items as a JSON array and the platform has no
    -- line table for these; kept whole rather than dropped
    lines             JSONB NOT NULL DEFAULT '[]'::jsonb,
    notes             TEXT NOT NULL DEFAULT '',
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_gov_project_requisitions_project ON public.gov_project_requisitions (project_id);

CREATE TABLE IF NOT EXISTS public.gov_equipment_requests (
    id                   UUID PRIMARY KEY,
    reference            TEXT NOT NULL UNIQUE,
    project_id           UUID,  -- references finance.projects; not an FK, see header
    project_name         TEXT NOT NULL DEFAULT '',
    phase                TEXT NOT NULL DEFAULT '',
    request_type         TEXT NOT NULL DEFAULT '',
    description          TEXT NOT NULL DEFAULT '',
    requested_by         TEXT NOT NULL DEFAULT '',
    status               TEXT NOT NULL DEFAULT 'Draft',
    needed_by            DATE,
    return_on            DATE,
    request_date         DATE,
    approved_by_accounts TEXT, approved_by_accounts_at TIMESTAMPTZ,
    notes                TEXT NOT NULL DEFAULT '',
    created_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT now()
);
