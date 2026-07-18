# Agent 配对与身份

CF Scanner Agent 通过一次性配对流程加入 Center。每个 Agent 都拥有独立长期凭据，不存在共享 Token 接入方式。

## 身份模型

```text
一次性 UUID 配对密钥
        ↓ 批准并领取
独立 Agent ID + 独立长期 Token
        ↓
HTTPS Bearer 鉴权
        ↓
心跳 / 领取任务 / 上传结果
```

配对密钥是 UUID v4，例如：

```text
8b8f3bf4-5ea2-49e0-b07e-087f69973223
```

它只在配对 URL、Agent 终端和一次性部署命令中出现。默认 10 分钟有效，只能领取一次；数据库只保存 SHA-256 哈希。

长期 Token 使用 32 字节随机 Secret。Agent 本地生成并保存完整 Token，Center 只保存 Secret 哈希。新 Token 绑定唯一 Agent，Center 会忽略请求体中试图冒充其他 Agent 的 `agent_id`。

## 默认方式：Agent 发起，Web 批准

在 Agent 机器运行：

```bash
cfscan-agent connect --server https://cfscan.example.com
```

Agent 输出配对 URL并等待：

```text
Waiting for administrator approval

Open: https://cfscan.example.com/agents/pair/<one-time-uuid>
Pairing key: <one-time-uuid>
Expires in: 10m0s

Waiting for approval...
```

浏览器未登录时会先进入登录页，登录成功后返回原审批路由。管理员确认设备信息，填写节点名称、地区、大洲和并发数，再点击“批准并连接”。

Agent 领取成功后会：

1. 在本地生成独立长期凭据；
2. 原子写入 `identity.json`；
3. 立即发送心跳；
4. 进入正常任务循环。

默认身份文件：

```text
Linux:   ~/.config/cfscan-agent/identity.json
Docker:  建议 /var/lib/cfscan-agent/identity.json
```

可通过 `CFSCAN_AGENT_IDENTITY_FILE` 或 `--identity-file` 修改。文件和目录权限分别为 `0600`、`0700`。

同一个 `connect` 命令重启时会优先读取现有身份，不会生成新的配对请求。只配对、不进入运行循环时使用：

```bash
cfscan-agent connect --server https://cfscan.example.com --pair-only
```

## 预授权方式：自动化部署

管理台“Agent 节点 → 添加 Agent → 自动化部署”先填写节点属性，再生成一次性 UUID 和部署命令。Agent 使用：

```bash
cfscan-agent join \
  --server https://cfscan.example.com \
  --token 8b8f3bf4-5ea2-49e0-b07e-087f69973223
```

该请求已经由管理员预授权，因此无需第二次 Web 审批。它与默认方式共用相同的领取、长期凭据、心跳和任务协议。

自动化环境优先通过文件或标准输入传递一次性密钥：

```bash
cfscan-agent join \
  --server https://cfscan.example.com \
  --token-file /run/secrets/cfscan-enrollment-token
```

或者：

```bash
printf '%s' "$CFSCAN_ENROLLMENT_TOKEN" |
  cfscan-agent join --server https://cfscan.example.com --token-stdin
```

避免把一次性密钥写入长期 Shell history、CI 日志或 Kubernetes Pod spec。

## Docker

默认配对：

```bash
docker run -d \
  --name cfscan-agent \
  --restart unless-stopped \
  -e CFSCAN_AGENT_IDENTITY_FILE=/var/lib/cfscan-agent/identity.json \
  -v cfscan-agent-data:/var/lib/cfscan-agent \
  ghcr.io/3011/cfscan-agent:v2.1.0 \
  connect --server https://cfscan.example.com
```

查看配对 URL：

```bash
docker logs -f cfscan-agent
```

身份卷必须持久化。删除 `cfscan-agent-data` 会丢失长期身份，需要重新配对。

## HTTPS 规则

`connect` 和 `join` 默认要求 HTTPS。以下本地地址允许 HTTP：

```text
http://localhost
http://127.0.0.1
http://[::1]
```

其他明文 HTTP 地址默认拒绝。确实需要在隔离测试网络使用时，必须显式传入：

```bash
--allow-insecure-http
```

生产环境不要使用该参数。

## 配对状态

| 状态 | 含义 |
|---|---|
| `pending` | 等待管理员批准 |
| `approved` | 已批准，等待原 Agent 领取 |
| `claimed` | 已创建 Agent 和独立凭据 |
| `rejected` | 管理员已拒绝 |
| `expired` | 一次性密钥已过期 |
| `revoked` | 一次性配对已撤销 |

配对请求和 Agent 运行状态相互独立。只有成功领取后才会创建正式 Agent，因此预授权但从未部署不会产生离线占位节点。

## 已删除的旧接入方式

从 v2.0.0 开始，Center 不再读取 `CFSCAN_AGENT_TOKEN`，Agent 不再读取共享 Token、地区或大洲注册环境变量，`POST /api/v1/agent/register` 也已删除。

升级前必须先确保每个节点已经拥有独立 `identity.json`。升级后，没有独立身份的旧 Agent 会停止工作，必须重新执行 `connect` 或 `join`。

## API 边界

公开配对接口：

```text
POST /api/v1/agent/enrollments
POST /api/v1/agent/enrollments/claim
```

管理端配对接口要求应用会话，批准、拒绝和预授权要求管理员：

```text
GET  /api/v1/agent-enrollments
GET  /api/v1/agent-enrollments/{pairingToken}
POST /api/v1/agent-enrollments/{pairingToken}/approve
POST /api/v1/agent-enrollments/{pairingToken}/reject
POST /api/v1/agent-enrollments/preauthorized
```

Agent 正常通讯仍使用：

```text
POST /api/v1/agent/heartbeat
POST /api/v1/agent/tasks/claim
POST /api/v1/agent/tasks/results
```
