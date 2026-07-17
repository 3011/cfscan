# UI 组件清单

本清单记录 `web/src/components/ui` 的职责和维护方式。所有组件均使用 Rhea 视觉令牌；“官方衍生”表示主体来自 shadcn/ui Rhea Registry，“项目封装”表示由项目维护但只能使用统一令牌。


## 数量与来源

当前 `components/ui` 共 **29 个文件**：

- **28 个官方 Rhea Registry 衍生组件**；
- **1 个项目维护组件**：`form.tsx`，用于把 React Hook Form 的既有 API 接到 Base UI 控件；
- `components/shared` 共 **14 个业务组合组件**，全部只组合 `components/ui`，不直接依赖 Base UI。

本次通过 shadcn CLI 4.13.0 的 `base-rhea` Registry 逐项核对。Alert、Card、Chart、Label、Skeleton、Sonner 和 Table 均有官方来源；其中 Table 与官方文件保持一致，Card 只修正官方 Registry 中的类名笔误，Sonner 仅将官网内部图标占位器替换为项目统一使用的 Lucide。

业务组合组件无法由 Registry 直接替代，例如服务端分页 DataTable、扫描结果筛选、服务端时钟、权限门和领域状态展示；这些组件必须继续只使用官方 primitive 和 Rhea 令牌。

## Base UI 交互组件

| 组件 | 类型 | 说明 |
|---|---|---|
| Alert Dialog | 官方衍生 | 危险或不可逆操作确认 |
| Button | 官方衍生 | 保留 Base UI `nativeButton` / `render` 语义 |
| Checkbox | 官方衍生 | 布尔多选 |
| Collapsible | 官方衍生 | 高级配置展开 |
| Combobox | 官方衍生 + 本地适配 | 搜索、Portal 与 live region 适配 |
| Dialog | 官方衍生 | 短表单与确认流程 |
| Dropdown Menu | 官方衍生 | 行操作、用户菜单、主题菜单 |
| Progress | 官方衍生 | 扫描任务进度 |
| Radio Group | 官方衍生 | 主题等互斥选项 |
| Select | 官方衍生 + 本地适配 | 固定短枚举，禁止意外 null |
| Sheet | 官方衍生 | 长表单、移动导航 |
| Sidebar | 官方衍生 | Rhea inset App Shell |
| Switch | 官方衍生 | 启停配置 |
| Tabs | 官方衍生 | 同页面视图与设置分区 |
| Tooltip | 官方衍生 | 折叠导航和辅助信息 |
| Form | 项目维护 | React Hook Form 与 Base UI 控件兼容层 |

## 展示与布局组件

| 组件 | 类型 | 说明 |
|---|---|---|
| Alert | 官方衍生 | 错误和警告消息 |
| Badge | 官方衍生 | 状态和紧凑元数据 |
| Breadcrumb | 官方衍生 | Header 当前页面 |
| Card | 官方衍生 + 笔误修正 | Rhea 内容表面和尺寸变体 |
| Chart | 官方衍生 | Recharts 3 图表容器 |
| Input / Textarea / Input Group | 官方衍生 | 统一表单控件 |
| Label | 官方衍生 | 表单标签 |
| Separator | 官方衍生 | 弱层级分隔 |
| Skeleton | 官方衍生 | 页面与表格加载状态 |
| Sonner | 官方衍生 + 图标替换 | Toast 样式和主题 |
| Table | 官方衍生 | TanStack Table 的视觉基础；业务边距由 DataTable 组合层处理 |

## Shared 组件

以下组件不属于 shadcn Registry，但构成项目设计系统的一部分：

- `PageHeader`：标题、描述和主操作；
- `MetricCard`：Dashboard 指标；
- `DataTable`：服务端/本地分页、排序、列控制和移动卡片；
- `DataTableResetButton`：常驻筛选重置按钮，无筛选时禁用；
- `SearchInput`：官方 Input Group 的统一搜索输入；
- `SearchableCombobox`：实体与级联维度选择；
- `EmptyState`、`ErrorState`、`PageSkeleton`：统一页面反馈；
- `LiveRelativeTime`：服务端时钟校准后的实时相对时间；
- `StatusBadge`、`ColoLocationLabel`：领域展示。

新增基础组件或显著修改本地适配时，必须同步更新本文件和 `design-system.md`，并执行：

```bash
cd web
pnpm check:ui-boundaries
pnpm test:ui
```

组件来源变化应同时记录官方 Registry 版本、必要适配原因和对应回归断言。

## 相关文档

- 设计系统和升级流程：[`design-system.md`](design-system.md)；
- 组件选择和交互规范：[`ui-component-guidelines.md`](ui-component-guidelines.md)。
