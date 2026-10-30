// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2026 Red Hat

package ovnrelay

import "testing"

func TestParseMemory(t *testing.T) {
	values := parseMemory("atoms:658 cells:433 monitors:0 n-weak-refs:14 txn-history:1 txn-history-atoms:644")
	want := map[string]float64{
		"atoms":             658,
		"cells":             433,
		"monitors":          0,
		"n-weak-refs":       14,
		"txn-history":       1,
		"txn-history-atoms": 644,
	}
	for key, expected := range want {
		got, ok := values[key]
		if !ok || got != expected {
			t.Errorf("%s = %v (present %v), want %v", key, got, ok, expected)
		}
	}
}

func TestCountRemotes(t *testing.T) {
	if got := countRemotes("ptcp:16642:[::]\npunix:/run/ovn/ovnsb_relay_db.sock\n"); got != 2 {
		t.Errorf("countRemotes = %v, want 2", got)
	}
	if got := countRemotes("\n\n"); got != 0 {
		t.Errorf("countRemotes empty = %v, want 0", got)
	}
}
