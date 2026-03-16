# Doc 2: Git 同步与文件扫描文档

## 概述

Dockit 的同步流程：克隆/拉取仓库 → 正则匹配文档文件 → 解析 Markdown 提取引用资产 → 复制到输出目录。

---

## Git 克隆策略

### 缓存目录

所有仓库克隆到 `cache_dir/<repo-name>/`，扁平存放：

```
.dockit_cache/
├── api-service/          ← 完整 git 仓库
├── auth/
├── deploy-scripts/
└── web-app/
```

缓存目录在多次同步间复用，避免重复克隆。

### 克隆方式

- **首次同步**：`git clone --depth 1 --single-branch --branch <branch> <url>`
  - Shallow clone（depth=1），只拉取最新一次提交，节省时间和空间
  - Single branch，只拉取配置指定的分支
- **后续同步**：`git fetch --depth 1` + `git checkout origin/<branch>`
  - 在已有缓存上拉取更新
  - 强制 checkout 到远程分支最新状态，丢弃本地变更

### 错误处理

- 克隆/拉取失败时记录错误，跳过该仓库，继续处理其他仓库
- 最终汇总报告哪些仓库同步失败
- 缓存目录损坏时（如 `.git` 目录不完整），删除后重新克隆

---

## 认证

### SSH 认证

```yaml
auth:
  type: ssh
  ssh_key_path: ~/.ssh/id_rsa
  ssh_key_passphrase: ""        # 可选
```

- 使用 go-git 的 `ssh.NewPublicKeysFromFile` 构建认证
- `ssh_key_path` 支持 `~` 展开为用户 home 目录
- 未指定 `ssh_key_path` 时默认使用 `~/.ssh/id_rsa`

### HTTP Token 认证

```yaml
auth:
  type: http
  token: ghp_xxxxxxxxxxxx
  username: git                  # 可选，默认 "git"
```

- 使用 go-git 的 `http.BasicAuth`，username + token 作为密码
- 适用于 GitHub/GitLab 的 Personal Access Token

### 无认证

`auth` 字段为空或不配置时，不携带认证信息。适用于公开仓库的 HTTPS 克隆。

---

## 文件扫描与匹配

### 扫描范围

1. 确定扫描根目录：`cache_dir/<repo-name>/<start_path>`
   - `start_path` 为空时，从仓库根目录开始
2. 遍历扫描根目录下的所有文件（递归）
3. 跳过 `.git` 目录

### 正则匹配

- 对每个文件，取其**相对于仓库根目录**的路径
- 用 `patterns` 中的正则逐个匹配，任一匹配即选中
- 使用 Go 标准库 `regexp`，模式需符合 RE2 语法
- 路径分隔符统一为 `/`（跨平台一致）

示例：仓库根目录结构：
```
docs/
  guide.md
  api.md
  images/
    arch.png
src/
  main.go
README.md
```

配置 `start_path: docs`，`patterns: [".*\\.md$"]`：
- 扫描范围限定在 `docs/` 下
- `docs/guide.md` → 相对路径 `docs/guide.md` → 匹配 `.*\.md$` ✓
- `docs/api.md` → 匹配 ✓
- `docs/images/arch.png` → 不匹配 ✗（由资产提取阶段处理）

### 资产文件提取

对每个匹配到的 Markdown 文件，解析内容提取本地资产引用：

**提取规则**：
- 图片语法：`![alt](path)` → 提取 `path`
- 链接语法：`[text](path)` → 提取 `path`（仅本地路径）
- HTML img 标签：`<img src="path">` → 提取 `path`

**过滤规则**：
- 跳过以 `http://`、`https://`、`//` 开头的外部 URL
- 跳过以 `#` 开头的锚点链接
- 跳过含 `://` 的其他协议链接

**路径解析**：
- 引用路径相对于 Markdown 文件所在目录解析
- 如 `docs/guide.md` 中引用 `./images/arch.png`，实际路径为 `docs/images/arch.png`
- 解析后检查文件是否存在于仓库中，存在则加入复制列表
- 不存在的引用记录警告日志，不中断流程

---

## 文件复制

### 输出目录结构

```
output_dir/<repo-name>/<文件相对于仓库根目录的路径>
```

保留仓库内部完整目录结构，包含 `start_path` 前缀。

### 复制行为

- 每次同步前**清空并重建** `output_dir`，确保产物干净
- 创建必要的中间目录
- 文件内容原封不动复制，保留文件权限
- Markdown 文件**不做任何修改**

### 去重

同一个资产文件可能被多个 Markdown 引用，复制时自动去重（同路径只复制一次）。

---

## 并发控制

### Worker Pool

- 默认 4 个并发 worker
- 每个 worker 独立处理一个仓库的完整流程：clone/pull → scan → extract assets → copy
- 仓库之间无依赖，可安全并行

### 同步状态

同步过程中维护状态信息：

```go
type SyncStatus struct {
    Running  bool         // 是否正在同步
    LastSync *time.Time   // 上次同步完成时间
    Results  []RepoResult // 每个仓库的同步结果
    Error    string       // 总体错误信息
}

type RepoResult struct {
    Name       string // 仓库名
    FilesCount int    // 同步的文件数（文档 + 资产）
    Error      string // 该仓库的错误信息
}
```

- 状态读写通过互斥锁保护，线程安全
- 同一时刻只允许一次同步运行，重复触发返回错误

---

## 日志

使用 `log/slog` 结构化日志：

| 事件 | 级别 | 内容 |
|------|------|------|
| 开始克隆 | INFO | repo name, url, branch |
| 克隆完成 | INFO | repo name |
| 开始更新 | INFO | repo name |
| 更新完成 | INFO | repo name |
| 文件匹配 | INFO | repo name, 匹配文件数 |
| 资产提取 | INFO | repo name, 资产文件数 |
| 同步完成 | INFO | 总仓库数, 成功数, 失败数 |
| 克隆/拉取失败 | ERROR | repo name, error |
| 资产文件不存在 | WARN | repo name, markdown file, missing asset path |

---

## 对应开发模块

确认此文档后，将开发：

- `internal/git/auth.go` — SSH / HTTP 认证构建
- `internal/git/git.go` — CloneOrPull 克隆与拉取
- `internal/sync/scanner.go` — 文件扫描与正则匹配
- `internal/sync/assets.go` — Markdown 资产引用解析与提取
- `internal/sync/copier.go` — 文件复制
- `internal/sync/sync.go` — 并发同步编排器 + 状态跟踪
