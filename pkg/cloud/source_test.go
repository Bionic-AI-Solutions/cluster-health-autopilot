// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

package cloud

import "testing"

func TestModeString(t *testing.T) {
	cases := []struct {
		mode Mode
		want string
	}{
		{ModeLive, "live"},
		{ModeSnapshot, "snapshot"},
		{Mode(99), "unknown"},
	}
	for _, c := range cases {
		if got := c.mode.String(); got != c.want {
			t.Errorf("Mode(%d).String() = %q, want %q", c.mode, got, c.want)
		}
	}
}
