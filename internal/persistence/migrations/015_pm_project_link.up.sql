-- Link governance contracts (and the requisitions that convert into them) to a
-- parent Project Management project. A Project owns Contracts; this id is what
-- the project-management consumer keys on to attach the contract to its project.
ALTER TABLE gov_contracts ADD COLUMN IF NOT EXISTS pm_project_id TEXT NOT NULL DEFAULT '';
ALTER TABLE gov_requisitions ADD COLUMN IF NOT EXISTS pm_project_id TEXT NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS idx_gov_contracts_pm_project_id
    ON gov_contracts (pm_project_id) WHERE pm_project_id <> '';
