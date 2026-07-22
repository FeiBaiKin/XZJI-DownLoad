-- 关键词资源状态：draft（草稿/下架）| published（已上架）
ALTER TABLE keyword_responses
  ADD COLUMN IF NOT EXISTS status varchar(20) NOT NULL DEFAULT 'published';

CREATE INDEX IF NOT EXISTS idx_keyword_responses_status ON keyword_responses (status);

-- 历史数据默认已上架
UPDATE keyword_responses SET status = 'published' WHERE status IS NULL OR status = '';
