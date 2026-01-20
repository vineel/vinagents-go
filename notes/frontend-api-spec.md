# Clauser Frontend API Specification

This document contains everything needed to build a frontend client for the Clauser API.

## Base Configuration

- **Base URL:** `http://localhost:3000/api/v1`
- **Content-Type:** `application/json`
- **Authentication:** Bearer token (JWT) in `Authorization` header

---

## Response Envelope

All successful responses follow this structure:

```json
{
  "status": "success",
  "data": { ... }
}
```

Error responses:

```json
{
  "error": "Error message here"
}
```

HTTP status codes:
- `200` - Success
- `201` - Created
- `202` - Accepted (async job started)
- `400` - Bad Request (validation error)
- `401` - Unauthorized (missing/invalid token)
- `404` - Not Found
- `409` - Conflict (e.g., job already running)
- `500` - Internal Server Error

---

## Authentication

### Register

```
POST /auth/register
```

**Request:**
```json
{
  "email": "user@example.com",
  "password": "password123",
  "firstName": "John",
  "lastName": "Doe"
}
```

- `email` - required, must be valid email
- `password` - required, min 6 characters
- `firstName` - optional
- `lastName` - optional

**Response (201):**
```json
{
  "status": "success",
  "data": {
    "user": {
      "userId": "uuid",
      "email": "user@example.com",
      "firstName": "John",
      "lastName": "Doe",
      "isActive": true,
      "createdAt": "2024-01-15T09:00:00Z",
      "updatedAt": "2024-01-15T09:00:00Z"
    },
    "accessToken": "eyJhbG...",
    "refreshToken": "eyJhbG..."
  }
}
```

### Login

```
POST /auth/login
```

**Request:**
```json
{
  "email": "user@example.com",
  "password": "password123"
}
```

**Response (200):** Same as Register

### Refresh Token

```
POST /auth/refresh
```

**Request:**
```json
{
  "refreshToken": "eyJhbG..."
}
```

**Response (200):** Same as Register (new tokens issued)

### Logout

```
POST /auth/logout
```

**Request:**
```json
{
  "refreshToken": "eyJhbG..."
}
```

**Response (200):**
```json
{
  "status": "success",
  "message": "Logged out successfully"
}
```

---

## Protected Routes

All routes below require the `Authorization` header:

```
Authorization: Bearer <accessToken>
```

---

## Clausers

A "clauser" is a clause negotiation workspace containing two agreement versions, two clause versions, context fields, and AI-generated outputs.

### Clauser Object

```typescript
interface Clauser {
  clauserId: string;              // UUID
  userId: string;                 // UUID
  title: string | null;
  agreementA: string | null;      // Full text of Agreement A (our version)
  agreementB: string | null;      // Full text of Agreement B (counterparty)
  clauseA: string | null;         // Clause A (our version)
  clauseB: string | null;         // Clause B (counterparty version)
  representedParty: string | null;      // Who we represent, e.g., "Acme Corp (Licensor)"
  draftingApproach: string | null;      // Strategy: "Aggressive", "Balanced", "Defensive"
  playbook: string | null;              // Reference playbook being used
  counterpartyRationale: string | null; // Context about counterparty's position
  businessContext: string | null;       // Business context for negotiation
  agentRunId: string | null;      // UUID of currently running job (null if none)
  clauseCHistory: ClauseCEntry[]; // History of generated clauses
  createdAt: string;              // ISO 8601
  updatedAt: string;              // ISO 8601
}

interface ClauseCEntry {
  text: string;
  createdAt: string;
  agentRunId: string;
}
```

### Create Clauser

```
POST /clausers
```

**Request:**
```json
{
  "title": "NDA Indemnification Clause"
}
```

- `title` - optional

**Response (201):**
```json
{
  "status": "success",
  "data": { /* Clauser object */ }
}
```

### List Clausers

```
GET /clausers
GET /clausers?limit=20&offset=0
```

**Query params:**
- `limit` - default 20, max 100
- `offset` - default 0

**Response (200):**
```json
{
  "status": "success",
  "data": {
    "clausers": [
      {
        "clauserId": "uuid",
        "title": "NDA Indemnification",
        "hasClauseC": true,
        "createdAt": "2024-01-15T09:00:00Z",
        "updatedAt": "2024-01-15T10:00:00Z"
      }
    ],
    "pagination": {
      "total": 42,
      "limit": 20,
      "offset": 0
    }
  }
}
```

### Get Clauser

```
GET /clausers/{clauserId}
```

**Response (200):**
```json
{
  "status": "success",
  "data": { /* Clauser object */ }
}
```

### Delete Clauser

```
DELETE /clausers/{clauserId}
```

**Response (200):**
```json
{
  "status": "success",
  "message": "Clauser deleted"
}
```

---

## Update Clauser Fields

All field update endpoints follow the same pattern:

```
PUT /clausers/{clauserId}/{field-name}
```

**Request:**
```json
{
  "value": "new value here"
}
```

**Response (200):** Returns the updated Clauser object

### Available Field Endpoints

| Endpoint | Field Updated |
|----------|---------------|
| `PUT /clausers/{id}/title` | title |
| `PUT /clausers/{id}/agreement-a` | agreementA |
| `PUT /clausers/{id}/agreement-b` | agreementB |
| `PUT /clausers/{id}/clause-a` | clauseA |
| `PUT /clausers/{id}/clause-b` | clauseB |
| `PUT /clausers/{id}/represented-party` | representedParty |
| `PUT /clausers/{id}/drafting-approach` | draftingApproach |
| `PUT /clausers/{id}/playbook` | playbook |
| `PUT /clausers/{id}/counterparty-rationale` | counterpartyRationale |
| `PUT /clausers/{id}/business-context` | businessContext |

---

## Screen State

Get everything needed to render the clauser workspace in one call.

```
GET /clausers/{clauserId}/screen
```

**Response (200):**
```json
{
  "status": "success",
  "data": {
    "clauser": { /* Clauser object */ },
    "outputs": [ /* ClauserOutput objects */ ],
    "favorites": { /* ClauserOutput or null */ },
    "activeRun": {
      "runId": "uuid",
      "status": "running",
      "currentStep": 1,
      "totalSteps": 3
    } // or null if no active job
  }
}
```

---

## Clauser Outputs

Outputs are AI-generated results stored in `clauser_outputs` table.

### ClauserOutput Object

```typescript
interface ClauserOutput {
  clauserOutputId: string;        // UUID
  agentRunId: string | null;      // UUID of the run that created this
  ordinal: number;                // Display order
  groupName: string;              // "lenses", "clause_c", "favorites"
  groupOrdinal: number;
  title: string;
  kind: string;                   // "lens_output", "clause_draft", "favorites"
  content: object;                // Structure varies by kind (see below)
  createdAt: string;
}
```

### Content Structures by Kind

**lens_output** (groupName: "lenses"):
```json
{
  "deltaResolutionMap": [
    {
      "clause_a": "Description of clause A position",
      "clause_b": "Description of clause B position",
      "clause_c": "Proposed compromise",
      "interestsProtected": "Both - explanation",
      "favorabilityPercent": "70"
    }
  ],
  "anotherLensName": [ ... ]
}
```

**clause_draft** (groupName: "clause_c"):
```json
{
  "text": "The generated clause text..."
}
```

**favorites** (groupName: "favorites"):
```json
{
  "items": [
    {
      "itemId": "uuid-of-the-original-item",
      "sourceOutputId": "uuid-of-clauser-output-row",
      "lens": "deltaResolutionMap",
      "data": {
        "priority": "high",
        "clause_a": "...",
        "clause_b": "...",
        "clause_c": "..."
      }
    }
  ]
}
```
Note: The `data` field contains the original item with `itemId` stripped out. The structure varies by lens type.

### Get All Outputs

```
GET /clausers/{clauserId}/outputs
```

**Response (200):**
```json
{
  "status": "success",
  "data": [ /* array of ClauserOutput objects */ ]
}
```

---

## AI Actions

### Run Lens Analysis

Triggers AI analysis using specified lenses.

```
POST /clausers/{clauserId}/run-lenses
```

**Request:**
```json
{
  "lenses": ["deltaResolutionMap", "risks", "opportunities"]
}
```

- `lenses` - required, array of lens names (at least one)

**Response (202):**
```json
{
  "status": "success",
  "data": {
    "runId": "uuid",
    "status": "pending",
    "pollUrl": "/api/v1/agents/runs/{runId}"
  }
}
```

**Error (409):** If a job is already running for this clauser.

### Rewrite Clause (Generate Clause C)

Triggers AI to generate a new clause draft.

```
POST /clausers/{clauserId}/rewrite
```

**Request:**
```json
{
  "instructions": "Create a balanced clause with mutual indemnification"
}
```

- `instructions` - optional, guidance for the AI

**Response (202):** Same as run-lenses

### Append Clause C (Manual)

Manually append a new clause C version to the history (without AI).

```
POST /clausers/{clauserId}/clause-c
```

**Request:**
```json
{
  "text": "The new clause text to append..."
}
```

- `text` - required, the clause text

**Response (200):**
```json
{
  "status": "success",
  "data": { /* Updated Clauser object with new entry in clauseCHistory */ }
}
```

### Clear Run

Cancels any active job and clears the `agentRunId` on the clauser. Useful when a job gets stuck or times out.

```
POST /clausers/{clauserId}/clear-run
```

**Request:** No body required

**Response (200):**
```json
{
  "status": "success",
  "data": { /* Updated Clauser object with agentRunId: null */ },
  "message": "Run cleared"
}
```

---

## Favorites

### Add to Favorites

```
POST /clausers/{clauserId}/outputs/{outputId}/favorite
```

**Request:**
```json
{
  "itemId": "uuid-of-item-to-favorite"
}
```

- `itemId` - The UUID of the item within the lens output (each item has a unique `itemId` field)

The API searches the output's content for an item with the matching `itemId`, copies it to favorites along with the lens name.

**Response (200):**
```json
{
  "status": "success",
  "data": { /* Updated favorites ClauserOutput */ }
}
```

**Response (404):** If `itemId` is not found in the output content.

### Remove from Favorites

```
DELETE /clausers/{clauserId}/favorites/{itemId}
```

- `itemId` - The UUID of the item to remove from favorites

**Response (200):**
```json
{
  "status": "success",
  "data": { /* Updated favorites ClauserOutput */ }
}
```

**Response (404):** If `itemId` is not found in favorites.

---

## Agent Runs (Job Status)

### Poll Job Status

After starting a job (run-lenses or rewrite), poll this endpoint until completion.

```
GET /agents/runs/{runId}
GET /agents/runs/{runId}?includeMessages=true
GET /agents/runs/{runId}?includeMessages=true&messagesSince=2024-01-15T10:00:00Z
```

**Query params:**
- `includeMessages` - if "true", include log messages
- `messagesSince` - ISO 8601 timestamp, only return messages after this time

**Response (200):**
```json
{
  "status": "success",
  "data": {
    "runId": "uuid",
    "agentType": "clauser_lens",
    "status": "running",
    "currentStep": 1,
    "totalSteps": 3,
    "input": { ... },
    "output": { ... },
    "error": null,
    "createdAt": "2024-01-15T10:00:00Z",
    "startedAt": "2024-01-15T10:00:01Z",
    "completedAt": null,
    "messages": [
      {
        "messageId": "uuid",
        "stepNumber": 1,
        "level": "info",
        "message": "Starting lens analysis",
        "details": null,
        "createdAt": "2024-01-15T10:00:01Z"
      }
    ]
  }
}
```

### Status Values

- `pending` - Job queued, not yet started
- `running` - Job in progress
- `completed` - Job finished successfully
- `failed` - Job failed (check `error` field)
- `cancelled` - Job was cancelled
- `cancel_requested` - Cancellation in progress

### Cancel Job

```
POST /agents/runs/{runId}/cancel
```

**Response (200):**
```json
{
  "status": "success",
  "data": {
    "runId": "uuid",
    "status": "cancel_requested",
    "message": "Cancellation requested. The run will be cancelled shortly if still in progress."
  }
}
```

### List Runs

```
GET /agents/runs
GET /agents/runs?status=running&agentType=clauser_lens&limit=20&offset=0
```

**Query params:**
- `status` - filter by status
- `agentType` - filter by agent type ("clauser_lens", "clauser_write")
- `limit` - default 20, max 100
- `offset` - default 0

**Response (200):**
```json
{
  "status": "success",
  "data": {
    "runs": [ /* AgentRun objects */ ],
    "pagination": {
      "total": 42,
      "limit": 20,
      "offset": 0
    }
  }
}
```

---

## Recommended Polling Strategy

When a job is started:

1. Receive `202 Accepted` with `pollUrl`
2. Poll `GET {pollUrl}` every 1-2 seconds
3. Check `status` field:
   - If `pending` or `running`: continue polling
   - If `completed`: fetch screen state to get new outputs
   - If `failed`: display error from `error` field
4. Optionally use `includeMessages=true` to show progress to user
5. Use `messagesSince` to avoid re-fetching old messages

---

## TypeScript Types Summary

```typescript
// Auth
interface User {
  userId: string;
  email: string;
  firstName: string | null;
  lastName: string | null;
  isActive: boolean;
  createdAt: string;
  updatedAt: string;
}

interface AuthResponse {
  user: User;
  accessToken: string;
  refreshToken: string;
}

// Clauser
interface Clauser {
  clauserId: string;
  userId: string;
  title: string | null;
  agreementA: string | null;
  agreementB: string | null;
  clauseA: string | null;
  clauseB: string | null;
  representedParty: string | null;
  draftingApproach: string | null;
  playbook: string | null;
  counterpartyRationale: string | null;
  businessContext: string | null;
  agentRunId: string | null;
  clauseCHistory: { text: string; createdAt: string; agentRunId: string }[];
  createdAt: string;
  updatedAt: string;
}

interface ClauserListItem {
  clauserId: string;
  title: string | null;
  hasClauseC: boolean;
  createdAt: string;
  updatedAt: string;
}

interface ClauserOutput {
  clauserOutputId: string;
  agentRunId: string | null;
  ordinal: number;
  groupName: string;
  groupOrdinal: number;
  title: string;
  kind: 'lens_output' | 'clause_draft' | 'favorites';
  content: Record<string, unknown>;
  createdAt: string;
}

interface ScreenState {
  clauser: Clauser;
  outputs: ClauserOutput[];
  favorites: ClauserOutput | null;
  activeRun: {
    runId: string;
    status: string;
    currentStep: number;
    totalSteps: number | null;
  } | null;
}

interface RunJobResponse {
  runId: string;
  status: string;
  pollUrl: string;
}

// Agent Runs
type RunStatus = 'pending' | 'running' | 'completed' | 'failed' | 'cancelled' | 'cancel_requested';

interface AgentRun {
  runId: string;
  agentType: string;
  status: RunStatus;
  currentStep: number;
  totalSteps: number | null;
  input: Record<string, unknown>;
  output: Record<string, unknown> | null;
  error: string | null;
  createdAt: string;
  startedAt: string | null;
  completedAt: string | null;
}

interface AgentRunMessage {
  messageId: string;
  stepNumber: number | null;
  level: 'debug' | 'info' | 'warn' | 'error';
  message: string;
  details: Record<string, unknown> | null;
  createdAt: string;
}

// API Response wrapper
interface ApiResponse<T> {
  status: 'success';
  data: T;
}

interface ApiError {
  error: string;
}

interface PaginatedResponse<T> {
  status: 'success';
  data: {
    [key: string]: T[];
    pagination: {
      total: number;
      limit: number;
      offset: number;
    };
  };
}
```
