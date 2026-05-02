package ipc

import "encoding/json"

// Op identifies a daemon operation.
type Op string

const (
	OpPing                Op = "Ping"
	OpCreateBuffer        Op = "CreateBuffer"
	OpGetBuffer           Op = "GetBuffer"
	OpListBuffers         Op = "ListBuffers"
	OpUpdateContent       Op = "UpdateContent"
	OpUpdateLabel         Op = "UpdateLabel"
	OpUpdateTags          Op = "UpdateTags"
	OpSoftDelete          Op = "SoftDelete"
	OpPermanentlyDelete   Op = "PermanentlyDelete"
	OpListTrash           Op = "ListTrash"
	OpRestoreFromTrash    Op = "RestoreFromTrash"
	OpSearch              Op = "Search"
	OpCount               Op = "Count"
	OpListBufferSummaries Op = "ListBufferSummaries"
)

// Request is sent from a client to the daemon.
type Request struct {
	ID      int64           `json:"id"`
	Op      Op              `json:"op"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

// Response is sent from the daemon back to the client.
type Response struct {
	ID      int64           `json:"id"`
	Ok      bool            `json:"ok"`
	Payload json.RawMessage `json:"payload,omitempty"`
	Error   string          `json:"error,omitempty"`
}

// ── Request payloads ──────────────────────────────────────────────

type CreateBufferPayload struct {
	Content string   `json:"content"`
	Label   string   `json:"label,omitempty"`
	Tags    []string `json:"tags,omitempty"`
}

type IDPayload struct {
	ID int64 `json:"id"`
}

type ListBuffersPayload struct {
	Keyword string `json:"keyword,omitempty"`
	Since   string `json:"since,omitempty"`
	Until   string `json:"until,omitempty"`
	Limit   int    `json:"limit,omitempty"`
	Offset  int    `json:"offset,omitempty"`
	SortBy  string `json:"sort_by,omitempty"`
	SortAsc bool   `json:"sort_asc,omitempty"`
}

type UpdateContentPayload struct {
	ID      int64  `json:"id"`
	Content string `json:"content"`
}

type UpdateLabelPayload struct {
	ID    int64  `json:"id"`
	Label string `json:"label"`
}

type UpdateTagsPayload struct {
	ID   int64    `json:"id"`
	Tags []string `json:"tags"`
}

type SoftDeletePayload struct {
	ID         int64 `json:"id"`
	TTLSeconds int   `json:"ttl_seconds,omitempty"`
}

type SearchPayload struct {
	Query string `json:"query"`
	Mode  string `json:"mode"` // "fuzzy", "literal", or "regex"
}

// ── Response payloads ─────────────────────────────────────────────

type PingResponse struct {
	Message string `json:"message"`
}

type IDResponse struct {
	ID int64 `json:"id"`
}

type CountResponse struct {
	Count int `json:"count"`
}

// ── Helpers ───────────────────────────────────────────────────────

// NewRequest builds a Request with the given op and payload. If payload
// is nil the Payload field is left empty (omitted from JSON).
func NewRequest(id int64, op Op, payload any) Request {
	req := Request{ID: id, Op: op}

	if payload != nil {
		raw, _ := json.Marshal(payload)
		req.Payload = raw
	}

	return req
}

// OKResponse builds a success Response. If payload is nil the Payload
// field is left empty.
func OKResponse(id int64, payload any) Response {
	resp := Response{ID: id, Ok: true}

	if payload != nil {
		raw, _ := json.Marshal(payload)
		resp.Payload = raw
	}

	return resp
}

// ErrorResponse builds an error Response.
func ErrorResponse(id int64, msg string) Response {
	return Response{ID: id, Ok: false, Error: msg}
}

// UnmarshalPayload decodes the response payload into v.
func (r *Response) UnmarshalPayload(v any) error {
	return json.Unmarshal(r.Payload, v)
}
