# XZJI 项目概况

> 面向 AI 助手与新成员的项目速览文档。修改代码前请先通读本文，尤其是「核心业务流程」与「关键约定与坑」两节。

## 1. 项目是什么

XZJI 是一个**关键词资源搜索 + 会员经济系统**的全栈网站（Go 后端 + Vue 3 前端）：

- 访客在首页输入**关键词**搜索，命中后返回内容块（文本 / 图片 / 网盘链接）；关键词也可按分类浏览。
- 链接块有三种访问权限：`free`（免费）、`balance`（余额付费）、`membership`（会员专属）。付费链接需登录后用余额购买，会员链接需有效会员身份且受每日下载次数限制。
- 完整经济体系：**余额充值**（微信支付 V3 / 支付宝 / 易支付）、**会员购买**（余额支付、支持折扣与续期叠加）、**每日签到**（连续签到奖励）、**经验等级**（充值/签到得经验，自动升级）。
- **管理后台**：关键词、分类、自定义页面、公告、用户、站点设置、邮件（SMTP）、签到 / 等级 / 会员 / 支付配置、充值订单管理、仪表盘统计。
- 内置 **Web 安装向导**（类 WordPress）：首次运行访问 `/install` 填写站点信息、管理员账号、数据库配置，自动建库建表、生成配置文件。
- 项目历史：由原 React + Node 项目 **Node-XZJI** 1:1 移植而来（前端 React→Vue3，后端 Node→Go），源码注释中多处提及"适配 GO 契约"等移植痕迹。

## 2. 技术栈

| 部分 | 技术 |
|---|---|
| 后端 `backend/` | Go 1.26 + Gin 1.12 + GORM 1.31 + MySQL；Viper（配置）、golang-jwt/v5（JWT HS256）、bcrypt（密码）、net/smtp 自封装（邮件，支持 465 隐式 TLS 与 STARTTLS） |
| 前端 `frontend/` | Vue 3.5 + TypeScript（strict）+ Vite 7 + Pinia 3 + vue-router 4 + axios + wangEditor 5（富文本）+ DOMPurify + qrcode |
| 测试 `pwtest/` | Playwright / Node fetch 编写的临时验证脚本（非正式测试框架） |
| 包管理 | 根目录配置了 pnpm workspace（packages 仅 `frontend`）；但启动脚本用 npm，`frontend/node_modules` 当前为 npm 安装。两者均可用 |

⚠️ `frontend/package.json` 中声明了 `naive-ui`，但**全项目未实际使用**——UI 全部为手写组件 + 全局 CSS（Brutalist 风格，CSS 变量实现深/浅主题，JetBrains Mono / Space Grotesk 字体）。

## 3. 仓库结构

```
E:\Work\XZJI\
├── backend/                # Go 后端（模块名 xzji/backend）
│   ├── main.go             # 入口：安装检查 → 双模式启动 → 优雅关闭
│   ├── config.yaml         # 实际配置（安装向导生成，勿提交）
│   ├── config.yaml.example # 配置示例
│   ├── .installed          # 安装锁文件（存在=已安装；删除可重装）
│   ├── migrations/         # 空目录（迁移用 GORM AutoMigrate，无 SQL 文件）
│   ├── uploads/            # 用户上传：头像 avatar_<uid>_<ts>.<ext>、Logo logo_<ts>.<ext>
│   └── internal/
│       ├── app/            # App 结构体：依赖注入容器（DB/JWTSecret/4个Service/ConfigCache）
│       ├── cache/          # 泛型 TTLCache + ConfigCache（邮件/支付/签到/等级配置缓存）
│       ├── config/         # Viper 配置加载/保存、JWT secret 生成、DSN 构造
│       ├── database/       # 连接池初始化、建库、AutoMigrate、Seed 种子数据
│       ├── handler/        # HTTP 处理器（无状态函数，经 getApp(c) 取依赖）
│       ├── middleware/     # JWT 鉴权、RequireAdmin、CORS、内存滑动窗口限流
│       ├── models/         # 全部 18 个 GORM 模型（单文件 models.go）
│       ├── router/         # 路由：SetupInstallOnly（未安装）/ SetupFull（已安装）
│       ├── scheduler/      # 定时任务：会员到期提醒（首次5s后，之后每24h）
│       ├── service/        # 业务层：auth/user/content/economy/misc/membership/email/install
│       │   └── payment/    # 支付 Provider 接口 + wechat/alipay/epay 三家实现
│       └── util/           # JWT 生成/解析
├── frontend/               # Vue 3 前端
│   ├── vite.config.ts      # dev 端口 3000；/api 与 /uploads 代理到 :4000；@ → ./src
│   └── src/
│       ├── api/            # axios 实例 + auth/content/install API 模块
│       ├── components/     # ConfirmDialog / ToastContainer / RichTextEditor / MembershipModal / RechargeModal
│       ├── constants/      # 支付方式标签、公告类型、8个网盘平台识别正则
│       ├── layouts/        # UserLayout（用户中心）/ AdminLayout（管理后台）
│       ├── router/         # 路由 + 守卫（requireAuth / requireAdmin / 全局 beforeEach）
│       ├── stores/         # Pinia：auth / siteSettings / theme / toast
│       ├── styles/         # 19 个全局 CSS（~6900行），无 scoped（仅 InstallView 例外）
│       ├── types/          # api.ts（全部接口类型）、error.ts（extractErrorMessage）
│       ├── utils/          # formatTime / formatMoney
│       └── views/          # public/（6页）、user/（4页）、admin/（13页）
├── deploy-package/         # Linux 部署包：xzji-linux-amd64 二进制 + config.yaml + public/（前端构建产物）+ uploads/ + migrations/
├── xzji-deploy.tar.gz      # deploy-package 的打包归档
├── pwtest/                 # Playwright/fetch 验证脚本（epay 签名、充值弹窗、支付配置等）
├── start-all.bat           # 一键启动前后端（分窗口）
├── start-backend.bat       # cd backend && go run main.go（:4000）
├── start-frontend.bat      # cd frontend && npm run dev（:3000）
├── stop-all.bat
└── .zcode/skills/          # 指向用户级 AI 技能的符号链接（与项目业务无关）
```

## 4. 启动与开发

```bash
# 后端（端口 4000）
cd backend && go run main.go

# 前端 dev（端口 3000，/api 和 /uploads 代理到 4000）
cd frontend && npm run dev

# 前端类型检查 / 构建
cd frontend && npm run typecheck   # vue-tsc -b
cd frontend && npm run build       # 产出 frontend/dist

# Windows 一键启动
start-all.bat
```

- **首次运行**：`backend/.installed` 不存在时，后端只挂载安装路由并提示访问 `http://localhost:4000/install`；完成安装（自动建库建表、写入管理员、生成 `config.yaml` 与 `.installed`）后**需重启后端**。
- **重装**：删除 `backend/.installed`（必要时连同 `config.yaml`）后重启。
- **生产模式**：Go 进程直接服务前端静态文件——优先 `backend/public/`，回退 `../frontend/dist/`；非 `/api` 的 GET 未命中文件时回落 `index.html`（SPA fallback）；`/uploads` 映射到 `backend/uploads/`。

## 5. 后端架构

### 5.1 启动流程（main.go 双模式）

1. `service.IsInstalled()` 检查 `backend/.installed` 锁文件。
2. **未安装** → `router.SetupInstallOnly()`：仅 `/api/install/status`、`/api/install`、`/api/health` + SPA fallback，监听 :4000。
3. **已安装** → 加载 `config.yaml`（Viper，支持 `XZJI_` 前缀环境变量）→ 注入 JWT secret 与 CORS 白名单 → `database.Init`（连接池 MaxIdle 10 / MaxOpen 100 / 生命周期 1h → AutoMigrate → Seed）→ `scheduler.Start`（会员到期提醒）→ `app.NewApp` 构建依赖 → `router.SetupFull` 监听配置端口（默认 4000）。
4. 优雅关闭：SIGINT/SIGTERM → 停调度器 → 15s 超时关 HTTP → 关数据库连接池。

### 5.2 分层与依赖注入

```
router → handler（无状态函数）→ service（业务逻辑）→ GORM models → MySQL
```

- `app.App` 持有 `DB`、`JWTSecret`、`AuthService`、`UserService`、`ContentSvc`、`EconomySvc`、`Cache(*ConfigCache)`。
- router 用中间件将 `*app.App` 以 key `"app"` 塞进 `gin.Context`；handler 通过 `handler/context.go` 的 `getApp(c)` 取用（无全局变量，便于测试）。

### 5.3 数据模型（`internal/models/models.go`，18 张表，均显式 TableName）

| 模型 | 表名 | 用途与关键字段 |
|---|---|---|
| `User` | `users` | 用户：Username(uniq)、Password(bcrypt,json隐藏)、Role(user/admin)、Balance(decimal 10,2)、Exp、Level、MembershipLevel、MembershipExpiresAt、DailyDownloads、DownloadDate、EmailVerified |
| `KeywordResponse` | `keyword_responses` | 关键词：Keyword(uniq)、**Response(text，存 JSON 内容块数组字符串)** |
| `KeywordCategory` | `keyword_categories` | 分类：Name、Description、SortOrder |
| `KeywordCategoryLink` | `keyword_category_links` | 关键词↔分类多对多（联合唯一，外键级联删除） |
| `Page` | `pages` | 自定义页面：Title、Slug(uniq)、Content(longtext HTML)、IsPublished |
| `Announcement` | `announcements` | 公告：Title、Content、Type(info/warning/success)、IsActive |
| `SiteSetting` | `site_settings` | KV 配置表：SettingKey 主键；承载站点信息、`email_*`、`payment_*` 全部动态配置 |
| `CheckInConfig` | `check_in_config` | 签到配置：DailyBalance、ConsecutiveDays、ConsecutiveBonusBalance、DailyExp |
| `CheckInRecord` | `check_in_records` | 签到记录：UserID+CheckInDate 联合唯一防重复 |
| `LevelConfig` | `level_config` | 等级配置：Level(uniq)、ExpRequired、Label |
| `MembershipConfig` | `membership_config` | 会员档位：LevelName(uniq)、Days、Price、DailyDownloadLimit(-1不限)、Discount、Color |
| `MembershipOrder` | `membership_orders` | 会员购买订单（余额支付，Status 默认 paid） |
| `PaymentOrder` | `payment_orders` | 充值订单：OrderNo(uniq)、Amount、**Points(实为赠送经验)**、Status(pending/paid)、PaymentMethod、PaymentScene、TradeNo |
| `LinkPurchase` | `link_purchases` | 链接购买记录：UserID+KeywordID+BlockIndex 联合唯一 |
| `PointsTransaction` | `points_transactions` | 余额/经验流水：Type(recharge/purchase/membership/checkin/bonus)、BalanceAmount、ExpEarned、RefID |
| `MembershipReminder` | `membership_reminders` | 到期提醒去重记录（user+days_left） |
| `EmailVerification` | `email_verification` | 邮箱验证码：Email+Purpose(register/forgot-password/login)、Code、ExpiresAt、Used |
| `SchemaMigration` | `schema_migrations` | 迁移版本表（已建表但当前无写入逻辑） |

### 5.4 API 路由（`internal/router/router.go`，前缀 `/api`）

**公开（无鉴权）**：
- `GET /health`、`GET /install/status`、`POST /install`（仅安装模式）
- `POST /search`（关键词搜索，**付费链接已脱敏**）
- `GET /page/:slug`、`GET /pages`、`GET /keywords?category=`、`GET /announcements/active`、`GET /categories`、`GET /settings`、`GET /site/config`

**认证 `/auth`（公开 + IP 限流）**：
- `POST /login`（10次/分）、`POST /register`（5次/时）、`POST /send-code`（3次/分）、`POST /verify-code`、`POST /passwordless-login`、`POST /forgot-password`、`POST /reset-password`（5次/分）

**用户 `/user`（`AuthRequired` JWT）**：
- 资料：`GET|PUT /profile`、`PUT /password`、`POST /avatar`
- 经济：`GET /payment-methods`、`POST /recharge`、`GET /recharge-status/:orderNo`、`POST /purchase-link`、`GET /purchase-status`、`GET /transactions`
- 会员：`GET /membership`、`POST /buy-membership`、`GET /downloads-today`
- 签到/等级：`GET /checkin-status`、`POST /checkin`、`GET /level-info`

**支付回调（公开）**：
- `POST /payment/callback/wechat`、`POST|GET /payment/callback/alipay`、`ANY /payment/callback/epay`

**管理 `/admin`（`AuthRequired` + `RequireAdmin`）**：
- 内容 CRUD：`/keywords`（含 `/:id/categories` 关联读写）、`/categories`、`/pages`、`/announcements`
- 订单：`GET /payment-orders`、`GET /payment-orders/stats`、`POST /payment-orders/batch-delete`、`DELETE /payment-orders/:id`
- 配置：`GET /stats`、`GET|PUT /settings`、`POST /upload-logo`、`GET|PUT /email-config`、`POST /email-test`、`GET|PUT /checkin-config`、`GET|PUT /level-config`、`GET|PUT /membership-config`、`GET|PUT /payment-config`、`POST /trigger-membership-reminders`
- 用户：`GET /users`、`GET|PUT|DELETE /users/:id`

### 5.5 中间件（`internal/middleware/`）

- `AuthRequired(jwtSecret)`：解析 `Authorization: Bearer <JWT>`，写 `userID/username/role` 进 context；JWT 有效期 **24h**（`util/jwt.go`，HS256）。
- `RequireAdmin()`：role != admin → 403。
- `CORS()`：白名单来自 `config.yaml` 的 `server.allowed_origins`，**为空则允许 `*`（仅限开发）**。
- 限流：内存滑动窗口、按客户端 IP（支持 X-Forwarded-For / X-Real-IP），单实例有效。

### 5.6 支付集成（`internal/service/payment/`）

统一 `Provider` 接口（`Name/Label/Scenes/CreatePayment/VerifyCallback`）+ `init()` 注册表。配置全部存于 `site_settings` 的 `payment_*` 键（后台「支付配置」页维护），`GetPaymentConfig` 读取并带 5 分钟缓存。

| Provider | 说明 |
|---|---|
| `wechat` 微信支付 | V3 Native 扫码：RSA 签名下单取 `code_url`；回调 AEAD_AES_256_GCM（APIv3Key）解密验签 |
| `alipay` 支付宝 | 场景 page（电脑跳转）/ wap（手机，按 UA 自动选）/ precreate（当面付扫码）；RSA2 签名，回调用支付宝公钥验签 |
| `epay` 易支付 | 彩虹易支付协议：MD5 签名跳转 `submit.php` 收银台；回调校验 MD5，仅 `TRADE_SUCCESS` 入账 |

**通用入账** `ProcessPaidOrder`：事务 + `FOR UPDATE` 锁单 → 金额校验（±0.01）→ 幂等（已 paid 跳过）→ 更新订单 → 加余额 + 赠经验 → 写流水 → 触发等级升级检查。

### 5.7 缓存与调度器

- `cache/`：泛型线程安全 `TTLCache`；`ConfigCache` 缓存邮件(5min)/支付(5min)/签到(10min)/等级(10min)配置，admin 修改配置后调用对应 `Invalidate*` 失效。
- `scheduler/`：单任务——会员到期提醒。启动 5s 后首查，之后每 24h 调 `CheckMembershipExpiryReminders`：扫描 7 天内到期会员，按 **7/3/1 天**阈值发邮件，`membership_reminders` 表去重；SMTP 未配置则跳过。

### 5.8 配置文件（`backend/config.yaml`）

```yaml
server:
  port: 4000
  allowed_origins: []   # CORS 白名单，空 = 允许所有（仅开发）
database:
  host: 127.0.0.1
  port: 3306
  user: ...
  password: ...
  name: xzji
jwt:
  secret: ...           # 安装时自动生成 32 字节 hex
```

由安装向导生成，已被 .gitignore 忽略；示例见 `config.yaml.example`。

### 5.9 数据库迁移与种子

- **无 SQL 迁移文件**（`migrations/` 为空），启动时 GORM `AutoMigrate` 全部模型，迁移前 `cleanOrphanedData()` 清理孤儿关联。
- `Seed`（幂等）：3 个默认页面（privacy/terms/about）、签到默认配置、10 级等级（初来乍到→至尊会员）、3 档会员（基础/高级/至尊：30/90/365 天，¥30/80/200，折扣 0.90/0.80/0.60）、5 个示例关键词、默认邮件设置（`email_force_verify=0`、`email_enable_passwordless=0` 等）。

## 6. 前端架构

### 6.1 路由（`src/router/index.ts`，History 模式，全部懒加载）

- 公开（meta `{public: true}`）：`/` 首页、`/install` 安装向导、`/login`、`/register`、`/forgot-password`、`/page/:slug` 动态页面。
- `/user`（`beforeEnter: requireAuth`，UserLayout）：`` 个人中心、`profile` 资料、`membership` 会员、`security` 改密。
- `/admin`（`beforeEnter: requireAdmin`，AdminLayout）：`` 仪表盘、`keywords`、`categories`、`pages`、`announcements`、`settings`、`email`、`users`、`checkin`、`levels`、`membership`、`payment-config`、`orders`。
- 兜底 `/:pathMatch(.*)*` → `/`。
- 全局 `beforeEach`：已有 token 但内存无 user 时先 `fetchProfile()`（刷新后恢复角色信息），失败清空登录态；非 public 且未登录 → `/login?redirect=...`。

### 6.2 状态管理（Pinia，setup 语法）

- `auth`：token + user（均持久化 localStorage）；`isLoggedIn`、`isAdmin`；login/register/logout/fetchProfile。
- `siteSettings`：站点名/描述/页脚/Logo（首次访问拉 `/api/settings`）。
- `theme`：深/浅主题（localStorage，默认 dark），App.vue 同步 `data-theme` 属性。
- `toast`：全局通知队列（success/error/info，3.5s 自动消失）。

### 6.3 API 层（`src/api/index.ts`）

- axios 实例：`baseURL: '/api'`、超时 15s。
- 请求拦截器：自动附加 `Authorization: Bearer <localStorage.token>`。
- 响应拦截器：**401 → 清 token、清 auth store、跳 `/login`**（当前在 /login、/install、/register 时除外）。
- 模块：`authApi`（认证+资料）、`contentApi`（admin CRUD + public 接口）、`installApi`；经济/签到/admin 配置类接口散落在各页面直接调用 `api` 实例。

### 6.4 页面职责速查

- **HomePage**（最大页面，678 行）：搜索、分类筛选、内容块渲染（含付费链接购买弹窗）、关键词卡片（点击复制）、公告弹窗、支付回跳提示（`?from=alipay|epay`）、主题切换、管理后台入口。
- **RechargeModal**（最复杂组件，361 行）：充值三步流程 select→paying→done；扫码类（微信 Native / 支付宝当面付）用 qrcode 生成二维码并 **2 秒轮询** `/user/recharge-status/:orderNo`；跳转类（支付宝网页 / 易支付）新窗口打开。
- **KeywordManager**（后台最大，413 行）：内容块编辑器（text/image/link 增删排序、权限设置），内置网盘链接解析器（自动识别百度/阿里/夸克/蓝奏/天翼/123/微云/UC 并提取提取码）。
- **RichTextEditor**：直接用 wangEditor 核心 API `createEditor/createToolbar`（`@wangeditor/editor-for-vue` 与 Vue 3.5 不兼容，勿改回）。
- 其余页面见第 3 节目录树，命名即职责。

### 6.5 样式约定

全局 CSS（`src/styles/` 19 个文件按序在 main.ts 导入），**不用 scoped**（唯一例外 InstallView.vue）；主题靠 CSS 变量 + `html[data-theme]`。新增页面样式应遵循此模式，按模块建独立 CSS 文件。

## 7. 核心业务流程

### 7.1 付费链接防泄漏（安全关键，勿破坏）

1. 公开 `POST /api/search` 返回前经 `ContentService.RedactPaidLinks` **清空**付费/会员链接块的 `url` 与 `code`（提取码）。
2. 真实地址只经两个鉴权端点下发：`POST /user/purchase-link`（余额购买或会员校验通过后，由 `ResolveBlock` 解析）、`GET /user/purchase-status`（已购买过）。
3. 购买逻辑（`EconomyService.PurchaseLink`）：已购 → 直接返回并计入今日下载；会员链接 → 校验会员有效 + 每日下载上限；余额链接 → 事务扣款（应用会员折扣 `Discount`）→ 写 `LinkPurchase` + 流水。

### 7.2 充值链路

前端 RechargeModal → `POST /user/recharge`（金额 0–50000，赠经验 = `floor(amount*2)`，订单号 `R<毫秒><rand>`）→ 对应 Provider `CreatePayment` 返回二维码/跳转链接 → 用户支付 → 第三方回调 `/api/payment/callback/<provider>` 验签 → `ProcessPaidOrder` 幂等入账（加余额+经验+流水+升级检查）→ 前端轮询 `recharge-status` 确认。

### 7.3 会员

`POST /user/buy-membership`：余额支付（无第三方支付）；到期时间**叠加续期**（未过期从原到期日累加）；事务更新用户会员字段 + 写 `MembershipOrder` + 流水。会员权益：每日下载上限（`DailyDownloadLimit`，-1 不限）+ 购买链接折扣（`Discount`）。

### 7.4 签到与等级

- 签到（`service.CheckIn`）：事务内 `FOR UPDATE` 锁用户防并发；`check_in_records` 联合唯一防重复；发余额+经验；连续签到满 `ConsecutiveDays` 天发额外奖励；连续天数在内存中计算避免 N+1。
- 等级：经验来自充值赠送/签到；`CheckLevelUp` 按 `level_config` 的 `ExpRequired` 自动升级。

### 7.5 邮箱验证码体系

`email_verification` 表按 purpose 区分：`register`（注册验证，受 `email_force_verify` 开关控制是否强制）、`forgot-password`（三步重置）、`login`（免密登录，受 `email_enable_passwordless` 开关控制）。发送有 60s 频率限制 + IP 限流（3 次/分）；验证码 6 位数字；`consumeCode` 校验并标记 used。

## 8. 部署

- `deploy-package/`：Linux 部署包 = `xzji-linux-amd64`（后端二进制）+ `config.yaml` + `public/`（前端 `vite build` 产物）+ `uploads/` + `migrations/`。二进制直接服务 `public/` 静态文件与 API，单进程即可上线。
- `xzji-deploy.tar.gz`：上述目录的打包归档。
- 交叉编译示例：`cd backend && GOOS=linux GOARCH=amd64 go build -o xzji-linux-amd64 main.go`；前端 `npm run build` 后把 `dist/` 内容放入部署包 `public/`。
- 生产注意：`config.yaml` 应配置 `allowed_origins` 白名单；支付回调地址在后台「支付配置」中基于站点 URL 配置。

## 9. 测试与验证（`pwtest/`）

临时性端到端验证脚本（需本地已安装并运行前后端，管理员账号 `admin/admin123` 为本地开发环境安装时所设）：

- `verify-epay.mjs`：纯 fetch + crypto 验证易支付签名/下单/回调闭环（含与 Go `epaySign` 一致的 MD5 签名实现）。
- `verify-home-admin.mjs` / `verify-payment-config.mjs` / `verify-recharge-modal.mjs` / `verify-recharge-wechat.mjs`：Playwright 无头浏览器验证对应 UI 流程。

运行方式：`node pwtest/<script>.mjs`（需要环境中可解析 `playwright` 依赖）。项目没有正式的单元测试/CI。

## 10. 关键约定与坑

1. **关键词 `response` 字段是 JSON 字符串**（内容块数组序列化），后端原样存取，前端需 `JSON.parse` / `JSON.stringify`。内容块结构见 `frontend/src/types/api.ts` 的 `KeywordBlock`（type: text/image/link；access: free/balance/membership；cost、url、code 等）。
2. **块权限判定逻辑前后端各有一份**（后端 `ContentService.blockAccess`，前端 HomePage）：`access` 显式值优先 → `cost>0` 视为 balance → 否则 free。两边必须保持一致。
3. **部分 admin 端点收发纯数组**（非对象包裹）：`PUT /admin/level-config`、`PUT /admin/membership-config`、`PUT /admin/keywords/:id/categories`——这是 Node→Go 移植时定下的契约，勿随意改形状。
4. `PaymentOrder.Points` 字段名有历史包袱，**实际含义是赠送经验**。
5. 金额一律 `decimal(10,2)`；微信回调金额单位是**分**（入账时已转元）。
6. 上传限制：头像/Logo ≤5MB（前端资料页另限头像 2MB）；扩展名白名单 jpg/jpeg/png/gif/webp（Logo 额外允许 svg）。
7. JWT 24h 过期，无 refresh token；401 由前端拦截器统一登出跳转。
8. 限流与配置缓存都是**内存实现**，仅单实例部署有效；多实例需改造。
9. `SiteSetting` 是万能 KV 表：站点信息、`email_*`、`payment_*` 都在里面，admin 保存后记得调 `ConfigCache.Invalidate*`（现有 handler 已处理，新增配置类须遵循）。
10. 前端**无 .env 文件**，运行时配置全部来自后端接口；开发期后端地址写死在 `vite.config.ts` 的 proxy（4000）。
11. `backendRoot()` 目录定位逻辑在 router/handler/service 三处各有一份类似实现（已知重复，改动路径逻辑时三处都要看）。
12. `.zcode/skills/` 是指向用户级 AI 技能目录的符号链接，与项目业务无关。
13. 数据库结构变更 = 改 `models.go` + 重启（AutoMigrate）；`SchemaMigration` 表已建但版本写入逻辑尚未实现。
