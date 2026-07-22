DROP INDEX IF EXISTS idx_keyword_responses_status;
ALTER TABLE keyword_responses DROP COLUMN IF EXISTS status;
