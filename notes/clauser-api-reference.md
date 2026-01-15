# Clauser API Reference

All endpoints require authentication via `Authorization: Bearer <token>` header.

Base URL: `http://localhost:3000/api/v1`

---

## Clausers

### Create a clauser

```
POST /clausers

{
  "title": "NDA Indemnification Clause"
}
```

Title is optional. Returns the created clauser.

---

### List clausers

```
GET /clausers
GET /clausers?limit=20&offset=0
```

Returns paginated list of user's clausers.

---

### Get a clauser

```
GET /clausers/{clauserId}
```

---

### Delete a clauser

```
DELETE /clausers/{clauserId}
```

---

## Update Fields

### Update title

```
PUT /clausers/{clauserId}/title

{
  "value": "Updated Title"
}
```

---

### Update Agreement A (our version)

```
PUT /clausers/{clauserId}/agreement-a

{
  "value": "Full text of our agreement..."
}
```

---

### Update Agreement B (counterparty version)

```
PUT /clausers/{clauserId}/agreement-b

{
  "value": "Full text of counterparty agreement..."
}
```

---

### Update Clause A (our version)

```
PUT /clausers/{clauserId}/clause-a

{
  "value": "Party A shall indemnify and hold harmless Party B from any claims arising from Party A's breach of this Agreement."
}
```

---

### Update Clause B (counterparty version)

```
PUT /clausers/{clauserId}/clause-b

{
  "value": "Each party shall indemnify the other for direct damages only, with total liability capped at fees paid in the prior 12 months."
}
```

---

## Screen State

### Get full screen state

```
GET /clausers/{clauserId}/screen
```

Returns everything needed to render the UI:
- `clauser` - The clauser with all fields
- `outputs` - All lens analyses and generated clauses
- `favorites` - The favorites collection (or null)
- `activeRun` - Currently running job status (or null)

---

### Get outputs only

```
GET /clausers/{clauserId}/outputs
```

Returns array of all outputs for this clauser.

---

## AI Actions

### Run lens analysis

```
POST /clausers/{clauserId}/run-lenses

{
  "lenses": ["risks", "opportunities"]
}
```

Available lenses:
- `risks` - Legal, business, operational risks
- `opportunities` - Areas for favorable negotiation
- `ambiguities` - Unclear or contested language
- `compliance` - Regulatory considerations
- `enforcement` - Enforceability issues
- `market_standard` - Comparison to market practice

Returns `202 Accepted` with:
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

Returns `409 Conflict` if a job is already running.

---

### Rewrite clause (generate Clause C)

```
POST /clausers/{clauserId}/rewrite

{
  "instructions": "Create a balanced clause with mutual indemnification and a 2x annual fees liability cap"
}
```

Instructions are optional. The AI will use:
- Clause A and Clause B as inputs
- Any favorited analysis items
- The provided instructions (if any)

Returns `202 Accepted` with job info.

---

## Favorites

### Add item to favorites

```
POST /clausers/{clauserId}/outputs/{outputId}/favorite

{
  "itemIndex": 0
}
```

`itemIndex` is the zero-based index of the item within the output's `content.items` array.

---

### Remove item from favorites

```
DELETE /clausers/{clauserId}/favorites/{index}
```

`index` is the zero-based position in the favorites list.

---

## Response Formats

### Clauser object

```json
{
  "clauserId": "uuid",
  "userId": "uuid",
  "title": "NDA Indemnification Clause",
  "agreementA": "Full agreement text...",
  "agreementB": "Counterparty agreement text...",
  "clauseA": "Our clause version...",
  "clauseB": "Their clause version...",
  "agentRunId": "uuid or null",
  "clauseCHistory": [
    {
      "text": "Generated clause text...",
      "createdAt": "2024-01-15T10:00:00Z",
      "agentRunId": "uuid"
    }
  ],
  "createdAt": "2024-01-15T09:00:00Z",
  "updatedAt": "2024-01-15T10:00:00Z"
}
```

---

### Lens output

```json
{
  "clauserOutputId": "uuid",
  "agentRunId": "uuid",
  "ordinal": 1,
  "groupName": "risks",
  "groupOrdinal": 0,
  "title": "Risk Assessment",
  "kind": "lens_output",
  "content": {
    "title": "Risk Assessment",
    "items": [
      {
        "title": "Unlimited liability exposure",
        "body": "Our version creates unlimited seller liability...",
        "severity": "high"
      },
      {
        "title": "No consequential damages exclusion",
        "body": "Neither clause excludes consequential damages...",
        "severity": "medium"
      }
    ]
  },
  "createdAt": "2024-01-15T10:00:00Z"
}
```

---

### Generated clause output

```json
{
  "clauserOutputId": "uuid",
  "agentRunId": "uuid",
  "ordinal": 3,
  "groupName": "clause_c",
  "groupOrdinal": 0,
  "title": "Generated Clause",
  "kind": "clause_draft",
  "content": {
    "text": "Each party's total liability shall not exceed two (2) times the amounts paid under this Agreement..."
  },
  "createdAt": "2024-01-15T10:05:00Z"
}
```

---

### Favorites output

```json
{
  "clauserOutputId": "uuid",
  "ordinal": 0,
  "groupName": "favorites",
  "groupOrdinal": 0,
  "title": "Favorites",
  "kind": "favorites",
  "content": {
    "items": [
      {
        "sourceOutputId": "uuid",
        "sourceItemIndex": 0,
        "sourceGroupName": "risks",
        "title": "Unlimited liability exposure",
        "body": "Our version creates unlimited seller liability..."
      }
    ]
  },
  "createdAt": "2024-01-15T10:02:00Z"
}
```

---

## Error Responses

```json
{"error": "Clauser not found"}
{"error": "Invalid request body: Key: 'value' Error:Field validation failed"}
{"error": "A job is already running for this clauser"}
{"error": "Unauthorized"}
```

---

## Polling for Job Status

After triggering `run-lenses` or `rewrite`, poll the returned `pollUrl`:

```
GET /api/v1/agents/runs/{runId}
```

Response:
```json
{
  "status": "success",
  "data": {
    "runId": "uuid",
    "status": "running",
    "currentStep": 1,
    "totalSteps": 3
  }
}
```

Status values: `pending`, `running`, `completed`, `failed`, `cancelled`, `cancel_requested`

When `status` is `completed`, fetch the screen state to get the new outputs.
