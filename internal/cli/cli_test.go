package cli_test

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/szdytom/tb/internal/cli"
	"github.com/szdytom/tb/internal/config"
	"github.com/szdytom/tb/internal/daemon"
	"github.com/szdytom/tb/internal/ipc"
)

// setupTestServer starts a daemon for testing and returns cleanup.
// It sets XDG_* env vars so config.Default() resolves to the temp dir.
func setupTestServer(t *testing.T) func() {
	t.Helper()

	dir, err := os.MkdirTemp("", "tmpbuffer-cli-test-*")
	if err != nil {
		t.Fatal(err)
	}

	origData := os.Getenv("XDG_DATA_HOME")
	origState := os.Getenv("XDG_STATE_HOME")
	origConfig := os.Getenv("XDG_CONFIG_HOME")

	os.Setenv("XDG_DATA_HOME", dir)
	os.Setenv("XDG_STATE_HOME", dir)
	os.Setenv("XDG_CONFIG_HOME", dir)

	// config.Default() resolves paths using XDG env vars, so we set them
	// before creating the config the daemon will use.
	cfg := config.Default()
	if err := cfg.EnsureDirs(); err != nil {
		os.RemoveAll(dir)
		t.Fatalf("ensure dirs: %v", err)
	}
	d, err := daemon.New(cfg)
	if err != nil {
		os.RemoveAll(dir)
		t.Fatalf("new daemon: %v", err)
	}

	if err := d.Serve(); err != nil {
		d.Shutdown()
		os.RemoveAll(dir)
		t.Fatalf("serve: %v", err)
	}

	// Wait for daemon to be ready and close the test connection.
	readyConn, err := ipc.Dial(cfg.SocketPath, 5*time.Second)
	if err != nil {
		d.Shutdown()
		os.RemoveAll(dir)
		t.Fatalf("dial daemon: %v", err)
	}
	readyConn.Close()

	cleanup := func() {
		d.Shutdown()
		os.RemoveAll(dir)
		os.Setenv("XDG_DATA_HOME", origData)
		os.Setenv("XDG_STATE_HOME", origState)
		os.Setenv("XDG_CONFIG_HOME", origConfig)
	}

	return cleanup
}

func TestVersion(t *testing.T) {
	out := captureStdout(func() {
		cli.Execute([]string{"version"})
	})
	if !strings.Contains(out, "tb version") {
		t.Errorf("version output = %q, want 'tb version'", out)
	}
}

func TestAddGet(t *testing.T) {
	defer setupTestServer(t)()

	out := captureStdout(func() {
		cli.Execute([]string{"add", "--text", "hello world", "--label", "test"})
	})
	idStr := strings.TrimSpace(out)
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		t.Fatalf("add output = %q, want numeric ID", idStr)
	}
	if id == 0 {
		t.Fatal("expected non-zero ID")
	}

	out = captureStdout(func() {
		cli.Execute([]string{"get", idStr})
	})
	if !strings.Contains(out, "hello world") {
		t.Errorf("get output = %q, want 'hello world'", out)
	}

	out = captureStdout(func() {
		cli.Execute([]string{"get", "--json", idStr})
	})
	if !strings.Contains(out, `"label": "test"`) {
		t.Errorf("get --json = %q, want label 'test'", out)
	}
}

func TestAddWithTags(t *testing.T) {
	defer setupTestServer(t)()

	out := captureStdout(func() {
		cli.Execute([]string{"add", "--text", "tagged", "--label", "mytag", "--tag", "a,b"})
	})
	idStr := strings.TrimSpace(out)

	out = captureStdout(func() {
		cli.Execute([]string{"get", "--json", idStr})
	})
	if !strings.Contains(out, `"a"`) || !strings.Contains(out, `"b"`) {
		t.Errorf("get --json = %q, want tags [a b]", out)
	}
}

func TestAddStdin(t *testing.T) {
	defer setupTestServer(t)()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		w.Write([]byte("from stdin"))
		w.Close()
	}()

	oldStdin := os.Stdin
	os.Stdin = r

	out := captureStdout(func() {
		cli.Execute([]string{"add", "--label", "stdin-test"})
	})
	idStr := strings.TrimSpace(out)

	os.Stdin = oldStdin

	out = captureStdout(func() {
		cli.Execute([]string{"get", idStr})
	})
	if !strings.Contains(out, "from stdin") {
		t.Errorf("get = %q, want 'from stdin'", out)
	}
}

func TestGetNotFound(t *testing.T) {
	defer setupTestServer(t)()

	code := cli.Execute([]string{"get", "99999"})
	if code == 0 {
		t.Fatal("expected non-zero exit for missing buffer")
	}
}

func TestList(t *testing.T) {
	defer setupTestServer(t)()

	cli.Execute([]string{"add", "--text", "apple"})
	cli.Execute([]string{"add", "--text", "banana"})

	out := captureStdout(func() {
		cli.Execute([]string{"list"})
	})
	if !strings.Contains(out, "apple") || !strings.Contains(out, "banana") {
		t.Errorf("list output = %q, want both 'apple' and 'banana'", out)
	}
}

func TestListJSON(t *testing.T) {
	defer setupTestServer(t)()

	cli.Execute([]string{"add", "--text", "json-test"})

	out := captureStdout(func() {
		cli.Execute([]string{"list", "--json"})
	})
	if !strings.Contains(out, `"content": "json-test"`) {
		t.Errorf("list --json = %q, want content field", out)
	}
}

func TestListFilter(t *testing.T) {
	defer setupTestServer(t)()

	cli.Execute([]string{"add", "--text", "apple banana"})
	cli.Execute([]string{"add", "--text", "cherry"})

	out := captureStdout(func() {
		cli.Execute([]string{"list", "--filter", "banana"})
	})
	if !strings.Contains(out, "apple") {
		t.Errorf("list --filter banana = %q, want 'apple'", out)
	}
	if strings.Contains(out, "cherry") {
		t.Errorf("list --filter banana = %q, should NOT contain 'cherry'", out)
	}
}

func TestSearch(t *testing.T) {
	defer setupTestServer(t)()

	cli.Execute([]string{"add", "--text", "error: timeout occurred"})
	cli.Execute([]string{"add", "--text", "everything is fine"})

	out := captureStdout(func() {
		cli.Execute([]string{"search", "error"})
	})
	if !strings.Contains(out, "error: timeout") {
		t.Errorf("search output = %q, want snippet with 'error: timeout'", out)
	}
	if strings.Contains(out, "everything is fine") {
		t.Errorf("search should not match 'everything is fine', got %q", out)
	}
}

func TestSearchRegex(t *testing.T) {
	defer setupTestServer(t)()

	cli.Execute([]string{"add", "--text", "error: timeout"})
	cli.Execute([]string{"add", "--text", "warning: slow"})

	out := captureStdout(func() {
		cli.Execute([]string{"search", "--regex", "error.*time"})
	})
	if !strings.Contains(out, "error: timeout") {
		t.Errorf("search --regex output = %q, want 'error: timeout'", out)
	}
}

func TestSearchJSON(t *testing.T) {
	defer setupTestServer(t)()

	cli.Execute([]string{"add", "--text", "searchable content"})

	out := captureStdout(func() {
		cli.Execute([]string{"search", "--json", "searchable"})
	})
	if !strings.Contains(out, `"snippet"`) {
		t.Errorf("search --json = %q, want snippet field", out)
	}
}

func TestRm(t *testing.T) {
	defer setupTestServer(t)()

	out := captureStdout(func() {
		cli.Execute([]string{"add", "--text", "delete me"})
	})
	idStr := strings.TrimSpace(out)

	cli.Execute([]string{"rm", idStr})

	out = captureStdout(func() {
		cli.Execute([]string{"list"})
	})
	if strings.Contains(out, idStr) {
		t.Errorf("list after rm contains deleted buffer ID %s", idStr)
	}
}

func TestRmPermanent(t *testing.T) {
	defer setupTestServer(t)()

	out := captureStdout(func() {
		cli.Execute([]string{"add", "--text", "delete permanently"})
	})
	idStr := strings.TrimSpace(out)

	cli.Execute([]string{"rm", "--permanent", idStr})

	code := cli.Execute([]string{"get", idStr})
	if code == 0 {
		t.Fatal("expected non-zero exit for permanently deleted buffer")
	}
}

func TestPipe(t *testing.T) {
	defer setupTestServer(t)()

	out := captureStdout(func() {
		cli.Execute([]string{"add", "--text", "hello world"})
	})
	idStr := strings.TrimSpace(out)

	// Working directory fix: pipe through tr -d '\n' to avoid newline-only args issue
	cli.Execute([]string{"pipe", idStr, "--command", "tr 'a-z' 'A-Z'"})

	out = captureStdout(func() {
		cli.Execute([]string{"get", idStr})
	})
	if !strings.Contains(out, "HELLO WORLD") {
		t.Errorf("pipe result = %q, want 'HELLO WORLD'", out)
	}
}

func TestPipeNew(t *testing.T) {
	defer setupTestServer(t)()

	out := captureStdout(func() {
		cli.Execute([]string{"add", "--text", "hello"})
	})
	idStr := strings.TrimSpace(out)

	out = captureStdout(func() {
		cli.Execute([]string{"pipe", idStr, "--command", "wc -c", "--new"})
	})
	newIDStr := strings.TrimSpace(out)

	if newIDStr == idStr {
		t.Fatal("--new should create a different buffer")
	}

	out = captureStdout(func() {
		cli.Execute([]string{"get", idStr})
	})
	if !strings.Contains(out, "hello") {
		t.Errorf("original should be 'hello', got %q", out)
	}
}

func TestEditPreservesContent(t *testing.T) {
	defer setupTestServer(t)()

	out := captureStdout(func() {
		cli.Execute([]string{"add", "--text", "original"})
	})
	idStr := strings.TrimSpace(out)

	cli.Execute([]string{"edit", idStr, "--editor", "cat"})

	out = captureStdout(func() {
		cli.Execute([]string{"get", idStr})
	})
	if !strings.Contains(out, "original") {
		t.Errorf("after edit with cat, content = %q, want 'original'", out)
	}
}

func TestCount(t *testing.T) {
	defer setupTestServer(t)()

	for i := 0; i < 3; i++ {
		cli.Execute([]string{"add", "--text", fmt.Sprintf("buf %d", i)})
	}

	out := captureStdout(func() {
		cli.Execute([]string{"list", "--json"})
	})
	count := strings.Count(out, `"id":`)
	if count != 3 {
		t.Errorf("expected 3 buffers, got %d", count)
	}
}

func TestHelp(t *testing.T) {
	out := captureStdout(func() {
		cli.Execute([]string{"--help"})
	})
	if !strings.Contains(out, "tmpbuffer") {
		t.Errorf("help output = %q, want 'tmpbuffer'", out)
	}
}

// ── Test helpers ──────────────────────────────────────────────────────

// captureStdout runs f and returns everything written to stdout.
func captureStdout(f func()) string {
	r, w, err := os.Pipe()
	if err != nil {
		panic(err)
	}
	orig := os.Stdout
	os.Stdout = w

	f()

	w.Close()
	os.Stdout = orig

	var buf strings.Builder
	io.Copy(&buf, r)
	return buf.String()
}
