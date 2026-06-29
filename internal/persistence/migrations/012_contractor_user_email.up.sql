-- Friendlier portal binding: link a governance contractor to a login by email
-- (admins know contractor emails; the JWT subject is opaque). Resolution
-- matches platform_user_id (subject) OR user_email, so both work.
ALTER TABLE gov_contractors ADD COLUMN IF NOT EXISTS user_email TEXT;

CREATE INDEX IF NOT EXISTS idx_gov_contractors_email
  ON gov_contractors (lower(user_email))
  WHERE user_email IS NOT NULL;
