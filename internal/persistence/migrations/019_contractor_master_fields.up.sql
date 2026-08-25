-- Contractor master data: the columns the Contract Manager app carries.
--
-- gov_contractors was created as a normalized parent for the monthly report —
-- just a unique name plus a free-text contact — and later grew the portal
-- binding (platform_user_id, user_email). The Contract Manager frontend keeps a
-- fuller contractor record: company, code, trade, address, phone, email,
-- payment details, and so on.
--
-- Those columns land here rather than staying behind on the shared ERP record
-- store because the portal binding lives on this row. Splitting one contractor
-- across two stores would make "which login is this contractor?" ambiguous
-- exactly where it has to be unambiguous.
--
-- Every column is nullable-with-default so pre-existing rows (including the
-- importer's name-only contractors) stay valid and the MR importer keeps
-- working untouched.
ALTER TABLE gov_contractors ADD COLUMN IF NOT EXISTS company         TEXT NOT NULL DEFAULT '';
ALTER TABLE gov_contractors ADD COLUMN IF NOT EXISTS code            TEXT NOT NULL DEFAULT '';
ALTER TABLE gov_contractors ADD COLUMN IF NOT EXISTS username        TEXT NOT NULL DEFAULT '';
ALTER TABLE gov_contractors ADD COLUMN IF NOT EXISTS type            TEXT NOT NULL DEFAULT '';
ALTER TABLE gov_contractors ADD COLUMN IF NOT EXISTS address         TEXT NOT NULL DEFAULT '';
ALTER TABLE gov_contractors ADD COLUMN IF NOT EXISTS phone           TEXT NOT NULL DEFAULT '';
ALTER TABLE gov_contractors ADD COLUMN IF NOT EXISTS email           TEXT NOT NULL DEFAULT '';
ALTER TABLE gov_contractors ADD COLUMN IF NOT EXISTS payment_details TEXT NOT NULL DEFAULT '';
ALTER TABLE gov_contractors ADD COLUMN IF NOT EXISTS trade           TEXT NOT NULL DEFAULT '';
ALTER TABLE gov_contractors ADD COLUMN IF NOT EXISTS status          TEXT NOT NULL DEFAULT 'Active';
ALTER TABLE gov_contractors ADD COLUMN IF NOT EXISTS notes           TEXT NOT NULL DEFAULT '';
ALTER TABLE gov_contractors ADD COLUMN IF NOT EXISTS attachments     TEXT NOT NULL DEFAULT '';

-- `email` is the contractor's business contact address and is deliberately NOT
-- the same column as `user_email`, which binds the row to a platform login.
-- They frequently hold the same value; conflating them would silently grant
-- portal access to whoever a contact address was last set to.
CREATE INDEX IF NOT EXISTS idx_gov_contractors_code
  ON gov_contractors (code)
  WHERE code <> '';
