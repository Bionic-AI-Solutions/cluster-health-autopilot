// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package gcp

import "testing"

// The two CIDR sizers encode DIFFERENT semantics and must not drift
// toward each other:
//   - usableIPsFromCIDR: PRIMARY-range capacity = 2^(32-mask) minus
//     GCP's 4 reserved addresses (network, gateway, second-to-last,
//     broadcast).
//   - rangeSizeFromCIDR: SECONDARY-(alias-)range capacity = the full
//     2^(32-mask) — GCP reserves nothing in secondary ranges.
func TestCIDRSizers_FullVsUsableSemantics(t *testing.T) {
	cases := []struct {
		cidr       string
		wantFull   int64
		wantUsable int64
	}{
		{"10.0.0.0/24", 256, 252},
		{"10.0.0.0/26", 64, 60},
		{"10.10.0.0/16", 65536, 65532},
		{"10.0.0.0/29", 8, 4},
		// /30 has 4 addresses; GCP's 4 reservations leave 0 usable.
		{"10.0.0.0/30", 4, 0},
		// /32: full size 1; usable clamps to 0 (mask leaves <=2 host bits).
		{"10.0.0.1/32", 1, 0},
	}
	for _, c := range cases {
		if got := rangeSizeFromCIDR(c.cidr); got != c.wantFull {
			t.Errorf("rangeSizeFromCIDR(%q) = %d, want %d (full range, no reservations)", c.cidr, got, c.wantFull)
		}
		if got := usableIPsFromCIDR(c.cidr); got != c.wantUsable {
			t.Errorf("usableIPsFromCIDR(%q) = %d, want %d (minus GCP's 4 reserved)", c.cidr, got, c.wantUsable)
		}
	}
}

func TestCIDRSizers_Unparseable(t *testing.T) {
	for _, cidr := range []string{"", "10.0.0.0", "10.0.0.0/", "10.0.0.0/abc", "10.0.0.0/33", "10.0.0.0/-1", "fd00::/64x"} {
		if got := rangeSizeFromCIDR(cidr); got != 0 {
			t.Errorf("rangeSizeFromCIDR(%q) = %d, want 0", cidr, got)
		}
		if got := usableIPsFromCIDR(cidr); got != 0 {
			t.Errorf("usableIPsFromCIDR(%q) = %d, want 0", cidr, got)
		}
	}
}
