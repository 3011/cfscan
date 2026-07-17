# CF Scanner

中心化 Cloudflare IP 扫描、地区测量、延迟排序与黑名单复检平台。
[English](README.md) · [项目文档](docs/README.md) · [安全策略](SECURITY.md) · [参与贡献](CONTRIBUTING.md)

> CF Scanner 是独立开源项目，与 Cloudflare, Inc. 不存在隶属、赞助或官方认可关系。Cloudflare 是 Cloudflare, Inc. 的商标。

![CF Scanner 运行总览](docs/images/dashboard.png)


## 文档导航

- 项目文档入口与阅读顺序：[`docs/README.md`](docs/README.md)；
- 架构与任务、结果、黑名单语义：[`docs/architecture.md`](docs/architecture.md)；
- 本地开发和完整测试：[`docs/development.md`](docs/development.md)；
- 环境变量、发布、回滚与故障检查：[`docs/operations.md`](docs/operations.md)。

## 架构

```text
React + shadcn/ui 管理台
          │
       Go Center ─── PostgreSQL
          │ HTTPS JSON（Agent 主动连接）
   ┌──────┼────────┐
 亚洲 Agent   欧洲 Agent   北美 Agent
```

中心负责 Cloudflare 官方 IP 前缀同步、任务采样和分发、结果筛选、按 Agent/地区/colo 排序，以及黑名单释放和复检。Agent 是轻量 Go 单文件程序，只负责领取任务、连接指定 IP、测量并回传结果。

## 已实现能力

- Cloudflare 官方 IPv4/IPv6 前缀、ASN 前缀和 colo 位置字典同步均可在自动化设置中启停、编辑频率和配置启动行为；
- 同一批 IP 向多个地区 Agent 下发，结果按 Agent 独立保存；
- 保持测试域名 Host 和 TLS SNI，底层强制连接待测 IP；
- TCP、TLS、TTFB、总耗时、HTTP、TLS、CF-RAY、colo 测量；
- 多次尝试计算失败率；
- 结果排行提供“最新结果 / 历史记录”双视图，支持真实总数、服务端分页与排序、任务、时间范围、Agent、可用状态、colo、大洲、国家和城市筛选；地理候选只展示当前匹配结果中真实存在的值和数量；
- 高延迟、高失败率和超时自动进入 Agent 级黑名单；
- 黑名单复查的执行频率、候选范围、选择比例、单轮上限、探测阈值和再次等待时间均可配置；
- React 19 + TypeScript 6 + Tailwind CSS 4 + shadcn/ui Rhea 管理端；
- 应用内账号登录、HttpOnly 会话和管理员 / 查看者两级权限；
- Agent 最后心跳使用共享秒级时钟实时更新，不依赖列表轮询刷新；
- PostgreSQL 任务租约和幂等结果回传；运行中任务支持“停止剩余任务”，晚到结果不会产生黑名单副作用；
- 自动化中心统一管理扫描计划、黑名单复查、数据源同步和带配置快照的执行记录。

## 前端架构

管理端已经按业务 feature 重构为原生 React + shadcn/ui 项目：

```text
web/src/
├── app/                 # 全局 Provider 与错误边界
├── components/
│   ├── ui/              # shadcn CLI 生成的基础组件
│   ├── layout/          # Sidebar、Header、App Shell
│   └── shared/          # Data Table、状态、空态与加载态
├── features/
│   ├── dashboard/
│   ├── scans/
│   ├── results/
│   ├── sources/
│   ├── blacklist/
│   ├── agents/
│   ├── settings/
│   ├── auth/
│   └── users/
├── routes/
├── hooks/
└── lib/
```

前端主要使用：

- React 19、TypeScript 6、Vite 8 和 React Router；
- shadcn/ui Rhea、Tailwind CSS 4、Base UI、OKLCH 主题和 Lucide React；
- TanStack Query 管理服务端数据；
- TanStack Table 实现排序、筛选、列显隐、分页和移动端卡片；
- React Hook Form + Zod 实现任务与 ASN 表单；
- next-themes 支持浅色、深色和跟随系统；
- Recharts + shadcn Chart 展示 colo 延迟；
- Sonner、错误边界、Skeleton、Empty State 和 Alert 提供统一反馈。

页面信息架构分为工作台、扫描运营、资源管理和系统。桌面端采用 Rhea inset Sidebar，小屏幕复用同一导航的 Sheet，并为复杂表格提供移动端卡片布局。业务代码只依赖项目内 UI 封装，不直接依赖 Base UI。设计系统、组件来源和升级流程见 [`docs/design-system.md`](docs/design-system.md) 与 [`docs/ui-component-inventory.md`](docs/ui-component-inventory.md)。

## 登录与权限

管理台使用应用内账号系统，不依赖 Ingress Basic Auth。登录会话使用随机令牌，浏览器仅保存 `HttpOnly + Secure + SameSite=Lax` Cookie，PostgreSQL 只保存令牌 SHA-256 哈希；默认会话有效期为 24 小时。密码使用 bcrypt 保存；同一来源短时间连续登录失败会触发临时限流并返回 `429 Too Many Requests`。

当前权限分为：

- **管理员**：查看全部数据，创建和停止任务，编辑自动化与数据源，管理 ASN、黑名单策略和平台账号；
- **查看者**：查看总览、任务、结果、黑名单、数据源、Agent 和自动化状态，但不能执行任何服务端写操作。

权限由中心 API 强制校验，查看者直接调用写接口会收到 `403 Forbidden`。前端同时隐藏对应操作入口。账号管理位于“系统 → 账号与权限”，支持创建、编辑角色、启停、重置密码和删除。系统禁止删除自己，以及停用、降级或删除最后一个有效管理员。

## 开发验证

运行完整门禁：

```bash
make check
```

该命令检查文档链接与环境变量、Go 测试和构建，以及前端 UI 边界、ESLint、TypeScript、Vitest 和生产构建。

浏览器测试需要已运行的 Vite、可访问的 API、管理员账号和 Chromium。完整步骤见 [`docs/development.md`](docs/development.md)。

## 扫描范围

创建手动任务或定时计划时，可以选择两种目标生成方式：

- **按数量采样**：从官方地址段和所有启用 ASN 前缀的去重并集中，轮询采样指定数量的 IP；
- **每个前缀取 1 个 IP**：对每条启用 CIDR 自动选择一个唯一 IP，覆盖当前全部前缀。

“每个前缀取 1 个 IP”模式中，`target_count` 不参与计算。是否包含 IPv6 由“包含 IPv6 采样”开关决定；若需要无视当前黑名单完成全量覆盖，还应打开“包含当前黑名单 IP”。多个 Agent 会扫描同一批 IP，因此最终任务数量约为：

```text
去重前缀数量 × 所选 Agent 数量
```

重叠前缀会分别获得一个目标，系统优先为更具体的前缀保留地址，并避免同一轮出现重复目标 IP。不同轮次会重新选择前缀内的地址。

## 自动化中心

管理台“设置”通过顶部单层 Tabs 集中展示所有非手动网络行为，分为：

- **总览**：显示已启用自动化、下一次执行、预计黑名单复查规模和即将执行时间线；
- **扫描计划**：按 Cron 和时区自动创建普通扫描任务；
- **黑名单复查**：配置候选范围、选择比例、单轮上限、避免重叠、尝试次数、超时、恢复阈值和再次失败等待时间；
- **数据源同步**：分别管理官方地址段、ASN 前缀与 colo 位置字典同步，支持启动时执行；
- **执行记录**：保存触发方式、执行状态、配置快照、生成任务和错误信息。

中心服务只保留一个每 30 秒检查数据库计划的调度循环，不再使用隐藏的固定同步或黑名单 Ticker。计划支持标准 5 段 Cron、`@hourly`、`@daily`、`@weekly` 和 IANA 时区。

黑名单复查默认保持旧行为：每 15 分钟检查已到期目标，每轮选择 50%，最多 500 个。上线后可直接在设置中修改或停用。复查成功且延迟、丢包符合阈值时移出黑名单；复查失败则按配置的等待时间重新进入候选。

自动执行配置存储在 PostgreSQL 的 `scan_schedules`、`blacklist_recheck_settings` 和 `source_sync_schedules` 表中。每次执行都会写入 `automation_runs`，并保留当时的配置快照。中心异常中断导致的运行中记录会在 15 分钟后自动标记为失败。

## 运行中心

```bash
CFSCAN_DATABASE_URL='postgres://cfscan:password@127.0.0.1:5432/cfscan?sslmode=disable' \
CFSCAN_AGENT_TOKEN='replace-me' \
CFSCAN_BOOTSTRAP_ADMIN_USERNAME='admin' \
CFSCAN_BOOTSTRAP_ADMIN_PASSWORD='replace-with-a-strong-password' \
CFSCAN_SESSION_TTL='24h' \
CFSCAN_COOKIE_SECURE='false' \
./bin/cfscan-server
```

`CFSCAN_BOOTSTRAP_ADMIN_*` 只在账号表为空时创建首个管理员；已有账号后不会覆盖密码。生产 HTTPS 环境应保持 `CFSCAN_COOKIE_SECURE=true`。

## 运行 Agent

```bash
CFSCAN_CENTER_URL='https://cfscan-agent.example.com' \
CFSCAN_AGENT_TOKEN='replace-me' \
CFSCAN_AGENT_NAME='hk-01' \
CFSCAN_AGENT_REGION='hong-kong' \
CFSCAN_AGENT_CONTINENT='asia' \
CFSCAN_AGENT_CONCURRENCY='64' \
./bin/cfscan-agent
```

## IP 数据源

中心维护两类 Cloudflare 地址来源，并额外维护一份 colo 地理位置字典：

- Cloudflare 官方公布的 IPv4/IPv6 代理前缀；
- ASN 当前通过 BGP 宣告的前缀；
- Cloudflare Status 公布的数据中心组件，用于映射 colo 对应的大洲、国家和城市。

内置 Cloudflare ASN：

```text
AS13335   CLOUDFLARENET
AS209242  CLOUDFLARESPECTRUM
AS14789   CLOUDFLARENET
AS394536  CLOUDFLARENET-SFO
AS395747  CLOUDFLARENET-SFO05
AS400095  CLOUDFLARENET
```

ASN 可以在“IP 数据源”页面单独同步和启停，也可以添加自定义 ASN。创建扫描任务时，中心对官方前缀和所有启用 ASN 前缀取去重并集。

## colo 获取

Agent 将待测 IP 作为 TCP 连接目标，同时保持任务配置中的 HTTP Host 和 TLS SNI。默认请求 `/cdn-cgi/trace`：

1. 优先读取响应正文中的 `colo=`；
2. 正文没有 colo 时，从响应头 `CF-RAY` 最后的三字母代码回退识别；
3. 原始 `CF-RAY` 和最终 colo 都写入扫描结果；
4. 中心使用可定时同步的 Cloudflare Status 位置字典补充城市、国家和大洲信息；未知代码明确显示为“位置未识别”。

结果页提供“大洲 → 国家/地区 → 城市 → colo”四级筛选。位置字典属于自动化数据源，可在“设置 → 数据源同步”中查看、编辑频率或立即同步。

结果页中的地理筛选来自扫描结果聚合，而不是完整位置字典：大洲、国家、城市和 colo 只在存在匹配结果时出现；切换 Agent 或可用状态会刷新候选，选择上级后会立即收窄下级。大洲名称保持 Cloudflare 官方英文分组。colo 统一显示为 `LAX · Los Angeles, United States`，悬停或键盘聚焦时补充 `North America`。

## 结果排行与历史记录

结果页不再一次性截取前 1000 条数据，而是使用数据库端分页、排序和准确总数：

- **最新结果**：按 Agent、目标 IP、协议、域名、路径、端口、尝试次数和超时配置分组，只保留最新一条；可用状态和 colo 筛选在选出最新结果后应用，不会用旧成功记录替代最新失败；
- **历史记录**：保留每次原始扫描结果，不做去重，默认按扫描时间倒序；
- 默认查看最近 24 小时的可用最新结果，按 TTFB 升序；切换到历史记录时默认展示全部状态并按扫描时间倒序；每页 50 条，可切换 50、100 或 200 条；
- 支持最近 1 小时、24 小时、7 天、30 天和全部历史，以及扫描任务、Agent、IP 搜索和四级 colo 地理筛选；
- API 返回 `total`、`page`、`page_size`、`total_pages`，并返回同一筛选条件下全部、可用和失败的准确数量。

## UI 设计系统

管理端以 Rhea 为唯一 Style、Base UI 为默认 primitive、Tailwind CSS 4 与 OKLCH 为令牌基础。完整架构、本地适配和升级流程见 [`docs/design-system.md`](docs/design-system.md)，组件清单见 [`docs/ui-component-inventory.md`](docs/ui-component-inventory.md)。

选择控件遵循“固定短枚举使用 Select；实体选择、级联维度和长列表使用 Popup Combobox；同一业务维度保持同一种交互组件”。完整规则和当前场景矩阵见 [`docs/ui-component-guidelines.md`](docs/ui-component-guidelines.md)。

完整代码审计、已处理项和后续维护建议见 [`docs/maintenance-audit.md`](docs/maintenance-audit.md)。

## 停止扫描任务

等待中任务可取消，运行中任务可执行“停止剩余任务”：已完成结果保留，尚未领取及当前租约中的未完成目标标记为取消，之后不再下发。Agent 已开始的短探测可能产生晚到回传，但中心会忽略这些结果，不更新任务进度和黑名单。已停止任务不会恢复，需重新创建任务。

## 许可证与负责任使用

项目采用 [Apache License 2.0](LICENSE)。仅扫描你拥有或获得明确授权的系统与地址范围；部署前请阅读 [`RESPONSIBLE_USE.md`](RESPONSIBLE_USE.md)。安全问题请按 [`SECURITY.md`](SECURITY.md) 私下报告。
