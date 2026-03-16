# Doc 3: Web 服务与 API 文档

## 概述

Dockit 内置 HTTP 服务器，提供：
- 树形导航页面（左侧导航 + 右侧内容）
- Markdown 实时渲染为 HTML
- 静态资产文件直接返回
- API 接口触发同步和查询状态

---

## URL 路由

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/` | 首页，左侧树形导航 + 右侧欢迎页 |
| GET | `/{repo}/{path...}` | 文件访问，.md 渲染为 HTML，其他原样返回 |
| GET | `/{repo}/{path...}?raw=1` | 强制返回原始文件（不渲染） |
| GET | `/api/tree` | 导航树 JSON |
| POST | `/api/sync` | 触发重新同步 |
| GET | `/api/status` | 同步状态 JSON |

### URL 与文件路径映射

`output_dir` 整体作为静态文件根目录，URL 路径直接对应文件路径：

```
请求：GET /api-service/docs/guide.md
文件：output_dir/api-service/docs/guide.md
```

这使得 Markdown 中的相对路径引用在浏览器中自然生效：

```markdown
<!-- docs/guide.md 中的引用 -->
![架构图](./images/arch.png)
```

浏览器访问 `/api-service/docs/guide.md` 时，将 `./images/arch.png` 解析为 `/api-service/docs/images/arch.png`，正好命中 `output_dir/api-service/docs/images/arch.png`。

---

## 文件访问行为

### Markdown 文件（`.md`）

- 默认渲染为 HTML，嵌入页面布局模板（含导航栏）
- 使用 goldmark 库渲染，支持：
  - GFM（GitHub Flavored Markdown）：表格、任务列表、删除线
  - 代码块语法高亮
  - 自动链接
- 加 `?raw=1` 参数时返回原始 Markdown 内容，Content-Type 为 `text/plain; charset=utf-8`

### 其他文件

- 根据文件扩展名设置 Content-Type（由 Go 标准库 `mime` 自动检测）
- 图片、PDF、字体等二进制文件直接返回
- 支持常见类型：`.png`、`.jpg`、`.svg`、`.gif`、`.pdf`、`.html`、`.css`、`.js`

### 目录访问

- 访问 `/{repo}/` 或 `/{repo}/docs/` 等目录路径时，列出该目录下的文件
- 如果目录下存在 `README.md`，自动渲染为页面内容
- 否则显示文件列表

### 404 处理

- 文件不存在时返回 404 页面

---

## 页面布局

### 整体结构

```
┌──────────────────────────────────────────────────┐
│  Dockit                              [Sync] 按钮 │
├──────────────┬───────────────────────────────────┤
│              │                                   │
│  树形导航     │          内容区域                  │
│              │                                   │
│  ▼ 后端服务   │   guide.md 的渲染内容              │
│    api-svc   │                                   │
│    auth      │   ![架构图](./images/arch.png)     │
│    ▼ 内部工具 │                                   │
│      deploy  │                                   │
│  ▼ 前端      │                                   │
│    web-app   │                                   │
│              │                                   │
├──────────────┴───────────────────────────────────┤
│  状态栏: Last sync: 2025-01-01 12:00:00          │
└──────────────────────────────────────────────────┘
```

### 左侧树形导航

- 根据 `repos.yaml` 中的 `path` 字段构建的树形结构
- Group 节点可折叠/展开
- Repo 节点下展示该仓库同步过来的文件列表
- 点击文件在右侧内容区加载
- 当前选中的文件高亮显示

### 右侧内容区

- 首页显示欢迎信息和仓库概览
- 点击 `.md` 文件显示渲染后的 HTML
- 点击其他文件类型提供下载

### 顶部栏

- 项目名称
- Sync 按钮：点击触发 POST /api/sync
- 同步状态指示（同步中显示 loading）

### 底部状态栏

- 显示上次同步时间
- 同步结果概要（成功/失败仓库数）

---

## 树形导航数据结构

### NavTree

```go
type NavTree struct {
    Name     string    `json:"name"`
    Children []NavTree `json:"children,omitempty"`
    Repos    []NavRepo `json:"repos,omitempty"`
}

type NavRepo struct {
    Name  string   `json:"name"`
    Files []string `json:"files"`  // 相对于 output_dir/<repo-name>/ 的路径
}
```

### 构建逻辑

1. 遍历所有 repo 的 `path` 字段，按 `/` 拆分
2. 逐级查找或创建树节点
3. 将 repo 挂载到对应节点
4. 扫描 `output_dir/<repo-name>/` 填充每个 repo 的实际文件列表

### GET /api/tree 返回示例

```json
{
  "name": "root",
  "children": [
    {
      "name": "后端服务",
      "repos": [
        {
          "name": "api-service",
          "files": ["docs/guide.md", "docs/api.md"]
        },
        {
          "name": "auth",
          "files": ["README.md"]
        }
      ],
      "children": [
        {
          "name": "内部工具",
          "repos": [
            {
              "name": "deploy-scripts",
              "files": ["README.md"]
            }
          ]
        }
      ]
    },
    {
      "name": "前端",
      "repos": [
        {
          "name": "web-app",
          "files": ["README.md"]
        }
      ]
    }
  ]
}
```

注意：`files` 列表只包含文档文件（由 patterns 匹配到的），不包含资产文件。资产文件通过文档中的相对引用访问，不需要在导航树中展示。

---

## API 接口

### POST /api/sync

触发重新同步。

**请求**：无 body。

**响应**：

- 成功触发：`202 Accepted`
  ```json
  { "message": "sync started" }
  ```
- 已在同步中：`409 Conflict`
  ```json
  { "message": "sync already in progress" }
  ```

同步在后台异步执行，通过 GET /api/status 查询进度。

### GET /api/status

查询同步状态。

**响应**：`200 OK`

```json
{
  "running": false,
  "last_sync": "2025-01-01T12:00:00Z",
  "results": [
    { "name": "api-service", "files_count": 5, "error": "" },
    { "name": "auth", "files_count": 1, "error": "" },
    { "name": "deploy-scripts", "files_count": 0, "error": "clone failed: authentication required" }
  ],
  "error": "some repos failed to sync"
}
```

### GET /api/tree

返回导航树 JSON（结构见上方）。

**响应**：`200 OK`，Content-Type: `application/json`

---

## HTML 模板

使用 Go 标准库 `html/template` + `embed` 嵌入模板文件。

### 模板文件

| 文件 | 用途 |
|------|------|
| `layout.html` | 基础布局：顶部栏 + 左侧导航 + 右侧内容区 + 底部状态栏 |
| `index.html` | 目录列表 / 欢迎页内容 |
| `markdown.html` | Markdown 渲染后的 HTML 内容容器 |

### 前端实现

- 纯 HTML/CSS/JS，不使用前端框架
- 树形导航通过 JS 从 `/api/tree` 获取数据，客户端渲染
- 文件点击通过 JS fetch 加载内容，无页面刷新（SPA 风格）
- 也支持直接 URL 访问（如 `/api-service/docs/guide.md`），服务端渲染完整页面
- CSS 内联在模板中，无外部依赖

### Markdown 渲染样式

- 基础排版：标题层级、段落间距、行高
- 代码块：等宽字体、背景色、内边距
- 表格：边框、斑马纹
- 图片：最大宽度 100%，自适应
- 链接：蓝色、hover 下划线
- 引用块：左边框、灰色背景
- 任务列表：复选框样式

---

## Graceful Shutdown

- 监听 SIGINT / SIGTERM 信号
- 收到信号后停止接受新请求
- 等待进行中的请求完成（超时 10 秒）
- 如果有同步任务在运行，等待其完成或取消
- 清理退出

---

## 命令行

```bash
dockit -config server_config.yaml -repos repos.yaml           # 同步 + 启动服务器
dockit -config server_config.yaml -repos repos.yaml -sync     # 仅同步不启服务器
```

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `-config` | string | `server_config.yaml` | 服务器配置文件路径 |
| `-repos` | string | `repos.yaml` | 仓库配置文件路径 |
| `-sync` | bool | `false` | 仅同步模式，不启动 Web 服务器 |

启动流程：
1. 解析命令行参数
2. 加载 server_config.yaml
3. 加载 repos.yaml
4. 合并继承，生成 ResolvedRepo 列表
5. 执行初始同步
6. 若 `-sync` 模式：输出同步结果，退出
7. 否则：启动 HTTP 服务器，监听 graceful shutdown

---

## 对应开发模块

确认此文档后，将开发：

- `internal/server/tree.go` — 导航树构建
- `internal/server/handler.go` — 文件服务 + Markdown 渲染
- `internal/server/api.go` — API 接口
- `internal/server/server.go` — HTTP 服务器 + 路由注册
- `internal/server/templates/layout.html` — 页面布局
- `internal/server/templates/index.html` — 目录/欢迎页
- `internal/server/templates/markdown.html` — Markdown 渲染页
- `cmd/dockit/main.go` — CLI 入口
