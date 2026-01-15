# VinAgents-Go Codebase Guide

A guide for developers coming from JS/TS to understand this Go codebase.

---

## What Is This Project?

VinAgents is an **AI agent execution platform** - it allows you to define, launch, and monitor AI agents that perform multi-step tasks using Claude (Anthropic's API). This is a Go port of a Node.js/TypeScript application.

---

## Project Structure

```
vinagents-go/
├── cmd/                          # Application entry points (like bin/ in Node)
│   ├── api/main.go              # REST API server
│   └── worker/main.go           # Background job processor
├── internal/                     # Private application code
│   ├── config/                  # Configuration loading
│   ├── db/                      # Database connection setup
│   ├── repository/              # Data access layer (like DAOs/repositories in TS)
│   ├── service/                 # Business logic layer
│   ├── handler/                 # HTTP request handlers (like controllers)
│   ├── middleware/              # HTTP middleware (auth, logging, errors)
│   ├── agent/                   # Agent framework & execution engine
│   │   ├── base.go             # Core interfaces
│   │   ├── registry.go         # Agent registration system
│   │   ├── executor.go         # Step orchestration
│   │   └── simple/             # Built-in "simple" agent
│   ├── worker/                  # River job queue handlers
│   └── testutil/                # Testing utilities
├── pkg/                         # Reusable public packages
│   └── jwt/                     # JWT token management
├── go.mod / go.sum              # Dependency management (like package.json)
├── Makefile                     # Build commands (like npm scripts)
└── .env                         # Environment config
```

---

## Key Go Concepts for JS/TS Developers

### 1. `internal/` vs `pkg/`

- **`internal/`**: Code that can ONLY be imported by this module. Go enforces this at compile time. Think of it as truly private code.
- **`pkg/`**: Code that CAN be imported by other modules. Public utilities.

### 2. No Classes, Just Structs + Methods

```go
// Go - struct with methods
type UserService struct {
    repo *UserRepository
}

func (s *UserService) GetUser(id string) (*User, error) {
    return s.repo.FindByID(id)
}
```

```typescript
// TypeScript equivalent
class UserService {
    constructor(private repo: UserRepository) {}

    getUser(id: string): Promise<User> {
        return this.repo.findById(id);
    }
}
```

### 3. Error Handling: No try/catch

```go
// Go - errors are return values
user, err := service.GetUser(id)
if err != nil {
    return nil, err  // propagate error up
}
// use user...
```

```typescript
// TypeScript equivalent
try {
    const user = await service.getUser(id);
} catch (err) {
    throw err;
}
```

### 4. Dependency Injection via Constructor Functions

```go
// Go pattern - "New" functions are constructors
func NewUserService(repo *UserRepository) *UserService {
    return &UserService{repo: repo}
}

// Usage in main.go
repo := repository.NewUserRepository(pool)
service := service.NewUserService(repo)
handler := handler.NewUserHandler(service)
```

### 5. Interfaces Are Implicit

```go
// Go - no "implements" keyword needed
type Agent interface {
    DefineSteps() []Step
    Initialize(ctx context.Context) error
}

// Any struct with these methods automatically implements Agent
type SimpleAgent struct{}

func (a *SimpleAgent) DefineSteps() []Step { ... }
func (a *SimpleAgent) Initialize(ctx context.Context) error { ... }
// SimpleAgent now implements Agent - no declaration needed!
```

### 6. Context for Cancellation & Timeouts

Go uses `context.Context` where you'd use AbortController or Promise cancellation:

```go
func (s *Service) DoWork(ctx context.Context) error {
    select {
    case <-ctx.Done():
        return ctx.Err()  // cancelled or timed out
    default:
        // do work
    }
}
```

---

## Architecture Overview

This codebase uses **layered architecture**:

```
HTTP Request
    ↓
[Middleware: auth, error handling, logging]
    ↓
Handler (HTTP parsing, validation, response formatting)
    ↓
Service (business logic, orchestration)
    ↓
Repository (database queries)
    ↓
Database (PostgreSQL)
```

Each layer has a single responsibility:
- **Handlers** understand HTTP (status codes, headers, JSON parsing)
- **Services** understand business rules
- **Repositories** understand SQL queries

---

## The Agent System

The core innovation is a pluggable agent framework:

### Agent Interface (`internal/agent/base.go`)

```go
type Agent interface {
    DefineSteps() []Step           // What steps does this agent have?
    Initialize(ctx context.Context) error   // Setup before running
    Cleanup(ctx context.Context) error      // Teardown after running
}

type Step struct {
    Name           string
    Execute        func(ctx context.Context, input interface{}, agentCtx *Context) (*StepResult, error)
    TransformInput func(previousOutput interface{}) interface{}  // Optional: transform previous step's output
    ShouldSkip     func(previousOutput interface{}, agentCtx *Context) bool  // Optional: conditional execution
}
```

### Agent Registry (`internal/agent/registry.go`)

Agents self-register using Go's `init()` function (runs automatically on import):

```go
// In simple/simple.go
func init() {
    agent.Register("simple", New)  // Register factory function
}

// Later, to create an agent:
factory := agent.Get("simple")
agentInstance := factory(agentContext)
```

### Executor (`internal/agent/executor.go`)

Orchestrates running an agent:
1. Load agent run from database
2. Get agent factory from registry
3. Call `Initialize()`
4. Execute each step from `DefineSteps()`, passing output forward
5. Update database after each step
6. Call `Cleanup()`

---

## Two Processes: API + Worker

### API Server (`cmd/api/main.go`)

Handles HTTP requests:
- User authentication (register, login, token refresh)
- Agent management (launch runs, check status, list runs, cancel)
- Uses **Gin** framework (like Express.js)

### Worker (`cmd/worker/main.go`)

Processes background jobs:
- Uses **River** job queue (PostgreSQL-based, like BullMQ but for Go)
- Picks up agent run jobs and executes them
- Runs 5 concurrent workers

**Why separate?**
- API stays responsive (doesn't block on long agent runs)
- Workers can be scaled independently
- Failed workers don't crash the API

---

## Database Patterns

### Raw SQL with pgx (No ORM)

Unlike TypeScript where you might use Prisma or TypeORM, this codebase uses raw SQL:

```go
func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*User, error) {
    query := `SELECT user_id, email, password, first_name, last_name, is_active, created_at, updated_at
              FROM users WHERE email = $1`

    var user User
    err := r.pool.QueryRow(ctx, query, email).Scan(
        &user.UserID, &user.Email, &user.Password,
        &user.FirstName, &user.LastName, &user.IsActive,
        &user.CreatedAt, &user.UpdatedAt,
    )
    if err != nil {
        return nil, err
    }
    return &user, nil
}
```

### Pointer Fields for Optional Values

```go
type User struct {
    FirstName *string  // nil = not set, *string = has value
    LastName  *string
}

type UpdateInput struct {
    Status *string  // nil = don't update this field
    Output *string  // non-nil = update to this value
}
```

This pattern allows partial updates - only non-nil fields get updated.

---

## Common Patterns You'll See

### 1. Factory Functions (Constructors)

```go
func NewAuthService(userRepo *UserRepository, jwtManager *jwt.Manager) *AuthService {
    return &AuthService{
        userRepo:   userRepo,
        jwtManager: jwtManager,
    }
}
```

### 2. Method Receivers

```go
// Value receiver - doesn't modify the struct
func (u User) FullName() string {
    return u.FirstName + " " + u.LastName
}

// Pointer receiver - can modify the struct, more efficient for large structs
func (s *AuthService) Login(ctx context.Context, email, password string) (*TokenPair, error) {
    // ...
}
```

### 3. Error Wrapping

```go
if err != nil {
    return fmt.Errorf("failed to create user: %w", err)  // %w wraps the original error
}
```

### 4. Defer for Cleanup

```go
func DoWork() error {
    file, err := os.Open("file.txt")
    if err != nil {
        return err
    }
    defer file.Close()  // Always runs when function returns

    // work with file...
}
```

---

## Key Files to Read First

1. **`cmd/api/main.go`** - See how everything is wired together
2. **`internal/agent/base.go`** - Understand the Agent interface
3. **`internal/agent/simple/simple.go`** - See a concrete agent implementation
4. **`internal/service/agent.go`** - See business logic patterns
5. **`internal/handler/agent.go`** - See HTTP handling patterns
6. **`internal/repository/agent_run.go`** - See database query patterns

---

## Workflow Example: Running an Agent

1. **User calls API**: `POST /api/v1/agents/simple/run` with `{"prompt": "Hello"}`

2. **Handler** (`internal/handler/agent.go`):
   - Extracts authenticated user from context
   - Parses JSON body
   - Calls `AgentService.LaunchRun()`

3. **Service** (`internal/service/agent.go`):
   - Validates agent type exists in registry
   - Creates `AgentRun` record (status: "pending")
   - Enqueues River job with the RunID
   - Returns response with RunID

4. **Worker** (separate process, `cmd/worker/main.go`):
   - River picks up the job
   - Creates `Executor` with RunID
   - Gets agent factory from registry
   - Runs: `Initialize()` → each step → `Cleanup()`
   - Updates database with progress/output

5. **User polls**: `GET /api/v1/agents/runs/{runId}`
   - Returns current status, output, and logs

---

## Quick Reference: Go vs TypeScript

| Concept | TypeScript | Go |
|---------|------------|-----|
| Package manager | npm/yarn/pnpm | go mod |
| Dependencies file | package.json | go.mod |
| Lock file | package-lock.json | go.sum |
| Entry point | index.ts | main.go in cmd/ |
| Class | `class Foo {}` | `type Foo struct {}` |
| Constructor | `constructor()` | `func NewFoo() *Foo` |
| Interface | `interface Foo {}` | `type Foo interface {}` |
| Implements | `class X implements Y` | Implicit (just have the methods) |
| Async/await | `async/await` | goroutines + channels |
| Error handling | try/catch | `if err != nil` |
| Null | `null` / `undefined` | `nil` |
| Optional field | `field?: string` | `field *string` |
| Private field | `private field` | lowercase name |
| Public field | `public field` | Uppercase name |
| Map | `Map<K, V>` or `{}` | `map[K]V` |
| Array | `T[]` | `[]T` |
| Generics | `<T>` | `[T any]` |
| String interpolation | `` `${var}` `` | `fmt.Sprintf("%s", var)` |

---

## Running the Project

```bash
# Install dependencies
go mod download

# Run API server
go run cmd/api/main.go

# Run worker (in another terminal)
go run cmd/worker/main.go

# Run tests
go test ./...

# Build binaries
go build -o bin/api cmd/api/main.go
go build -o bin/worker cmd/worker/main.go
```

---

## Tips for JS/TS Developers

1. **Embrace explicit error handling** - It feels verbose at first, but it makes error flows very clear.

2. **Don't fight the type system** - Go is stricter than TypeScript. If it won't compile, there's usually a good reason.

3. **Use `go fmt`** - Go has one official formatting style. Let the tool handle it.

4. **Read the standard library** - Go's stdlib is excellent. You often don't need external packages.

5. **Pointers aren't scary** - `*Type` means "pointer to Type", `&value` gets a pointer, `*pointer` dereferences it.

6. **Channels are like async queues** - When you see `chan`, think of it as a thread-safe queue for passing data between goroutines.

7. **Context is everywhere** - Always pass `context.Context` as the first parameter. It enables cancellation and timeouts.
