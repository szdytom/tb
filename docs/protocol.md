# tmpbuffer IPC Protocol

## Transport

Unix domain socket (stream). Default location: `$XDG_STATE_HOME/tmpbuffer/tmpbuffer.sock`. Socket permissions: `0600`.

## Framing

Newline-delimited JSON. Each message is a single JSON object terminated by `\n`. Messages are encoded with `json.Encoder` and decoded with `json.Decoder`.

## Envelope

### Request

```
{"id": <int64>, "op": "<string>", "payload": {<op-specific>}}
```

| Field | Type | Description |
|---|---|---|
| `id` | number | Client-chosen correlation ID. Echoed back in the response. |
| `op` | string | One of the operation codes below. |
| `payload` | object | Per-operation payload (see below). Omitted if empty. |

### Response (success)

```
{"id": <int64>, "ok": true, "payload": {<op-specific>}}
```

### Response (error)

```
{"id": <int64>, "ok": false, "error": "<message>"}
```

---

## Operations

### Ping

Check that the daemon is alive.

**Request:** no payload
**Response payload:** `{"message": "pong"}`

### CreateBuffer

Create a new buffer.

**Request payload:**
```
{"content": "<string>", "label": "<string>", "tags": ["<string>"]}
```
`label` and `tags` are optional.

**Response payload:** `{"id": <int64>}` — the assigned buffer ID.

### GetBuffer

Retrieve a single buffer by ID.

**Request payload:** `{"id": <int64>}`
**Response payload:** full `Buffer` object (see below).

Returns error if ID is not found.

### ListBuffers

List active (non-trashed) buffers.

**Request payload:** (all fields optional)
```
{"keyword": "<string>", "since": "<RFC3339>", "until": "<RFC3339>",
 "limit": <int>, "offset": <int>, "sort_by": "<field>", "sort_asc": <bool>}
```

`sort_by` accepts: `"updated_at"` (default), `"created_at"`, `"label"`, `"id"`. Default order is most-recent-first (DESC); set `sort_asc: true` to reverse.

`since` and `until` are RFC 3339 timestamps.

**Response payload:** `[{<Buffer>}, ...]`

### UpdateContent

Replace a buffer's content.

**Request payload:** `{"id": <int64>, "content": "<string>"}`
**Response payload:** none

### UpdateLabel

Change a buffer's label.

**Request payload:** `{"id": <int64>, "label": "<string>"}`
**Response payload:** none

### UpdateTags

Replace a buffer's tags.

**Request payload:** `{"id": <int64>, "tags": ["<string>"]}`
**Response payload:** none

### SoftDelete

Move a buffer to the trash.

**Request payload:**
```
{"id": <int64>, "ttl_seconds": <int>}
```
`ttl_seconds` is optional. Default TTL is 24 hours.

**Response payload:** none

### PermanentlyDelete

Remove a buffer from the database entirely.

**Request payload:** `{"id": <int64>}`
**Response payload:** none

### ListTrash

List all trashed buffers.

**Request:** no payload
**Response payload:** `[{<Buffer>}, ...]`

### RestoreFromTrash

Move a buffer from trash back to active.

**Request payload:** `{"id": <int64>}`
**Response payload:** none

### Search

Full-text search across all active buffers.

**Request payload:**
```
{"query": "<string>", "is_regex": <bool>}
```
If `is_regex` is true, `query` is treated as a Go regular expression.

**Response payload:** `[{"buffer": {<Buffer>}, "snippet": "<string>"}, ...]`

### Count

Return the number of active buffers.

**Request:** no payload
**Response payload:** `{"count": <int>}`

---

## Buffer Object

Returned by `GetBuffer`, `ListBuffers`, `ListTrash`, and embedded in `SearchResult`.

```
{"id": <int64>, "label": "<string>", "content": "<string>",
 "metadata": {"line_count": <int>, "byte_count": <int>},
 "tags": ["<string>"],
 "created_at": "<RFC3339>", "updated_at": "<RFC3339>",
 "trash_status": <int>,
 "trashed_at": "<RFC3339>", "expires_at": "<RFC3339>"}
```

`trash_status`: `0` = active, `1` = trashed.
`label`, `tags`, `trashed_at`, `expires_at` are omitted when empty/null.
