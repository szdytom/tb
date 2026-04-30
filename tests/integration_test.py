#!/usr/bin/env python3
"""tmpbuffer CLI integration tests.

Drives the tb CLI as an external process (black-box) and asserts on
stdout, stderr, and exit codes. Requires Go toolchain.

Usage:
    python3 tests/integration_test.py
    python3 tests/integration_test.py --tb ./tb --daemon ./tmpbufferd
    python3 tests/integration_test.py -v
"""

import argparse
import os
import pathlib
import shutil
import signal
import subprocess
import sys
import tempfile
import time

REPO_ROOT = pathlib.Path(__file__).resolve().parent.parent


# ── Test harness ──────────────────────────────────────────────────────


class Tester:
    """Black-box CLI tester. Manages daemon lifecycle and provides `tb()`."""

    def __init__(self, tb_bin: str, daemon_bin: str, work_dir: str, verbose: bool):
        self.tb_bin = tb_bin
        self.daemon_bin = daemon_bin
        self.work_dir = pathlib.Path(work_dir)
        self.verbose = verbose
        self._proc: subprocess.Popen | None = None

        # XDG isolation: all state goes under work_dir/tmpbuffer/
        self.env = os.environ.copy()
        self.env["XDG_DATA_HOME"] = str(self.work_dir)
        self.env["XDG_STATE_HOME"] = str(self.work_dir)
        self.env["XDG_CONFIG_HOME"] = str(self.work_dir)

        self.socket_path = self.work_dir / "tmpbuffer" / "tmpbuffer.sock"
        self.pid_path = self.work_dir / "tmpbuffer" / "tmpbuffer.pid"

    # ── Daemon lifecycle ──────────────────────────────────────────

    def start_daemon(self):
        """Fork the daemon and wait until its socket is ready."""
        if self._proc is not None:
            return

        self.log(f"starting daemon: {self.daemon_bin}")
        self._proc = subprocess.Popen(
            [self.daemon_bin],
            env=self.env,
            stdout=subprocess.DEVNULL,
            stderr=subprocess.PIPE,
        )

        # Poll for the socket (up to 5 s).
        for _ in range(50):
            if self.socket_path.exists():
                # Give it one more tick to start accepting.
                time.sleep(0.1)
                self.log(f"daemon ready (pid={self._proc.pid})")
                return
            time.sleep(0.1)

        self.abort("daemon did not create socket within 5 s")

    def stop_daemon(self):
        """Send SIGTERM and wait for clean exit."""
        if self._proc is None:
            return
        pid = self._proc.pid
        self.log(f"stopping daemon (pid={pid})")
        self._proc.send_signal(signal.SIGTERM)
        try:
            self._proc.wait(timeout=5)
        except subprocess.TimeoutExpired:
            self._proc.kill()
            self._proc.wait()
            self.log("daemon killed (timeout)")
        self._proc = None

    # ── CLI execution ─────────────────────────────────────────────

    def tb(self, *args: str, stdin: str | None = None) -> subprocess.CompletedProcess:
        """Run a tb subcommand and return the CompletedProcess."""
        if self.verbose:
            desc = " ".join(f"'{a}'" if " " in a else a for a in args)
            self.log(f"tb {desc}")

        result = subprocess.run(
            [self.tb_bin, *args],
            env=self.env,
            input=stdin,
            capture_output=True,
            text=True,
        )

        if self.verbose and result.stderr:
            for line in result.stderr.strip().split("\n"):
                self.log(f"  stderr: {line}")

        return result

    # ── Helpers ───────────────────────────────────────────────────

    def log(self, msg: str):
        print(f"  [{_ts()}] {msg}", file=sys.stderr)

    def abort(self, msg: str):
        print(f"\n  ABORT: {msg}", file=sys.stderr)
        self.stop_daemon()
        sys.exit(1)

    def must(self, result: subprocess.CompletedProcess, msg: str = ""):
        """Assert that a command succeeded, or abort the whole suite."""
        if result.returncode != 0:
            label = msg or f"exit code {result.returncode}"
            print(f"\n  ABORT: {label}", file=sys.stderr)
            if result.stderr:
                for line in result.stderr.strip().split("\n"):
                    print(f"    stderr: {line}", file=sys.stderr)
            self.stop_daemon()
            sys.exit(1)
        return result


# ── Assertion helpers ──────────────────────────────────────────────────


class Fail(Exception):
    """Raised when a test assertion fails."""


def require(cond: bool, msg: str):
    if not cond:
        raise Fail(msg)


def require_ok(result: subprocess.CompletedProcess, label: str = ""):
    prefix = f"{label}: " if label else ""
    require(result.returncode == 0, f"{prefix}expected rc=0, got {result.returncode}")
    if result.stderr and "Error:" in result.stderr:
        require(False, f"{prefix}unexpected stderr: {result.stderr.strip()}")


def require_fail(result: subprocess.CompletedProcess, label: str = ""):
    prefix = f"{label}: " if label else ""
    require(result.returncode != 0, f"{prefix}expected non-zero exit, got 0")


def require_stdout_contains(result: subprocess.CompletedProcess, substr: str, label: str = ""):
    prefix = f"{label}: " if label else ""
    require(substr in result.stdout, f"{prefix}expected {substr!r} in stdout:\n{result.stdout}")


def require_stdout_not_contains(result: subprocess.CompletedProcess, substr: str, label: str = ""):
    prefix = f"{label}: " if label else ""
    require(substr not in result.stdout, f"{prefix}expected {substr!r} NOT in stdout:\n{result.stdout}")


def require_stderr_contains(result: subprocess.CompletedProcess, substr: str, label: str = ""):
    prefix = f"{label}: " if label else ""
    require(substr in result.stderr, f"{prefix}expected {substr!r} in stderr:\n{result.stderr}")


def parse_id(result: subprocess.CompletedProcess) -> str:
    """Extract a numeric ID from a successful tb add."""
    id_str = result.stdout.strip()
    require(id_str.isdigit(), f"expected numeric ID, got {id_str!r}")
    return id_str


# ── Test functions ─────────────────────────────────────────────────────
# Each test receives a Tester instance. Raises Fail on failure.

def test_version(t: Tester):
    r = t.tb("version")
    require_ok(r, "version")
    require_stdout_contains(r, "tb version", "version")


def test_add_text(t: Tester):
    r = t.tb("add", "--text", "hello world", "--label", "greeting")
    require_ok(r, "add --text")
    id_str = parse_id(r)
    assert int(id_str) > 0  # sanity

    r = t.tb("get", id_str)
    require_ok(r, "get")
    require_stdout_contains(r, "hello world", "get content")

    r = t.tb("get", "--json", id_str)
    require_ok(r, "get --json")
    require_stdout_contains(r, '"label": "greeting"', "get --json label")


def test_add_stdin(t: Tester):
    r = t.tb("add", "--label", "stdin-test", stdin="from stdin")
    require_ok(r, "add stdin")
    id_str = parse_id(r)

    r = t.tb("get", id_str)
    require_stdout_contains(r, "from stdin", "get after stdin add")


def test_add_empty(t: Tester):
    r = t.tb("add")
    require_ok(r, "add empty")
    id_str = parse_id(r)

    r = t.tb("get", id_str)
    require_ok(r, "get after empty add")


def test_add_tags(t: Tester):
    r = t.tb("add", "--text", "tagged", "--label", "mytag", "--tag", "x,y")
    require_ok(r, "add with tags")
    id_str = parse_id(r)

    r = t.tb("get", "--json", id_str)
    require_stdout_contains(r, '"x"', "tag x")
    require_stdout_contains(r, '"y"', "tag y")


def test_get_not_found(t: Tester):
    r = t.tb("get", "99999")
    require_fail(r, "get not found")
    require_stderr_contains(r, "Error:", "get not found error message")


def test_list(t: Tester):
    t.tb("add", "--text", "apple")
    t.tb("add", "--text", "banana")

    r = t.tb("list")
    require_ok(r, "list")
    require_stdout_contains(r, "apple", "list apple")
    require_stdout_contains(r, "banana", "list banana")


def test_list_json(t: Tester):
    t.tb("add", "--text", "json content")

    r = t.tb("list", "--json")
    require_ok(r, "list --json")
    require_stdout_contains(r, '"content": "json content"', "list --json content")


def test_list_filter(t: Tester):
    t.tb("add", "--text", "apple banana")
    t.tb("add", "--text", "cherry")

    r = t.tb("list", "--filter", "banana")
    require_ok(r, "list --filter")
    require_stdout_contains(r, "apple", "filter shows apple")
    require_stdout_not_contains(r, "cherry", "filter hides cherry")


def test_search(t: Tester):
    t.tb("add", "--text", "error: timeout occurred")
    t.tb("add", "--text", "everything is fine")

    r = t.tb("search", "error")
    require_ok(r, "search")
    require_stdout_contains(r, "timeout", "search matches timeout")
    require_stdout_not_contains(r, "fine", "search excludes no-match")


def test_search_regex(t: Tester):
    t.tb("add", "--text", "error: timeout")
    t.tb("add", "--text", "warning: slow")

    r = t.tb("search", "--regex", "error.*time")
    require_ok(r, "search --regex")
    require_stdout_contains(r, "timeout", "search regex matches")


def test_search_json(t: Tester):
    t.tb("add", "--text", "findable content")

    r = t.tb("search", "--json", "findable")
    require_ok(r, "search --json")
    require_stdout_contains(r, '"snippet"', "search --json has snippet")


def test_search_no_match(t: Tester):
    t.tb("add", "--text", "something")
    r = t.tb("search", "xyznonexistent")
    require_ok(r, "search no match")  # search succeeds, just 0 results
    require_stderr_contains(r, "No results", "search no match message")


def test_rm(t: Tester):
    r = t.tb("add", "--text", "delete me")
    id_str = parse_id(r)

    r = t.tb("rm", id_str)
    require_ok(r, "rm")

    r = t.tb("list")
    require_stdout_not_contains(r, id_str, "rm removes from list")


def test_rm_permanent(t: Tester):
    r = t.tb("add", "--text", "permanent delete")
    id_str = parse_id(r)

    r = t.tb("rm", "--permanent", id_str)
    require_ok(r, "rm --permanent")

    r = t.tb("get", id_str)
    require_fail(r, "get after permanent rm")


def test_edit(t: Tester):
    """Edit with cat as editor: content should stay unchanged."""
    r = t.tb("add", "--text", "original")
    id_str = parse_id(r)

    r = t.tb("edit", id_str, "--editor", "cat")
    require_ok(r, "edit")

    r = t.tb("get", id_str)
    require_stdout_contains(r, "original", "edit preserved content")


def test_pipe(t: Tester):
    r = t.tb("add", "--text", "hello world")
    id_str = parse_id(r)

    r = t.tb("pipe", id_str, "--command", "tr a-z A-Z")
    require_ok(r, "pipe")

    r = t.tb("get", id_str)
    require_stdout_contains(r, "HELLO WORLD", "pipe transformed content")


def test_pipe_new(t: Tester):
    r = t.tb("add", "--text", "hello")
    id_str = parse_id(r)

    r = t.tb("pipe", id_str, "--command", "wc -c", "--new")
    require_ok(r, "pipe --new")
    new_id = r.stdout.strip()

    # new_id must be different from the original
    require(new_id != id_str, f"pipe --new produced same id ({id_str})")

    # Original unchanged
    r = t.tb("get", id_str)
    require_stdout_contains(r, "hello", "original preserved after pipe --new")

    # New buffer exists
    r = t.tb("get", new_id)
    require_ok(r, "pipe --new created valid buffer")


def test_daemon_status_running(t: Tester):
    """Daemon is already running from setUp."""
    r = t.tb("daemon", "status")
    # status writes messages to stderr
    require(r.returncode == 0, f"daemon status expected rc=0, got {r.returncode}")


def test_help(t: Tester):
    r = t.tb("--help")
    require_ok(r, "--help")
    require_stdout_contains(r, "tmpbuffer", "--help shows description")


# ── Test registry ─────────────────────────────────────────────────────


TESTS = [
    ("version",                test_version),
    ("add --text",             test_add_text),
    ("add stdin",              test_add_stdin),
    ("add empty",              test_add_empty),
    ("add --tag",              test_add_tags),
    ("get not found",          test_get_not_found),
    ("list",                   test_list),
    ("list --json",            test_list_json),
    ("list --filter",          test_list_filter),
    ("search",                 test_search),
    ("search --regex",         test_search_regex),
    ("search --json",          test_search_json),
    ("search no match",        test_search_no_match),
    ("rm",                     test_rm),
    ("rm --permanent",         test_rm_permanent),
    ("edit",                   test_edit),
    ("pipe",                   test_pipe),
    ("pipe --new",             test_pipe_new),
    ("daemon status",          test_daemon_status_running),
    ("--help",                 test_help),
]

# Sort by name for consistent ordering.
TESTS.sort(key=lambda t: t[0])


# ── main ──────────────────────────────────────────────────────────────


def build_binaries(tb_bin: str, daemon_bin: str):
    """Build Go binaries if they don't exist."""
    tb_path = pathlib.Path(tb_bin)
    daemon_path = pathlib.Path(daemon_bin)

    if not tb_path.exists():
        print(f"  building tb → {tb_bin}")
        subprocess.run(
            ["go", "build", "-o", tb_bin, "./cmd/tb"],
            cwd=REPO_ROOT, check=True, capture_output=True,
            env={**os.environ, "CGO_ENABLED": "0"},
        )

    if not daemon_path.exists():
        print(f"  building tmpbufferd → {daemon_bin}")
        subprocess.run(
            ["go", "build", "-o", daemon_bin, "./cmd/tmpbufferd"],
            cwd=REPO_ROOT, check=True, capture_output=True,
            env={**os.environ, "CGO_ENABLED": "0"},
        )

    # Verify both exist.
    for p in [tb_path, daemon_path]:
        if not p.exists():
            print(f"  Error: {p} not found and could not be built", file=sys.stderr)
            sys.exit(1)


def _ts() -> str:
    return time.strftime("%H:%M:%S", time.localtime())


def main():
    ap = argparse.ArgumentParser(description="tmpbuffer CLI integration tests")
    ap.add_argument("--tb", help="path to tb binary (default: build from source)")
    ap.add_argument("--daemon", help="path to tmpbufferd binary (default: build from source)")
    ap.add_argument("-v", "--verbose", action="store_true", help="show CLI stderr output")
    args = ap.parse_args()

    tb_bin = args.tb or shutil.which("tb") or str(REPO_ROOT / "tb")
    daemon_bin = args.daemon or shutil.which("tmpbufferd") or str(REPO_ROOT / "tmpbufferd")

    print(f"tmpbuffer CLI integration tests\n", file=sys.stderr)
    print(f"  tb binary:       {tb_bin}", file=sys.stderr)
    print(f"  daemon binary:   {daemon_bin}", file=sys.stderr)
    print(f"  working dir:     (temp)", file=sys.stderr)
    print(file=sys.stderr)

    build_binaries(tb_bin, daemon_bin)

    work_dir = tempfile.mkdtemp(prefix="tb-integration-")
    tester = Tester(tb_bin, daemon_bin, work_dir, verbose=args.verbose)

    # Ensure cleanup on early exit.
    exit_clean = False

    def _cleanup():
        nonlocal exit_clean
        if exit_clean:
            return
        exit_clean = True
        tester.stop_daemon()
        shutil.rmtree(work_dir, ignore_errors=True)

    try:
        # ── Start daemon ──────────────────────────────────────────
        print("  [setup] starting daemon...", file=sys.stderr)
        tester.start_daemon()

        # ── Run tests ─────────────────────────────────────────────
        passed = 0
        failed = 0
        failures: list[tuple[str, str]] = []

        print(file=sys.stderr)
        for name, fn in TESTS:
            try:
                fn(tester)
                print(f"  PASS  {name}", file=sys.stderr)
                passed += 1
            except Fail as e:
                print(f"  FAIL  {name}: {e}", file=sys.stderr)
                failed += 1
                failures.append((name, str(e)))
            except Exception as e:
                print(f"  FAIL  {name} (exception): {e}", file=sys.stderr)
                failed += 1
                failures.append((name, f"unexpected exception: {e}"))

        # ── Summary ───────────────────────────────────────────────
        total = len(TESTS)
        print(file=sys.stderr)
        print(f"  {'──' * 20}", file=sys.stderr)
        print(f"  {passed} passed, {failed} failed, {total} total", file=sys.stderr)

        if failures:
            print(file=sys.stderr)
            print(f"  Failures:", file=sys.stderr)
            for name, reason in failures:
                print(f"    • {name}: {reason}", file=sys.stderr)

        _cleanup()
        sys.exit(1 if failed > 0 else 0)

    except KeyboardInterrupt:
        print("\n  interrupted", file=sys.stderr)
        _cleanup()
        sys.exit(130)
    except Exception as e:
        print(f"\n  unexpected error: {e}", file=sys.stderr)
        _cleanup()
        sys.exit(1)


if __name__ == "__main__":
    main()
