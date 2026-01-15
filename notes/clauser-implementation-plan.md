# Clauser Feature Implementation Plan

## Overview

Implement the "Clauser" feature - a tool for lawyers to iterate on contract clauses using AI. Users provide agreements and clause versions, run AI "lenses" for analysis, curate favorites, and generate refined clause drafts.

## Database Schema

Create `notes/clauser-schema.sql`:

```sql
-- Clauser session table
CREATE TABLE app.clausers (
    clauser_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES app.users(user_id) ON DELETE CASCADE,
    title TEXT,
    agreement_a TEXT,
    agreement_b TEXT,
    clause_a TEXT,
    clause_b TEXT,
    agent_run_id UUID REFERENCES app.agent_runs(agent_run_id) ON DELETE SET NULL,
    clause_c_history JSONB NOT NULL DEFAULT '[]'::JSONB,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_clausers_user_id ON app.clausers(user_id);

-- Clauser outputs table (immutable)
CREATE TABLE app.clauser_outputs (
    clauser_output_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    clauser_id UUID NOT NULL REFERENCES app.clausers(clauser_id) ON DELETE CASCADE,
    agent_run_id UUID REFERENCES app.agent_runs(agent_run_id) ON DELETE SET NULL,
    ordinal INTEGER NOT NULL,
    group_name TEXT NOT NULL,
    group_ordinal INTEGER NOT NULL DEFAULT 0,
    title TEXT NOT NULL,
    kind TEXT NOT NULL,
    content JSONB NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_clauser_outputs_clauser_id ON app.clauser_outputs(clauser_id);

-- Trigger for updated_at
CREATE TRIGGER update_clausers_updated_at
    BEFORE UPDATE ON app.clausers
    FOR EACH ROW
    EXECUTE FUNCTION app.update_updated_at_column();
```

### Data Structures

**clause_c_history** (JSONB array on clausers):
```json
[
  {"text": "clause text...", "createdAt": "2024-01-15T...", "agentRunId": "uuid"}
]
```

**content** (JSONB on clauser_outputs):
- For lens outputs: `{"items": [{"title": "...", "body": "...", "severity": "high"}]}`
- For favorites: `{"items": [{"sourceOutputId": "uuid", "sourceItemIndex": 0, "title": "...", "body": "..."}]}`
- For clause_c: `{"text": "...", "version": 1}`

### Favorites Mechanism

- A dedicated row with `group_name = "favorites"` and `kind = "favorites"` holds all favorited items
- When user favorites an item from a lens output, copy it to the favorites row's `content.items` array
- Unfavoriting removes from that array
- The favorites row is created lazily on first favorite action

## File Structure

```
internal/
├── repository/
│   ├── models.go              # Add Clauser, ClauserOutput, input structs
│   ├── clauser.go             # ClauserRepository
│   └── clauser_output.go      # ClauserOutputRepository
├── service/
│   └── clauser.go             # ClauserService
├── handler/
│   └── clauser.go             # ClauserHandler + routes
├── worker/
│   ├── tasks.go               # Add ClauserWriteArgs, ClauserLensArgs
│   ├── clauser_write.go       # ClauserWriteWorker
│   └── clauser_lens.go        # ClauserLensWorker
└── prompt/
    └── loader.go              # Template loading

prompts/
├── clauser-write-clause.prompt.tpl
└── clauser-lenses.prompt.tpl

notes/
└── clauser-schema.sql
```

## API Routes

All routes under `/api/v1/clausers`, protected by auth middleware.

| Method | Path | Description |
|--------|------|-------------|
| POST | `/clausers` | Create new clauser |
| GET | `/clausers` | List user's clausers |
| GET | `/clausers/:id` | Get clauser |
| DELETE | `/clausers/:id` | Delete clauser |
| PUT | `/clausers/:id/agreement-a` | Update agreement A |
| PUT | `/clausers/:id/agreement-b` | Update agreement B |
| PUT | `/clausers/:id/clause-a` | Update clause A |
| PUT | `/clausers/:id/clause-b` | Update clause B |
| GET | `/clausers/:id/screen` | Get full screen state |
| POST | `/clausers/:id/run-lenses` | Trigger lens job |
| POST | `/clausers/:id/rewrite` | Trigger rewrite job |
| POST | `/clausers/:id/outputs/:outputId/items/:index/favorite` | Add item to favorites |
| DELETE | `/clausers/:id/favorites/:index` | Remove from favorites |

### Key Request/Response Shapes

**GET /clausers/:id/screen**
```json
{
  "status": "success",
  "data": {
    "clauser": { "clauserId", "title", "agreementA", "agreementB", "clauseA", "clauseB", "clauseCHistory", "agentRunId" },
    "outputs": [{ "clauserOutputId", "ordinal", "groupName", "groupOrdinal", "title", "kind", "content", "createdAt" }],
    "favorites": { "clauserOutputId", "content": { "items": [...] } },
    "activeRun": { "runId", "status", "currentStep", "totalSteps" } | null
  }
}
```

**POST /clausers/:id/run-lenses**
```json
{ "lenses": ["risks", "opportunities", "ambiguities"] }
```

## Worker Jobs

### ClauserWriteArgs
```go
type ClauserWriteArgs struct {
    ClauserID    string `json:"clauser_id"`
    AgentRunID   string `json:"agent_run_id"`
    Instructions string `json:"instructions,omitempty"`
}
func (ClauserWriteArgs) Kind() string { return "clauser_write" }
```

### ClauserLensArgs
```go
type ClauserLensArgs struct {
    ClauserID  string   `json:"clauser_id"`
    AgentRunID string   `json:"agent_run_id"`
    Lenses     []string `json:"lenses"`
}
func (ClauserLensArgs) Kind() string { return "clauser_lens" }
```

### Worker Flow

**ClauserWriteWorker:**

1. Load clauser (agreements, clauses) and favorites from DB
2. Load `clauser-write-clause.prompt.tpl`
3. Substitute variables, call Claude
4. Append result to `clause_c_history` JSONB array
5. Create clauser_output row with `group_name="clause_c"`
6. Update agent_run status to completed

**ClauserLensWorker:**

1. Load clauser and favorites
2. Load `clauser-lenses.prompt.tpl`
3. Call Claude with selected lenses
4. Parse structured output
5. Create clauser_output rows per lens
6. Update agent_run status to completed

## Implementation Order

### Phase 1: Schema & Models
1. Write `notes/clauser-schema.sql`
2. Run SQL against database
3. Add to `internal/repository/models.go`:
   - `Clauser` struct
   - `ClauserOutput` struct
   - `CreateClauserInput`, `UpdateClauserInput`
   - `CreateClauserOutputInput`

### Phase 2: Repository Layer
4. Create `internal/repository/clauser.go`:
   - `Create`, `FindByID`, `FindByIDAndUserID`, `FindByUserID`
   - `Update`, `UpdateField`, `AppendClauseCHistory`, `Delete`
5. Create `internal/repository/clauser_output.go`:
   - `Create`, `FindByClauserID`, `FindFavorites`
   - `GetOrCreateFavorites`, `AppendFavorite`, `RemoveFavorite`
   - `GetNextOrdinal`

### Phase 3: Prompt System
6. Create `internal/prompt/loader.go`:
   - `LoadTemplate(name string) (*template.Template, error)`
   - `Execute(name string, data interface{}) (string, error)`
7. Create prompt templates in `prompts/`

### Phase 4: Workers
8. Add job args to `internal/worker/tasks.go`
9. Create `internal/worker/clauser_write.go`
10. Create `internal/worker/clauser_lens.go`
11. Register workers in `cmd/worker/main.go`

### Phase 5: Service Layer
12. Create `internal/service/clauser.go`:
    - CRUD methods
    - `GetScreenState`
    - `RunLenses`, `Rewrite` (create agent_run, enqueue job)
    - `AddToFavorites`, `RemoveFromFavorites`

### Phase 6: Handler Layer
13. Create `internal/handler/clauser.go`:
    - All route handlers
    - `RegisterRoutes(rg *gin.RouterGroup, authMiddleware gin.HandlerFunc)`
14. Register in `cmd/api/main.go`

## Critical Files to Modify

- `internal/repository/models.go` - Add Clauser and ClauserOutput types
- `internal/worker/tasks.go` - Add job args types
- `cmd/worker/main.go` - Register new workers
- `cmd/api/main.go` - Register clauser routes

## Verification

1. **Database:** Run schema SQL, verify tables exist with `\dt app.*`
2. **API:** Use curl/httpie to test CRUD operations
3. **Workers:**
   - Create a clauser, add agreements/clauses
   - POST to `/run-lenses`, check agent_runs table for pending job
   - Run worker, verify outputs appear in clauser_outputs
4. **Screen endpoint:** Verify aggregated response includes all data
5. **Favorites:** Add/remove favorites, verify persistence
