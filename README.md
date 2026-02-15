# kview

A Kubernetes TUI (Terminal User Interface) that provides intelligent cluster management with problem detection, resource relationship graphs, and workflow automation.

## Features

- **Auto Problem Detection**: Automatically surfaces issues with root cause analysis
- **Resource Relationships**: Visual dependency graphs showing how resources connect
- **History & Timeline**: Full event timeline with diff capability
- **Multi-Cluster Support**: Side-by-side comparison of multiple clusters
- **Command Palette**: Fast fuzzy-search command access (Ctrl+P)
- **Workflow Engine**: Saved workflows/runbooks for common operations

## Installation

### From Source

```bash
# Clone the repository
git clone https://github.com/bijaya/kview.git
cd kview

# Install dependencies
go mod download

# Build
make build

# Run
./bin/kview
```

### Requirements

- Go 1.22 or later
- A valid kubeconfig file

## Usage

### Basic Navigation

| Key | Action |
|-----|--------|
| `↑/↓` or `j/k` | Navigate up/down |
| `Tab` | Switch between views |
| `1/2/3` | Switch to Pods/Deployments/Services |
| `Enter` | Select resource |
| `/` | Filter current view |
| `Ctrl+P` | Open command palette |
| `?` | Show help |
| `q` | Quit |

### Views

- **Pods (1)**: List and manage pods
- **Deployments (2)**: List and manage deployments
- **Services (3)**: List and manage services
- **Logs (L)**: View pod logs
- **Describe (d)**: Show resource details
- **Diagnosis (D)**: View detected problems
- **Graph (g)**: Resource relationship graph
- **Timeline (t)**: Event timeline

### Actions

| Key | Action |
|-----|--------|
| `r` | Restart deployment |
| `s` | Scale deployment |
| `Ctrl+X` | Delete resource |
| `L` | View logs |
| `d` | Describe resource |
| `n` | Change namespace |
| `c` | Change context |
| `Ctrl+R` | Refresh |

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

## Workflows

Create workflow files in `~/.kview/workflows/`:

```yaml
name: restart-deployment
description: Safely restart a deployment
steps:
  - name: Scale down
    action: scale
    confirm: true
    args:
      replicas: "0"

  - name: Wait
    action: wait
    args:
      duration: "10s"

  - name: Scale up
    action: scale
    args:
      replicas: "1"
```

## Architecture

```
kview/
├── cmd/kview/          # Entry point
├── internal/
│   ├── k8s/            # Kubernetes client layer
│   ├── store/          # SQLite persistence
│   ├── cache/          # In-memory caching
│   ├── analyzer/       # Problem detection
│   ├── graph/          # Resource relationships
│   ├── workflow/       # Workflow engine
│   └── ui/             # TUI layer (Bubble Tea)
└── pkg/config/         # Configuration
```

## Development

```bash
# Run tests
make test

# Run linter
make lint

# Format code
make fmt

# Build for all platforms
make build-all
```

## License

MIT
