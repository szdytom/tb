# tmpbuffer (tb) – Phase 1 Requirements Document

**Product Name:** tmpbuffer  
**Command Name:** `tb`  
**Version:** 1.0.0 (Phase 1)  
**Document Status:** Draft  
**Date:** 2026-04-30  

---

## 1. Introduction

### 1.1 Purpose
tmpbuffer is a terminal-based text buffer manager designed for developers, operators, and power users who frequently work with transient text snippets. It acts as a “scratchpad multiplexer” – managing a collection of text buffers with persistent history, full-text search, and seamless shell integration, while completely delegating text editing to the user’s preferred external editor (e.g., Vim, Emacs, Helix, VSCode).

Phase 1 establishes the minimal viable core: a daemon-backed buffer store, an interactive TUI browser, external editor integration, basic CLI commands, and pipeline operations. This foundation ensures that the tool is immediately useful without competing with or duplicating existing editor functionality.

### 1.2 Scope
Phase 1 includes:
- Persistent, auto-saved buffer storage (no manual save required)
- A TUI for browsing, creating, deleting, and previewing buffers
- Full-text search (literal and regex) within buffers
- Editing of buffer content via an external editor (`$EDITOR` convention)
- A CLI (`tb`) for scriptable access: add, list, get, search
- Pipeline operations: sending buffer content to external commands and capturing output as a new buffer
- Basic session continuity: all buffers are restored exactly as left after a restart

### 1.3 Out of Scope (for Phase 1)
- Format conversion (JSON ↔ CSV, etc.)
- AI-powered semantic search
- Clipboard monitoring
- Multi-cursor editing (delegated to the external editor)
- Plugin/extension system
- Network sharing or collaboration
- Syntax highlighting in the TUI preview (plain text only)
- Snapshots and version time-travel beyond the last edited state

### 1.4 Definitions
- **Buffer:** A named or unnamed text container managed by tmpbuffer. Every buffer holds a plain text content and associated metadata (creation time, last modification time, tags, etc.).
- **TUI:** The terminal user interface that serves as the primary interactive environment.
- **Daemon:** A background process that owns the buffer database and serves the TUI and CLI requests.
- **External Editor:** Any text editor invoked as a child process to edit buffer content, determined by the `$EDITOR` environment variable or user configuration.

---

## 2. System Overview

tmpbuffer consists of two components:
1. **Daemon (`tmpbufferd` or embedded in `tb daemon`):** Manages buffer persistence, indexing, and inter-process communication.
2. **Client:**  
   - `tb` CLI for scripting and direct operations.  
   - `tb tui` interactive terminal browser (default command when no arguments given).

The daemon starts automatically on the first `tb` invocation if not already running, and can be explicitly controlled with `tb daemon start|stop|status`.

---

## 3. Functional Requirements

### 3.1 Buffer Management

**FR-3.1.1 Buffer Creation**
- The user shall be able to create a new, empty buffer instantly without specifying a name, file path, or destination.
- In the TUI, a dedicated keybinding (e.g., `n`) creates a new blank buffer and adds it to the list.
- Via CLI, `tb add` with no input creates an empty buffer; with input from stdin or a string argument, it creates a buffer with that content.

**FR-3.1.2 Buffer Persistence**
- Every buffer shall be automatically persisted to local storage immediately upon creation and whenever its content changes.
- There shall be no “save” action; closing the TUI, exiting an editor, or killing the daemon must not cause data loss.
- On daemon restart, all previously existing buffers (including their latest content) must be fully restored and visible in the TUI list.

**FR-3.1.3 Buffer Deletion / Archiving**
- The user shall be able to delete a buffer from the TUI with a confirmation step. Deletion moves the buffer to an internal “trash” (archived state) for a configurable retention period, after which it is permanently removed.
- The CLI shall provide `tb rm <id>` with a `--permanent` flag to skip the trash if desired.

**FR-3.1.4 Buffer Metadata**
- Each buffer shall automatically record:
  - Unique identifier (UUID or sequential ID)
  - Creation timestamp
  - Last modification timestamp
  - Byte count and line count
  - A user-assignable short label (optional)
- Metadata shall be searchable and displayable in the TUI list.

### 3.2 TUI (Terminal User Interface)

**FR-3.2.1 Main Layout**
- The TUI shall present a split view:
  - **Left pane (buffer list):** Scrollable list of all active buffers, sorted by last modification time (most recent first). Each entry shows ID, first line preview (truncated to fit), timestamp, and an optional label.
  - **Right pane (preview):** Displays the full content of the currently highlighted buffer in read-only plain text. Supports vertical/horizontal scrolling.

**FR-3.2.2 Navigation and Interaction**
- Keyboard shortcuts shall include (among others):
  - `j` / `k` or arrow keys: move selection up/down
  - `Enter`: open highlighted buffer in the external editor
  - `n`: new buffer
  - `d`: delete highlighted buffer (with confirmation prompt)
  - `/`: open a search bar to filter the buffer list by literal substring or regex
  - `:q` or `Ctrl+C`: quit the TUI (daemon continues running)
  - `:` prefix for command mode (see FR-3.5 Pipeline)

**FR-3.2.3 Search in TUI**
- The `/` key shall open an inline search prompt. Typing filters the buffer list in real time to show only buffers whose content or metadata matches.
- Search terms can be literal strings or, if the search string starts with `~`, interpreted as a regular expression.
- Clearing the search field restores the full buffer list.

**FR-3.2.4 Preview Pane Behavior**
- The preview pane shall update instantly when the selection changes.
- Long lines shall be soft-wrapped; horizontal scrolling shall be supported via `h`/`l` or arrow keys when the preview pane is focused.
- Line numbers shall be displayed optionally (toggle with a key or configuration).

### 3.3 External Editor Integration

**FR-3.3.1 Editor Invocation**
- When a buffer is opened for editing (via `Enter` in TUI or `tb edit <id>`), the daemon shall:
  1. Write the current buffer content to a temporary file (preserving line endings).
  2. Launch the editor defined by the `$EDITOR` environment variable (or `$VISUAL`, or a configured editor) with that temporary file path.
  3. Block and wait for the editor process to terminate.
  4. Read the modified temporary file content back into the buffer.
  5. Delete the temporary file.
- If the editor process exits with a non-zero code, the TUI shall prompt the user whether to keep changes or revert to the original content.

**FR-3.3.2 Editor Configuration**
- The user may configure different editors per file extension in a configuration file (e.g., `*.md` → `typora`, `*.json` → `code --wait`). If no match, `$EDITOR` is used.
- The user may specify the editor directly via `tb edit <id> --editor vim`.

**FR-3.3.3 Content Conflict Handling**
- If an external editor modifies the buffer while another `tb` process is also editing it, the last write wins without merging. The daemon shall log a warning.
- The user is responsible for avoiding concurrent edits.

### 3.4 CLI (Command-Line Interface)

**FR-3.4.1 `tb add`**
- Create a new buffer with content from:
  - Standard input: `echo "hello" | tb add`
  - Direct argument: `tb add --text "hello"`
  - No input: creates an empty buffer and prints its ID.
- Options:
  - `--label "name"` assigns a human-readable label.
  - `--tag tag1,tag2` adds tags for later filtering.

**FR-3.4.2 `tb list`**
- List all active buffers with their ID, label, timestamp, and preview.
- Options:
  - `--filter <keyword>` simple text match
  - `--regex <pattern>` regex match
  - `--since "2 hours ago"`, `--until "2026-04-30"`
  - `--limit <n>`
  - `--json` outputs as JSON array for further scripting.

**FR-3.4.3 `tb get <id>`**
- Print the full content of the specified buffer to stdout.
- Exit code 0 on success; non-zero if ID not found.

**FR-3.4.4 `tb search <query>`**
- Full-text search across all buffers, returns list of matching buffer IDs and a snippet of the match.
- Supports regex with `--regex` flag.
- Default output is human-readable; `--json` available.

**FR-3.4.5 `tb edit <id>`**
- Open a buffer in the external editor from the CLI (similar to TUI `Enter`). Blocks until editor closes.

**FR-3.4.6 `tb rm <id>`**
- Delete a buffer (moved to trash).

### 3.5 Pipeline Operations

**FR-3.5.1 Pipe Buffer to Command**
- In the TUI, a dedicated keybinding (e.g., `!`) on a selected buffer opens a command prompt. The user enters a shell command. The buffer content is piped to that command’s stdin, and the stdout output is captured.
- The user may choose to:
  - Replace the current buffer’s content with the output (and save the previous version as an automatic snapshot).
  - Create a new buffer containing the output.
- In CLI, `tb pipe <id> --command "jq ." --new` creates a new buffer; without `--new` it overwrites.

**FR-3.5.2 Preview Mode**
- Before applying the pipeline, the TUI shall show a preview of the command output in a modal, allowing the user to confirm or cancel.

**FR-3.5.3 Error Handling**
- If the command exits non-zero or produces empty output, the user shall be notified and given the option to proceed or abort.

**FR-3.5.4 Security**
- No implicit shell interpretation: the command shall be passed to the user’s default shell (`sh -c` or `$SHELL -c`) with proper escaping to prevent injection. The user is responsible for the content of the buffer being safe to pass to a shell.

### 3.6 Session Continuity

**FR-3.6.1 Daemon Lifecycle**
- The daemon can be started and managed by a external tool, such as systemd.
- If there is no external tool, the tb command should launch a daemon silently on first invocation.
- If the daemon crashes or is killed, all data shall be recoverable to the state of the last automatic snapshot (maximum 2 seconds before the incident).

**FR-3.6.2 Restoration**
- Upon daemon restart, the TUI and CLI shall see the exact same set of buffers as before, with the same IDs, metadata, and content.

---

## 4. Non-Functional Requirements

### 4.1 Performance
- **NFR-1:** TUI startup time (cold) shall be under 200ms on a typical modern machine with up to 10,000 stored buffers.
- **NFR-2:** Buffer list filtering and search shall respond in less than 50ms for up to 10,000 buffers.
- **NFR-3:** The daemon’s memory footprint shall not exceed 80MB in idle state.
- **NFR-4:** Disk storage shall be optimized with SQLite (with WAL mode) and optional compression of older buffer content.

### 4.2 Reliability
- **NFR-5:** All buffer modifications shall be atomic from the user’s perspective; if an editor crashes, the original content must remain intact.
- **NFR-6:** Auto-save must occur within 2 seconds of any change; after an unexpected termination, no more than 2 seconds of data shall be lost.

### 4.3 Privacy & Security
- **NFR-7:** All data, including buffer content and metadata, shall be stored only on the local machine. No network communication shall be performed by the daemon.
- **NFR-8:** The database file and any temporary files shall be created with restrictive permissions (0600) to prevent access by other users on a multi-user system.
- **NFR-9:** The pipeline feature shall warn the user before executing commands, and the TUI shall display the exact command string about to be run.

### 4.4 Usability
- **NFR-10:** The TUI shall offer Vim-like keybindings by default, but provide a configuration option for alternative key mappings.
- **NFR-11:** A help screen accessible via `?` key shall list all shortcuts and commands.
- **NFR-12:** The command `tb` without arguments shall launch the TUI, providing an immediate interactive experience.

### 4.5 Cross-Platform Compatibility
- **NFR-13:** tmpbuffer shall run on Linux and macOS. Windows support is a non-goal for Phase 1 but should not be architecturally precluded.
- **NFR-14:** The TUI shall adapt to terminal resizing and different color depth capabilities.

### 4.6 Development & Extensibility
- **NFR-15:** The codebase shall be structured to allow future addition of conversion plugins, AI indexing, and a GUI frontend without major refactor.
- **NFR-16:** A configuration file (e.g., `$XDG_CONFIG_HOME/tmpbuffer/config.toml`) shall be read on startup, with sensible defaults for all options.

---

## 5. Acceptance Criteria

Each functional requirement must be verified by the following concrete tests.

### AC-1: Instant Buffer Creation
- **Given** the TUI is open  
- **When** the user presses `n`  
- **Then** a new empty buffer appears at the top of the list with a unique ID and current timestamp. No prompt for filename or path appears.

### AC-2: No-Save Persistence
- **Given** a buffer exists with the text “sample”  
- **When** the user kills the daemon process with `SIGKILL`  
- **And** restarts the daemon / launches TUI again  
- **Then** the buffer is present and contains “sample”. No data was lost.

### AC-3: External Editor Round-Trip
- **Given** a buffer with content “original”  
- **When** the user presses `Enter`, the `$EDITOR` (e.g., Vim) opens with a temporary file containing “original”, the user changes it to “modified”, and saves + exits with zero code  
- **Then** the buffer content is updated to “modified” in the TUI preview, and the modification timestamp is updated.

### AC-4: CLI Add and Get
- **Given** the daemon is running  
- **When** the user executes `echo "hello world" | tb add --label "greeting"`  
- **Then** a new buffer is created, an ID is printed to stdout, and `tb get <that-id>` outputs “hello world” with a zero exit code.

### AC-5: Pipeline Operation
- **Given** a buffer containing `{"a": 1}`  
- **When** in TUI, the user selects that buffer, presses `!`, enters `jq .` as the command, and chooses “create new buffer”  
- **Then** a new buffer is created containing pretty-printed JSON (the output of `jq .`), and the original buffer remains unchanged.

### AC-6: Regex Search
- **Given** multiple buffers exist, one containing “error: timeout” and others without “error”  
- **When** the user presses `/` and types `~error.*time`  
- **Then** only the buffer with “error: timeout” is displayed in the list, with the matching snippet highlighted.

### AC-7: TUI Responsiveness
- **Given** 10,000 buffers are stored  
- **When** the user launches the TUI  
- **Then** the list renders within 200ms, and scrolling/j/filtering has no perceptible lag (sub-50ms feedback).

### AC-8: Editor Exit Code Handling
- **Given** a buffer is opened in an editor that exits with code 1 (e.g., Vim with `:cquit`)  
- **When** the user returns to the TUI  
- **Then** a prompt asks “Editor exited with code 1. Keep changes? [y/N]”. If the user selects ‘N’, the buffer remains unchanged.

### AC-9: Concurrent Access Safety
- **Given** the same buffer is edited via two separate `tb edit` commands (one from TUI, one from CLI) and both modify content  
- **When** the second editor saves after the first  
- **Then** the content reflects the second save (last write wins), and the daemon logs a warning to the log file.

---

## 6. Technical Constraints (for reference)

- Storage engine: SQLite 3.x, with WAL journal mode.
- IPC mechanism: Unix domain socket on Linux/macOS.
- Implementation language: C++ or Go (to be finalized; chosen for performance and binary distribution simplicity).
- Optionally no dynamic linking to heavy runtimes; final binary should be static or depend only on system libc and SQLite.
- Minimal system dependencies: terminal, `$EDITOR` (or equivalent), and a POSIX shell.

---

## 7. Glossary

- **Daemon:** Long-lived background process that manages buffer data and communicates with clients.
- **TUI:** Text-based User Interface rendered inside a terminal emulator.
- **Buffer ID:** A unique string (UUID or sequential integer) assigned to each buffer.
- **Trash/Archive:** A temporary holding area for deleted buffers, allowing recovery before permanent deletion.

---

**Document Approval**

| Role | Name | Date |
|------|------|------|
| Product Owner | F. | 26 Apr. 30 |
| Lead Developer | (pending) | |
| QA Lead | (pending) | |

---

*End of Phase 1 Requirements Document for tmpbuffer*
