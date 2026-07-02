-- Baseline: 安装向导 / GORM AutoMigrate 建立的初始表结构。
-- 此文件为基准文档，首次运行时由 RunMigrations 自动标记为已应用（不会重复执行）。
-- 后续对表结构的增删改请新建编号递增的 SQL 迁移文件。

-- 用户表
CREATE TABLE IF NOT EXISTS `users` (
  `id` bigint unsigned AUTO_INCREMENT PRIMARY KEY,
  `username` varchar(50) NOT NULL,
  `password` varchar(255) NOT NULL,
  `role` varchar(20) NOT NULL DEFAULT 'user',
  `email` varchar(255) DEFAULT '',
  `nickname` varchar(100) DEFAULT '',
  `avatar` varchar(500) DEFAULT '',
  `balance` decimal(10,2) NOT NULL DEFAULT 0,
  `exp` int NOT NULL DEFAULT 0,
  `level` int NOT NULL DEFAULT 1,
  `membership_level` varchar(50) DEFAULT NULL,
  `membership_expires_at` datetime DEFAULT NULL,
  `daily_downloads` int NOT NULL DEFAULT 0,
  `download_date` date DEFAULT NULL,
  `email_verified` tinyint(1) NOT NULL DEFAULT 0,
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime DEFAULT NULL ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY `uni_users_username` (`username`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 关键词响应
CREATE TABLE IF NOT EXISTS `keyword_responses` (
  `id` bigint unsigned AUTO_INCREMENT PRIMARY KEY,
  `keyword` varchar(255) NOT NULL,
  `response` text NOT NULL,
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime DEFAULT NULL ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY `uni_keyword_responses_keyword` (`keyword`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 关键词分类
CREATE TABLE IF NOT EXISTS `keyword_categories` (
  `id` bigint unsigned AUTO_INCREMENT PRIMARY KEY,
  `name` varchar(255) NOT NULL,
  `description` varchar(500) DEFAULT '',
  `sort_order` int NOT NULL DEFAULT 0,
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime DEFAULT NULL ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 关键词↔分类多对多
CREATE TABLE IF NOT EXISTS `keyword_category_links` (
  `id` bigint unsigned AUTO_INCREMENT PRIMARY KEY,
  `keyword_id` bigint unsigned NOT NULL,
  `category_id` bigint unsigned NOT NULL,
  UNIQUE KEY `unique_link` (`keyword_id`, `category_id`),
  FOREIGN KEY (`keyword_id`) REFERENCES `keyword_responses`(`id`) ON DELETE CASCADE,
  FOREIGN KEY (`category_id`) REFERENCES `keyword_categories`(`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 自定义页面
CREATE TABLE IF NOT EXISTS `pages` (
  `id` bigint unsigned AUTO_INCREMENT PRIMARY KEY,
  `title` varchar(255) NOT NULL,
  `slug` varchar(255) NOT NULL,
  `content` longtext,
  `is_published` tinyint(1) NOT NULL DEFAULT 0,
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime DEFAULT NULL ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY `uni_pages_slug` (`slug`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 公告
CREATE TABLE IF NOT EXISTS `announcements` (
  `id` bigint unsigned AUTO_INCREMENT PRIMARY KEY,
  `title` varchar(255) NOT NULL,
  `content` text,
  `type` varchar(20) NOT NULL DEFAULT 'info',
  `is_active` tinyint(1) NOT NULL DEFAULT 1,
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime DEFAULT NULL ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- KV 配置
CREATE TABLE IF NOT EXISTS `site_settings` (
  `setting_key` varchar(100) PRIMARY KEY,
  `setting_value` text
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 签到配置
CREATE TABLE IF NOT EXISTS `check_in_config` (
  `id` bigint unsigned AUTO_INCREMENT PRIMARY KEY,
  `daily_balance` decimal(10,2) NOT NULL DEFAULT 0.50,
  `consecutive_days` int NOT NULL DEFAULT 7,
  `consecutive_bonus_balance` decimal(10,2) NOT NULL DEFAULT 2.00,
  `daily_exp` int NOT NULL DEFAULT 10,
  `updated_at` datetime DEFAULT NULL ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 签到记录
CREATE TABLE IF NOT EXISTS `check_in_records` (
  `id` bigint unsigned AUTO_INCREMENT PRIMARY KEY,
  `user_id` bigint unsigned NOT NULL,
  `check_in_date` date NOT NULL,
  `balance_earned` decimal(10,2) NOT NULL DEFAULT 0,
  `exp_earned` int NOT NULL DEFAULT 0,
  `is_bonus` tinyint(1) NOT NULL DEFAULT 0,
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE KEY `unique_daily` (`user_id`, `check_in_date`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 等级配置
CREATE TABLE IF NOT EXISTS `level_config` (
  `id` bigint unsigned AUTO_INCREMENT PRIMARY KEY,
  `level` int NOT NULL,
  `exp_required` int NOT NULL,
  `label` varchar(100) NOT NULL DEFAULT '',
  `updated_at` datetime DEFAULT NULL ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY `uni_level_config_level` (`level`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 会员配置
CREATE TABLE IF NOT EXISTS `membership_config` (
  `id` bigint unsigned AUTO_INCREMENT PRIMARY KEY,
  `level_name` varchar(50) NOT NULL,
  `days` int NOT NULL DEFAULT 30,
  `price` decimal(10,2) NOT NULL DEFAULT 0,
  `daily_download_limit` int NOT NULL DEFAULT 20,
  `discount` decimal(5,2) NOT NULL DEFAULT 1.00,
  `color` varchar(20) DEFAULT '',
  `updated_at` datetime DEFAULT NULL ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY `uni_membership_config_level_name` (`level_name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 会员订单
CREATE TABLE IF NOT EXISTS `membership_orders` (
  `id` bigint unsigned AUTO_INCREMENT PRIMARY KEY,
  `order_no` varchar(64) NOT NULL,
  `user_id` bigint unsigned NOT NULL,
  `membership_level` varchar(50) NOT NULL,
  `amount` decimal(10,2) NOT NULL DEFAULT 0,
  `days` int NOT NULL DEFAULT 30,
  `status` varchar(20) NOT NULL DEFAULT 'paid',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `paid_at` datetime DEFAULT NULL,
  UNIQUE KEY `uni_membership_orders_order_no` (`order_no`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 充值订单
CREATE TABLE IF NOT EXISTS `payment_orders` (
  `id` bigint unsigned AUTO_INCREMENT PRIMARY KEY,
  `order_no` varchar(64) NOT NULL,
  `user_id` bigint unsigned NOT NULL,
  `amount` decimal(10,2) NOT NULL DEFAULT 0,
  `points` int NOT NULL DEFAULT 0,
  `status` varchar(20) NOT NULL DEFAULT 'pending',
  `payment_method` varchar(20) DEFAULT '',
  `payment_scene` varchar(20) DEFAULT '',
  `trade_no` varchar(100) DEFAULT '',
  `open_id` varchar(100) DEFAULT '',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `paid_at` datetime DEFAULT NULL,
  UNIQUE KEY `uni_payment_orders_order_no` (`order_no`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 链接购买记录
CREATE TABLE IF NOT EXISTS `link_purchases` (
  `id` bigint unsigned AUTO_INCREMENT PRIMARY KEY,
  `user_id` bigint unsigned NOT NULL,
  `keyword_id` bigint unsigned NOT NULL,
  `block_index` int NOT NULL,
  `balance_spent` decimal(10,2) NOT NULL DEFAULT 0,
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE KEY `unique_purchase` (`user_id`, `keyword_id`, `block_index`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 余额/经验流水
CREATE TABLE IF NOT EXISTS `points_transactions` (
  `id` bigint unsigned AUTO_INCREMENT PRIMARY KEY,
  `user_id` bigint unsigned NOT NULL,
  `type` varchar(20) NOT NULL DEFAULT '',
  `amount` decimal(10,2) NOT NULL DEFAULT 0,
  `balance_amount` decimal(10,2) NOT NULL DEFAULT 0,
  `exp_earned` int NOT NULL DEFAULT 0,
  `description` varchar(500) DEFAULT '',
  `ref_id` bigint unsigned DEFAULT NULL,
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 会员到期提醒
CREATE TABLE IF NOT EXISTS `membership_reminders` (
  `id` bigint unsigned AUTO_INCREMENT PRIMARY KEY,
  `user_id` bigint unsigned NOT NULL,
  `days_left` int NOT NULL DEFAULT 0,
  `membership_level` varchar(50) DEFAULT '',
  `sent_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  KEY `idx_user_days` (`user_id`, `days_left`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 邮箱验证码
CREATE TABLE IF NOT EXISTS `email_verifications` (
  `id` bigint unsigned AUTO_INCREMENT PRIMARY KEY,
  `email` varchar(255) NOT NULL,
  `code` varchar(10) NOT NULL,
  `purpose` varchar(20) NOT NULL DEFAULT '',
  `expires_at` datetime NOT NULL,
  `used` tinyint(1) NOT NULL DEFAULT 0,
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  KEY `idx_email_verifications` (`email`, `purpose`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 迁移版本
CREATE TABLE IF NOT EXISTS `schema_migrations` (
  `id` bigint unsigned AUTO_INCREMENT PRIMARY KEY,
  `version` varchar(255) NOT NULL,
  `applied_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE KEY `uni_schema_migrations_version` (`version`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
