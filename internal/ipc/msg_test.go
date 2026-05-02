package ipc_test

import (
	"encoding/json"
	"testing"

	"github.com/szdytom/tb/internal/ipc"
)

func TestRequestMarshal(t *testing.T) {
	req := ipc.NewRequest(1, ipc.OpPing, nil)

	b, err := json.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}

	var decoded ipc.Request
	if err := json.Unmarshal(b, &decoded); err != nil {
		t.Fatal(err)
	}

	if decoded.ID != 1 {
		t.Errorf("ID = %d, want 1", decoded.ID)
	}

	if decoded.Op != ipc.OpPing {
		t.Errorf("Op = %q, want %q", decoded.Op, ipc.OpPing)
	}
}

func TestRequestMarshalWithPayload(t *testing.T) {
	req := ipc.NewRequest(2, ipc.OpCreateBuffer, ipc.CreateBufferPayload{
		Content: "hello",
		Label:   "test",
		Tags:    []string{"a"},
	})

	b, err := json.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}

	var decoded struct {
		ID      int64           `json:"id"`
		Op      ipc.Op          `json:"op"`
		Payload json.RawMessage `json:"payload"`
	}
	if err := json.Unmarshal(b, &decoded); err != nil {
		t.Fatal(err)
	}

	var p ipc.CreateBufferPayload
	if err := json.Unmarshal(decoded.Payload, &p); err != nil {
		t.Fatal(err)
	}

	if p.Content != "hello" || p.Label != "test" || len(p.Tags) != 1 || p.Tags[0] != "a" {
		t.Errorf("payload = %+v, want Content=hello Label=test Tags=[a]", p)
	}
}

func TestResponseMarshalSuccess(t *testing.T) {
	resp := ipc.OKResponse(3, ipc.PingResponse{Message: "pong"})

	b, err := json.Marshal(resp)
	if err != nil {
		t.Fatal(err)
	}

	var decoded struct {
		ID      int64           `json:"id"`
		Ok      bool            `json:"ok"`
		Payload json.RawMessage `json:"payload"`
		Error   string          `json:"error"`
	}
	if err := json.Unmarshal(b, &decoded); err != nil {
		t.Fatal(err)
	}

	if decoded.ID != 3 || !decoded.Ok || decoded.Error != "" {
		t.Errorf("unexpected fields: id=%d ok=%v error=%q", decoded.ID, decoded.Ok, decoded.Error)
	}
}

func TestResponseMarshalError(t *testing.T) {
	resp := ipc.ErrorResponse(4, "something went wrong")

	b, err := json.Marshal(resp)
	if err != nil {
		t.Fatal(err)
	}

	var decoded ipc.Response
	if err := json.Unmarshal(b, &decoded); err != nil {
		t.Fatal(err)
	}

	if decoded.ID != 4 || decoded.Ok || decoded.Error != "something went wrong" {
		t.Errorf("unexpected fields: id=%d ok=%v error=%q", decoded.ID, decoded.Ok, decoded.Error)
	}
}

func TestPayloadRoundTrips(t *testing.T) {
	tests := []struct {
		name    string
		payload any
	}{
		{"CreateBufferPayload", ipc.CreateBufferPayload{Content: "c", Label: "l", Tags: []string{"t"}}},
		{"IDPayload", ipc.IDPayload{ID: 42}},
		{"ListBuffersPayload", ipc.ListBuffersPayload{Keyword: "hi", Limit: 10, SortBy: "label", SortAsc: true}},
		{"UpdateContentPayload", ipc.UpdateContentPayload{ID: 1, Content: "new"}},
		{"UpdateLabelPayload", ipc.UpdateLabelPayload{ID: 1, Label: "x"}},
		{"UpdateTagsPayload", ipc.UpdateTagsPayload{ID: 1, Tags: []string{"x"}}},
		{"SoftDeletePayload", ipc.SoftDeletePayload{ID: 1, TTLSeconds: 3600}},
		{"SearchPayload", ipc.SearchPayload{Query: "hello", Mode: "regex"}},
		{"PingResponse", ipc.PingResponse{Message: "pong"}},
		{"IDResponse", ipc.IDResponse{ID: 5}},
		{"CountResponse", ipc.CountResponse{Count: 3}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, err := json.Marshal(tt.payload)
			if err != nil {
				t.Fatal(err)
			}
			// Verify it unmarshals into the same type
			switch p := tt.payload.(type) {
			case ipc.CreateBufferPayload:
				var v ipc.CreateBufferPayload
				if err := json.Unmarshal(b, &v); err != nil {
					t.Fatal(err)
				}

				if v.Content != p.Content || v.Label != p.Label {
					t.Errorf("got %+v, want %+v", v, p)
				}
			case ipc.IDPayload:
				var v ipc.IDPayload
				json.Unmarshal(b, &v)

				if v.ID != p.ID {
					t.Errorf("got %+v, want %+v", v, p)
				}
			case ipc.CountResponse:
				var v ipc.CountResponse
				json.Unmarshal(b, &v)

				if v.Count != p.Count {
					t.Errorf("got %+v, want %+v", v, p)
				}
			}
		})
	}
}
