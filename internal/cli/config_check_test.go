package cli

import "testing"

func TestPiSubagentsPinned(t *testing.T) {
	cases := []struct {
		name string
		pkgs []any
		want bool
	}{
		{"exact pin", []any{"npm:pi-subagents@0.45.1"}, true},
		{"bare name unpinned", []any{"npm:pi-subagents"}, false},
		{"caret range unpinned", []any{"npm:pi-subagents@^0.45.1"}, false},
		{"tilde range unpinned", []any{"npm:pi-subagents@~0.45.1"}, false},
		{"missing package", []any{"npm:pi-web-access"}, false},
		{"empty packages list", []any{}, false},
		{"multiple with pinned last", []any{"npm:pi-web-access", "npm:pi-subagents@0.45.1"}, true},
		{"multiple with unpinned subagents", []any{"npm:pi-web-access", "npm:pi-subagents"}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := map[string]any{"packages": tc.pkgs}
			if got := piSubagentsPinned(s); got != tc.want {
				t.Fatalf("piSubagentsPinned(%v) = %v, want %v", tc.pkgs, got, tc.want)
			}
		})
	}
}

func TestPiSubagentsPinnedMissingPackagesKey(t *testing.T) {
	if piSubagentsPinned(map[string]any{}) {
		t.Fatal("expected false when packages key is absent")
	}
}
