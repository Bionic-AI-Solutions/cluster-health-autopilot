// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

package investigator

import "testing"

func TestParseKindNsName(t *testing.T) {
	cases := []struct{ subj, ns, name string }{
		{"CronJob/livekit-agents/retention-sweep", "livekit-agents", "retention-sweep"},
		{"Pod/immersive/immersive-engine-abc", "immersive", "immersive-engine-abc"},
		{"Secret/mcp/x/key", "mcp", "x"},
		{"Node/worker-1", "", ""},
		{"", "", ""},
	}
	for _, c := range cases {
		ns, name := parseKindNsName(c.subj)
		if ns != c.ns || name != c.name {
			t.Errorf("parseKindNsName(%q) = (%q,%q), want (%q,%q)", c.subj, ns, name, c.ns, c.name)
		}
	}
}
