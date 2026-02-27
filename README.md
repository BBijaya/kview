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
- **Delta Marking** — Color-coded rows for new (blue), modified (steel blue), and unhealthy (coral) resources with `Ctrl+Z` error zoom
- **21 Built-in Themes** — Dracula, Catppuccin, Tokyo Night, Nord, Gruvbox, and more with full color customization
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
| `PgUp`/`Ctrl+U` | Page up |
| `PgDn` | Page down |
| `Home`/`g` | Jump to top |
| `End`/`G` | Jump to bottom |
| `←`/`→` | Horizontal scroll (table columns) |
| `Ctrl+←`/`Ctrl+→` | Previous/next category |
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
| `D` | Diagnostics view |
| `t` | Event timeline |
| `F` | Port forward |
| `v` | Helm values (Helm views) |
| `m` | Helm manifest (Helm views) |
| `c` | Copy resource name / content |
| `Ctrl+D` | Delete resource (with confirmation) |
| `Ctrl+R` | Refresh current view |
| `Ctrl+A` | Toggle auto-refresh |

### Search & Filter

| Key | Action |
|-----|--------|
| `/` | Filter (table views) or search (detail views) |
| `/!term` | Inverse filter (exclude matches) |
| `/-f term` | Fuzzy filter |
| `/-l key=val` | Label selector filter |
| `n`/`N` | Next/previous search match (detail views) |
| `Ctrl+Z` | Toggle error-only filter (show only unhealthy rows) |

### Sorting

| Key | Action |
|-----|--------|
| `S` | Toggle sort direction (unsorted → asc → desc) |
| `[` / `]` | Previous/next sort column |

### UI

| Key | Action |
|-----|--------|
| `:` | Command mode |
| `Ctrl+P` | Command palette |
| `Ctrl+K` | Switch context |
| `?` | Show help |
| `q` / `Ctrl+C` | Quit |

### Log Viewer

| Key | Action |
|-----|--------|
| `/` | Search logs (regex) |
| `n`/`N` | Next/previous match |
| `Ctrl+S` | Export/save logs |
| `t` | Toggle timestamps |
| `p` | Previous container logs |
| `Ctrl+T` | Time range filter |
| `w` | Toggle line wrap |
| `f` | Toggle auto-scroll (tail) |

### Mouse

kview does not capture mouse input, matching k9s behavior:

- **Text highlighting** — Click and drag to select text natively (no Shift required)
- **Mouse wheel** — Scrolls via the terminal's alternate screen scroll conversion (terminal-dependent)
- **Copy/paste** — Use your terminal's native selection and clipboard

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

### Resource Views

| Command | Aliases | Action |
|---------|---------|--------|
| `:pods` | `:pod`, `:po` | Switch to Pods view |
| `:deployments` | `:deployment`, `:deploy` | Switch to Deployments view |
| `:services` | `:service`, `:svc` | Switch to Services view |
| `:configmaps` | `:configmap`, `:cm` | Switch to ConfigMaps view |
| `:secrets` | `:secret`, `:sec` | Switch to Secrets view |
| `:ingresses` | `:ingress`, `:ing` | Switch to Ingresses view |
| `:pvcs` | `:pvc`, `:persistentvolumeclaim` | Switch to PVCs view |
| `:statefulsets` | `:statefulset`, `:sts` | Switch to StatefulSets view |
| `:nodes` | `:node`, `:no` | Switch to Nodes view |
| `:events` | `:event`, `:ev` | Switch to Events view |
| `:replicasets` | `:replicaset`, `:rs` | Switch to ReplicaSets view |
| `:daemonsets` | `:daemonset`, `:ds` | Switch to DaemonSets view |
| `:jobs` | `:job` | Switch to Jobs view |
| `:cronjobs` | `:cronjob`, `:cj` | Switch to CronJobs view |
| `:hpa` | `:hpas` | Switch to HPAs view |
| `:pv` | `:pvs`, `:persistentvolume` | Switch to PVs view |
| `:rolebindings` | `:rolebinding`, `:rb` | Switch to RoleBindings view |
| `:helm` | `:releases`, `:rel`, `:hr` | Switch to Helm Releases view |

### Navigation & Actions

| Command | Action |
|---------|--------|
| `:ns <namespace>` | Switch namespace (`:ns all` or `:ns -` for all) |
| `:ns` | Open namespace picker |
| `:ctx <context>` | Switch context |
| `:describe` / `:desc` | Describe selected resource |
| `:yaml` | Show YAML view |
| `:logs` / `:log` | View pod logs |
| `:shell` / `:sh` / `:exec` | Shell into container (optional: `:shell <container>`) |
| `:edit` | Edit resource in `$EDITOR` |
| `:delete` / `:del` | Delete selected resource |
| `:scale <name> <replicas>` | Scale a deployment |
| `:refresh` / `:r` | Refresh current view |

### Advanced Views

| Command | Action |
|---------|--------|
| `:xray <kind>` | Xray tree for resource type (e.g. `:xray deploy`) |
| `:xray <name>` | Xray for specific resource (e.g. `:xray nginx`) |
| `:xray <kind>/<name>` | Xray with kind (e.g. `:xray svc/nginx`) |
| `:xray <ns>/<kind>/<name>` | Xray with namespace (e.g. `:xray default/deploy/nginx`) |
| `:graph` | Alias for `:xray` |
| `:timeline` / `:tl` | Event timeline view |
| `:diagnosis` / `:diag` | Diagnostics view |
| `:health` | Cluster health dashboard |
| `:pulse` | Pulse dashboard |
| `:pf` / `:portforwards` | Port forwards management view |
| `:pf-stop <id\|all>` | Stop port forward session(s) |
| `:api-resources` / `:ar` | Browse API resources (CRDs) |
| `:themes` | Display all available themes with color swatches (read-only — set theme in config file) |
| `:help` / `:h` | Show help |
| `:q` / `:quit` | Quit |

### CRD / Generic Resources

Any unknown command is looked up against the cluster's discovered API resources. This means custom resources (CRDs) and built-in types not in the resource views table above are all accessible. For example:

- `:networkpolicies` — NetworkPolicy resources
- `:storageclasses` — StorageClass resources
- `:serviceaccounts` — ServiceAccount resources
- `:certificates` — cert-manager Certificate CRDs
- Any CRD singular/plural name available on the cluster

The generic view provides the same table interface with describe (`d`), YAML (`y`), edit (`e`), xray (`X`), delete (`Ctrl+D`), filter (`/`), sort (`S`/`[`/`]`), and error filter (`Ctrl+Z`). You can also browse all available API resources with `:api-resources`.

## View-Specific Shortcuts

| View | Actions |
|------|---------|
| Pods | `d` describe, `y` yaml, `e` edit, `l` logs, `s` shell, `F` port-forward, `c` copy, `Ctrl+D` delete |
| Deployments | `d` describe, `y` yaml, `e` edit, `r` restart, `s` scale, `c` copy, `Ctrl+D` delete |
| Services | `d` describe, `y` yaml, `e` edit, `F` port-forward, `c` copy, `Ctrl+D` delete |
| ConfigMaps | `d` describe, `y` yaml, `e` edit, `c` copy, `Ctrl+D` delete |
| Secrets | `d` describe, `y` yaml, `e` edit, `x` decode, `c` copy, `Ctrl+D` delete |
| Helm Releases | `Enter` history, `d` describe, `v` values, `m` manifest, `y` yaml, `Ctrl+D` delete |
| Helm History | `Enter` detail, `d` describe, `v` values, `m` manifest, `y` yaml |
| Containers | `d` describe, `l` logs, `s` shell |
| Xray | `Enter` expand/collapse, `d` describe, `y` yaml, `l` logs, `Ctrl+D` delete |
| Port Forwards | `Ctrl+D` stop |
| Detail Views | `/` search, `n`/`N` next/prev match, `Escape` clear search |
| All Table Views | `/` filter, `S`/`[`/`]` sort, `Ctrl+Z` error filter, `e` edit |

## Configuration

Configuration is stored in `~/.kview/config.yaml`:

```yaml
# UI settings
ui:
  # Theme name — see "Themes" section for all 21 options
  theme: default

  # Show all namespaces on startup
  showAllNamespaces: true

  # Optional color overrides — see "Custom Colors" section
  colors: {}

  # Table display options
  table:
    showLineNumbers: false
    compactMode: false

# Default namespace (empty = all namespaces)
defaultNamespace: ""

# Auto-refresh interval in seconds
refreshInterval: 30

# SQLite database path for event history and snapshots
databasePath: ~/.kview/data.db
```

## Themes

kview ships with 21 built-in themes. Set the theme in your config file or view them at runtime with `:themes`.

### Available Themes

| Theme | Description |
|-------|-------------|
| `default` | Dark indigo with purple/cyan accents |
| `dracula` | Popular dark theme with pastel colors |
| `catppuccin` | Soothing pastel tones |
| `tokyo-night` | Modern dark theme inspired by Tokyo lights |
| `nord` | Arctic blue tones |
| `gruvbox` | Retro groove warm colors |
| `solarized` | Precision colors (dark variant) |
| `one-dark` | Atom One Dark inspired |
| `monokai` | Classic syntax highlighting colors |
| `rose-pine` | Rose and pine color harmony |
| `kanagawa` | Japanese wave inspired |
| `everforest` | Green forest tones |
| `palenight` | Pale night dark variant |
| `ayu` | Ayu mirage theme |
| `horizon` | Horizon dark |
| `midnight` | Pure black background |
| `night-owl` | Optimized for contrast |
| `synthwave` | Retro neon |
| `oxocarbon` | IBM Carbon design |
| `github-dark` | GitHub dark mode |
| `github-light` | GitHub light mode |

### Custom Colors

Override individual colors in your config. All fields are optional — unset fields use the theme's defaults:

```yaml
ui:
  theme: dracula
  colors:
    # 12 base colors
    primary: "#BD93F9"
    accent: "#8BE9FD"
    background: "#282A36"
    surface: "#21222C"
    text: "#F8F8F2"
    muted: "#6272A4"
    border: "#44475A"
    highlight: "#BD93F9"
    success: "#50FA7B"
    warning: "#F1FA8C"
    error: "#FF5555"
    info: "#8BE9FD"

    # Derived colors (normally auto-computed from base colors)
    selectionBg: "#44475A"
    selectionFg: "#F8F8F2"
    frameBorder: "#6272A4"
    surfaceAlt: "#34353E"
    searchHighlightBg: "#F1FA8C"
    searchHighlightFg: "#282A36"

    # Delta row marking colors
    deltaAdd: "#87CEEB"       # New resources
    deltaModify: "#B0C4DE"    # Modified resources
    deltaError: "#E08080"     # Unhealthy resources
    deltaDelete: "#708090"    # Deleted resources (future use)
```

### Color Derivation

When derived colors are not explicitly set, they are automatically computed from your base colors:

| Derived Color | Computed From |
|---------------|---------------|
| `selectionBg` | `accent` darkened 60% |
| `selectionFg` | Contrast-based white/black against `selectionBg` |
| `frameBorder` | `border` lightened 30% |
| `surfaceAlt` | Blend of `background` + `surface` |
| `searchHighlightBg` | Same as `warning` |
| `searchHighlightFg` | Luminance-based white/black |
| `deltaAdd` | `info` lightened 30% |
| `deltaModify` | Blend of `highlight` + `muted` |
| `deltaError` | `error` lightened 20% |
| `deltaDelete` | Same as `muted` |

This means switching themes gives you harmonious delta marking, search highlights, and selection colors automatically.

## Delta Marking

During rolling updates or deployments, kview color-codes rows to show what changed since the last refresh:

| State | Color | Meaning |
|-------|-------|---------|
| Add | Sky blue | Resource appeared since last refresh |
| Modify | Steel blue | Resource values changed |
| Error | Coral | Resource is unhealthy (CrashLoopBackOff, Failed, OOMKilled, etc.) |

Press `Ctrl+Z` to toggle **error zoom** — filters the table to show only unhealthy rows. Composes with text/fuzzy/label filters. The header shows `[ERRORS]` when active.

On first load, only error states are highlighted (not everything marked as "new").

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
