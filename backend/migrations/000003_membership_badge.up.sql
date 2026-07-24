-- 会员套餐自定义角标（如「推荐」「超值」），空值不显示
ALTER TABLE membership_config
  ADD COLUMN IF NOT EXISTS badge varchar(20) DEFAULT '';

-- 存量数据沿用旧版「中间档位标注推荐」的展示口径，避免升级后角标凭空消失
-- idx / total 与前端旧逻辑 recommendedIndex 对齐：按价格升序取 floor(total / 2)，不足两档不标注
WITH ordered AS (
  SELECT id, ROW_NUMBER() OVER (ORDER BY price) - 1 AS idx, COUNT(*) OVER () AS total
  FROM membership_config
)
UPDATE membership_config m
SET badge = '推荐'
FROM ordered o
WHERE m.id = o.id AND o.total >= 2 AND o.idx = o.total / 2;
