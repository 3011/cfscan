# Rhea 设计系统

Cloudflare IP Scanner 的管理端以 **shadcn/ui Rhea** 为唯一设计风格，以 **Base UI** 为默认 primitive，以 **Tailwind CSS 4** 和 **OKLCH** 设计令牌作为基础。业务代码只依赖仓库内的 `web/src/components/ui` 与 `web/src/components/shared`，不得直接依赖 Base UI 或其他 primitive 库。

## 当前基线

| 项目 | 基线 |
|---|---|
| shadcn CLI / Registry | 4.13.x |
| Style | `base-rhea` |
| Primitive | Base UI 1.6.x |
| React | 19.2.x |
| Tailwind CSS | 4.x，Vite 插件模式 |
| TypeScript | 6.x |
| Vite | 8.x |
| 颜色 | OKLCH CSS Variables |
| 字体 | Inter Variable |
| 图标 | Lucide，默认 `1.75` 描边 |

`web/components.json` 是组件生成配置的唯一来源。项目不再保留 `tailwind.config.js` 和 `postcss.config.js`。

## 视觉方向

Rhea 用于高信息密度的管理平台：控件紧凑、圆角柔和、表面层次轻量。页面不通过大量边框划分区域，而主要使用背景、ring、阴影、留白和排版建立层级。

全局外壳遵循：

- 桌面端使用 15.5rem 的可折叠 inset Sidebar；
- Sidebar 中的账号入口固定在底部，Header 只承载当前页面、运行状态和主题；
- Sidebar 展开时使用分组标题，折叠时隐藏分组语义并形成连续图标轨道；折叠图标中心距统一为 34px，品牌与首项保留独立间隔；Sidebar Rail 使用宽点击区和居中短提示线；
- Header 高度统一为 48px，Sidebar Trigger 后不再附加装饰性竖向分隔线；
- 主内容最大宽度为 1600px，桌面保留紧凑内容边距；
- 移动端使用同一 Sidebar 内容的 Drawer，不另建一套导航；
- 页面标题、描述和主操作统一由 `PageHeader` 承载；
- 数据表格使用服务端分页表面，移动端使用同语义的紧凑卡片。

## 组件边界

业务和 shared 目录只能这样导入：

```tsx
import { Button } from "@/components/ui/button"
import { Dialog } from "@/components/ui/dialog"
import { SearchableCombobox } from "@/components/shared/searchable-combobox"
```

禁止这样导入：

```tsx
import { Dialog } from "@base-ui/react"
import * as DialogPrimitive from "@radix-ui/react-dialog"
```

运行以下命令验证边界：

```bash
cd web
pnpm check:ui-boundaries
```

该检查同时确保：

- 业务层没有直接 primitive 依赖；
- 业务和 shared 层不直接使用原生 `button`、`select` 或 `textarea`，交互统一经过项目 UI 组件；
- 仓库没有 Radix 依赖或导入；
- Style 保持 `base-rhea`；
- Tailwind 4 的旧配置文件没有重新出现。

## 组件来源与本地适配

`components/ui` 的 29 个文件中有 28 个来自官方 Rhea Registry，只有 `form.tsx` 是项目维护的 React Hook Form 兼容层。以下适配必须在后续升级时保留：

| 组件/模块 | 本地适配 | 原因 |
|---|---|---|
| `form.tsx` | 保留 React Hook Form API，并使用 Base UI `mergeProps` 合并控件属性 | 业务表单无需感知 primitive 迁移 |
| `select.tsx` | 忽略 Base UI 可清空产生的 `null`；业务 Select 必须通过官方 `items` 映射显示标签 | 防止 `true`、`24h` 等内部值泄漏到界面 |
| `dropdown-menu.tsx` | Label 保持可直接放在 Content 中，不要求业务额外包裹 Group | 兼容既有 shadcn 菜单 API，并避免 Base UI Group Context 错误 |
| `combobox.tsx` / `searchable-combobox.tsx` | Popup 自动挂载到最近的 Sheet/Dialog；搜索输入不重复显示触发图标；业务 Trigger 与 Select 使用同一 `bg-input/50` 表面并左对齐文字 | 保证模态隔离、点击、键盘操作及同级筛选控件视觉一致 |
| Combobox Empty | 非空列表时 live region 高度归零但不卸载 | 兼顾可访问性并消除空白区域 |
| `sheet.tsx` 与业务 Sheet | 固定 Header、滚动 Content、固定不透明 Footer | 避免底部内容透出和移动端安全区问题 |
| `use-mobile.ts` | 使用 `useSyncExternalStore` | 符合 React 19 与严格 Hook 规则 |
| `sidebar.tsx` | 折叠分组标题禁用 pointer events；Tooltip 使用 350ms 主动悬停延迟，并只在折叠桌面状态挂载内容 | 避免透明分组标题遮挡导航，并防止折叠瞬间批量出现提示 |
| `index.css` | 保留成功、警告、失败及图表语义色；Lucide 描边 1.75 | 数据产品需要稳定的状态辨识度 |
| `server-clock` / `LiveRelativeTime` | 相对时间使用服务端校准时钟 | 避免客户端时钟偏差导致一直显示 0 秒前 |
| Data Table | 服务端分页、排序、准确总数、移动卡片、首尾列 16px 内容间距、32px 工具栏、官方 Checkbox Menu 列选择及常驻重置按钮 | 保持官方 Table primitive 不变，同时适配大圆角数据表面和稳定工具栏布局 |

## 页面层级

- **Sidebar / Header**：全局导航和会话信息；Sidebar Rail 保留官方宽点击热区，但视觉提示必须是居中的短竖线，不得使用贯穿整个页面的全高分隔线；
- **Settings Tabs**：标题下方的单层官方横向 Tabs，窄屏可横向滚动，不再嵌套导航或占用第二列；
- **PageHeader**：页面标题、描述与主操作；
- **Card / Table surface**：业务内容；
- **Muted surface**：说明、摘要和弱分组；
- **Dialog / Sheet**：短流程使用 Dialog，长表单和编辑流程使用 Sheet；
- **页面内部导航**：同一页面的设置分类使用顶部单层 Tabs；不得在全局 Sidebar 之外再增加占据内容宽度的二级侧栏；
- **Toast / AlertDialog**：结果反馈与不可逆操作确认。

不得在业务页面临时创造新的按钮高度、圆角体系、图标描边或独立 Sidebar 规格。新增页面先复用上述层级。

## 文档与代码同步

设计系统改动必须同步：

- `ui-component-guidelines.md`：交互和视觉规则；
- `ui-component-inventory.md`：来源、数量和本地适配；
- `CHANGELOG.md`：用户可见变化；
- `web/scripts/smoke-ui.mjs`：可重复浏览器回归；
- `web/scripts/check-ui-boundaries.mjs`：可自动检查的架构边界。

## 官方组件升级流程

不要在产品目录直接执行全量覆盖。正确流程：

1. 从干净工作区生成当前官方 `base-rhea` 参考项目；
2. 将需要更新的组件生成到临时目录；
3. 对比官方实现与当前本地适配；
4. 合并官方修复，同时保留上表中的行为；
5. 运行 `pnpm check`；
6. 完成桌面、移动端、浅色、深色、键盘和弹层浏览器验收；
7. 删除临时参考目录后再提交。

禁止使用 `shadcn add --all --overwrite` 或类似命令直接覆盖产品组件。

## 完成标准

设计系统变更只有同时满足以下条件才可发布：

- UI 边界检查通过；
- ESLint、TypeScript、Vitest 和生产构建通过；
- 业务代码没有直接 primitive 导入；
- 桌面和移动端没有横向溢出；
- 浅色、深色和跟随系统正常；
- Combobox、Select、Dropdown、Tooltip、Dialog、Sheet 和 AlertDialog 键盘可操作；
- 管理员和查看者的可见操作与后端权限一致；
- 浏览器没有运行时错误；
- 本地适配清单与代码保持同步。

## 相关文档

- 文档入口：[`README.md`](README.md)；
- 组件与交互规范：[`ui-component-guidelines.md`](ui-component-guidelines.md)；
- 组件清单：[`ui-component-inventory.md`](ui-component-inventory.md)；
- 开发与浏览器回归：[`development.md`](development.md)。
