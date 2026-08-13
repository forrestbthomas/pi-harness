//go:build !windows

package cli

import (
	"syscall"
	"time"
)

// newProcessGroupAttr returns the SysProcAttr that puts the spawned pi process
// (and everything it forks) into its own process group, so a watchdog can kill
// the whole tree with one negative-pgid signal. Only used for supervised
// (non-interactive) spawns; interactive chat keeps the caller's group.
func newProcessGroupAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{Setpgid: true}
}

// terminateProcessGroup escalates a watchdog termination: SIGTERM the group,
// wait the grace window, then SIGKILL the survivors. ESRCH is treated as
// success (the group is already gone, or the leader exited between the
// deadline and the kill — the pgid-reuse guard). EPERM on the SIGKILL step is
// also treated as success: it means the group now holds only the leader's
// zombie (dead already; the caller's cmd.Wait reaps it), so there is nothing
// left to kill. Returns the last other error, if any, so the caller can
// record partial-kill evidence.
func terminateProcessGroup(pgid int, grace time.Duration) error {
	if err := killProcessGroup(pgid, syscall.SIGTERM); err != nil {
		return err
	}
	time.Sleep(grace)
	if err := killProcessGroup(pgid, syscall.SIGKILL); err != nil && err != syscall.EPERM {
		return err
	}
	return nil
}

// killProcessGroup sends sig to every process in pgid's group. A negative pgid
// with syscall.Kill targets the whole group. ESRCH is swallowed: it means the
// group no longer exists (nothing left to kill), which is the goal.
func killProcessGroup(pgid int, sig syscall.Signal) error {
	if err := syscall.Kill(-pgid, sig); err == syscall.ESRCH {
		return nil
	} else if err != nil {
		return err
	}
	return nil
}

// processGroupAlive reports whether any process remains in the group. Used by
// tests to prove grandchildren were reaped (kill(-pgid, 0) → ESRCH when empty)
// and by the escalation report to record partial-kill evidence.
func processGroupAlive(pgid int) bool {
	return syscall.Kill(-pgid, 0) != syscall.ESRCH
}
