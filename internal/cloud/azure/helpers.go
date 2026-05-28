// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package azure

import (
	"fmt"

	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/pkg/probe"
)

// skipped is the canonical "Azure not configured" probe result.
func skipped(name, reason string) probe.Result {
	return probe.Result{
		Component: probe.ComponentResult{Component: name, Status: "SKIPPED", Detail: reason},
	}
}

// probeFailed is the canonical "we tried, the API said no" result.
func probeFailed(name, op string, err error) probe.Result {
	return probe.Result{
		Component: probe.ComponentResult{Component: name, Status: "PROBE_FAILED", Detail: fmt.Sprintf("%s: %v", op, err)},
	}
}
