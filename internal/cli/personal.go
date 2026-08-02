package cli

import "os"

// personalMode reports whether personal-machine checks are enabled. When unset
// or "0", checks that assert facts about a specific developer's machine
// (symlinks into ~/bin, dotfile contents, installed skill counts) are skipped
// so the harness passes on a fresh clone.
func personalMode() bool {
	return os.Getenv("PI_RUN_PERSONAL") == "1"
}
