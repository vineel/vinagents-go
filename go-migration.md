# VinAgents Go Rewrite Plan

## Overview

Rewrite the Node.js/TypeScript VinAgents application in Go, using:
- **Gin** for HTTP routing
- **pgx** for PostgreSQL
- **River** for job queue
- **Anthropic Go SDK** for LLM calls
- **golang-jwt/jwt** for authentication
- Standard Go testing + testify

Target directory: `../vinagents-go`

## Project Structure

```
vinagents-go/
├── cmd/
│   ├── api/
│   │   └── main.go           # API server entrypoint
│   └── worker/
│       └── main.go           # Worker process entrypoint
├── internal/
│   ├── config/
│   │   └── config.go         # Environment configuration
│   ├── db/
│   │   ├── db.go             # Connection pool setup
│   │   └── migrations/       # SQL migration files
│   ├── repository/           # Data access layer (DAO equivalent)
│   │   ├── user.go
│   │   ├── refresh_token.go
│   │   ├── agent_run.go
│   │   └── agent_run_message.go
│   ├── service/              # Business logic
│   │   ├── auth.go
│   │   └── agent.go
│   ├── handler/              # HTTP handlers (controller equivalent)
│   │   ├── auth.go
│   │   ├── user.go
│   │   ├── agent.go
│   │   └── health.go
│   ├── middleware/
│   │   ├── auth.go
│   │   ├── error.go
│   │   └── logging.go
│   ├── agent/                # Agent framework
│   │   ├── base.go           # Base agent interface
│   │   ├── registry.go       # Agent type registry
│   │   ├── context.go        # Agent execution context
│   │   ├── executor.go       # Step execution orchestrator
│   │   └── simple/
│   │       └── simple.go     # Simple agent implementation
│   └── worker/
│       └── tasks.go          # River job definitions
├── pkg/
│   └── jwt/
│       └── jwt.go            # JWT utilities
├── go.mod
├── go.sum
├── .env.example
└── Makefile
```

## Database Schema Changes

Add column to `agent_runs` for River job tracking:

```sql
ALTER TABLE app.agent_runs
ADD COLUMN river_job_id BIGINT;
```

River will create its own schema (`river_job` by default) automatically.

## Implementation Phases

### Phase 1: Project Scaffolding
1. Initialize Go module at `../vinagents-go`
2. Create directory structure
3. Set up `go.mod` with dependencies:
   - `github.com/gin-gonic/gin`
   - `github.com/jackc/pgx/v5`
   - `github.com/riverqueue/river`
   - `github.com/anthropics/anthropic-sdk-go`
   - `github.com/golang-jwt/jwt/v5`
   - `github.com/stretchr/testify`
   - `golang.org/x/crypto/bcrypt`
4. Create Makefile with common commands

### Phase 2: Configuration & Database
1. Implement config loading from environment (matching Node version's env vars)
2. Set up pgx connection pool
3. Create migration for `river_job_id` column
4. Implement repository layer:
   - UserRepository
   - RefreshTokenRepository
   - AgentRunRepository
   - AgentRunMessageRepository

### Phase 3: Authentication
1. Implement JWT token generation/validation
2. Implement auth middleware
3. Implement auth handlers:
   - POST /api/v1/auth/register
   - POST /api/v1/auth/login
   - POST /api/v1/auth/refresh
   - POST /api/v1/auth/logout
4. Implement auth service (password hashing, token management)

### Phase 4: User & Health Endpoints
1. Implement user handlers:
   - GET /api/v1/users/me
   - GET /api/v1/users
2. Implement health handler:
   - GET /api/v1/health

### Phase 5: Agent Framework
1. Define agent interfaces:
   - `Agent` interface with `DefineSteps()`, `Initialize()`, `Cleanup()`
   - `Step` struct with name, type, execute function
   - `Context` struct with run info, LLM client, logging
2. Implement agent registry
3. Implement SimpleAgent (single LLM call step)
4. Implement AgentExecutor (step orchestration, status updates)

### Phase 6: River Queue Integration
1. Set up River client and worker
2. Define `AgentRunJob` type
3. Implement job handler that invokes AgentExecutor
4. Wire up job enqueueing in agent service

### Phase 7: Agent API Endpoints
1. Implement agent handlers:
   - POST /api/v1/agents/:agentType/run
   - GET /api/v1/agents/runs
   - GET /api/v1/agents/runs/:runId
   - POST /api/v1/agents/runs/:runId/cancel

### Phase 8: API Server Assembly
1. Create Gin router with middleware chain:
   - Recovery
   - Logging
   - CORS
   - Rate limiting (if desired)
2. Mount all route groups
3. Implement graceful shutdown
4. Create `cmd/api/main.go` entrypoint

### Phase 9: Worker Process
1. Create standalone worker process
2. Configure River worker with concurrency settings
3. Implement graceful shutdown
4. Create `cmd/worker/main.go` entrypoint

### Phase 10: Testing
1. Set up test database utilities
2. Write tests for:
   - Auth endpoints
   - User endpoints
   - Agent launch/status/cancel/list
   - Agent executor logic
3. Aim for comparable coverage to Node version

## Key Mappings: Node.js → Go

| Node.js | Go |
|---------|-----|
| Express middleware | Gin middleware |
| DAO classes | Repository structs with methods |
| Controller functions | Handler functions |
| Service classes | Service structs with methods |
| Zod validation | Gin binding + custom validators |
| Winston logger | Go `log/slog` (stdlib) |
| bcrypt (npm) | `golang.org/x/crypto/bcrypt` |
| Graphile Worker | River |
| Class-based agents | Interface-based agents |

## Environment Variables (same as Node)

```
NODE_ENV → GO_ENV (or just check for "production")
PORT
API_PREFIX
DATABASE_URL
JWT_SECRET
JWT_EXPIRES_IN
JWT_REFRESH_SECRET
JWT_REFRESH_EXPIRES_IN
ANTHROPIC_API_KEY
CORS_ORIGIN
```

## Verification

After implementation, verify:

1. **Database connectivity**: API and worker can connect to PostgreSQL
2. **Auth flow**: Register → Login → Access protected endpoint → Refresh → Logout
3. **Agent run**: Launch simple agent → Poll status → Verify completion with output
4. **Cancellation**: Launch agent → Request cancel → Verify cancelled status
5. **Run tests**: `go test ./...` passes
6. **Concurrent runs**: Launch multiple agents, verify worker handles them

## Files to Create (in order)

1. `go.mod`, `Makefile`
2. `internal/config/config.go`
3. `internal/db/db.go`
4. `internal/repository/*.go`
5. `pkg/jwt/jwt.go`
6. `internal/middleware/*.go`
7. `internal/service/auth.go`
8. `internal/handler/auth.go`, `health.go`, `user.go`
9. `internal/agent/*.go`
10. `internal/worker/tasks.go`
11. `internal/service/agent.go`
12. `internal/handler/agent.go`
13. `cmd/api/main.go`
14. `cmd/worker/main.go`
15. Tests for each layer
