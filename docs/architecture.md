# 系统架构

Cloudflare IP Scanner 是一个中心化、Agent 执行的网络质量测量平台。中心负责配置、调度、持久化和策略；地区 Agent 只负责主动领取任务、直接连接目标 IP、测量并回传结果。

## 组件与边界

```text
浏览器
  │ 应用会话 Cookie
  ▼
Web（React / Nginx）
  │ /api/v1
  ▼
Center（Go / chi） ───────── PostgreSQL
  ▲                              │
  │ Bearer Agent Token           ├─ 配置与账号
  │ HTTPS JSON                   ├─ 任务租约
  │                              ├─ 扫描结果
地区 Agent ──────────────────────└─ 黑名单与自动化记录
```

### Center

- 唯一业务控制面；
- 同步 Cloudflare 官方前缀、ASN 前缀与 colo 字典；
- 生成目标、分发 Agent 级任务、管理租约；
- 保存结果、计算状态、维护临时黑名单；
- 执行扫描计划、黑名单复查和数据源同步；
- 提供应用登录、RBAC 和管理 API。

### Agent

- 无持久状态、只主动向中心发起出站请求；
- 以配置的名称注册，同名 Agent 重启会更新原记录；
- 默认每 15 秒心跳、每 5 秒轮询任务；
- 直接连接目标 IP，同时保留任务 Host 和 TLS SNI；
- 并发执行探测并批量回传结果；
- 不承担调度、去重、黑名单或排名逻辑。

### PostgreSQL

PostgreSQL 是配置、账号、会话、任务、结果和自动化状态的唯一事实来源。系统不依赖 Redis、消息队列或 Kubernetes Controller。

## 请求与认证边界

### 管理端 API

路径位于 `/api/v1`，使用 `cfscan_session` HttpOnly Cookie：

- 查看接口要求有效会话；
- 写接口要求管理员角色；
- 查看者直接调用写接口返回 `403 Forbidden`；
- 登录接口有来源级失败限流。

### Agent API

路径位于 `/api/v1/agent/*`，使用统一 Bearer Token：

- `register`：注册或更新 Agent；
- `heartbeat`：刷新在线时间；
- `tasks/claim`：领取租约批次；
- `tasks/results`：幂等回传结果。

Agent API 与管理会话相互独立。不要把 Agent Token 放入浏览器、URL、日志或仓库。

## API 权限矩阵

| 范围 | 认证 | 代表接口 |
|---|---|---|
| 健康检查 | 无 | `GET /healthz` |
| 登录 | 无，带失败限流 | `POST /api/v1/auth/login` |
| 当前会话 | Session Cookie | `GET /auth/me`、`POST /auth/logout` |
| 读取管理数据 | Session Cookie，管理员或查看者 | overview、agents、sources、jobs、schedules、results、colos、blacklist、automation |
| 修改业务数据 | Session Cookie，管理员 | 创建/停止任务、计划、数据源、同步和自动化配置 |
| 账号管理 | Session Cookie，管理员 | `/api/v1/users/*` |
| Agent | Bearer Agent Token | `/api/v1/agent/*` |

所有 JSON 错误使用统一结构：

```json
{
  "error": {
    "code": "permission_denied",
    "message": "当前账号只有查看权限"
  }
}
```

请求体最大读取 4 MiB，拒绝未知 JSON 字段和多余 JSON 对象。所有 API 响应带 `X-CFScan-Server-Time` 毫秒时间头，用于 Web 时钟校准。

## 任务生命周期

### 扫描任务

任务由手动创建、定时计划或黑名单复查生成。核心状态为：

```text
pending → running → completed
    └────────────→ stopped
```

每个目标按 Agent 生成独立任务行，因此多 Agent 扫描会放大任务数量。租约用于避免同一任务被同时重复领取；过期租约可以重新进入待领取状态。

### 停止语义

- 等待中的任务显示“取消任务”；
- 运行中的任务显示“停止剩余任务”；
- 未领取和仍在租约中的未完成目标标记为取消；
- 已完成结果保留；
- Agent 已经开始的短探测可能晚到，但中心不会让已停止任务恢复，也不会让晚到结果更新黑名单；
- 停止后不能继续原任务，需要复制配置重新创建。

当前不支持暂停/继续，因为 Agent 批次短、无持续可恢复执行上下文，暂停会显著增加协议和状态复杂度。

## 目标与数据源

目标来自以下启用来源的去重并集：

- Cloudflare 官方 IPv4/IPv6 前缀；
- 内置或管理员添加的 Cloudflare ASN BGP 前缀。

目标生成支持：

- 按数量采样；
- 每个前缀选择一个唯一 IP。

IPv6 只有在 Agent 所在节点具备可用 IPv6 出站能力时才有实际意义。

## 结果语义

结果必须按 Agent 维度理解，因为 Cloudflare 使用 Anycast，同一 IP 在不同地区的网络质量和 colo 可能不同。

### 最新结果

按以下组合分组，只保留最新一条：

```text
Agent + IP + scheme + hostname + path + port + attempts + timeout
```

可用状态和地理筛选在选出最新记录后应用，因此最新失败不会被旧成功替代。

### 历史记录

保留每次原始探测，不做去重，默认按扫描时间倒序。结果页的分页、排序和准确总数均由数据库处理。

### colo

Agent 优先读取 `/cdn-cgi/trace` 中的 `colo=`，没有时回退到 `CF-RAY` 后缀。中心使用同步的位置字典补充城市、国家和大洲。地理筛选候选来自当前结果聚合，而不是完整字典。

## 黑名单

黑名单按 `Agent + IP` 维护，而不是全局 IP：

- 高延迟、高丢包、超时或不可用可进入临时黑名单；
- 到期目标由黑名单复查自动化抽样；
- 恢复且满足阈值时移除；
- 失败时根据配置延后下一次复查；
- 已停止扫描的晚到结果不得产生黑名单副作用。

## 自动化

中心只有一个每 30 秒执行的数据库调度循环：

- 扫描计划；
- 黑名单复查；
- 官方前缀、ASN 前缀与 colo 字典同步。

每次自动执行写入 `automation_runs`，保存触发方式、状态、配置快照、摘要和错误。启动时自动化可按配置执行；异常遗留的运行中记录会在超时后收敛为失败。

## 时间语义

数据库和中心时间是运行状态的基准。所有 API 响应返回服务器时间，Web 使用请求时钟偏移校准相对时间，因此 Agent 心跳和账号最后登录不依赖用户电脑时钟是否准确。

Agent 在 `last_seen_at` 距数据库当前时间 45 秒内视为在线。

## 数据库迁移

中心启动顺序：

1. 创建连接池并 Ping PostgreSQL；
2. 执行嵌入的 `internal/store/postgres/schema.sql`；
3. 确保首个管理员存在；
4. 启动调度器和 HTTP 服务。

任何一步失败都会终止中心启动。当前运行时不遍历 `migrations/` 目录；修改 schema 时必须更新 `internal/store/postgres/schema.sql`，并验证旧数据库向前升级和必要的回滚兼容性。

## 非目标

除非有测量数据证明需要，否则不要引入：

- 微服务拆分；
- Redis、Kafka、NATS 或 ClickHouse；
- Agent 入站控制端口；
- Kubernetes 自定义 Controller；
- 浏览器直接访问 Agent；
- 只依赖前端隐藏按钮的权限控制。
