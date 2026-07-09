-- 014: Coffee supply (off-take) contracts.
--
-- A commodity off-take agreement between a cooperative/exporter and a farmer:
-- committed volume, base price/kg and a quality bonus rate, plus the agreement
-- text and the farmer's signature. This is deliberately a separate domain from
-- the construction-oriented governance contracts (gov_contracts, lump-sum value
-- + retention + IPC valuations) — the two share nothing but the word "contract".
CREATE TABLE IF NOT EXISTS supply_contracts (
  id                     TEXT PRIMARY KEY,
  farmer_business_id     TEXT NOT NULL DEFAULT '',
  farmer_name            TEXT NOT NULL DEFAULT '',
  variety                TEXT NOT NULL DEFAULT '',
  committed_weight_kg    NUMERIC NOT NULL DEFAULT 0,
  base_price_per_kg_ugx  BIGINT NOT NULL DEFAULT 0,
  quality_bonus_rate_ugx BIGINT NOT NULL DEFAULT 0,
  sign_date              TIMESTAMPTZ,
  status                 TEXT NOT NULL DEFAULT 'Draft',
  contract_text          TEXT NOT NULL DEFAULT '',
  signature              TEXT NOT NULL DEFAULT '',
  created_at             TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at             TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_supply_contracts_farmer ON supply_contracts (farmer_business_id);
CREATE INDEX IF NOT EXISTS idx_supply_contracts_status ON supply_contracts (status);
