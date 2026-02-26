# kview

A powerful terminal-based Kubernetes cluster viewer inspired by [k9s](https://k9scli.io/), built with Go and the [Bubble Tea](https://github.com/charmbracelet/bubbletea) TUI framework.

## Features

- **18 Resource Views** — Pods, Deployments, Services, ConfigMaps, Secrets, Ingresses, PVCs, StatefulSets, Nodes, Events, ReplicaSets, DaemonSets, Jobs, CronJobs, HPAs, PVs, RoleBindings, and Helm Releases
- **Generic/CRD Support** — Browse any API resource discovered via the Kubernetes API
- **Xray View** — k9s-style interactive resource relationship tree with expandable nodes, RS skipping, container env refs, volume mounts, and emoji icons
- **Helm Integration** — Release listing, revision history, values/manifest viewer (reads Helm Secrets directly, no Helm SDK)
- **Auto Diagnostics** — 8 analyzer rules (CrashLoopBackOff, ImagePull errors, OOMKilled, Pending pods, missing resource limits, image tags, health probes, security policy)
- **Log Viewer** — Streaming logs with regex search, export, timestamps, previous container, time-range filter, wrap toggle, and auto-scroll
- **Port Forwarding** — 3-field overlay for pods and services, dedicated management view, input validation
- **Vim-Style Commands** — `:pods`, `:deploy`, `:ns default`, `:xray deploy`, `:scale nginx 3`, and more
- **Command Palette** — Fuzzy-search command access (`Ctrl+P`)
- **Live Search & Filter** — `/` for search in detail views and table filtering, with inverse (`!`), fuzzy (`-f`), and label selector (`-l`) modes
- **Column Sorting** — `S`/`[`/`]` keys, numeric-aware, persists across refresh
- **Secret Decode** — Base64 decode with `x` key in Secrets view
- **Edit Resources** — `e` key opens resource in `$EDITOR`, diffs and applies changes
- **Event Timeline** — Chronological event view with correlation
- **Health & Pulse Dashboards** — Cluster health overview and k9s-style pulse dashboard
- **Multi-Cluster Support** — Side-by-side cluster comparison
- **Workflow Engine** — YAML-based runbook automation (13 action types)
- **SQLite Persistence** — Event history and snapshots
- **Flicker-Free Rendering** — DEC synchronized output and SGR reset interception for clean visuals

## Installation

### From Source

```bash
git clone https://github.com/bijaya/kview.git
cd kview

# Build
make build

# Run
./bin/kview
```

Or build and run directly with Go:

```bash
go build ./cmd/kview && ./kview
```

### Requirements

- Go 1.22 or later
- A valid kubeconfig file (`~/.kube/config` or `$KUBECONFIG`)
- CGO enabled (required for SQLite — `CGO_ENABLED=1`)

## UI Layout

```
╭─ kview ──────────────────────────────────────────────────────── v0.1.0 ─╮
│  Context: minikube       ↑↓:nav  enter:select        d:describe y:yaml │
│  Cluster: kubernetes     /:filter  ?:help             l:logs  s:shell  │
│  User: minikube-user     ctrl+r:refresh  q:quit       F:pf ctrl+d:del  │
│  K8s Rev: v1.28.0        ctrl+p:palette                                │
│  CPU: 250m / 4.0                                                       │
│  MEM: 512Mi / 8Gi                                                      │
│  ► Workloads  Network  Config  Cluster  Helm  [1]Pods [2]Deploy ...    │
│─────────────────────────────────────────────────────────────────────────│
│  NAME              READY   STATUS    RESTARTS   AGE                     │
│  nginx-pod         1/1     Running   0          5d                      │
│  redis-master      1/1     Running   0          3d                      │
╰─────────────────────────────────────────────────────────────────────────╯
```

- **Header** (7 lines) — Cluster info, navigation shortcuts, view-specific actions, category/resource tabs
- **Body** (remaining) — Resource table or detail view
- **Command Line** (conditional) — Appears on `:` for vim-style commands

## Key Bindings

### Navigation

| Key | Action |
|-----|--------|
| `↑`/`↓` or `j`/`k` | Navigate up/down |
| `PgUp`/`PgDn` | Page up/down |
| `Home`/`g` / `End`/`G` | Jump to top/bottom |
| `←`/`→` | Horizontal scroll (table columns) |
| `Tab`/`Shift+Tab` | Next/previous resource view |
| `1`–`9` | Select resource by number in current category |
| `Enter` | Select / expand (xray) |
| `Escape` | Go back / clear filter |

### Actions

| Key | Action |
|-----|--------|
| `d` | Describe selected resource |
| `y` | Show YAML view |
| `l` | View pod logs |
| `s` | Shell into container (pods) / Scale (deployments) |
| `e` | Edit resource in `$EDITOR` |
| `r` | Restart deployment |
| `x` | Decode secret |
| `X` | Xray view (resource relationships) |
| `F` | Port forward |
| `c` | Copy resource name / content |
| `Ctrl+D` | Delete resource (with confirmation) |
| `Ctrl+R` | Refresh current view |

### Search & Filter

| Key | Action |
|-----|--------|
| `/` | Filter (table views) or search (detail views) |
| `/!term` | Inverse filter (exclude matches) |
| `/-f term` | Fuzzy filter |
| `/-l key=val` | Label selector filter |
| `n`/`N` | Next/previous search match (detail views) |

### Sorting

| Key | Action |
|-----|--------|
| `S` | Toggle sort direction (unsorted → asc → desc) |
| `[` / `]` | Previous/next sort column |

### UI

| Key | Action |
|-----|--------|
| `Ctrl+P` | Command palette |
| `Ctrl+K` | Switch context |
| `?` | Show help |
| `q` | Quit |

## Resource Categories

| Category | Resources |
|----------|-----------|
| **Workloads** | Pods, Deployments, ReplicaSets, DaemonSets, StatefulSets, Jobs, CronJobs, HPAs |
| **Network** | Services, Ingresses |
| **Config** | ConfigMaps, Secrets, PVCs |
| **Cluster** | Nodes, Events, PVs, RoleBindings |
| **Helm** | Releases |

## Vim-Style Commands

Open the command line with `:` (Shift+colon):

```
:pods, :deploy, :svc, :cm        Switch to resource view
:sec, :ing, :pvc, :sts           More resource views
:nodes, :events, :rs, :ds        Cluster resource views
:jobs, :cj, :helm                Batch and Helm views
:ns <namespace>                  Switch namespace
:ctx <context>                   Switch context
:describe, :yaml, :logs          View actions on selected resource
:shell, :sh, :exec               Shell into pod container
:edit                            Edit resource in $EDITOR
:delete                          Delete selected resource
:scale <name> <replicas>         Scale a deployment
:xray <kind>                     Xray tree for resource type
:xray <name>                     Xray relationships for specific resource
:graph                           Alias for :xray
:pf                              Show port forwards view
:pf-stop <id|all>                Stop port forward session(s)
:timeline                        Event timeline
:diagnosis                       Diagnostics view
:health                          Health dashboard
:pulse                           Pulse dashboard
:api-resources                   Browse API resources (CRDs)
:themes                          Display available themes
:refresh                         Refresh current view
:help                            Show help
:q                               Quit
```

## View-Specific Shortcuts

| View | Actions |
|------|---------|
| Pods | `d` describe, `y` yaml, `l` logs, `s` shell, `F` port-forward, `Ctrl+D` delete |
| Deployments | `d` describe, `y` yaml, `r` restart, `s` scale, `Ctrl+D` delete |
| Services | `d` describe, `y` yaml, `F` port-forward, `Ctrl+D` delete |
| Secrets | `d` describe, `y` yaml, `x` decode, `Ctrl+D` delete |
| Helm Releases | `Enter` history, `d` describe, `v` values, `m` manifest, `y` yaml, `Ctrl+D` delete |
| Xray | `Enter` expand/collapse, `d` describe, `y` yaml, `l` logs, `Ctrl+D` delete |
| Detail Views | `/` search, `n`/`N` next/prev match, `Escape` clear |

## Architecture

```
kview/
├── cmd/kview/              # Entry point + syncWriter (flicker-free rendering)
├── internal/
│   ├── k8s/                # Kubernetes client (typed + dynamic), informers, watchers
│   │   ├── client.go       # Client interface, K8sClient struct
│   │   ├── resources.go    # 20+ typed resource info structs
│   │   ├── informer.go     # Background cache with watch + 2s polling
│   │   ├── portforward.go  # Port forward session management
│   │   ├── exec.go         # Shell exec into containers
│   │   ├── edit.go         # Edit resource in $EDITOR
│   │   └── ...             # Metrics, Helm, context, discovery, helpers
│   ├── ui/
│   │   ├── app*.go         # App core (init, update, view, commands, helpers)
│   │   ├── theme/          # Colors, styles, key bindings (50+ pre-computed styles)
│   │   ├── views/          # 46 view files (18 resource lists + specialized views)
│   │   ├── components/     # 21 reusable components (table, header, picker, toast, etc.)
│   │   └── commands/       # Command palette & registry
│   ├── analyzer/           # 8 diagnostic rules with event correlation
│   ├── graph/              # Resource dependency graph (nodes, edges, queries)
│   ├── store/              # SQLite persistence layer
│   ├── cache/              # In-memory caching with index support
│   └── workflow/           # YAML-based workflow/runbook engine
└── pkg/config/             # Configuration types and loading
```

## Configuration

Configuration is stored in `~/.kview/config.yaml`:

```yaml
ui:
  theme: default
  showAllNamespaces: true

defaultNamespace: ""
refreshInterval: 30
databasePath: ~/.kview/data.db
```

## Key Dependencies

| Package | Version | Purpose |
|---------|---------|---------|
| [bubbletea](https://github.com/charmbracelet/bubbletea) | v1.1.0 | TUI framework |
| [bubbles](https://github.com/charmbracelet/bubbles) | v0.19.0 | TUI components |
| [lipgloss](https://github.com/charmbracelet/lipgloss) | v0.13.0 | Terminal styling |
| [chroma](https://github.com/alecthomas/chroma) | v2.23.1 | Syntax highlighting |
| [client-go](https://github.com/kubernetes/client-go) | v0.31.0 | Kubernetes client |
| [go-sqlite3](https://github.com/mattn/go-sqlite3) | v1.14.22 | SQLite persistence |

## Development

```bash
make build          # Build the binary
make run            # Run the application
make test           # Run tests
make test-coverage  # Run tests with coverage report
make lint           # Run golangci-lint
make fmt            # Format code
make deps           # Download and tidy dependencies
make build-all      # Cross-compile for linux, darwin, windows
make install        # Install binary to $GOPATH/bin
make dev            # Hot reload with air
make clean          # Clean build artifacts
```

## License

MIT
