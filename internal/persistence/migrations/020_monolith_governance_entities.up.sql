-- Tables the ERP monolith clone needed and this service had no model for.
--
-- These were created directly against the platform database during the
-- migration and are recorded here so a fresh environment reproduces them. The
-- DDL is identical; IF NOT EXISTS makes it a no-op where they already stand.
--
-- gov_payments was the obvious home for payment certificates and does not fit:
-- milestone_id is NOT NULL and UNIQUE while every migrated certificate has an
-- empty milestone, and contract_id is NOT NULL while the source references a
-- project. It models milestone-based contract payments; these are interim
-- certificates against a project.
--
-- All of these carry the four-stage approval the source records (PM -> GM ->
-- CEO -> Accounts) as explicit columns rather than JSON, because "who approved
-- this and when" is the question these records exist to answer.

CREATE TABLE IF NOT EXISTS public.gov_general_requests (
    id                  UUID PRIMARY KEY,
    reference           TEXT NOT NULL UNIQUE,
    subject             TEXT NOT NULL DEFAULT '',
    category            TEXT NOT NULL DEFAULT '',
    department          TEXT NOT NULL DEFAULT '',
    description         TEXT NOT NULL DEFAULT '',
    priority            TEXT NOT NULL DEFAULT '',
    status              TEXT NOT NULL DEFAULT 'Submitted',
    amount              NUMERIC(18,4),
    currency            TEXT NOT NULL DEFAULT 'UGX',
    requested_by        TEXT NOT NULL DEFAULT '',
    approver            TEXT NOT NULL DEFAULT '',
    needed_by           DATE,
    request_date        DATE,
    approved_by_pm       TEXT, approved_by_pm_at       TIMESTAMPTZ,
    approved_by_gm       TEXT, approved_by_gm_at       TIMESTAMPTZ,
    approved_by_ceo      TEXT, approved_by_ceo_at      TIMESTAMPTZ,
    approved_by_accounts TEXT, approved_by_accounts_at TIMESTAMPTZ,
    rejected_by         TEXT, rejected_at TIMESTAMPTZ, rejection_reason TEXT,
    notes               TEXT NOT NULL DEFAULT '',
    created_by          TEXT NOT NULL DEFAULT '',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_gov_general_requests_status   ON public.gov_general_requests (status);
CREATE INDEX IF NOT EXISTS idx_gov_general_requests_category ON public.gov_general_requests (category);

CREATE TABLE IF NOT EXISTS public.gov_payment_certificates (
    id                  UUID PRIMARY KEY,
    reference           TEXT NOT NULL UNIQUE,
    project_id          UUID,
    project_name        TEXT NOT NULL DEFAULT '',
    contractor          TEXT NOT NULL DEFAULT '',
    payee               TEXT NOT NULL DEFAULT '',
    pay_to              TEXT NOT NULL DEFAULT '',
    description         TEXT NOT NULL DEFAULT '',
    phase               TEXT NOT NULL DEFAULT '',
    milestone           TEXT NOT NULL DEFAULT '',
    amount              NUMERIC(18,4) NOT NULL DEFAULT 0,
    currency            TEXT NOT NULL DEFAULT 'UGX',
    status              TEXT NOT NULL DEFAULT 'Draft',
    department          TEXT NOT NULL DEFAULT '',
    bank_account        TEXT NOT NULL DEFAULT '',
    certificate_date    DATE,
    due_date            DATE,
    paid_at             DATE,
    paid_by             TEXT,
    approved_by_pm       TEXT, approved_by_pm_at       TIMESTAMPTZ,
    approved_by_gm       TEXT, approved_by_gm_at       TIMESTAMPTZ,
    approved_by_ceo      TEXT, approved_by_ceo_at      TIMESTAMPTZ,
    approved_by_accounts TEXT, approved_by_accounts_at TIMESTAMPTZ,
    rejected_by         TEXT, rejected_at TIMESTAMPTZ, rejection_reason TEXT,
    source_module       TEXT NOT NULL DEFAULT '',
    source_entity       TEXT NOT NULL DEFAULT '',
    source_reference    TEXT NOT NULL DEFAULT '',
    notes               TEXT NOT NULL DEFAULT '',
    created_by          TEXT NOT NULL DEFAULT '',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_gov_payment_certificates_project ON public.gov_payment_certificates (project_id);
CREATE INDEX IF NOT EXISTS idx_gov_payment_certificates_status  ON public.gov_payment_certificates (status);

CREATE TABLE IF NOT EXISTS public.gov_work_variations (
    id                UUID PRIMARY KEY,
    reference         TEXT NOT NULL UNIQUE,
    project_id        UUID,
    project_name      TEXT NOT NULL DEFAULT '',
    activity          TEXT NOT NULL DEFAULT '',
    description       TEXT NOT NULL DEFAULT '',
    phase             TEXT NOT NULL DEFAULT '',
    -- a variation can reduce scope as well as add to it, so this is signed
    amount_impact     NUMERIC(18,4) NOT NULL DEFAULT 0,
    currency          TEXT NOT NULL DEFAULT 'UGX',
    schedule_days     INTEGER NOT NULL DEFAULT 0,
    progress_percent  NUMERIC(5,2),
    status            TEXT NOT NULL DEFAULT 'Draft',
    variation_date    DATE,
    notes             TEXT NOT NULL DEFAULT '',
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_gov_work_variations_project ON public.gov_work_variations (project_id);
