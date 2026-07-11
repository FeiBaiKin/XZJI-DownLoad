-- PostgreSQL baseline: 安装向导 / GORM AutoMigrate 建立的初始表结构。
-- 此文件为基准文档，首次运行时由 RunMigrations 自动标记为已应用（不会重复执行）。
-- 后续对表结构的增删改请新建编号递增的 SQL 迁移文件。

-- 用户表
CREATE TABLE IF NOT EXISTS "users" (
  "id" bigserial PRIMARY KEY,
  "username" varchar(50) NOT NULL,
  "password" varchar(255) NOT NULL,
  "role" varchar(20) NOT NULL DEFAULT 'user',
  "email" varchar(100) DEFAULT '',
  "nickname" varchar(50) DEFAULT '',
  "avatar" varchar(255) DEFAULT '',
  "balance" decimal(10,2) NOT NULL DEFAULT 0,
  "exp" int NOT NULL DEFAULT 0,
  "level" int NOT NULL DEFAULT 1,
  "membership_level" varchar(50) DEFAULT NULL,
  "membership_expires_at" timestamptz DEFAULT NULL,
  "daily_downloads" int NOT NULL DEFAULT 0,
  "download_date" date DEFAULT NULL,
  "email_verified" boolean NOT NULL DEFAULT false,
  "created_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "updated_at" timestamptz DEFAULT NULL,
  CONSTRAINT "uni_users_username" UNIQUE ("username")
);

-- 关键词响应
CREATE TABLE IF NOT EXISTS "keyword_responses" (
  "id" bigserial PRIMARY KEY,
  "keyword" varchar(255) NOT NULL,
  "response" text NOT NULL,
  "icon" varchar(255) NOT NULL DEFAULT '',
  "attributes" text NOT NULL DEFAULT '',
  "created_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "updated_at" timestamptz DEFAULT NULL,
  CONSTRAINT "uni_keyword_responses_keyword" UNIQUE ("keyword")
);

-- 关键词分类
CREATE TABLE IF NOT EXISTS "keyword_categories" (
  "id" bigserial PRIMARY KEY,
  "name" varchar(100) NOT NULL,
  "description" varchar(500) DEFAULT '',
  "sort_order" int NOT NULL DEFAULT 0,
  "created_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "updated_at" timestamptz DEFAULT NULL
);

-- 关键词↔分类多对多
CREATE TABLE IF NOT EXISTS "keyword_category_links" (
  "id" bigserial PRIMARY KEY,
  "keyword_id" bigint NOT NULL,
  "category_id" bigint NOT NULL,
  CONSTRAINT "unique_link" UNIQUE ("keyword_id", "category_id"),
  FOREIGN KEY ("keyword_id") REFERENCES "keyword_responses"("id") ON DELETE CASCADE,
  FOREIGN KEY ("category_id") REFERENCES "keyword_categories"("id") ON DELETE CASCADE
);

-- 自定义页面
CREATE TABLE IF NOT EXISTS "pages" (
  "id" bigserial PRIMARY KEY,
  "title" varchar(200) NOT NULL,
  "slug" varchar(100) NOT NULL,
  "content" text,
  "is_published" boolean NOT NULL DEFAULT false,
  "created_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "updated_at" timestamptz DEFAULT NULL,
  CONSTRAINT "uni_pages_slug" UNIQUE ("slug")
);

-- 公告
CREATE TABLE IF NOT EXISTS "announcements" (
  "id" bigserial PRIMARY KEY,
  "title" varchar(200) NOT NULL,
  "content" text,
  "type" varchar(20) NOT NULL DEFAULT 'info',
  "is_active" boolean NOT NULL DEFAULT true,
  "created_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "updated_at" timestamptz DEFAULT NULL
);

-- KV 配置
CREATE TABLE IF NOT EXISTS "site_settings" (
  "setting_key" varchar(100) PRIMARY KEY,
  "setting_value" text
);

-- 签到配置
CREATE TABLE IF NOT EXISTS "check_in_config" (
  "id" bigserial PRIMARY KEY,
  "daily_balance" decimal(10,2) NOT NULL DEFAULT 0.50,
  "consecutive_days" int NOT NULL DEFAULT 7,
  "consecutive_bonus_balance" decimal(10,2) NOT NULL DEFAULT 2.00,
  "daily_exp" int NOT NULL DEFAULT 10,
  "updated_at" timestamptz DEFAULT NULL
);

-- 签到记录
CREATE TABLE IF NOT EXISTS "check_in_records" (
  "id" bigserial PRIMARY KEY,
  "user_id" bigint NOT NULL,
  "check_in_date" date NOT NULL,
  "balance_earned" decimal(10,2) NOT NULL DEFAULT 0,
  "exp_earned" int NOT NULL DEFAULT 0,
  "is_bonus" boolean NOT NULL DEFAULT false,
  "created_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT "unique_daily" UNIQUE ("user_id", "check_in_date")
);

-- 等级配置
CREATE TABLE IF NOT EXISTS "level_config" (
  "id" bigserial PRIMARY KEY,
  "level" int NOT NULL,
  "exp_required" int NOT NULL,
  "label" varchar(50) NOT NULL DEFAULT '',
  "updated_at" timestamptz DEFAULT NULL,
  CONSTRAINT "uni_level_config_level" UNIQUE ("level")
);

-- 会员配置
CREATE TABLE IF NOT EXISTS "membership_config" (
  "id" bigserial PRIMARY KEY,
  "level_name" varchar(50) NOT NULL,
  "days" int NOT NULL DEFAULT 30,
  "price" decimal(10,2) NOT NULL DEFAULT 0,
  "daily_download_limit" int NOT NULL DEFAULT 20,
  "discount" decimal(5,2) NOT NULL DEFAULT 0,
  "color" varchar(20) DEFAULT '',
  "updated_at" timestamptz DEFAULT NULL,
  CONSTRAINT "uni_membership_config_level_name" UNIQUE ("level_name")
);

-- 会员订单
CREATE TABLE IF NOT EXISTS "membership_orders" (
  "id" bigserial PRIMARY KEY,
  "order_no" varchar(64) NOT NULL,
  "user_id" bigint NOT NULL,
  "membership_level" varchar(50) NOT NULL,
  "amount" decimal(10,2) NOT NULL DEFAULT 0,
  "days" int NOT NULL DEFAULT 30,
  "status" varchar(20) NOT NULL DEFAULT 'paid',
  "created_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "paid_at" timestamptz DEFAULT NULL,
  CONSTRAINT "uni_membership_orders_order_no" UNIQUE ("order_no")
);

-- 充值订单
CREATE TABLE IF NOT EXISTS "payment_orders" (
  "id" bigserial PRIMARY KEY,
  "order_no" varchar(64) NOT NULL,
  "user_id" bigint NOT NULL,
  "amount" decimal(10,2) NOT NULL DEFAULT 0,
  "points" int NOT NULL DEFAULT 0,
  "status" varchar(20) NOT NULL DEFAULT 'pending',
  "payment_method" varchar(20) DEFAULT '',
  "payment_scene" varchar(20) DEFAULT '',
  "trade_no" varchar(64) DEFAULT '',
  "open_id" varchar(128) DEFAULT '',
  "created_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "paid_at" timestamptz DEFAULT NULL,
  CONSTRAINT "uni_payment_orders_order_no" UNIQUE ("order_no")
);

-- 链接购买记录
CREATE TABLE IF NOT EXISTS "link_purchases" (
  "id" bigserial PRIMARY KEY,
  "user_id" bigint NOT NULL,
  "keyword_id" bigint NOT NULL,
  "block_index" int NOT NULL,
  "balance_spent" decimal(10,2) NOT NULL DEFAULT 0,
  "created_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT "unique_purchase" UNIQUE ("user_id", "keyword_id", "block_index")
);

-- 关键词收藏
CREATE TABLE IF NOT EXISTS "keyword_favorites" (
  "id" bigserial PRIMARY KEY,
  "user_id" bigint NOT NULL,
  "keyword_id" bigint NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT "unique_favorite" UNIQUE ("user_id", "keyword_id")
);
-- 余额/经验流水
CREATE TABLE IF NOT EXISTS "points_transactions" (
  "id" bigserial PRIMARY KEY,
  "user_id" bigint NOT NULL,
  "type" varchar(20) NOT NULL DEFAULT '',
  "amount" decimal(10,2) NOT NULL DEFAULT 0,
  "balance_amount" decimal(10,2) NOT NULL DEFAULT 0,
  "exp_earned" int NOT NULL DEFAULT 0,
  "description" varchar(255) DEFAULT '',
  "ref_id" bigint DEFAULT NULL,
  "created_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 会员到期提醒
CREATE TABLE IF NOT EXISTS "membership_reminders" (
  "id" bigserial PRIMARY KEY,
  "user_id" bigint NOT NULL,
  "days_left" int NOT NULL DEFAULT 0,
  "membership_level" varchar(50) DEFAULT '',
  "sent_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 邮箱验证码
CREATE TABLE IF NOT EXISTS "email_verification" (
  "id" bigserial PRIMARY KEY,
  "email" varchar(100) NOT NULL,
  "code" varchar(10) NOT NULL,
  "purpose" varchar(30) NOT NULL DEFAULT '',
  "expires_at" timestamptz NOT NULL,
  "used" boolean NOT NULL DEFAULT false,
  "created_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 媒体库
CREATE TABLE IF NOT EXISTS "media_files" (
  "id" bigserial PRIMARY KEY,
  "url" varchar(255) NOT NULL,
  "filename" varchar(255) NOT NULL,
  "origin_name" varchar(255) NOT NULL DEFAULT '',
  "mime" varchar(100) NOT NULL DEFAULT '',
  "size" bigint NOT NULL DEFAULT 0,
  "width" integer NOT NULL DEFAULT 0,
  "height" integer NOT NULL DEFAULT 0,
  "source" varchar(20) NOT NULL DEFAULT 'local',
  "uploader_id" bigint NOT NULL DEFAULT 0,
  "created_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP
);
-- 迁移版本
CREATE TABLE IF NOT EXISTS "schema_migrations" (
  "id" bigserial PRIMARY KEY,
  "version" varchar(255) NOT NULL,
  "applied_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT "uni_schema_migrations_version" UNIQUE ("version")
);


CREATE INDEX IF NOT EXISTS "idx_user_days" ON "membership_reminders" ("user_id", "days_left");
CREATE INDEX IF NOT EXISTS "idx_email_purpose" ON "email_verification" ("email", "purpose");
CREATE INDEX IF NOT EXISTS "idx_media_files_url" ON "media_files" ("url");
