-- Contract-governance Phase 3: the remaining governance modules — requisitions,
-- obligations, approval rules, templates, clauses, budgets, and closeout.

-- Requisitions: pre-contract requests with a PM→Dept→Finance→Management approval
-- chain; an approved requisition can be converted into a governance contract.
CREATE TABLE IF NOT EXISTS gov_requisitions (
    id                TEXT PRIMARY KEY,
    no                TEXT NOT NULL UNIQUE,
    title             TEXT NOT NULL,
    department        TEXT NOT NULL DEFAULT '',
    requester         TEXT NOT NULL DEFAULT '',
    type              TEXT NOT NULL DEFAULT '',
    procurement_method TEXT NOT NULL DEFAULT '',
    supplier          TEXT NOT NULL DEFAULT '',
    estimate          BIGINT NOT NULL DEFAULT 0,
    budget_code       TEXT NOT NULL DEFAULT '',
    urgency           TEXT NOT NULL DEFAULT 'Medium',
    status            TEXT NOT NULL DEFAULT 'Pending Approval',
    stage             INT NOT NULL DEFAULT 0,
    approvals         JSONB NOT NULL DEFAULT '[]'::jsonb,
    justification     TEXT NOT NULL DEFAULT '',
    docs              JSONB NOT NULL DEFAULT '[]'::jsonb,
    linked_contract   TEXT,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Obligations: bonds, insurance, HSE etc. owed against a contract, with a due
-- date, recurrence, evidence requirement, and escalation owner.
CREATE TABLE IF NOT EXISTS gov_obligations (
    id          TEXT PRIMARY KEY,
    contract_id TEXT NOT NULL REFERENCES gov_contracts (id) ON DELETE CASCADE,
    type        TEXT NOT NULL DEFAULT '',
    owner       TEXT NOT NULL DEFAULT '',
    due_date    TEXT,
    frequency   TEXT NOT NULL DEFAULT 'Once',
    evidence    TEXT NOT NULL DEFAULT '',
    status      TEXT NOT NULL DEFAULT 'Open',
    escalation  TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_gov_obligations_contract ON gov_obligations (contract_id);

-- Approval rules: value-banded routing config (the approval engine).
CREATE TABLE IF NOT EXISTS gov_approval_rules (
    id         TEXT PRIMARY KEY,
    name       TEXT NOT NULL,
    applies    TEXT NOT NULL DEFAULT '',
    threshold  TEXT NOT NULL DEFAULT '',
    min_value  BIGINT NOT NULL DEFAULT 0,
    max_value  BIGINT,
    route      JSONB NOT NULL DEFAULT '[]'::jsonb,
    status     TEXT NOT NULL DEFAULT 'Active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Contract templates and the clause library (the "risk" module).
CREATE TABLE IF NOT EXISTS gov_templates (
    id         TEXT PRIMARY KEY,
    name       TEXT NOT NULL,
    type       TEXT NOT NULL DEFAULT '',
    owner      TEXT NOT NULL DEFAULT '',
    version    TEXT NOT NULL DEFAULT 'v1.0',
    status     TEXT NOT NULL DEFAULT 'Draft',
    clauses    JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS gov_clauses (
    id         TEXT PRIMARY KEY,
    title      TEXT NOT NULL,
    risk       TEXT NOT NULL DEFAULT 'Medium',
    approved   BOOLEAN NOT NULL DEFAULT FALSE,
    owner      TEXT NOT NULL DEFAULT '',
    text       TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Budgets: CAPEX/OPEX lines with approved/committed/paid totals.
CREATE TABLE IF NOT EXISTS gov_budgets (
    code       TEXT PRIMARY KEY,
    name       TEXT NOT NULL DEFAULT '',
    owner      TEXT NOT NULL DEFAULT '',
    approved   BIGINT NOT NULL DEFAULT 0,
    committed  BIGINT NOT NULL DEFAULT 0,
    paid       BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Closeout: a per-contract completion checklist.
CREATE TABLE IF NOT EXISTS gov_closeouts (
    contract_id          TEXT PRIMARY KEY REFERENCES gov_contracts (id) ON DELETE CASCADE,
    final_account        BOOLEAN NOT NULL DEFAULT FALSE,
    retention_decision   BOOLEAN NOT NULL DEFAULT FALSE,
    defects_liability    BOOLEAN NOT NULL DEFAULT FALSE,
    documents_complete   BOOLEAN NOT NULL DEFAULT FALSE,
    unresolved_variations INT NOT NULL DEFAULT 0,
    final_report         BOOLEAN NOT NULL DEFAULT FALSE,
    status               TEXT NOT NULL DEFAULT 'Open',
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
