package daemon

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/szdytom/tb/internal/buffer"
	"github.com/szdytom/tb/internal/config"
	"github.com/szdytom/tb/internal/ipc"
	"github.com/szdytom/tb/internal/store"
)

func setupTestDaemon(t *testing.T) (*Daemon, *ipc.Conn, func()) {
	t.Helper()

	dir, err := os.MkdirTemp("", "tmpbuffer-daemon-test-*")
	if err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{
		DataDir:    dir,
		ConfigDir:  dir,
		SocketDir:  dir,
		DBPath:     filepath.Join(dir, "test.db"),
		SocketPath: filepath.Join(dir, "test.sock"),
	}

	d, err := New(cfg)
	if err != nil {
		os.RemoveAll(dir)
		t.Fatalf("new daemon: %v", err)
	}

	if err := d.Serve(); err != nil {
		d.Shutdown()
		os.RemoveAll(dir)
		t.Fatalf("serve: %v", err)
	}

	// Give the server a moment to start listening.
	conn, err := ipc.Dial(cfg.SocketPath, 5*time.Second)
	if err != nil {
		d.Shutdown()
		os.RemoveAll(dir)
		t.Fatalf("dial: %v", err)
	}

	cleanup := func() {
		conn.Close()
		d.Shutdown()
		os.RemoveAll(dir)
	}

	return d, conn, cleanup
}

func sendRequest(t *testing.T, conn *ipc.Conn, req ipc.Request) ipc.Response {
	t.Helper()
	if err := conn.Send(req); err != nil {
		t.Fatalf("send: %v", err)
	}
	var resp ipc.Response
	if err := conn.Receive(&resp); err != nil {
		t.Fatalf("receive: %v", err)
	}
	return resp
}

func TestPing(t *testing.T) {
	_, conn, cleanup := setupTestDaemon(t)
	defer cleanup()

	resp := sendRequest(t, conn, ipc.NewRequest(1, ipc.OpPing, nil))
	if !resp.Ok {
		t.Fatalf("Ping failed: %s", resp.Error)
	}
}

func TestCreateAndGetBuffer(t *testing.T) {
	_, conn, cleanup := setupTestDaemon(t)
	defer cleanup()

	// Create
	resp := sendRequest(t, conn, ipc.NewRequest(1, ipc.OpCreateBuffer, ipc.CreateBufferPayload{
		Content: "hello world",
		Label:   "test",
		Tags:    []string{"a", "b"},
	}))
	if !resp.Ok {
		t.Fatalf("CreateBuffer failed: %s", resp.Error)
	}
	var idResp ipc.IDResponse
	if err := resp.UnmarshalPayload(&idResp); err != nil {
		t.Fatal(err)
	}
	if idResp.ID == 0 {
		t.Fatal("expected non-zero ID")
	}

	// Get
	resp = sendRequest(t, conn, ipc.NewRequest(2, ipc.OpGetBuffer, ipc.IDPayload{ID: idResp.ID}))
	if !resp.Ok {
		t.Fatalf("GetBuffer failed: %s", resp.Error)
	}
	var buf buffer.Buffer
	if err := resp.UnmarshalPayload(&buf); err != nil {
		t.Fatal(err)
	}
	if buf.Content != "hello world" || buf.Label != "test" {
		t.Errorf("got content=%q label=%q, want content=hello world label=test", buf.Content, buf.Label)
	}
	if len(buf.Tags) != 2 {
		t.Errorf("got %d tags, want 2", len(buf.Tags))
	}
}

func TestGetBufferNotFound(t *testing.T) {
	_, conn, cleanup := setupTestDaemon(t)
	defer cleanup()

	resp := sendRequest(t, conn, ipc.NewRequest(1, ipc.OpGetBuffer, ipc.IDPayload{ID: 999}))
	if resp.Ok {
		t.Fatal("expected error for non-existent buffer")
	}
}

func TestListBuffers(t *testing.T) {
	_, conn, cleanup := setupTestDaemon(t)
	defer cleanup()

	// Create 3 buffers
	for i := 0; i < 3; i++ {
		resp := sendRequest(t, conn, ipc.NewRequest(int64(i+1), ipc.OpCreateBuffer, ipc.CreateBufferPayload{
			Content: "content",
		}))
		if !resp.Ok {
			t.Fatalf("create %d: %s", i, resp.Error)
		}
	}

	// List all
	resp := sendRequest(t, conn, ipc.NewRequest(10, ipc.OpListBuffers, ipc.ListBuffersPayload{}))
	if !resp.Ok {
		t.Fatalf("ListBuffers failed: %s", resp.Error)
	}
	var bufs []buffer.Buffer
	if err := resp.UnmarshalPayload(&bufs); err != nil {
		t.Fatal(err)
	}
	if len(bufs) != 3 {
		t.Fatalf("got %d buffers, want 3", len(bufs))
	}
}

func TestListBuffersWithKeyword(t *testing.T) {
	_, conn, cleanup := setupTestDaemon(t)
	defer cleanup()

	sendRequest(t, conn, ipc.NewRequest(1, ipc.OpCreateBuffer, ipc.CreateBufferPayload{Content: "apple banana"}))
	sendRequest(t, conn, ipc.NewRequest(2, ipc.OpCreateBuffer, ipc.CreateBufferPayload{Content: "banana cherry"}))
	sendRequest(t, conn, ipc.NewRequest(3, ipc.OpCreateBuffer, ipc.CreateBufferPayload{Content: "date"}))

	resp := sendRequest(t, conn, ipc.NewRequest(10, ipc.OpListBuffers, ipc.ListBuffersPayload{
		Keyword: "banana",
	}))
	if !resp.Ok {
		t.Fatalf("ListBuffers failed: %s", resp.Error)
	}
	var bufs []buffer.Buffer
	if err := resp.UnmarshalPayload(&bufs); err != nil {
		t.Fatal(err)
	}
	if len(bufs) != 2 {
		t.Fatalf("got %d buffers for keyword 'banana', want 2", len(bufs))
	}
}

func TestUpdateContent(t *testing.T) {
	_, conn, cleanup := setupTestDaemon(t)
	defer cleanup()

	resp := sendRequest(t, conn, ipc.NewRequest(1, ipc.OpCreateBuffer, ipc.CreateBufferPayload{Content: "original"}))
	var idResp ipc.IDResponse
	resp.UnmarshalPayload(&idResp)

	resp = sendRequest(t, conn, ipc.NewRequest(2, ipc.OpUpdateContent, ipc.UpdateContentPayload{
		ID:      idResp.ID,
		Content: "modified",
	}))
	if !resp.Ok {
		t.Fatalf("UpdateContent failed: %s", resp.Error)
	}

	resp = sendRequest(t, conn, ipc.NewRequest(3, ipc.OpGetBuffer, ipc.IDPayload{ID: idResp.ID}))
	var buf buffer.Buffer
	resp.UnmarshalPayload(&buf)
	if buf.Content != "modified" {
		t.Errorf("Content = %q, want %q", buf.Content, "modified")
	}
}

func TestUpdateLabelAndTags(t *testing.T) {
	_, conn, cleanup := setupTestDaemon(t)
	defer cleanup()

	resp := sendRequest(t, conn, ipc.NewRequest(1, ipc.OpCreateBuffer, ipc.CreateBufferPayload{
		Content: "c", Label: "old", Tags: []string{"a"},
	}))
	var idResp ipc.IDResponse
	resp.UnmarshalPayload(&idResp)

	sendRequest(t, conn, ipc.NewRequest(2, ipc.OpUpdateLabel, ipc.UpdateLabelPayload{ID: idResp.ID, Label: "new"}))
	sendRequest(t, conn, ipc.NewRequest(3, ipc.OpUpdateTags, ipc.UpdateTagsPayload{ID: idResp.ID, Tags: []string{"x", "y"}}))

	resp = sendRequest(t, conn, ipc.NewRequest(4, ipc.OpGetBuffer, ipc.IDPayload{ID: idResp.ID}))
	var buf buffer.Buffer
	resp.UnmarshalPayload(&buf)
	if buf.Label != "new" {
		t.Errorf("Label = %q, want %q", buf.Label, "new")
	}
	if len(buf.Tags) != 2 || buf.Tags[0] != "x" {
		t.Errorf("Tags = %v, want [x y]", buf.Tags)
	}
}

func TestSoftDeleteAndRestore(t *testing.T) {
	_, conn, cleanup := setupTestDaemon(t)
	defer cleanup()

	resp := sendRequest(t, conn, ipc.NewRequest(1, ipc.OpCreateBuffer, ipc.CreateBufferPayload{Content: "c"}))
	var idResp ipc.IDResponse
	resp.UnmarshalPayload(&idResp)
	id := idResp.ID

	// Soft delete
	resp = sendRequest(t, conn, ipc.NewRequest(2, ipc.OpSoftDelete, ipc.SoftDeletePayload{ID: id}))
	if !resp.Ok {
		t.Fatalf("SoftDelete failed: %s", resp.Error)
	}

	// Active list should be empty
	resp = sendRequest(t, conn, ipc.NewRequest(3, ipc.OpListBuffers, ipc.ListBuffersPayload{}))
	var bufs []buffer.Buffer
	resp.UnmarshalPayload(&bufs)
	if len(bufs) != 0 {
		t.Errorf("expected 0 active buffers, got %d", len(bufs))
	}

	// Trash list should have it
	resp = sendRequest(t, conn, ipc.NewRequest(4, ipc.OpListTrash, nil))
	var trash []buffer.Buffer
	resp.UnmarshalPayload(&trash)
	if len(trash) != 1 || trash[0].ID != id {
		t.Errorf("expected 1 trashed buffer with ID %d, got %d items", id, len(trash))
	}

	// Restore
	resp = sendRequest(t, conn, ipc.NewRequest(5, ipc.OpRestoreFromTrash, ipc.IDPayload{ID: id}))
	if !resp.Ok {
		t.Fatalf("RestoreFromTrash failed: %s", resp.Error)
	}

	resp = sendRequest(t, conn, ipc.NewRequest(6, ipc.OpListBuffers, ipc.ListBuffersPayload{}))
	resp.UnmarshalPayload(&bufs)
	if len(bufs) != 1 {
		t.Errorf("expected 1 buffer after restore, got %d", len(bufs))
	}
}

func TestPermanentlyDelete(t *testing.T) {
	_, conn, cleanup := setupTestDaemon(t)
	defer cleanup()

	resp := sendRequest(t, conn, ipc.NewRequest(1, ipc.OpCreateBuffer, ipc.CreateBufferPayload{Content: "c"}))
	var idResp ipc.IDResponse
	resp.UnmarshalPayload(&idResp)

	resp = sendRequest(t, conn, ipc.NewRequest(2, ipc.OpPermanentlyDelete, ipc.IDPayload{ID: idResp.ID}))
	if !resp.Ok {
		t.Fatalf("PermanentlyDelete failed: %s", resp.Error)
	}

	resp = sendRequest(t, conn, ipc.NewRequest(3, ipc.OpGetBuffer, ipc.IDPayload{ID: idResp.ID}))
	if resp.Ok {
		t.Fatal("expected error for deleted buffer")
	}
}

func TestSearch(t *testing.T) {
	_, conn, cleanup := setupTestDaemon(t)
	defer cleanup()

	sendRequest(t, conn, ipc.NewRequest(1, ipc.OpCreateBuffer, ipc.CreateBufferPayload{Content: "error: timeout occurred"}))
	sendRequest(t, conn, ipc.NewRequest(2, ipc.OpCreateBuffer, ipc.CreateBufferPayload{Content: "everything is fine"}))

	resp := sendRequest(t, conn, ipc.NewRequest(10, ipc.OpSearch, ipc.SearchPayload{Query: "error"}))
	if !resp.Ok {
		t.Fatalf("Search failed: %s", resp.Error)
	}
	var results []store.SearchResult
	if err := resp.UnmarshalPayload(&results); err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
	if results[0].Buffer.Content != "error: timeout occurred" {
		t.Errorf("result content = %q, want %q", results[0].Buffer.Content, "error: timeout occurred")
	}
}

func TestSearchRegex(t *testing.T) {
	_, conn, cleanup := setupTestDaemon(t)
	defer cleanup()

	sendRequest(t, conn, ipc.NewRequest(1, ipc.OpCreateBuffer, ipc.CreateBufferPayload{Content: "error: timeout"}))
	sendRequest(t, conn, ipc.NewRequest(2, ipc.OpCreateBuffer, ipc.CreateBufferPayload{Content: "warning: slow"}))

	resp := sendRequest(t, conn, ipc.NewRequest(10, ipc.OpSearch, ipc.SearchPayload{
		Query:   "error.*time",
		IsRegex: true,
	}))
	if !resp.Ok {
		t.Fatalf("Search failed: %s", resp.Error)
	}
	var results []store.SearchResult
	resp.UnmarshalPayload(&results)
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
}

func TestCount(t *testing.T) {
	_, conn, cleanup := setupTestDaemon(t)
	defer cleanup()

	resp := sendRequest(t, conn, ipc.NewRequest(1, ipc.OpCount, nil))
	var countResp ipc.CountResponse
	if err := resp.UnmarshalPayload(&countResp); err != nil {
		t.Fatal(err)
	}
	if countResp.Count != 0 {
		t.Errorf("initial count = %d, want 0", countResp.Count)
	}

	sendRequest(t, conn, ipc.NewRequest(2, ipc.OpCreateBuffer, ipc.CreateBufferPayload{Content: "a"}))
	sendRequest(t, conn, ipc.NewRequest(3, ipc.OpCreateBuffer, ipc.CreateBufferPayload{Content: "b"}))

	resp = sendRequest(t, conn, ipc.NewRequest(4, ipc.OpCount, nil))
	resp.UnmarshalPayload(&countResp)
	if countResp.Count != 2 {
		t.Errorf("count = %d, want 2", countResp.Count)
	}
}

func TestUnknownOp(t *testing.T) {
	_, conn, cleanup := setupTestDaemon(t)
	defer cleanup()

	req := ipc.Request{ID: 1, Op: "BadOp"}
	resp := sendRequest(t, conn, req)
	if resp.Ok {
		t.Fatal("expected error for unknown operation")
	}
}

func TestConsecutiveRequests(t *testing.T) {
	_, conn, cleanup := setupTestDaemon(t)
	defer cleanup()

	// Send two requests and match responses by ID.
	if err := conn.Send(ipc.NewRequest(10, ipc.OpCount, nil)); err != nil {
		t.Fatal(err)
	}
	if err := conn.Send(ipc.NewRequest(20, ipc.OpPing, nil)); err != nil {
		t.Fatal(err)
	}

	var r1, r2 ipc.Response
	if err := conn.Receive(&r1); err != nil {
		t.Fatal(err)
	}
	if err := conn.Receive(&r2); err != nil {
		t.Fatal(err)
	}

	if r1.ID != 10 || r2.ID != 20 {
		t.Errorf("response order: got IDs %d and %d, want 10 then 20", r1.ID, r2.ID)
	}
}

func TestMultipleConnections(t *testing.T) {
	d, first, cleanup := setupTestDaemon(t)
	defer cleanup()

	// Open a second connection.
	second, err := ipc.Dial(d.cfg.SocketPath, 5*time.Second)
	if err != nil {
		t.Fatalf("second dial: %v", err)
	}
	defer second.Close()

	// Both connections should work independently.
	resp1 := sendRequest(t, first, ipc.NewRequest(1, ipc.OpPing, nil))
	resp2 := sendRequest(t, second, ipc.NewRequest(1, ipc.OpPing, nil))

	if !resp1.Ok || !resp2.Ok {
		t.Fatal("both connections should succeed")
	}
}
