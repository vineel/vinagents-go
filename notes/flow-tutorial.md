# Multi-Step Agent Workflow Tutorial

This document explains how multi-step agents work in this codebase, using a concrete example: the **Rewriter Agent**.

## The Example: Rewriter Agent

```
Input: Clause A, Clause B
        ↓
   ┌────┴────┐
   ▼         ▼
Step A     Step B      (parallel - both LLM calls)
   │         │
   └────┬────┘
        ▼
     Step C            (sequential - depends on A & B)
        ↓
Output: Clause C (persisted, shown to user)
```

---

## Database Tables Involved

### `app.agent_runs` — The Run Record

When a user kicks off the Rewriter, a row is created here. This is the source of truth for the entire execution.

| Column | Role in Workflow |
|--------|------------------|
| `agent_run_id` | UUID identifying this specific run |
| `agent_type` | `"rewriter"` — used to look up the agent factory |
| `input_payload` | JSONB containing `{clauseA: "...", clauseB: "...", promptA: "...", promptB: "...", promptC: "..."}` |
| `output_payload` | JSONB where final result lives after completion (`{clauseC: "..."}`) |
| `status` | State machine: `pending` → `running` → `completed` (or `failed`/`cancelled`) |
| `current_step` | Integer tracking progress (0, 1, 2...) — lets UI show "Step 2 of 3" |
| `total_steps` | Set when agent starts — Rewriter would set this to 3 |
| `river_job_id` | Links to the River job that's executing this run |
| `started_at` | Timestamp when executor picked it up |
| `completed_at` | Timestamp when final step finished |

### `app.agent_run_messages` — The Audit Trail

Every significant event gets logged here. This powers the UI's progress display and post-mortem debugging.

| Column | Role in Workflow |
|--------|------------------|
| `agent_run_id` | Foreign key back to the run |
| `step_number` | Which step emitted this (null for run-level events) |
| `level` | `debug`, `info`, `warn`, `error` |
| `message` | Human-readable: `"Starting Step A: Process Clause A"` |
| `details` | JSONB for structured data: `{tokens_in: 500, tokens_out: 200, model: "claude-sonnet-4"}` |

---

## How River Fits In

River is a Postgres-backed job queue. Key mental model:

1. **Worker process** — a long-running Go process that calls `riverClient.Start()` and blocks, waiting for jobs
2. **Job pickup** — River uses Postgres `LISTEN/NOTIFY` + polling to detect new jobs, then calls your `Work()` method
3. **Goroutine pool** — River manages a pool (configured via `MaxWorkers`); each job runs in one goroutine
4. **Atomic execution** — to River, a job is a black box. It either succeeds or fails. River knows nothing about "steps"

The step abstraction is purely application-level. River sees:
```
Job started → (30 seconds pass) → Job completed
```

We see:
```
Step 0: running → completed
Step 1: running → completed
Step 2: running → completed
```

---

## How a Run Moves Forward

### Phase 1: Enqueue (API Layer)

1. User calls `POST /api/v1/agents/rewriter/run` with input JSON
2. Service layer creates an `agent_runs` row with `status = 'pending'`
3. Service enqueues a River job containing just `{run_id: "<uuid>"}`
4. Service updates `river_job_id` on the run row
5. API returns immediately with `{runId, status: "pending", pollUrl}`

The run is now sitting in the queue. The user can poll for updates.

### Phase 2: Pickup (Worker Process)

The worker process is running River's job loop. When it grabs our job:

1. Worker receives `AgentRunArgs{RunID: "..."}` from River
2. Worker loads the `agent_runs` row by ID
3. Worker updates: `status = 'running'`, `started_at = now()`
4. Worker looks up agent factory in registry using `agent_type = 'rewriter'`
5. Worker instantiates the agent and calls `DefineSteps()` to get the step list
6. Worker sets `total_steps = len(steps)` on the run row

### Phase 3: Step Execution (Executor)

The executor loops through steps. For each step:

```
1. Check: Is status = 'cancel_requested'? → If yes, bail out
2. Check: Should this step be skipped? (conditional logic)
3. Transform: Reshape previous output if needed
4. Execute: Run the step's function
5. Log: Write to agent_run_messages
6. Update: Set current_step = N on the run row
7. Pass: Hand output to next step as input (in-memory)
```

#### How Rewriter Steps Would Execute

**Step 0 (Parallel LLM calls):**
- This is conceptually "steps A and B" but implemented as a single step
- The step function internally launches both LLM calls concurrently (goroutines or errgroup)
- Waits for both to complete
- Returns combined output: `{resultA: "...", resultB: "..."}`
- Logs: `"Completed parallel processing of Clause A and Clause B"`
- Updates: `current_step = 0`

**Step 1 (Sequential LLM call):**
- Receives `{resultA, resultB}` from previous step (in-memory)
- Calls LLM with Prompt C, feeding in both results
- Returns: `{clauseC: "..."}`
- Logs: `"Generated final Clause C"`
- Updates: `current_step = 1`

### Phase 4: Completion

After all steps finish:

1. Executor stores final step's output into `output_payload`
2. Updates: `status = 'completed'`, `completed_at = now()`
3. Worker returns `nil` to River (job done)

The UI can now fetch the completed run and display Clause C to the user.

---

## Handling Parallelism

The step framework executes sequentially. For parallel work like the A/B processing, make it a single step that internally parallelizes:

```
DefineSteps() returns:
  Step 0: "Process Clauses A & B" (internally parallel via errgroup)
  Step 1: "Generate Clause C"
```

The step function uses `errgroup.Group` to run both LLM calls concurrently. From the executor's perspective, it's one step.

**Trade-off:** `current_step` only shows "step 0" while both A and B run. If you need finer progress visibility, log to `agent_run_messages` as each sub-task completes.

---

## Key State Transitions

```
┌─────────┐    ┌─────────┐    ┌───────────┐
│ pending │───▶│ running │───▶│ completed │
└─────────┘    └────┬────┘    └───────────┘
                    │
                    ├────────▶ failed
                    │
                    └────────▶ cancel_requested ───▶ cancelled
```

- `pending → running`: Worker picks up job
- `running → completed`: All steps succeed
- `running → failed`: Any step throws, or unrecoverable error
- `running → cancel_requested`: User calls cancel endpoint
- `cancel_requested → cancelled`: Executor detects at next step boundary

---

## What Gets Persisted Where

| Data | Location | Column |
|------|----------|--------|
| User's input (clauses, prompts) | `agent_runs` | `input_payload` |
| Final output (Clause C) | `agent_runs` | `output_payload` |
| Intermediate results (resultA, resultB) | In-memory | passed between steps |
| Progress indicators | `agent_runs` | `current_step`, `total_steps` |
| Detailed logs | `agent_run_messages` | `message`, `details` |
| Token usage per step | `agent_run_messages` | `details` JSONB |
| Error details | `agent_runs` | `error_message`, `error_details` |

**Note:** Intermediate results are not persisted by default. If you need to debug or want visibility, log them to `agent_run_messages.details`.

---

## Polling for Progress

The UI calls:
```
GET /api/v1/agents/runs/{runId}?includeMessages=true&messagesSince=2024-01-15T10:00:00Z
```

Response includes:
- `status`: current state
- `currentStep` / `totalSteps`: for progress bar
- `messages`: new log entries since last poll

For the Rewriter mid-execution, this might return:
```json
{
  "status": "running",
  "currentStep": 0,
  "totalSteps": 2,
  "messages": [
    {"level": "info", "message": "Starting parallel processing", "stepNumber": 0},
    {"level": "info", "message": "Clause A processed: 450 tokens", "stepNumber": 0}
  ]
}
```

---

## Future: Resumability

If you later need mid-job re-enterability (resume after crash), you can:

1. Store checkpoint data in `agent_runs.output_payload` (or add a `checkpoint` JSONB column)
2. On job retry, check for existing checkpoint
3. Skip already-completed work based on checkpoint state

This keeps the schema simple while allowing opt-in durability for long-running agents.

---

## Summary: Mental Model

1. **`agent_runs`** — the run's home base (status, progress, input, output)
2. **`agent_run_messages`** — audit log for debugging and UI progress
3. **River** — job durability and async execution; sees jobs as atomic black boxes
4. **Steps** — application-level abstraction; executed sequentially in one goroutine
5. **Parallelism** — handled within a step via goroutines/errgroup
6. **Intermediate results** — in-memory; log to messages if you need visibility

The Rewriter agent would be roughly 100 lines of Go: define two steps, implement the parallel LLM calls in step 0, implement the combining LLM call in step 1, register it, done.
