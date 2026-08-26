-- Purge the governance dataset seeded by 009_seed_monthly_report_may2026.up.sql and
-- 010_monthly_report_meta.up.sql.
--
-- 009 transcribed the May-2026 monthly-report workbook into gov_contractors,
-- gov_contracts, gov_progress_reports, gov_valuations and gov_requisitions; 010 added
-- the challenges and action items that accompany it. Every row carries a '-SEED-'
-- identifier, so records captured through the app since — which use generated ids —
-- are untouched by these predicates.
--
-- Dependents added after 009 (milestones, obligations, variations, payments,
-- closeouts) reference gov_contracts, so they are cleared for the seeded contracts
-- first. Nothing here touches the legacy zone-works contract tables, which hold no
-- seeded governance rows.
--
-- The runner wraps each migration in its own transaction, so this file does not open
-- one: a COMMIT here would end that transaction early.

-- ---- dependents of the seeded contracts -----------------------------------
DELETE FROM gov_payments     WHERE contract_id LIKE 'GCT-SEED-%';
DELETE FROM gov_milestones   WHERE contract_id LIKE 'GCT-SEED-%';
DELETE FROM gov_obligations  WHERE contract_id LIKE 'GCT-SEED-%';
DELETE FROM gov_variations   WHERE contract_id LIKE 'GCT-SEED-%';
DELETE FROM gov_closeouts    WHERE contract_id LIKE 'GCT-SEED-%';

-- ---- the seeded workbook rows ---------------------------------------------
DELETE FROM gov_progress_reports WHERE id LIKE 'PRG-SEED-%' OR contract_id LIKE 'GCT-SEED-%';
DELETE FROM gov_valuations       WHERE id LIKE 'VAL-SEED-%';
DELETE FROM gov_requisitions     WHERE id LIKE 'GRQ-SEED-%';
DELETE FROM gov_action_items     WHERE id LIKE 'GACT-SEED-%';
DELETE FROM gov_challenges       WHERE id LIKE 'GCHL-SEED-%';

-- ---- parents last ----------------------------------------------------------
DELETE FROM gov_contracts   WHERE id LIKE 'GCT-SEED-%';
DELETE FROM gov_contractors WHERE id LIKE 'GCON-SEED-%';
