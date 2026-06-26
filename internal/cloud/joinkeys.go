// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

package cloud

import (
	"fmt"
	"strings"
)

// CROSS-REPO CONTRACT — Srenix Enterprise's cross-resource RCA matchers
// (ai/cloudcontext, PR #65) join Kubernetes resources to cloud findings
// by parsing these exact tokens out of the finding MESSAGE. The format
// is frozen: single leading space, literal "(lb: " / "(domains: ",
// comma-separated domains with NO spaces, closing paren. Changing it
// breaks Srenix Enterprise's fixtures — see contract_test.go in this package.

// JoinKeyLB returns the " (lb: <value>)" message suffix the LB probes
// (aws-alb-target-health, gcp-lb-backends, azure-appgw-backends) append
// to their 0-healthy-backend findings. Empty value → empty string (the
// suffix is omitted entirely; never an empty "(lb: )").
func JoinKeyLB(value string) string {
	if value == "" {
		return ""
	}
	return fmt.Sprintf(" (lb: %s)", value)
}

// JoinKeyDomains returns the " (domains: <d1>,<d2>)" message suffix the
// azure-certs probe appends to its certificate findings. Empty entries
// are dropped; no usable domains → empty string (the suffix is omitted
// entirely; never an empty "(domains: )").
func JoinKeyDomains(domains []string) string {
	vals := make([]string, 0, len(domains))
	for _, d := range domains {
		if d != "" {
			vals = append(vals, d)
		}
	}
	if len(vals) == 0 {
		return ""
	}
	return fmt.Sprintf(" (domains: %s)", strings.Join(vals, ","))
}
