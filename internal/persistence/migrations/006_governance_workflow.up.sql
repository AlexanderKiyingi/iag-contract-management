-- Contract-governance Phase 2: milestone payments + contract variations, each
-- with a multi-stage workflow whose sequencing is enforced server-side.

-- One payment per milestone, advancing through PM Approval → Finance Review →
-- Payment Authorization → Paid. `history` records who completed each stage.
CREATE TABLE IF NOT EXISTS gov_payments (
    id          TEXT PRIMARY KEY,
    milestone_id TEXT NOT NULL UNIQUE REFERENCES gov_milestones (id) ON DELETE CASCADE,
    contract_id TEXT NOT NULL,
    amount      BIGINT NOT NULL DEFAULT 0,
    retention   INT NOT NULL DEFAULT 0,
    payable     BIGINT NOT NULL DEFAULT 0,
    stage       INT NOT NULL DEFAULT 0,
    status      TEXT NOT NULL DEFAULT 'PM Approval',
    history     JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_gov_payments_contract ON gov_payments (contract_id);

-- Variations advance through Project Manager → Department Head → Procurement →
-- Management. On full approval the contract value is adjusted by `amount`.
CREATE TABLE IF NOT EXISTS gov_variations (
    id             TEXT PRIMARY KEY,
    contract_id    TEXT NOT NULL REFERENCES gov_contracts (id) ON DELETE CASCADE,
    number         TEXT NOT NULL DEFAULT '',
    title          TEXT NOT NULL DEFAULT '',
    amount         BIGINT NOT NULL DEFAULT 0,
    extension_days INT NOT NULL DEFAULT 0,
    description    TEXT NOT NULL DEFAULT '',
    reason         TEXT NOT NULL DEFAULT '',
    impact         TEXT NOT NULL DEFAULT '',
    status         TEXT NOT NULL DEFAULT 'Pending',
    stage          INT NOT NULL DEFAULT 0,
    approvals      JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_gov_variations_contract ON gov_variations (contract_id);
