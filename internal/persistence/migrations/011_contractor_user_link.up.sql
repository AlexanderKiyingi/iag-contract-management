-- Link a governance contractor to a platform user (JWT subject) so a
-- logged-in contractor can be scoped to their own contracts in the portal.
ALTER TABLE gov_contractors ADD COLUMN IF NOT EXISTS platform_user_id TEXT;

CREATE INDEX IF NOT EXISTS idx_gov_contractors_user
  ON gov_contractors (platform_user_id)
  WHERE platform_user_id IS NOT NULL;
