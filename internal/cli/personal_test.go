package cli

import "testing"

func TestPersonalModeDefaultOff(t *testing.T) {
	t.Setenv("PI_RUN_PERSONAL", "")
	if personalMode() {
		t.Fatal("personalMode should be false when PI_RUN_PERSONAL unset")
	}
}

func TestPersonalModeOn(t *testing.T) {
	t.Setenv("PI_RUN_PERSONAL", "1")
	if !personalMode() {
		t.Fatal("personalMode should be true when PI_RUN_PERSONAL=1")
	}
}
