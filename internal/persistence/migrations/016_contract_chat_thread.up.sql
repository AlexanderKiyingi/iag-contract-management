-- iag-chat discussion thread per governance contract. The conversation id is
-- find-or-created (S2S) on contract creation and stored here so the UI can load
-- the thread directly.
ALTER TABLE gov_contracts ADD COLUMN IF NOT EXISTS conversation_id TEXT NOT NULL DEFAULT '';
