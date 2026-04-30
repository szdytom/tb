# tmpbuffer (tb) – Phase 1 Implementation Plan

**Language:** Go 1.22+  
**Status:** Implementation Plan (Draft)  
**Based on:** spec-phase1.md v1.0  

---

## 1. Tech Stack

| Layer | Choice | Rationale |
|---|---|---|
| Language | Go 1.22+ | Single-binary distribution, static linking, excellent stdlib for IPC/CLI, goroutine-based daemon concurrency |
| Storage | SQLite via `modernc.org/sqlite` (pure-Go, no CGO) or `mattn/go-sqlite3` (CGO) | WAL mode, zero-bloat, embedded. Prefer pure-Go variant to avoid CGO cross-compilation issues |
| TUI | `github.com/charmbracelet/bubbletea` + `bubbles` | Elm-architecture, lightweight, active ecosystem; natural fit for split-pane TUI with keyboard-driven interaction |
| IPC | Unix domain socket (stdlib `net.Listen` + `encoding/json`) | Documented constraint; Go stdlib makes UDS trivial |
| Config | `github.com/BurntSushi/toml` | Matches requirement (TOML); mature, no-frills |
| CLI | `github.com/spf13/cobra` | De facto standard for Go CLI; subcommand support, help generation |
| Testing | `testing` stdlib + `github.com/stretchr/testify` | Standard; testify/assert for readability |

### Key design constraints

- **Zero runtime dependencies beyond libc** (or fully static with `modernc.org/sqlite`).
- **No network listeners** (NFR-7). UDS only.
- **No external DB processes.** SQLite is linked into the daemon binary.
- **`$XDG_DATA_HOME`/`$XDG_CONFIG_HOME`/`$XDG_STATE_HOME`** convention for file layout.

---

## 2. Directory Layout

```
tb/
├── cmd/                    # Entry points
│   ├── tb/                 # Client CLI binary
│   │   └── main.go
│   └── tmpbufferd/         # Standalone daemon binary (optional)
│       └── main.go
├── internal/               # Not exported; all business logic
│   ├── buffer/             # Buffer data model, CRUD, validation
│   ├── store/              # SQLite persistence layer (repository)
│   ├── daemon/             # Daemon lifecycle, IPC server, request routing
│   ├── ipc/                # Shared IPC protocol (message types, UDS helpers)
│   ├── tui/                # Bubbletea TUI: model, views, keybindings, search
│   ├── editor/             # External editor invocation & temp file mgmt
│   ├── pipe/               # Pipeline: shell command execution, output capture
│   ├── config/             # TOML config loading, defaults, XDG paths
│   └── cli/                # Cobra command tree & flag wiring
├── go.mod
├── go.sum
└── docs/
    ├── spec-phase1.md
    └── plan-phase1.md      # ← this file
```

### Why `internal/`?

Per NFR-15 (future extensibility), sealing the core as `internal/` means future frontends (GUI, AI plugin) import a clean public API surface, while internal packages can be refactored freely.

---

## 3. Macro-Level Implementation Steps

Each step produces a working, incrementally testable artifact. Steps are ordered by dependency — later steps depend on earlier ones.

### Step 1 — Project Scaffolding & Data Model

**Goal:** Go module initialized, directory tree created, buffer struct and SQLite schema defined, config file paths resolved, empty daemon skeleton that starts/stops cleanly.

**Artifacts:**
- `go.mod`, directory structure
- `internal/buffer/model.go` — `Buffer` struct, `Metadata` struct, `TrashStatus` enum
- `internal/store/schema.go` — SQLite DDL (migration), WAL PRAGMAs
- `internal/store/db.go` — Open/close, connection pooling
- `internal/config/paths.go` — XDG path resolution, defaults
- `internal/config/config.go` — Config struct with shell defaults
- `internal/daemon/daemon.go` — Start/stop skeleton (no IPC yet)

**Acceptance:** `go build ./...` succeeds; daemon binary starts, creates DB file, and exits cleanly on SIGTERM.

---

### Step 2 — Storage Layer (CRUD)

**Goal:** Full buffer persistence — create, read, update, delete (soft-delete to trash), list with sorting, count stats.

**Artifacts:**
- `internal/store/buffer_repo.go` — Insert, Get, List (with filter/sort/pagination), UpdateContent, SoftDelete, PermanentlyDelete, ListTrash, RestoreFromTrash
- `internal/store/metadata.go` — Line count, byte count computation at write time
- `internal/store/search.go` — Full-text search (LIKE for literal, `REGEXP` for regex), snippet extraction
- Migration: add `trash_expires_at` column, auto-purge goroutine in daemon

**Acceptance:** Unit tests for each CRUD path with an in-memory SQLite DB; verify WAL mode is active; verify search returns correct matches for literal and regex.

---

### Step 3 — IPC Protocol & Daemon Server Loop

**Goal:** Daemon listens on UDS (`$XDG_RUNTIME_DIR/tmpbuffer.sock`), accepts JSON-encoded requests, dispatches to store layer, returns JSON responses.

**Artifacts:**
- `internal/ipc/msg.go` — Request/Response structs, `OpCode` enum
- `internal/ipc/conn.go` — `SendMsg`/`RecvMsg` helpers (JSON lines over UDS)
- `internal/daemon/server.go` — `net.Listen("unix", ...)`, accept loop, goroutine-per-conn, request router
- `internal/daemon/handlers.go` — Wire IPC ops to `store.*` calls
- `internal/daemon/autostart.go` — Client-side: connect to UDS, if fail → fork daemon, retry

**Acceptance:** Manual test: start daemon, `socat` or Go test client sends JSON request, receives response. Client-side auto-start: `tb list` with daemon dead auto-launches it.

---

### Step 4 — CLI Command Tree (All Commands)

**Goal:** `tb` binary with all subcommands working against the daemon via UDS. No TUI in this step.

**Artifacts:**
- `cmd/tb/main.go` — Root command, auto-start daemon, dispatch
- `internal/cli/root.go` — Cobra root command, global flags
- `internal/cli/add.go` — `tb add`, stdin/`--text`/`--label`/`--tag`
- `internal/cli/list.go` — `tb list`, filtering/formatting flags
- `internal/cli/get.go` — `tb get <id>`
- `internal/cli/search.go` — `tb search <query>`
- `internal/cli/edit.go` — `tb edit <id>`
- `internal/cli/rm.go` — `tb rm <id> [--permanent]`
- `internal/cli/pipe.go` — `tb pipe <id> --command <cmd> [--new]`
- `internal/cli/daemon.go` — `tb daemon start|stop|status`
- `internal/cli/version.go` — `tb version`

**Acceptance:** Every AC that involves CLI (AC-4) passes. Shell pipelines work: `echo "x" | tb add --label test && tb get $(tb list --json | jq -r '.[0].id')`.

---

### Step 5 — TUI: Basic Layout & Navigation

**Goal:** Launchable TUI showing buffer list in left pane, preview in right pane. Keyboard navigation works. Quit is clean.

**Artifacts:**
- `internal/tui/model.go` — Bubbletea `Model` struct, `Init`/`Update`/`View`
- `internal/tui/buffer_list.go` — List view: ID, timestamp, first-line preview, label
- `internal/tui/preview.go` — Scrollable read-only preview, soft-wrap, line numbers
- `internal/tui/keymap.go` — Default keybindings (j/k, arrows, Enter, n, d, /, :q, ?)
- `internal/tui/help.go` — Help modal (? key)
- `internal/tui/update.go` — Message routing, buffer state management
- Wire into `cmd/tb/main.go` as default command

**Acceptance:** AC-1 (instant buffer creation via `n`), AC-7 (200ms startup with 10k buffers — test with DB seeded with 10k rows).

---

### Step 6 — TUI: Search & Filter

**Goal:** Inline search triggered by `/`, real-time filtering, literal/regex mode.

**Artifacts:**
- `internal/tui/search.go` — Search prompt model, debounced input
- Wire to daemon search endpoint; update buffer list on results
- Regex detection (prefix `~`)
- Clear search restores full list

**Acceptance:** AC-6 (regex search) passes.

---

### Step 7 — External Editor Integration

**Goal:** `Enter` in TUI (and `tb edit <id>` in CLI) opens `$EDITOR` on a temp file, reads it back on editor exit, handles non-zero exit.

**Artifacts:**
- `internal/editor/editor.go` — Resolve editor command ($EDITOR/$VISUAL/config), temp file creation, process execution, content read-back
- `internal/editor/config.go` — Per-extension editor mapping
- Non-zero exit handling: prompt user in TUI
- Conflict warning (last-write-wins) with daemon log

**Acceptance:** AC-3 (editor round-trip), AC-8 (non-zero exit handling) pass.

---

### Step 8 — Pipeline Operations

**Goal:** `!` keybinding in TUI triggers command prompt, pipes buffer content to shell command, shows preview, applies (replace or new buffer).

**Artifacts:**
- `internal/pipe/exec.go` — `sh -c` execution, stdin piping, stdout capture, stderr capture
- `internal/pipe/security.go` — Command string display, confirmation prompt
- `internal/tui/pipe.go` — Command input modal, preview modal, confirm/cancel
- CLI `tb pipe` wired to same logic

**Acceptance:** AC-5 (pipeline operation) passes. Non-zero exit handling (FR-3.5.3) works.

---

### Step 9 — Session Continuity & Resilience

**Goal:** Auto-save on every mutation (≤2s window), crash recovery, trash auto-purge, daemon lifecycle via external tools (systemd).

**Artifacts:**
- Auto-save: every mutation goes directly to SQLite (no in-memory-only window); re-access after crash restores from DB
- Daemon: SIGHUP/SIGTERM graceful shutdown, flush DB, clean up UDS socket file
- Trash auto-purge goroutine (configurable TTL)
- systemd user unit example in repo (`contrib/tmpbufferd.service`)

**Acceptance:** AC-2 (SIGKILL tolerance) passes. `tb daemon status` reports running/stopped correctly.

---

### Step 10 — Polish, Testing & Acceptance

**Goal:** All acceptance criteria verified, edge cases handled, documentation ready.

**Artifacts:**
- Integration tests covering all ACs (1–9)
- Benchmark test: 10k buffer startup time, search latency
- Config file examples
- `--help` output review
- Cross-platform smoke test (Linux + macOS)
- Configuration for NFR-10 (vim-like keybindings default, configurable)
- Terminal resize handling (NFR-14)

**Acceptance:** Green CI; all ACs checked off in test matrix.

---

## 4. Key Architectural Decisions

### Daemon-client model: request-response over UDS

```
CLI/TUI  ── UDS (JSON) ──>  Daemon (SQLite)
   ^                              │
   └──────────────────────────────┘
```

- **Why UDS + JSON and not a Go-internal API?** The daemon survives client restarts. CLI and TUI are separate processes that connect to the daemon. JSON over UDS is debuggable (can inspect with `socat`).
- **Why not gRPC?** Added dependency weight for no benefit at this scale. JSON is trivially inspectable and sufficient.
- **Why not embedded the DB in the client directly?** Simultaneous access from TUI + CLI + pipes would require file-level locking. The daemon serializes writes and avoids SQLite `SQLITE_BUSY`.

### Auto-save strategy

Every buffer mutation (create, edit, pipe) writes to SQLite synchronously within the handler. There is no delayed write-back. This means:
- After a crash, the data loss window is bounded by the time between the handler writing to DB and the OS flushing the WAL (typically < 100ms, well under the 2s requirement).
- No additional goroutine needed for "auto-save"; it's just "save on every operation."

### External editor flow

```
TUI → request "edit buffer X" → daemon writes temp file → launches $EDITOR
  → blocks (goroutine waits on process) → on exit: reads file → updates buffer
  → sends response back to TUI/CLI
```

The blocking happens in a goroutine so the daemon remains responsive to other requests during editing.

### TUI vs CLI code sharing

CLI commands and TUI components both talk to the daemon via the same IPC protocol (`internal/ipc`). There is no shared UI code — the TUI is a Bubbletea app, the CLI is Cobra. But the daemon handlers are shared: `tb add` (CLI) and `n` key (TUI) both send the same IPC `CreateBuffer` request.

---

## 5. Risks & Mitigations

| Risk | Mitigation |
|---|---|
| CGO dependency complicates cross-compilation | Favor `modernc.org/sqlite` (pure Go); benchmark to verify performance is acceptable |
| TUI with 10k buffers exceeds 200ms startup | Lazy-load preview content; list only loads IDs + first line + timestamps; batch fetch |
| Editor blocks daemon's UDS handler | Edit operation runs in its own goroutine; daemon accept loop remains non-blocking |
| SQLite WAL file grows unbounded | Periodic `PRAGMA wal_checkpoint(TRUNCHECK)` after mutations; configurable threshold |
| Conflicting concurrent edits (FR-3.3.3) | Last-write-wins is by design; daemon logs warning. No locking needed |

---

## 6. Up Next (Phase 2 considerations, not in scope)

- Format conversion engine (JSON ↔ CSV ↔ YAML ↔ TOML)
- AI semantic search plugin
- Clipboard monitoring daemon plugin
- Snapshots / time-travel per buffer
- Networking / collaboration (would be a new service, not the daemon)
