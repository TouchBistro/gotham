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

## Package Structure

```
gotham/
├── http/          # Auth policy, JWT, principals, roles, middleware, gin & net/http handlers
├── cache/         # Cache interface + memory, Redis, nil implementations + serde helpers
├── util/          # JWT and general utility helpers
├── sql/
│   └── qb/        # Query builder: automated CRUD SQL generation for PostgreSQL (uses pq.Array for batch ops)
│       └── tmp/   # Temporary holding package for pointer helpers (pending future refactoring)
├── doc.go         # Package-level godoc
├── go.mod / go.sum
└── Makefile
```

## Testing

- **Framework**: Standard library `testing` package
- **Test files**: `*_test.go` co-located with source
- **Coverage**: Run via `go test ./...` or Makefile targets

## Build & Tooling

- `Makefile` for common tasks
- `CHANGELOG.md` for release tracking
- GitHub Actions for CI (assumed)
- `coverage/` directory for coverage output artifacts
