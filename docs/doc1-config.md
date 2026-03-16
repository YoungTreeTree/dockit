# Doc 1: 配置格式文档

## 配置文件

Dockit 使用两个 YAML 配置文件：

| 文件 | 说明 |
|------|------|
| `server_config.yaml` | 服务器、目录、默认值等全局配置 |
| `repos.yaml` | 仓库列表，扁平结构，通过 `path` 字段组织树形导航 |

通过命令行参数指定：

```bash
dockit -config server_config.yaml -repos repos.yaml
```

---

## server_config.yaml

```yaml
output_dir: ./docs_output
cache_dir: .dockit_cache

server:
  port: 9090
  host: 0.0.0.0

defaults:
  branch: main
  patterns:
    - '.*\.md$'
  auth:
    type: ssh
    ssh_key_path: ~/.ssh/id_rsa
```

### 字段说明

| 字段 | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| `output_dir` | string | 否 | `./docs_output` | 同步后文件的输出目录 |
| `cache_dir` | string | 否 | `.dockit_cache` | Git 仓库的缓存目录 |
| `server.port` | int | 否 | `9090` | 监听端口 (1-65535) |
| `server.host` | string | 否 | `0.0.0.0` | 监听地址 |
| `defaults.branch` | string | 否 | `main` | 默认拉取的分支 |
| `defaults.start_path` | string | 否 | 空（仓库根目录） | 默认扫描起始目录 |
| `defaults.patterns` | []string | 否 | `[".*\\.md$"]` | 默认文档文件匹配正则 |
| `defaults.auth` | object | 否 | `null` | 默认认证配置 |

### auth 字段

| 字段 | 类型 | 说明 |
|------|------|------|
| `type` | string | **必填**，`ssh` 或 `http` |
| `ssh_key_path` | string | SSH 私钥路径，支持 `~` 展开，默认 `~/.ssh/id_rsa` |
| `ssh_key_passphrase` | string | SSH 密钥密码（可选） |
| `token` | string | HTTP Token（用于 `type: http`） |
| `username` | string | HTTP 用户名，默认 `git` |

---

## repos.yaml

**扁平列表**，每个条目描述一个仓库。通过 `path` 字段定义其在导航树中的位置。

```yaml
repos:
  - path: 后端服务
    url: git@github.com:myorg/api-service.git
    branch: develop
    start_path: docs
    patterns:
      - '.*\.md$'

  - path: 后端服务
    url: git@github.com:myorg/auth-service.git
    name: auth
    branch: develop

  - path: 后端服务/内部工具
    url: https://github.com/myorg/deploy-scripts.git
    branch: main
    auth:
      type: http
      token: ghp_xxxxxxxxxxxx

  - path: 前端
    url: git@github.com:myorg/web-app.git
```

### 字段说明

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `path` | string | **是** | 导航树路径，用 `/` 分隔层级，如 `后端服务/内部工具` |
| `url` | string | **是** | 仓库 URL（SSH 或 HTTP） |
| `name` | string | 否 | 仓库别名，默认从 URL 提取（最后一段去 `.git`） |
| `branch` | string | 否 | 覆盖 defaults 中的分支 |
| `start_path` | string | 否 | 仓库内扫描起始目录，覆盖 defaults |
| `patterns` | []string | 否 | 覆盖 defaults 中的文档文件匹配正则 |
| `auth` | object | 否 | 覆盖 defaults 中的认证配置 |

### path 字段规则

- 用 `/` 分隔层级，如 `平台/基础设施/监控`
- 运行时根据所有 repo 的 path 自动构建树形导航
- 同一 path 下的多个 repo 归入同一个导航节点
- path 本身不需要预先声明，按需自动创建

### start_path 字段规则

- 指定仓库内的扫描起始目录，只在该目录下匹配文件
- 为空时从仓库根目录开始扫描
- 复制到 output 时**保留完整相对路径**（包含 start_path 前缀），确保 Markdown 内部引用不断裂

示例：仓库结构为 `src/`、`docs/guide.md`、`docs/images/arch.png`、`README.md`

- `start_path: docs` → 只扫描 `docs/` 下的文件，输出保留 `docs/` 前缀：
  ```
  output_dir/<repo-name>/docs/guide.md
  output_dir/<repo-name>/docs/images/arch.png
  ```
- `start_path` 为空 → 扫描整个仓库

### patterns 与资产文件

`patterns` **仅用于匹配文档文件**（如 `.md`），不需要配置图片等资产类型。

同步时会自动解析匹配到的 Markdown 文件内容，提取其中引用的本地资产文件（图片、附件等），一并复制到输出目录。

处理流程：
1. 用 `patterns` 正则匹配文档文件（如 `.*\.md$`）
2. 解析每个匹配到的 Markdown 文件，提取引用的本地路径：
   - 图片：`![alt](./images/arch.png)`
   - 链接：`[text](./other.pdf)`
3. 将文档文件 + 引用的资产文件一起复制到 output

这样即使 `patterns` 只配了 `.*\.md$`，Markdown 中引用的图片也会自动带过来。

### 继承规则

每个 repo 条目从 `server_config.yaml` 的 `defaults` 继承，repo 级别字段非空则覆盖：

- `branch`：字符串，repo 非空则覆盖
- `start_path`：字符串，repo 非空则覆盖
- `auth`：对象，repo 非 null 则**整体替换**
- `patterns`：数组，repo 非空则**整体替换**

### Repo Name 提取

未指定 `name` 时，从 URL 自动提取：

| URL | 提取结果 |
|-----|----------|
| `git@github.com:myorg/api-service.git` | `api-service` |
| `https://github.com/myorg/web-app.git` | `web-app` |

规则：取 URL 最后一段路径，去掉 `.git` 后缀。

所有 repo name（展平后）必须唯一，重复则报错。

---

## 输出目录结构

同步后 `output_dir` 按 repo name 扁平存放，保留仓库内部完整目录结构：

```
output_dir/
├── api-service/
│   └── docs/
│       ├── guide.md
│       └── images/
│           └── arch.png          ← 从 guide.md 中解析出的引用，自动复制
├── auth/
│   └── README.md
├── deploy-scripts/
│   └── README.md
└── web-app/
    ├── README.md
    └── assets/
        └── logo.svg              ← 从 README.md 中解析出的引用，自动复制
```

Markdown 文件**原封不动复制**，不做任何内容修改。

---

## 树形导航构建

导航树根据 `repos.yaml` 中每个 repo 的 `path` 字段自动构建，上述示例生成：

```
root/
├── 后端服务/
│   ├── [api-service]
│   ├── [auth]
│   └── 内部工具/
│       └── [deploy-scripts]
└── 前端/
    └── [web-app]
```

构建逻辑：
1. 收集所有 repo 的 `path` 字段
2. 按 `/` 拆分，逐级创建树节点
3. 每个 repo 挂载到对应节点下
4. 扫描 `output_dir/<repo-name>/` 获取实际文件列表，挂到 repo 节点下

---

## 配置示例

### 最小配置

**server_config.yaml**:
```yaml
defaults:
  branch: main
```

**repos.yaml**:
```yaml
repos:
  - path: 项目
    url: https://github.com/user/repo.git
```

所有字段使用默认值：output_dir=./docs_output, branch=main, patterns=[".*\\.md$"], port=9090。

### 多层嵌套

**repos.yaml**:
```yaml
repos:
  - path: 平台/基础设施/监控
    url: https://github.com/org/prometheus-config.git
    branch: release

  - path: 平台/基础设施/监控
    url: https://github.com/org/grafana-dashboards.git
    branch: release

  - path: 平台/中间件
    url: https://github.com/org/message-queue.git
```

导航树：
```
平台/
  基础设施/
    监控/
      prometheus-config
      grafana-dashboards
  中间件/
    message-queue
```

### 指定 start_path + 自定义 patterns

**repos.yaml**:
```yaml
repos:
  - path: 后端
    url: git@github.com:myorg/api-service.git
    start_path: docs
    patterns:
      - '.*\.(md|txt)$'
```

只扫描仓库 `docs/` 目录下的 `.md` 和 `.txt` 文件，加上它们引用的资产文件。

---

## 校验规则

1. `repos.yaml` 中 `repos` 不能为空
2. 每个 repo 的 `url` 和 `path` 不能为空
3. `server.port` 在 1-65535 范围内
4. 展平后所有 repo name 不能重复
5. `auth.type` 只能是 `ssh` 或 `http`（或为空表示无认证）

---

## 对应开发模块

确认此文档后，将开发 `config/` 包：

- `config/types.go` — 所有结构体定义
- `config/config.go` — LoadServerConfig() + LoadRepos() 读取 YAML + 校验
- `config/resolve.go` — 从 defaults + repo 条目合并继承 → []ResolvedRepo；从 path 构建导航树
