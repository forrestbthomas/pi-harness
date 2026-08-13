//go:build windows

package cli

import (
	"errors"
	"syscall"
	"time"
)

// newProcessGroupAttr is a no-op on Windows: the POSIX process-group model
// (Setpgid, negative-pgid kill) does not exist here. Spawns still work, they
// are just not group-supervisable.
func newProcessGroupAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{}
}

// terminateProcessGroup is not supported on Windows. The supervised spawn
// paths (benchmark agent timeout, output-stall watchdog) degrade to the
// wall-clock CommandContext-free timeout only: the direct child is killed but
// grandchildren cannot be reaped as a group.
func terminateProcessGroup(pgid int, grace time.Duration) error {
	return errors.New("process-group termination is not supported on this platform (windows)")
}

// killProcessGroup is not supported on Windows.
func killProcessGroup(pgid int, sig syscall.Signal) error {
	return errors.New("process-group termination is not supported on this platform (windows)")
}

// processGroupAlive is not supported on Windows (always reports no group).
func processGroupAlive(pgid int) bool { return false }
