# Tech Stack — gotham

## Language & Runtime

| Item | Details |
|------|---------|
| Language | Go 1.22 |
| Module | `github.com/TouchBistro/gotham` |
| Organization | TouchBistro |

## Core Dependencies

| Dependency | Purpose |
|-----------|---------|
| `github.com/gin-gonic/gin` | HTTP framework — handlers and middleware for gin-based services |
| `github.com/lestrrat-go/jwx/v2` | JWT parsing, validation, and key management |
| `github.com/redis/go-redis/v9` | Redis client for cache implementation |
| `github.com/sirupsen/logrus` | Structured logging |
| `github.com/spf13/viper` | Configuration management |
| `github.com/pkg/errors` | Error wrapping with stack traces |
| `github.com/TouchBistro/goutils` | Internal TouchBistro Go utilities |
| `github.com/lib/pq` | PostgreSQL driver — provides `pq.Array` for passing Go slices as PostgreSQL array parameters in batch INSERT, UPDATE, and DELETE operations (used by `sql/qb`) |
| `golang.org/x/sync` | Structured concurrency with errgroup for bulk operations |
| `github.com/aws/aws-sdk-go-v2` | AWS SDK v2 core — `aws.String` and shared types (used by `aws/cereport`) |
| `github.com/aws/aws-sdk-go-v2/service/costexplorer` | Cost Explorer API types and `GetCostAndUsage` client (used by `aws/cereport`; the library takes a `CostExplorerAPI` interface, callers build the client — `aws-sdk-go-v2/config` is deliberately **not** a gotham dependency) |

## Package Structure

```
gotham/
├── aws/
│   └── cereport/  # Cost Explorer saved-report library: console URL → Spec → GetCostAndUsage → CSV grid
├── http/          # Auth policy, JWT, principals, roles, middleware, gin & net/http handlers
├── cache/         # Cache interface + memory, Redis, nil implementations + serde helpers
├── circleci/      # CircleCI API client (v1.1 and v2) — project, pipeline, workflow, and insights operations
├── util/          # JWT and general utility helpers
├── sql/
│   └── qb/        # Query builder: automated CRUD SQL generation for PostgreSQL (uses pq.Array for batch ops)
│       └── tmp/   # Temporary holding package for pointer helpers (pending future refactoring)
├── doc.go         # Package-level godoc
├── go.mod / go.sum
└── Makefile
```

## Change Log

- **2026-09-14 (DEVOPS-8987):** first AWS dependency. Added `aws-sdk-go-v2` core and
  `service/costexplorer` for the new `aws/cereport` package, moved from `devops-go-tools`.
  Consumers that do not import `aws/cereport` compile no AWS code (module-graph pruning);
  only their `go.sum` grows.

## Testing

- **Framework**: Standard library `testing` package
- **Test files**: `*_test.go` co-located with source
- **Coverage**: Run via `go test ./...` or Makefile targets

## Build & Tooling

- `Makefile` for common tasks
- `CHANGELOG.md` for release tracking
- GitHub Actions for CI (assumed)
- `coverage/` directory for coverage output artifacts
