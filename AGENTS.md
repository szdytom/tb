# AGENTS.md

tmpbuffer is a terminal-based text buffer manager designed for developers, operators, and power users who frequently work with transient text snippets. It acts as a “scratchpad multiplexer” – managing a collection of text buffers with persistent history, full-text search, and seamless shell integration, while completely delegating text editing to the user’s preferred external editor. The code is written in Go.

## General Instructions

Read [Requirements Doc](./docs/spec-phase1.md).

- Don't assume. Don't hide confusion. Surface tradeoffs.
- State your assumptions explicitly. If uncertain, ask. If multiple interpretations exist, present them - don't pick silently.
- No error handling for impossible scenarios.
- If you write 200 lines and it could be 50, rewrite it.
- Write tests. Not too many. Mostly integration.
- Define success criteria. Loop until verified.

## Code Style

- Use Go idioms. Follow Go conventions. Run `go fmt` on all code.
