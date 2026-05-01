# Progress Report

## Status Overview

Build: ✅ passes (`go build ./...`, `go vet ./...`)
Tests: ✅ all pass (`go test ./...`)

---

## Architecture Change: Bubbletea → Vaxis (2026-05-01)

**Decision:** Replace `github.com/charmbracelet/bubbletea` + custom `internal/vt/` with `git.sr.ht/~rockorager/vaxis`.

**Why:** Building a PTY-based terminal emulator from scratch is too complex. Vaxis's `widgets/term` provides a complete, production-quality terminal emulator (PTY lifecycle, VT500 ANSI parser, double-buffered rendering, window system) off the shelf.

**Impact:**
- Step 5 must be **redone**: vaxis event loop instead of bubbletea Model/Update/View
- `internal/vt/` package eliminated entirely
- Old Step 6 (VT Infrastructure) subsumed into new Step 5
- Steps 7-11 renumbered to 6-10
- Editor tabs and pipeline preview use `term.Model` directly

**See also:** `docs/Creating Terminal Multiplexer with Vaxis.md`, `docs/plan-phase1.md`

---

## Step 1 — Project Scaffolding & Data Model

**Status: COMPLETE**

| Artifact | File | Status |
|---|---|---|
| Go module, directory tree | `go.mod`, `cmd/tb/main.go`, `cmd/tmpbufferd/main.go` | ✅ |
| Buffer struct, Metadata, TrashStatus | `internal/buffer/model.go` | ✅ |
| SQLite DDL + migration framework | `internal/store/schema.go` | ✅ |
| DB open/close, WAL pragmas | `internal/store/db.go` | ✅ |
| XDG path resolution | `internal/config/paths.go` | ✅ |
| Config struct with defaults | `internal/config/config.go` | ✅ |
| Daemon start/stop skeleton | `internal/daemon/daemon.go` | ✅ |



---

## Step 2 — Storage Layer (CRUD)

**Status: COMPLETE**

| Artifact | File | Status |
|---|---|---|
| Insert, Get, List with filter/sort/pagination | `internal/store/buffer_repo.go` | ✅ |
| UpdateContent, UpdateLabel, UpdateTags | `internal/store/buffer_repo.go` | ✅ |
| SoftDelete, PermanentlyDelete | `internal/store/buffer_repo.go` | ✅ |
| ListTrash, RestoreFromTrash | `internal/store/buffer_repo.go` | ✅ |
| Count, DeleteExpiredTrash | `internal/store/buffer_repo.go` | ✅ |
| Line/byte count at write time | `internal/buffer/model.go` (`ComputeMetadata`) | ✅* |
| Literal + regex full-text search | `internal/store/search.go` | ✅ |
| Tests for CRUD + search | `internal/store/buffer_repo_test.go`, `search_test.go` | ✅ |

*\*Planned as `internal/store/metadata.go`; implemented in `buffer/model.go` — a better home architecturally.*

Daemon auto-purge goroutine added as modification to `internal/daemon/daemon.go` (trash expiration cleanup).

---

## Step 3 — IPC Protocol & Daemon Server Loop

**Status: COMPLETE**

| Artifact | File | Status |
|---|---|---|
| Request/Response types, Op constants, payload structs | `internal/ipc/msg.go` | ✅ |
| Conn wrapper with Send/Receive/Dial | `internal/ipc/conn.go` | ✅ |
| UDS listener, accept loop, per-connection goroutine | `internal/daemon/server.go` | ✅ |
| Request dispatch (Op → store.* mapping) | `internal/daemon/handlers.go` | ✅ |
| Client-side autostart (dial or fork daemon) | `internal/daemon/autostart.go` | ✅ |
| Daemon struct extended (listener, WaitGroup) | `internal/daemon/daemon.go` | ✅ |
| Message serialization tests | `internal/ipc/msg_test.go` | ✅ |
| Conn IO tests | `internal/ipc/conn_test.go` | ✅ |
| Integration tests (all 13 operations) | `internal/daemon/handlers_test.go` | ✅ |

---

## Step 4 — CLI Command Tree (All Commands)

**Status: COMPLETE**

| Artifact | File | Status |
|---|---|---|
| IPC client wrapper (13 typed methods) | `internal/cli/client.go` | ✅ |
| Cobra root command + Execute entry point | `internal/cli/root.go` | ✅ |
| Output formatting helpers | `internal/cli/output.go` | ✅ |
| `tb add` — stdin/--text/--label/--tag | `internal/cli/add.go` | ✅ |
| `tb list` — filter/regex/since/until/limit/json | `internal/cli/list.go` | ✅ |
| `tb get <id>` | `internal/cli/get.go` | ✅ |
| `tb search <query>` — regex/json | `internal/cli/search.go` | ✅ |
| `tb edit <id>` — $EDITOR integration, exit-code handling | `internal/cli/edit.go` | ✅ |
| `tb rm <id>` — soft delete / --permanent | `internal/cli/rm.go` | ✅ |
| `tb pipe <id> --command` — pipe/new | `internal/cli/pipe.go` | ✅ |
| `tb daemon {start|stop|status}` | `internal/cli/daemon.go` | ✅ |
| `tb version` | `internal/cli/version.go` | ✅ |
| `cmd/tb/main.go` — wired to cli.Execute | `cmd/tb/main.go` | ✅ |
| PID file support in daemon | `internal/daemon/daemon.go` | ✅ |
| `PidFilePath()` on Config | `internal/config/config.go` | ✅ |
| Exported `FindDaemonBinary` | `internal/daemon/autostart.go` | ✅ |
| Exported `Daemon.Serve()` | `internal/daemon/server.go` | ✅ |
| Integration tests (18 tests, all pass) | `internal/cli/cli_test.go` | ✅ |
| Dependency: cobra | `go.mod` | ✅ |

---

## Step 5 — TUI with Vaxis: Layout, Navigation & VT Preview

**Status: COMPLETE**

**Change:** The bubbletea TUI was replaced with a vaxis-based implementation. The `internal/vt/` package is eliminated — vaxis's `widgets/term` provides terminal emulation for VT preview.

**Kept artifacts (unchanged, reused from prior steps):**
| Artifact | File | Status |
|---|---|---|
| `BufferSummary` type + `NewBufferSummary` | `internal/buffer/model.go` | ✅ |
| `ListBufferSummaries` (lightweight SQL query) | `internal/store/buffer_repo.go` | ✅ |
| `OpListBufferSummaries` IPC constant | `internal/ipc/msg.go` | ✅ |
| Daemon handler + dispatch case | `internal/daemon/handlers.go` | ✅ |
| `ListBufferSummaries` on `cli.Client` | `internal/cli/client.go` | ✅ |

**New vaxis-based artifacts:**

| Artifact | File | Lines | Status |
|---|---|---|---|
| App struct, vaxis init, event loop + style vars | `internal/tui/app.go` | ~255 | ✅ |
| Buffer list pane (vaxis.Window rendering) | `internal/tui/buffer_list.go` | ~80 | ✅ |
| Preview state + text/term.Model VT rendering | `internal/tui/preview.go` | ~130 | ✅ |
| Keybinding mapping (vaxis.Key → action) | `internal/tui/keymap.go` | ~60 | ✅ |
| Help overlay (centered border box) | `internal/tui/help.go` | ~60 | ✅ |
| Event routing + IPC goroutines + state mutations | `internal/tui/update.go` | ~190 | ✅ |

**Deleted (old bubbletea files):**
| File | Status |
|---|---|
| `internal/tui/model.go` | 🗑️ Cleared (kept as empty placeholder) |
| Dependencies: bubbletea, lipgloss, creack/pty | 🗑️ Removed from go.mod |

**New dependencies:** `git.sr.ht/~rockorager/vaxis` (via local replace: `/tmp/vaxis`)
**Key design decisions:**
- Async IPC via goroutines + `vx.PostEvent()` (custom event types work since vaxis.Event is `interface{}`)
- Event loop re-draws every frame unconditionally — vaxis double-buffered diff rendering makes it efficient
- VT preview uses `term.Model` with `cat` piping content through stdin; ANSI escapes automatically parsed
- Help overlay drawn with box-drawing characters (`┌┐└┘─│`) via `win.SetCell`

---

## Step 6 — TUI: Search & Filter

**Status: COMPLETE**

| Artifact | File | Status |
|---|---|---|
| Search state management, debounced IPC, key handling | `internal/tui/search.go` | ✅ |
| `Search` added to `Client` interface | `internal/tui/app.go` | ✅ |
| `stateSearch` routing, `/` key handler, search result dispatch | `internal/tui/update.go` | ✅ |
| `vaxis/widgets/textinput` for search input (proper key handling) | `internal/tui/search.go` | ✅ |
| Status bar renders `textinput.Model` during search | `internal/tui/app.go` | ✅ |
| Paste support during search | `internal/tui/update.go` | ✅ |

**Design:**
- `/` key enters search mode; `textinput.Model` renders in the status bar with `"/"` prompt
- Literal search uses `ListBufferSummaries` with `Keyword` (server-side LIKE, returns `[]BufferSummary`)
- Regex search (prefix `~`) uses `Search` endpoint with `isRegex=true`, converts to summaries
- Debounce 150ms timer resets on each keystroke; generation counter discards stale results
- `allSummaries` cached so Escape clears instantly without daemon round-trip
- `searchGen` incremented on exit to invalidate in-flight goroutines
- Create/delete while filtered updates `allSummaries` and re-triggers search

---

## Step 7 — External Editor Integration (PTY Tab)

**Status: COMPLETE**

| Artifact | File | Status |
|---|---|---|
| Editor resolution, temp file creation, command building | `internal/editor/editor.go` | ✅ |
| `EditorTab` struct wrapping `term.Model` for PTY editor | `internal/tui/edit_tab.go` | ✅ |
| Tab bar rendering (list + editor tabs) | `internal/tui/tab.go` | ✅ |
| Multi-tab tracking, tab-aware draw/event dispatch | `internal/tui/app.go` | ✅ |
| Editor lifecycle events (`editorStarted`, `editorExited`) | `internal/tui/update.go` | ✅ |
| Non-zero exit confirmation dialog | `internal/tui/update.go` | ✅ |
| Tab switch keybindings (Tab, Shift+Tab, Alt+&lt;n&gt;) | `internal/tui/keymap.go` | ✅ |
| Help overlay updated with tab/editor entries | `internal/tui/help.go` | ✅ |
| `UpdateContent` added to TUI `Client` interface | `internal/tui/app.go` | ✅ |
| Editor command passed from config through to TUI | `internal/cli/root.go` | ✅ |

**Key design decisions:**
- Deferred-start pattern (aerc-style): `EditorTab.Start()` called on first `Draw()` with real window dimensions
- `EditorTab.onExit` callback posts `editorExited` event to vaxis event loop for thread-safe state mutation
- Zero exit code → auto-save and close tab; non-zero → prompt user with `stateEditorExitConfirm`
- Tab index tracked separately from editor tab slice; `handleTabSwitch` intercepted at `handleKey()` level before mode dispatch
- `updateTabFocus()` calls `Focus()`/`Blur()` on editor `term.Model` for PTY focus notifications
- All editor tabs closed on quit; `Ctrl+C` in editor tab forwarded to PTY (SIGINT to editor), not intercepted
- Tab bar drawn at row 0 (replaces old top bar); content area unchanged
