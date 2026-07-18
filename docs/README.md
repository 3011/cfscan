# 项目文档

本目录是 CF Scanner 的长期维护入口。README 用于了解产品能力；本目录用于理解架构、开发、运维、设计系统和后续维护边界。

## 推荐阅读顺序

### 新接手项目

1. [`../AGENTS.md`](../AGENTS.md)：改动边界、必跑检查和提交要求；
2. [`architecture.md`](architecture.md)：中心、Agent、数据库和任务生命周期；
3. [`agent-enrollment.md`](agent-enrollment.md)：Agent 配对、独立凭据、CLI 和 Docker；
4. [`development.md`](development.md)：本地启动、目录、测试和调试；
5. [`operations.md`](operations.md)：环境变量、健康检查、发布与回滚。

### 前端与 UI

1. [`design-system.md`](design-system.md)：Rhea、Base UI、Tailwind 4 和本地适配；
2. [`ui-component-guidelines.md`](ui-component-guidelines.md)：组件选择、布局和交互规范；
3. [`ui-component-inventory.md`](ui-component-inventory.md)：官方组件与项目组合组件清单。


### 开源社区

- [`../CONTRIBUTING.md`](../CONTRIBUTING.md)：贡献流程和提交要求；
- [`../SECURITY.md`](../SECURITY.md)：漏洞私下报告；
- [`../RESPONSIBLE_USE.md`](../RESPONSIBLE_USE.md)：主动测量的授权与责任边界；
- [`../SUPPORT.md`](../SUPPORT.md)：社区支持渠道；
- [`../GOVERNANCE.md`](../GOVERNANCE.md)：维护者决策模型。

### 维护与发布

- [`maintenance-audit.md`](maintenance-audit.md)：已完成审计、当前技术债和建议优先级；
- [`../CHANGELOG.md`](../CHANGELOG.md)：用户可见版本变更；
- [`operations.md`](operations.md)：镜像标签、GitOps、验证和回滚流程。

## 文档职责

| 文档 | 负责回答的问题 |
|---|---|
| `README.md` | 这个平台做什么、已经实现什么 |
| `architecture.md` | 系统为什么这样设计、数据如何流转 |
| `agent-enrollment.md` | Agent 如何配对和保存独立身份 |
| `development.md` | 如何在本地开发和验证 |
| `operations.md` | 如何配置、部署、监控和回滚 |
| `design-system.md` | UI 技术基线和官方组件升级方式 |
| `ui-component-guidelines.md` | 新界面应使用什么组件和交互 |
| `ui-component-inventory.md` | 哪些组件来自官方、哪些由项目维护 |
| `maintenance-audit.md` | 还有哪些技术债，应如何排期 |
| `CHANGELOG.md` | 每个发布版本改变了什么 |

## 权威来源

文档不能替代代码。发生冲突时按以下顺序判断，并立即修正文档：

| 内容 | 权威来源 |
|---|---|
| 服务端与 Agent 环境变量 | `internal/config/config.go` |
| HTTP API 路由与权限 | `internal/api/api.go`、`internal/api/auth.go`、`internal/api/users.go` |
| 数据库启动迁移 | `internal/store/postgres/schema.sql` 与 `Store.Migrate` |
| Web 路由和导航 | `web/src/routes/app-router.tsx`、`web/src/components/layout/app-sidebar.tsx` |
| UI Style 与组件生成配置 | `web/components.json` |
| 前端依赖与检查命令 | `web/package.json` |
| 本地容器启动 | `compose.yaml` |
| 公开发布版本 | Git Tag、GitHub Release 与不可变镜像标签 |

`migrations/000001_init.sql` 不是当前运行时迁移入口。中心启动时执行嵌入的 `internal/store/postgres/schema.sql`。

## 文档同步规则

以下改动必须在同一个提交中更新文档：

- 新增或修改环境变量：更新 `.env.example` 和 `operations.md`；
- 新增 API、角色权限或任务状态：更新 `architecture.md`；
- 修改 Agent配对、凭据或 CLI：更新 `agent-enrollment.md`；
- 新增页面、路由或导航：更新 README 和相关 UI 文档；
- 新增 UI primitive 或本地适配：更新 `design-system.md` 与 `ui-component-inventory.md`；
- 改变组件选择规则：更新 `ui-component-guidelines.md`；
- 发布用户可见功能：更新 `CHANGELOG.md`；
- 发现新的技术债或完成审计项：更新 `maintenance-audit.md`。

运行以下命令检查内部链接、必需文档和环境变量覆盖：

```bash
make docs-check
```
