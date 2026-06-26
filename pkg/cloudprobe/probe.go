// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

// Package cloudprobe defines the Probe interface cloud-resource
// probes implement. Parallel to pkg/probe.Probe but takes a
// cloud.Source instead of a snapshot.Source.
//
// Reuses probe.ComponentResult and probe.Result so downstream
// rendering (Slack, Alertmanager, DriftReport, ticketing) needs zero
// changes. Subject convention for findings:
//
//	"aws-rds/<region>/<db-id>"
//	"gcp-cloudsql/<project>/<instance>"
//	"azure-sql/<rg>/<server>/<db>"
//
// See docs/design/2026-05-cloud-probe-framework.md for the full
// design.
package cloudprobe

import (
	"context"

	"github.com/srenix-ai/agentic-sre/pkg/cloud"
	"github.com/srenix-ai/agentic-sre/pkg/probe"
)

// Probe is the cloud-resource counterpart to pkg/probe.Probe. Shape
// matches Probe (Name + Run) so the catalog can register and iterate
// K8s and cloud probes uniformly.
//
// Implementations:
//   - Inspect a single resource type (e.g., AWS RDS instances, GCP
//     Cloud SQL instances, Azure Disk attachments)
//   - Return probe.Result; findings flow through the existing
//     report.AssembleEntries path unchanged
//   - Refuse cleanly (return Result with Component.Status="SKIPPED")
//     when their required cloud sub-client is nil — never panic on nil
//   - Are SAFE to call concurrently with other probes; per-cycle
//     rate limiting is applied at the catalog level, not per-probe
//   - Must NEVER mutate cloud resources — that lands in a separate
//     design with its own safety envelope (M4)
type Probe interface {
	// Name returns the component label rendered in Slack, DriftReport,
	// Alertmanager. Convention: "aws-rds", "gcp-cloudsql", "azure-sql"
	// — kebab-case, provider-prefixed.
	Name() string

	// Run inspects the cloud Source and returns zero or more findings.
	// Must not mutate. Must handle nil sub-client by returning a
	// probe.Result with Component.Status="SKIPPED".
	Run(ctx context.Context, src cloud.Source) probe.Result
}
