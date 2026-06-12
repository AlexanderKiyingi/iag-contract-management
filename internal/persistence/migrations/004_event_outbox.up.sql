-- Transactional outbox for iag.commercial domain events.
--
-- Before this, controllers published to Kafka synchronously inside the HTTP
-- request: if Kafka was unreachable after the domain write committed, the
-- event was lost with only a log line. The Bus now persists each event here
-- and a background publisher drains the table to Kafka with retry/backoff, so
-- a Kafka outage delays delivery instead of dropping it. Idempotent consumers
-- already dedupe, so at-least-once delivery is safe.
CREATE TABLE IF NOT EXISTS contract_event_outbox (
    id            BIGSERIAL PRIMARY KEY,
    event_type    TEXT        NOT NULL,
    event_key     TEXT,
    payload       JSONB       NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    available_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    attempts      INT         NOT NULL DEFAULT 0,
    last_error    TEXT,
    dispatched_at TIMESTAMPTZ
);

-- Partial index: the publisher only ever scans undispatched, due rows.
CREATE INDEX IF NOT EXISTS idx_contract_event_outbox_pending
    ON contract_event_outbox (available_at)
    WHERE dispatched_at IS NULL;
