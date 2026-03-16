# Dockit

Dockit is a tool that aggregates Markdown documentation from multiple Git repositories into a single browsable web interface with tree navigation.

## Features

- **Multi-repo sync** — Clone/pull multiple Git repositories concurrently via SSH or HTTPS
- **Regex file matching** — Extract files matching configurable patterns (default: `.*\.md$`)
- **Asset extraction** — Automatically detects and copies images/assets referenced in Markdown files
- **Tree navigation** — Organize repos into groups via `path` field, rendered as a collapsible sidebar tree
- **Markdown rendering** — GFM support, Mermaid diagrams (server-side), CJK-friendly heading anchors
- **SPA navigation** — Client-side routing with browser history sync
- **API** — Trigger re-sync and query status via REST endpoints

## Quick Start

```bash
# Build
go build -o dockit ./cmd/dockit/

# Run
./dockit -config server_config.yaml -repos repos.yaml

# Sync only (no web server)
./dockit -config server_config.yaml -repos repos.yaml -sync
```

## Configuration

### server_config.yaml

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

### repos.yaml

```yaml
repos:
  - path: Backend/user-service
    url: git@github.com:myorg/user-service.git
    start_path: docs

  - path: Backend/order-service
    url: git@github.com:myorg/order-service.git

  - path: Frontend/web-app
    url: git@github.com:myorg/web-app.git

  - path: Infra/deploy-scripts
    url: https://github.com/myorg/deploy-scripts.git
    branch: main
    auth:
      type: http
      token: your-token-here
```

The `path` field controls tree navigation structure. For example, `Backend/user-service` creates a "Backend" group containing a "user-service" repo node.

### Field Inheritance

Repos inherit `branch`, `patterns`, and `auth` from `defaults` in server_config.yaml. Repo-level fields override defaults.

## API

| Method | Path | Description |
|--------|------|-------------|
| GET | `/` | Web UI with tree navigation |
| GET | `/api/tree` | Navigation tree JSON |
| GET | `/api/status` | Sync status |
| POST | `/api/sync` | Trigger re-sync |
| GET | `/{repo}/{path}` | View file (Markdown rendered as HTML) |
| GET | `/{repo}/{path}?raw=1` | Raw file download |

## Auth

| Type | Fields |
|------|--------|
| SSH | `type: ssh`, `ssh_key_path`, `ssh_key_passphrase` (optional) |
| HTTP | `type: http`, `token`, `username` (optional, defaults to "git") |
| None | Omit `auth` block — works for public HTTPS repos |

## Dependencies

- [go-git](https://github.com/go-git/go-git) — Pure Go git implementation
- [goldmark](https://github.com/yuin/goldmark) — Markdown rendering with GFM
- [goldmark-mermaid](https://github.com/abhinav/goldmark-mermaid) — Mermaid diagram support (requires `mmdc` CLI for server-side rendering)
