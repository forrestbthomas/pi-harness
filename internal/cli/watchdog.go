package cli

import (
	"io"
	"os"
	"strconv"
	"sync"
	"time"
)

// Watchdog constants for self-healing agent runs (spec §"Watchdog-terminated
// exit code" / "Assumptions").
const (
	// exitWatchdogTerminated is the exit code returned when a run is terminated
	// by the watchdog (output stall) or by the wall-clock timeout of a
	// supervised spawn, distinct from the generic error code 1 so CI can grep
	// for watchdog kills.
	exitWatchdogTerminated = 9
	// defaultStallTimeoutSec is the default output-stall silent window, chosen
	// to match the benchmark task default (300s) so supervised benchmark runs
	// keep their existing wall-clock bound while print runs gain a first bound.
	defaultStallTimeoutSec = 300
	// defaultGrace is the SIGTERM→SIGKILL escalation window after a watchdog
	// termination is decided.
	defaultGrace = 10 * time.Second
)

// stallTimeout returns the output-stall silent window: PI_STALL_TIMEOUT_SECS
// when set to a positive integer, else the 300s default. A non-positive or
// unparsable env value falls back to the default rather than erroring (the
// harness must never refuse to run because of a bad tuning var).
func stallTimeout() time.Duration {
	if v := os.Getenv("PI_STALL_TIMEOUT_SECS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return time.Duration(n) * time.Second
		}
	}
	return defaultStallTimeoutSec * time.Second
}

// watchdogGrace returns the SIGTERM→SIGKILL escalation window:
// PI_WATCHDOG_GRACE_SECS when set to a non-negative integer (0 = immediate
// SIGKILL), else the 10s default. Same fallback rule as stallTimeout.
func watchdogGrace() time.Duration {
	if v := os.Getenv("PI_WATCHDOG_GRACE_SECS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			return time.Duration(n) * time.Second
		}
	}
	return defaultGrace
}

// selfHealEnabled reports whether --self-heal observability is on. It is
// env-gated (PI_SELF_HEAL=1) so CI can set it for the nightly live job while
// the deterministic job leaves it off — no behavior change in deterministic
// tests, events are logged only when enabled.
func selfHealEnabled() bool {
	return os.Getenv("PI_SELF_HEAL") == "1"
}

// stallEvidence is the trigger-evidence block of an escalation report: what
// the watchdog observed when it decided the run was stalled.
type stallEvidence struct {
	// LastOutputAt is when the last stdout byte was observed (zero when the
	// run produced no output at all).
	LastOutputAt time.Time `json:"lastOutputAt"`
	// SilentSeconds is how long the run was silent when the watchdog fired.
	SilentSeconds float64 `json:"silentSeconds"`
	// BytesRead is the total stdout bytes the watchdog consumed.
	BytesRead int64 `json:"bytesRead"`
}

// watchdog supervises a spawned process group's stdout for output activity.
// Any byte read from the tee'd pipe resets the stall clock; when no bytes
// arrive for the window, killCh is closed exactly once (the caller owns the
// actual group termination, so the watchdog stays testable without processes).
//
// The watchdog is only attached to non-interactive spawns (print/benchmark):
// interactive chat keeps manual control and is never auto-killed.
type watchdog struct {
	pgid   int
	window time.Duration
	grace  time.Duration

	lastByte time.Time
	bytes    int64
	mu       sync.Mutex

	killCh chan struct{}
	done   chan struct{}
	once   sync.Once
	pr     io.Reader
	pw     io.WriteCloser
}

// newWatchdog creates a watchdog for the process group pgid (set before
// Start, or wired by the caller right after). pr is the tee'd stdout pipe the
// watchdog consumes; pw is closed by stop() to unblock the reader on exit.
func newWatchdog(window, grace time.Duration, pgid int, pr io.Reader, pw io.WriteCloser) *watchdog {
	return &watchdog{
		pgid:     pgid,
		window:   window,
		grace:    grace,
		lastByte: time.Now(),
		killCh:   make(chan struct{}),
		done:     make(chan struct{}),
		pr:       pr,
		pw:       pw,
	}
}

// start launches the reader and timer goroutines.
func (w *watchdog) start() {
	go w.readLoop()
	go w.timerLoop()
}

// stop halts the watchdog and closes the tee pipe so the reader goroutine
// unblocks after the child exits. Safe to call more than once.
func (w *watchdog) stop() {
	w.once.Do(func() {
		close(w.done)
		_ = w.pw.Close()
	})
}

// stalled returns the channel that closes when the watchdog decides the run
// is stalled. The caller then terminates the group (w.grace) and reports.
func (w *watchdog) stalled() <-chan struct{} { return w.killCh }

// evidence snapshots the stall observations (last output, silent seconds,
// bytes) for the escalation report.
func (w *watchdog) evidence() stallEvidence {
	w.mu.Lock()
	defer w.mu.Unlock()
	return stallEvidence{
		LastOutputAt:  w.lastByte,
		SilentSeconds: time.Since(w.lastByte).Seconds(),
		BytesRead:     w.bytes,
	}
}

// readLoop consumes the tee'd stdout, resetting the stall clock on every byte
// and counting bytes for the evidence block. Exits when the pipe reaches EOF
// (pw closed by stop()).
func (w *watchdog) readLoop() {
	buf := make([]byte, 32*1024)
	for {
		n, err := w.pr.Read(buf)
		if n > 0 {
			w.mu.Lock()
			w.lastByte = time.Now()
			w.bytes += int64(n)
			w.mu.Unlock()
		}
		if err != nil {
			return
		}
	}
}

// timerLoop checks the stall clock once per second and closes killCh when the
// window has passed with no output.
func (w *watchdog) timerLoop() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-w.done:
			return
		case <-ticker.C:
			w.mu.Lock()
			idle := time.Since(w.lastByte)
			w.mu.Unlock()
			if idle >= w.window {
				close(w.killCh)
				return
			}
		}
	}
}
