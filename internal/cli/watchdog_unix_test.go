//go:build !windows

package cli

import (
	"os/exec"
	"syscall"
	"testing"
	"time"
)

// TestTerminateProcessGroupReapsGrandchild proves a supervised spawn's process
// group termination kills the WHOLE tree — the direct child AND a grandchild
// it forked — fixing the direct-child-only kill gap (Codex #4337 class).
//
// The child is `sh -c 'sleep 100 & wait'`: sh spawns a background sleep
// grandchild and then waits. Terminating the group must reap both. The sleep
// is bounded (100s) so even a fully failed test self-terminates; the t.Cleanup
// group kill bounds it tighter.
func TestTerminateProcessGroupReapsGrandchild(t *testing.T) {
	cmd := exec.Command("sh", "-c", "sleep 100 & wait")
	cmd.SysProcAttr = newProcessGroupAttr()
	if err := cmd.Start(); err != nil {
		t.Fatalf("spawn sh: %v", err)
	}
	pgid := cmd.Process.Pid
	// Cleanup safety net: if the test fails before termination, kill the group
	// so no process outlives the test binary.
	t.Cleanup(func() {
		_ = terminateProcessGroup(pgid, 0)
		_ = cmd.Wait()
	})

	// Give the grandchild a moment to actually start before killing.
	time.Sleep(150 * time.Millisecond)

	// Terminate the group with a tiny grace window so the test is fast.
	if err := terminateProcessGroup(pgid, 50*time.Millisecond); err != nil {
		t.Fatalf("terminateProcessGroup: %v", err)
	}

	// Reap the direct child. The group leader (sh) died from the group
	// SIGTERM and is now a zombie; only cmd.Wait reaps it. Until reaped, the
	// zombie keeps the group technically non-empty, so Wait must happen before
	// the emptiness poll.
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case <-done:
		// reaped
	case <-time.After(5 * time.Second):
		t.Fatal("cmd.Wait did not return after group termination")
	}

	// Poll until the group is gone (kill(-pgid, 0) → ESRCH when empty), bounded
	// by a deadline so a broken kill fails loudly instead of hanging.
	deadline := time.Now().Add(5 * time.Second)
	for processGroupAlive(pgid) {
		if time.Now().After(deadline) {
			t.Fatal("process group still alive after termination (grandchild leaked)")
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// TestKillProcessGroupEsrchTolerant proves killing an already-gone group is
// not an error (the leader can exit between the deadline and the kill — the
// pgid-reuse guard in the spec).
func TestKillProcessGroupEsrchTolerant(t *testing.T) {
	// Spawn and reap a short-lived process so its pgid is definitely gone.
	cmd := exec.Command("sh", "-c", "exit 0")
	cmd.SysProcAttr = newProcessGroupAttr()
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	pgid := cmd.Process.Pid
	if err := cmd.Wait(); err != nil {
		t.Fatalf("wait: %v", err)
	}
	// Give the OS a beat to fully reap before asserting ESRCH tolerance.
	time.Sleep(50 * time.Millisecond)
	if err := killProcessGroup(pgid, syscall.SIGTERM); err != nil {
		t.Fatalf("killProcessGroup on gone group returned error: %v", err)
	}
	if err := terminateProcessGroup(pgid, time.Millisecond); err != nil {
		t.Fatalf("terminateProcessGroup on gone group returned error: %v", err)
	}
}

// TestProcessGroupAliveReflectsReality proves the liveness probe distinguishes
// a live group from an empty one (used by the reap test and partial-kill
// evidence).
func TestProcessGroupAliveReflectsReality(t *testing.T) {
	cmd := exec.Command("sh", "-c", "sleep 30")
	cmd.SysProcAttr = newProcessGroupAttr()
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	pgid := cmd.Process.Pid
	t.Cleanup(func() { _ = terminateProcessGroup(pgid, 0) })
	if !processGroupAlive(pgid) {
		t.Fatal("live group reported not alive")
	}
	_ = terminateProcessGroup(pgid, 50*time.Millisecond)
	_ = cmd.Wait() // reap the leader zombie before polling emptiness
	deadline := time.Now().Add(5 * time.Second)
	for processGroupAlive(pgid) {
		if time.Now().After(deadline) {
			t.Fatal("group still alive after termination")
		}
		time.Sleep(20 * time.Millisecond)
	}
}
