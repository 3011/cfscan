# 开发指南

## 环境要求

| 工具 | 建议版本 |
|---|---|
| Go | 1.25 or newer |
| Node.js | 22 |
| pnpm | 11.12.x（以 `web/package.json` 为准） |
| PostgreSQL | 17 |
| Docker / Compose | 用于本地一键环境和镜像构建 |
| Chromium | 运行 `pnpm test:ui` 时需要 |

Go 可使用官方工具链自动选择功能。保持 `GOTOOLCHAIN=auto`，然后运行 `make check`；CI 会根据 `go.mod` 安装所需版本。

## 快速启动

复制本地配置：

```bash
cp .env.example .env
```

一键启动 PostgreSQL、Center、Web 和本地 Agent：

```bash
docker compose --profile agent up -d --build
```

默认入口：

```text
Web:    http://localhost:18081
API:    http://localhost:18080
Health: http://localhost:18080/healthz
```

登录账号由 `.env` 中的 `CFSCAN_BOOTSTRAP_ADMIN_USERNAME` 和 `CFSCAN_BOOTSTRAP_ADMIN_PASSWORD` 决定。Compose 中的默认值只用于本机开发，不能复制到生产。

停止环境：

```bash
docker compose --profile agent down
```

保留数据库卷；需要完全清空本地数据时再显式执行 `docker compose down -v`。

## 分离开发

### 只启动 PostgreSQL

```bash
docker compose up -d postgres
```

加载本地环境变量并运行 Center：

```bash
set -a
. ./.env
set +a
make build
./bin/cfscan-server
```

`.env.example` 的数据库地址使用宿主机端口 `55432`，Center 监听宿主机端口 `18080`，与 Vite 代理一致。如果使用本机 PostgreSQL或其他端口，请自行调整。

### 启动 Web

Vite 默认把 `/api` 和 `/healthz` 代理到 `http://127.0.0.1:18080`：

```bash
cd web
pnpm install --frozen-lockfile
pnpm dev -- --port 4173
```

### 启动 Agent

```bash
set -a
. ./.env
set +a
CFSCAN_CENTER_URL=http://127.0.0.1:18080 ./bin/cfscan-agent
```

Agent Token 必须与 Center 一致。

## 目录结构

```text
cmd/server/                    Center 入口
cmd/agent/                     Agent 入口
internal/api/                  HTTP 路由、中间件和 Handler
internal/auth/                 登录、会话和校验
internal/automation/           黑名单与数据源自动化
internal/cloudflare/           官方前缀、ASN 和 colo 同步
internal/probe/                TCP/TLS/HTTP 探测
internal/scans/                扫描输入校验与创建流程
internal/scheduling/           Cron 扫描计划
internal/store/                Store interface
internal/store/postgres/       PostgreSQL 实现与运行时 schema
internal/targets/              目标采样
web/src/components/ui/         官方 Rhea primitive 与必要适配
web/src/components/shared/     项目级业务组合组件
web/src/features/              按业务领域组织的页面与 Hooks
docs/                          架构、运维和设计规范
scripts/check_docs.py          文档一致性检查
```

## 常用命令

```bash
make fmt          # Go 格式化
make test         # Go 单元测试
make build        # 构建 Center 与 Agent
make build-web    # 构建 Web
make docs-check   # 检查文档链接和环境变量覆盖
make check        # 文档 + Go + 前端完整门禁
```

前端单独执行：

```bash
cd web
pnpm check:ui-boundaries
pnpm lint
pnpm typecheck
pnpm test
pnpm build
pnpm check
```

## 浏览器回归

前置条件：

- API 可从 Vite 代理地址访问；
- Vite 运行在 `BASE_URL` 指定地址，默认 `http://127.0.0.1:4173`；
- 提供管理员账号；
- 系统安装 Chromium，或设置 `CHROMIUM_PATH`。

```bash
cd web
CFSCAN_UI_USERNAME=admin \
CFSCAN_UI_PASSWORD='your-local-password' \
BASE_URL=http://127.0.0.1:4173 \
pnpm test:ui
```

永久回归覆盖八个桌面路由、八个移动路由、权限、Sidebar、Tabs、Select、Combobox、列管理、重置、Tooltip、Dialog、Sheet、AlertDialog、主题、横向溢出和全局错误边界。

测试只应读取生产等价数据，涉及表单时应打开、输入后取消，不应在共享环境创建或删除真实业务数据。

## 开发约束

- 业务代码不得直接导入 `@base-ui/react`；
- 不得重新引入 Radix；
- UI 规则见 `design-system.md` 和 `ui-component-guidelines.md`；
- SQL 必须参数化；
- 新增环境变量必须同步 `.env.example` 和 `operations.md`；
- 新增运行时 schema 必须修改 `internal/store/postgres/schema.sql`；
- 不要把行为修改、格式化大重构和领域拆分混在同一提交；
- 不要提交凭据、Cookie、Token、测试账号或生产导出的数据库内容。

## 版本说明

公开版本由 Git Tag 和 GitHub Release 决定，例如 `v1.0.0`。同一 Release 会发布对应的二进制包和 GHCR 镜像；生产部署应使用不可变版本标签或镜像 digest。`web/package.json` 的版本用于前端包元数据，不应单独作为部署版本来源。用户可见变更记录在 `CHANGELOG.md`。
