// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package ticketing

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

// Fingerprint computes the stable dedup key for a ticket. The
// implementation matches the convention used by DriftReport CR naming
// (see internal/report/driftreport.go:nameForSubject) so an operator
// looking at a DriftReport and the ticket it produced can correlate
// them by hash.
//
// Inputs are normalised to lowercase and trimmed; empty cluster is
// allowed (single-cluster deployments) but produces a different
// fingerprint than cluster="default".
func Fingerprint(subject, cluster string) string {
	subject = strings.TrimSpace(strings.ToLower(subject))
	cluster = strings.TrimSpace(strings.ToLower(cluster))
	h := sha256.Sum256([]byte(cluster + "|" + subject))
	return "cha-" + hex.EncodeToString(h[:8])
}
