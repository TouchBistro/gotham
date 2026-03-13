# Go Styleguide

Standards for writing idiomatic Go code following Effective Go and community conventions.

## General Principles

- Run `gofmt` on all code (non-negotiable)
- Follow Effective Go guidelines
- Prefer simplicity over cleverness
- Accept interfaces, return structs

## Naming Conventions

| Element | Convention | Example |
|---------|------------|---------|
| Packages | lowercase, singular | `user`, `http` |
| Exported | MixedCase | `UserService`, `GetUser` |
| Unexported | mixedCase | `userCache`, `parseInput` |
| Interfaces | -er suffix | `Reader`, `Stringer` |
| Acronyms | All caps | `HTTPServer`, `userID` |
| Local variables | Short in small scopes | `i`, `u`, `ctx` |

### Short Variable Names

```go
// Good: Short names in small scopes
for i, v := range items {
    process(v)
}

// Good: Descriptive names in larger scopes
func ProcessUserRegistration(registrationRequest *Request) error {
    ...
}
```

## Package Organization

### Standard Layout

```
myproject/
├── cmd/
│   └── server/
│       └── main.go
├── internal/
│   ├── user/
│   │   ├── service.go
│   │   └── repository.go
│   └── auth/
│       └── middleware.go
├── pkg/
│   └── validation/
│       └── email.go
├── go.mod
└── go.sum
```

### Package Guidelines

```go
// Package user provides user management functionality.
package user

// Keep packages focused on a single responsibility
// Avoid package names like "util", "common", "helpers"
```

## Error Handling

### Explicit Error Checking

```go
// Always check errors
result, err := doSomething()
if err != nil {
    return fmt.Errorf("failed to do something: %w", err)
}
```

### Custom Error Types

```go
type ValidationError struct {
    Field   string
    Message string
}

func (e *ValidationError) Error() string {
    return fmt.Sprintf("validation error on %s: %s", e.Field, e.Message)
}

// Check with errors.As
var valErr *ValidationError
if errors.As(err, &valErr) {
    log.Printf("field %s invalid", valErr.Field)
}
```

### Sentinel Errors

```go
var (
    ErrNotFound     = errors.New("not found")
    ErrUnauthorized = errors.New("unauthorized")
)

// Check with errors.Is
if errors.Is(err, ErrNotFound) {
    return http.StatusNotFound
}
```

### Error Wrapping

```go
// Wrap errors with context
if err != nil {
    return fmt.Errorf("creating user %s: %w", name, err)
}
```

## Concurrency

### Goroutine Patterns

```go
// Use errgroup for concurrent operations
g, ctx := errgroup.WithContext(ctx)

g.Go(func() error {
    return fetchUser(ctx, userID)
})

g.Go(func() error {
    return fetchOrders(ctx, userID)
})

if err := g.Wait(); err != nil {
    return err
}
```

### Channel Usage

```go
// Prefer unbuffered channels for synchronization
done := make(chan struct{})

// Use buffered channels for known workloads
results := make(chan Result, workerCount)

// Always close channels from sender side
close(results)
```

### Context Propagation

```go
func ProcessRequest(ctx context.Context, req *Request) error {
    // Always accept context as first parameter
    // Pass it down to all called functions

    ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
    defer cancel()

    return db.QueryContext(ctx, query)
}
```

## Interfaces

### Accept Interfaces, Return Structs

```go
// Good: Accept interface
func ProcessData(r io.Reader) error {
    ...
}

// Good: Return concrete type
func NewUserService(repo *UserRepository) *UserService {
    return &UserService{repo: repo}
}
```

### Keep Interfaces Small

```go
// Good: Single-method interface
type Validator interface {
    Validate() error
}

// Avoid: Large interfaces
// type UserManager interface {
//     CreateUser(...) error
//     UpdateUser(...) error
//     DeleteUser(...) error
//     GetUser(...) (*User, error)
//     ListUsers(...) ([]*User, error)
// }
```

## Testing Patterns

### Table-Driven Tests

```go
func TestAdd(t *testing.T) {
    tests := []struct {
        name     string
        a, b     int
        expected int
    }{
        {"positive numbers", 2, 3, 5},
        {"negative numbers", -2, -3, -5},
        {"zero", 0, 0, 0},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := Add(tt.a, tt.b)
            if result != tt.expected {
                t.Errorf("Add(%d, %d) = %d; want %d",
                    tt.a, tt.b, result, tt.expected)
            }
        })
    }
}
```

### Test Naming

```go
// Test function names: Test<Function>_<scenario>
func TestUserService_CreateUser_ValidInput(t *testing.T) { ... }
func TestUserService_CreateUser_InvalidEmail(t *testing.T) { ... }
```

### Benchmarks

```go
func BenchmarkProcess(b *testing.B) {
    for i := 0; i < b.N; i++ {
        Process(testData)
    }
}
```

## Documentation

### Package Comments

```go
// Package user provides user management functionality including
// creation, authentication, and profile management.
//
// Basic usage:
//
//     svc := user.NewService(repo)
//     u, err := svc.Create(ctx, req)
package user
```

### Function Comments

```go
// CreateUser creates a new user with the given details.
// It returns ErrDuplicateEmail if the email already exists.
func (s *Service) CreateUser(ctx context.Context, req CreateRequest) (*User, error) {
    ...
}
```

## Quick Reference

| Aspect | Recommendation |
|--------|----------------|
| Formatter | gofmt (required) |
| Linter | golangci-lint |
| Indentation | Tabs |
| Line length | No hard limit, use judgment |
| Error handling | Always explicit |
| Interfaces | Small, accept don't return |
| Tests | Table-driven |
