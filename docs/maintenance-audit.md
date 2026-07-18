# 代码维护审计

本文件记录首个公开版本前的全仓维护审计结果，避免后续重复检查或误删必要适配。

## 审计范围

- Go：31 个源码文件，约 5,782 行；
- Web：132 个 TypeScript / TSX / CSS 文件；
- UI primitive：29 个文件，其中 28 个来自 shadcn/ui 4.13.0 `base-rhea` Registry；
- shared 业务组合组件：14 个；
- 桌面与移动端各 8 个核心路由；
- 管理员认证、筛选、分页、列管理、业务 Dialog、移动 Sidebar Sheet、AlertDialog、Tooltip 和 Sidebar。

## 已完成优化

- 所有 8 个表格筛选工具栏使用常驻重置按钮，无筛选时禁用；
- 列管理改为官方 `DropdownMenuCheckboxItem`；
- 外观主题改为官方 Base UI `RadioGroup`；
- Select 与 Combobox 的表面、标签和可访问名称统一；
- DataTable 空状态使用当前可见列数作为 `colSpan`；
- 列定义补齐 `meta.label`，避免展示内部字段名；
- UI 边界检查禁止业务层直接导入 primitive 或使用原生交互标签；
- 永久浏览器回归覆盖上述交互；
- 文档按架构、开发、运维、设计系统、组件和维护职责重新整理；
- 本地 Compose 补齐 Bootstrap 管理员、Cookie 和 Agent 周期配置；
- `make docs-check` 自动验证内部链接、必需文档和环境变量覆盖。

## 官方 Registry 对比

28 个官方来源组件中：

- 12 个与当前 Registry 规范化后完全一致；
- 16 个包含已记录适配，主要是 Lucide 图标替换、Vite 项目无需 `use client`、Select null 防护、Combobox 模态 Portal、Dropdown Label 兼容、长 Dialog 固定布局、Sidebar Rail 与 Tooltip；
- `form.tsx` 是唯一项目维护的 primitive 兼容层。

所有差异均已记录在 [`design-system.md`](design-system.md)，没有发现未说明的自定义 primitive。

## 当前质量门禁

- Go vet：通过；
- Go test：通过；
- Go race：通过；
- 前端 UI boundaries：通过；
- ESLint / TypeScript / Vitest / production build：通过；
- 生产依赖审计：0 个已知漏洞；
- 桌面和移动端浏览器运行时错误：0；
- 横向溢出：0。

## 后续建议单独排期

### 1. 拆分 PostgreSQL Store

`internal/store/postgres/store.go` 约 1,600 行。建议按领域拆分为：

- `agents.go`；
- `scans.go`；
- `results.go`；
- `sources.go`；
- `automation.go`；
- `blacklist.go`。

只移动实现，不改变 Store interface 和 SQL 语义，降低多人修改冲突。

### 2. 拆分 API Handler

`internal/api/api.go` 约 800 行。建议按路由领域拆分，并保留统一 Router 和中间件入口。

### 3. 提升后端测试覆盖

当前重点覆盖率：

- `internal/api`：9.6%；
- `internal/store/postgres`：4.6%；
- `internal/automation`：0%；
- `cmd/agent` / `cmd/server`：0%。

优先补充：停止任务竞态、晚到结果、自动化重入、权限矩阵、结果分页与数据库迁移测试。

### 4. 前端包体积

Dashboard / Recharts 路由块约 354 KB（gzip 约 105 KB），主入口约 350 KB（gzip 约 109 KB）。当前已按路由懒加载，暂不影响功能；后续可以评估图表进一步延迟加载和依赖拆包。

### 5. 可读性重构

部分早期表格组件仍包含密集单行 JSX。建议在功能稳定后单独格式化与拆分移动卡片，避免与行为修改混在同一提交。
