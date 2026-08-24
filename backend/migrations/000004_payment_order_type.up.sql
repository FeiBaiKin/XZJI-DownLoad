-- 支付订单区分用途：recharge 充值入余额；membership 直接开通会员（免去先充值再扣余额）
ALTER TABLE payment_orders
  ADD COLUMN IF NOT EXISTS order_type varchar(20) DEFAULT 'recharge',
  ADD COLUMN IF NOT EXISTS membership_level varchar(50) DEFAULT '',
  ADD COLUMN IF NOT EXISTS membership_days integer DEFAULT 0;

-- 存量订单一律视为充值（历史上仅有充值一种用途），避免 NULL 走进会员开通分支
UPDATE payment_orders SET order_type = 'recharge' WHERE order_type IS NULL OR order_type = '';

CREATE INDEX IF NOT EXISTS idx_payment_orders_order_type ON payment_orders (order_type);
