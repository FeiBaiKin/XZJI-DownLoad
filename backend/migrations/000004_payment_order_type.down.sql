DROP INDEX IF EXISTS idx_payment_orders_order_type;
ALTER TABLE payment_orders
  DROP COLUMN IF EXISTS order_type,
  DROP COLUMN IF EXISTS membership_level,
  DROP COLUMN IF EXISTS membership_days;
