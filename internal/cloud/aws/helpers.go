// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"fmt"

	"github.com/srenix-ai/agentic-sre/pkg/probe"
)

// skipped is the canonical "AWS not configured" probe result.
func skipped(name, reason string) probe.Result {
	return probe.Result{
		Component: probe.ComponentResult{
			Component: name,
			Status:    "SKIPPED",
			Detail:    reason,
		},
	}
}

// probeFailed is the canonical "we tried, the API said no" probe result.
func probeFailed(name, op string, err error) probe.Result {
	return probe.Result{
		Component: probe.ComponentResult{
			Component: name,
			Status:    "PROBE_FAILED",
			Detail:    fmt.Sprintf("%s: %v", op, err),
		},
	}
}
