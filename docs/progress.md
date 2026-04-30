# Progress Report

## Status Overview

Build: ✅ passes (`go build ./...`, `go vet ./...`)
Tests: ✅ all pass (`go test ./...`)

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
