# 运维与发布

## Server 环境变量

| 变量 | 默认值 | 说明 |
|---|---|---|
| `CFSCAN_HTTP_ADDR` | `:8080` | Center 监听地址 |
| `CFSCAN_DATABASE_URL` | 本机 PostgreSQL | PostgreSQL DSN；生产必须显式设置 |
| `CFSCAN_AGENT_TOKEN` | `change-me` | Agent API Bearer Token；生产必须替换 |
| `CFSCAN_BOOTSTRAP_ADMIN_USERNAME` | `admin` | 空账号表时创建首个管理员 |
| `CFSCAN_BOOTSTRAP_ADMIN_PASSWORD` | 空 | 空账号表时必填，长度 8–128；已有账号后不覆盖密码 |
| `CFSCAN_SESSION_TTL` | `24h` | 应用会话有效期，Go duration |
| `CFSCAN_COOKIE_SECURE` | `true` | HTTPS 生产应保持 `true`；本地 HTTP 设为 `false` |

当数据库中没有账号且 Bootstrap 用户名或密码为空时，Center 会拒绝启动。

## Agent 环境变量

| 变量 | 默认值 | 说明 |
|---|---|---|
| `CFSCAN_CENTER_URL` | `http://localhost:8080` | Agent API 根地址，不要包含尾部 `/` |
| `CFSCAN_AGENT_TOKEN` | `change-me` | 必须与 Center 一致 |
| `CFSCAN_AGENT_NAME` | `local-agent` | 稳定唯一名称；同名重启更新原记录 |
| `CFSCAN_AGENT_REGION` | `local` | 地区标识 |
| `CFSCAN_AGENT_CONTINENT` | `local` | 大洲标识 |
| `CFSCAN_AGENT_CONCURRENCY` | `64` | 单批探测并发，必须为正整数 |
| `CFSCAN_AGENT_HEARTBEAT_INTERVAL` | `15s` | 心跳周期 |
| `CFSCAN_AGENT_POLL_INTERVAL` | `5s` | 无任务时轮询周期 |

Agent 距数据库当前时间超过 45 秒未心跳时，管理台显示离线。

## 启动和迁移

Center 启动时自动：

1. 连接并 Ping PostgreSQL；
2. 执行嵌入的 `internal/store/postgres/schema.sql`；
3. 在账号表为空时创建 Bootstrap 管理员；
4. 启动自动化调度和 HTTP 服务。

当前没有独立迁移进程，也不会遍历 `migrations/`。生产升级前必须：

- 备份 PostgreSQL；
- 在旧数据库副本上启动新 Center 验证向前迁移；
- 确认旧 Server 是否仍能读取迁移后的 schema，再决定是否允许直接回滚二进制。

示例备份：

```bash
pg_dump --format=custom --file=cfscan-$(date +%F-%H%M).dump "$CFSCAN_DATABASE_URL"
```

## 健康检查

```text
GET /healthz → 200 {"status":"ok"}
```

Center 只有在数据库连接、schema 和首个管理员初始化成功后才开始监听，因此启动阶段失败不会出现健康 Pod。运行中的 `/healthz` 是轻量进程检查，不会在每次请求中重新查询数据库。

目前没有独立 `/readyz`。如果未来需要区分存活和数据库可用性，应新增独立 readiness，而不是把高成本查询加入 `/healthz`。

## 日志与排障

Center 和 Agent 使用 JSON 结构化日志写到 stdout。重点字段包括：

- Center：监听地址、调度执行数量、同步失败、数据库和认证启动错误；
- Agent：Agent ID、领取任务、批次目标数、成功/失败数、心跳与回传错误。

常见检查顺序：

1. `/healthz` 是否正常；
2. Center 是否能连接 PostgreSQL；
3. Agent Token 和 Center URL 是否一致；
4. Agent 最近心跳是否在 45 秒内；
5. Agent 节点是否具备目标网络和 DNS/HTTPS 出站；
6. IPv6 扫描时节点是否具备 IPv6 路由；
7. 任务是否已停止、租约是否过期或 Agent 是否被限定。

## 安全要求

- 生产必须使用 HTTPS；
- 保持 `CFSCAN_COOKIE_SECURE=true`；
- Bootstrap 密码、Agent Token 和数据库 DSN 只通过 Secret 注入；
- 不在命令参数、Git、日志、截图或测试报告中输出凭据；
- 管理入口与 Agent 入口可以使用不同域名，但都由同一 Center 提供；
- Agent 只需出站访问中心，无需开放入站控制端口；
- 定期备份 PostgreSQL，因为它是唯一事实来源。

## 公开发布版本

公开版本由 Git Tag 与 GitHub Release 共同定义：

```text
v1.0.0
```

Release 工作流会发布：

- Linux amd64 / arm64 的 Center 与 Agent 压缩包；
- SHA-256 校验文件；
- `ghcr.io/3011/cfscan-server:<tag>`；
- `ghcr.io/3011/cfscan-agent:<tag>`；
- `ghcr.io/3011/cfscan-web:<tag>`；
- 多架构镜像 provenance 与 SBOM。

生产环境必须使用不可变版本标签或 digest，不要依赖 `latest`。

## 发布流程

1. 确认工作区只包含目标改动；
2. 更新 `CHANGELOG.md`；
3. 运行完整门禁：

   ```bash
   make check
   ```

4. 合并到受保护的 `main`；
5. 创建并推送签名或受保护的 SemVer Tag；
6. 等待 `Release` Workflow 完成二进制、镜像和 GitHub Release；
7. 核对 Release 附件校验和以及三个 GHCR 多架构镜像；
8. 在部署仓库或环境中更新不可变标签/digest；
9. 执行 API、Agent 和浏览器回归；
10. 确认日志、截图和发布说明中没有凭据或生产数据。

只修改 Web 时，部署层可以只滚动 Web；Agent 协议未变化时不应无意义滚动 Agent。

## 回滚

### Web

把部署中的 Web 镜像恢复到上一已验证 Tag 或 digest。Web 不直接修改数据库，通常可以独立回滚。

### Center

回滚前先确认新版本执行的 schema 与旧二进制兼容。当前 Center 启动时执行幂等 schema，但任何破坏性 schema 变化都必须附带明确迁移和恢复说明。无法确认兼容时，优先发布向前修复版本。

### Agent

Agent 与 Center API 应保持向后兼容。协议变更必须先发布兼容 Center，再分批滚动 Agent，最后移除旧协议兼容。

## Kubernetes 与 GitOps

本仓库不绑定特定 Kubernetes 发行版、GitOps 产品或私有 Registry。使用 Helm、Kustomize、Argo CD 或 Flux 的维护者应：

- 只修改本应用部署路径；
- 使用 Secret 注入凭据，不把 Secret 明文提交到 Git；
- 使用不可变镜像 Tag 或 digest；
- 记录数据库备份、发布 revision、实际镜像和回滚目标；
- 在滚动后验证健康检查、Agent 在线状态和管理端关键流程。

