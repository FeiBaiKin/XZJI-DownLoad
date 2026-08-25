# AGENTS.md

## 基本协作规则

- 始终使用中文回复。
- 在 Windows 环境优先使用 Git Bash 或 CMD，尽量少用 PowerShell。
- 修改前先检查 `git status`，不得覆盖或回退用户未明确要求处理的改动。

## 运行时上传文件：最高优先级规则

- `backend/uploads/` 是运行时数据目录，里面可能包含用户头像、站点 Logo、关键词图标及媒体文件。
- 构建、测试、打包、Git 暂存、提交、推送时，严禁包含 `backend/uploads/` 下的任何真实文件。
- Git 仓库中该目录只允许保留 `backend/uploads/.gitkeep`。
- 发布包中只允许保留空目录占位符 `uploads/.gitkeep`，不得把本机 `backend/uploads/` 复制到发布包。
- 不得使用 `git add -f backend/uploads/...` 绕过忽略规则。
- 打包时不得为了清空发布包而删除本机 `backend/uploads/`；只能重建独立的 `deploy-package/` 输出目录。

提交或推送前必须检查：

```bash
git status --short --ignored
git ls-files backend/uploads
git diff --cached --name-only
```

`git ls-files backend/uploads` 的唯一允许输出是：

```text
backend/uploads/.gitkeep
```

如果暂存区出现其他 `backend/uploads/` 文件，必须先取消暂存，禁止提交和推送。

## 前端构建

前端生产构建固定使用以下命令，不要替换为其他构建命令：

```bash
cd frontend
npx vite build
```

构建产物位于 `frontend/dist/`，该目录不提交 Git。

## 前端设计规范

- 图标：统一使用 Lucide 图标（`@lucide/vue`），界面全程禁止使用表情符号；全站只保留这一个图标族，禁止混用其他图标库或手绘 SVG 图标。
- 图标一律经由 `src/components/AppIcon.vue` 渲染：尺寸只允许五档 `xs / sm / md / lg / xl`（14 / 16 / 20 / 24 / 32 px），强调态传 `strong`（2px 线宽），实心开关态传 `filled`；超大纯装饰底纹可直接渲染图标组件，不受档位限制。
- 图标对比度：禁止手动调细线宽来"弱化"图标，小尺寸细线会糊成灰影；辅助信息一律靠颜色降权（不低于 `text-zinc-500` / 深色 `dark:text-zinc-400`）。
- 设计标准：对标 Awwwards 顶级网站水准，达到 Awwwards、FWA、CSS Design Awards 每日最佳网站同等设计品质。
- 创意自由度：将浏览器视作交互式艺术画布，跳出传统布局框架，追求先锋视觉风格、实验性排版、流畅物理动效、极具冲击力的文字版式。
- 沉浸式体验：融合代码、高级渲染逻辑，打造统一完整的精品页面，做出突破常规 UI 认知、令人惊艳的数字交互体验。

## 后端检查与构建

常规检查：

```bash
cd backend
go test ./...
```

Linux AMD64 发布二进制由仓库根目录的 `package-deploy.cmd` 统一构建。

## 发布打包

统一从仓库根目录运行：

```bat
package-deploy.cmd
```

脚本必须完成：

1. 使用 `npx vite build` 构建前端；
2. 构建 Linux AMD64 Go 后端；
3. 重建 `deploy-package/`；
4. 仅复制后端二进制、`config.yaml.example`、迁移文件和前端 `dist`；
5. 发布包内只创建 `uploads/.gitkeep`，不复制任何真实上传文件；
6. 生成 `xzji-deploy.tar.gz`。

打包后检查归档内容：

```bash
tar -tzf xzji-deploy.tar.gz
```

归档中 `uploads/` 下只能出现 `uploads/.gitkeep`。

## Git 提交与推送

- 只有用户明确要求时才执行提交或推送。
- 提交前先运行必要测试/构建，并检查工作树和暂存区。
- 优先暂存明确的源码和文档路径；即使使用 `git add -A`，也必须再次检查暂存区。
- 提交前确认没有包含：真实上传文件、`frontend/dist/`、`deploy-package/`、发布压缩包、数据库备份、`config.yaml`、`.installed`、`license.json` 或其他运行时机密文件。
- 推送前再次运行：

```bash
git diff --cached --name-only
git status --short
git ls-files backend/uploads
```

- 确认无误后才可执行 `git commit` 和 `git push`。
