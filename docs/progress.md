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

Remaining: `cmd/tb/main.go` is a stub (just prints error — proper CLI comes in Step 4).

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

