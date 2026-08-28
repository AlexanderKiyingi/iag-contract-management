-- Valuation commercial fields: the columns the Contract Manager app's
-- "Contractor Invoices" form has always collected and had nowhere to put.
--
-- gov_valuations was modelled on one sheet of the monthly report — the
-- contractor, the period, and the four money columns that sheet reconciles.
-- The frontend form is an invoice form: it also asks for the contractor's own
-- reference, a due date, a currency, a division, a tax amount, a document
-- status and supporting files.
--
-- Until now the adapter mapped those seven fields to constants: `reference` was
-- overwritten with the row id on every read, `status` was recomputed as
-- Paid/Open, and the other five round-tripped as empty strings. A required
-- field the user filled in was discarded on save, which is worse than not
-- asking for it.
--
-- balanceDue is deliberately NOT a column. It is verified_value_owed minus
-- amount_paid; storing it would create a second answer to a question that
-- already has one, and the two would drift the first time either operand was
-- corrected. The API computes it on read instead.
--
-- Every column is NOT NULL with a default so the MR importer, the May-2026
-- seed rows and any existing valuation stay valid without a backfill.
ALTER TABLE gov_valuations ADD COLUMN IF NOT EXISTS reference   TEXT   NOT NULL DEFAULT '';
ALTER TABLE gov_valuations ADD COLUMN IF NOT EXISTS due_date    TEXT   NOT NULL DEFAULT '';
ALTER TABLE gov_valuations ADD COLUMN IF NOT EXISTS currency    TEXT   NOT NULL DEFAULT '';
ALTER TABLE gov_valuations ADD COLUMN IF NOT EXISTS division    TEXT   NOT NULL DEFAULT '';
ALTER TABLE gov_valuations ADD COLUMN IF NOT EXISTS tax         BIGINT NOT NULL DEFAULT 0;
ALTER TABLE gov_valuations ADD COLUMN IF NOT EXISTS attachments TEXT   NOT NULL DEFAULT '';

-- Existing rows predate the concept of a valuation status. The column defaults
-- to 'Draft', which would be a lie for a seeded row that has already been
-- verified and part-paid — so immediately after adding it, every row (all of
-- which are pre-existing, by definition of just having added the column) is
-- given the status the old derived Paid/Open rule would have shown. Rows
-- created from here on start at Draft for real.
ALTER TABLE gov_valuations ADD COLUMN IF NOT EXISTS status TEXT NOT NULL DEFAULT 'Draft';
UPDATE gov_valuations
   SET status = CASE
                  WHEN amount_paid >= verified_value_owed AND verified_value_owed > 0 THEN 'Paid'
                  ELSE 'Submitted'
                END
 WHERE status = 'Draft';

-- A contractor's own invoice reference is how finance finds the row when the
-- contractor calls about it, so it needs to be searchable. Not unique: two
-- contractors may each number their invoices from 1.
CREATE INDEX IF NOT EXISTS idx_gov_valuations_reference
  ON gov_valuations (reference)
  WHERE reference <> '';
