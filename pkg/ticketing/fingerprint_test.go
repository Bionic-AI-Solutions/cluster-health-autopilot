// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package ticketing

import "testing"

func TestFingerprintStable(t *testing.T) {
	subj := "Secret/mcp/openproject-secrets/openproject-url"
	a := Fingerprint(subj, "gpu-cluster")
	b := Fingerprint(subj, "gpu-cluster")
	if a != b {
		t.Fatalf("fingerprint not stable: %q != %q", a, b)
	}
	if len(a) != len("cha-")+16 {
		t.Fatalf("unexpected fingerprint length: %q", a)
	}
}

func TestFingerprintCaseAndSpaceInsensitive(t *testing.T) {
	a := Fingerprint("Secret/mcp/openproject-secrets/openproject-url", "gpu-cluster")
	b := Fingerprint("  secret/mcp/openproject-secrets/openproject-url  ", "GPU-Cluster")
	if a != b {
		t.Fatalf("normalisation failed: %q != %q", a, b)
	}
}

func TestFingerprintDistinguishesClusters(t *testing.T) {
	subj := "Pod/default/example"
	a := Fingerprint(subj, "prod")
	b := Fingerprint(subj, "staging")
	if a == b {
		t.Fatalf("cluster scope ignored: %q == %q", a, b)
	}
}

func TestFingerprintEmptyClusterAllowed(t *testing.T) {
	a := Fingerprint("Pod/default/example", "")
	if a == "" {
		t.Fatal("empty cluster produced empty fingerprint")
	}
	b := Fingerprint("Pod/default/example", "default")
	if a == b {
		t.Fatal("empty cluster should differ from cluster=default")
	}
}
