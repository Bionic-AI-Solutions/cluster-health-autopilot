// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

package ai

import (
	"context"
	"time"

	"github.com/srenix-ai/agentic-sre/pkg/diagnose"
	"github.com/srenix-ai/agentic-sre/pkg/probe"
)

// Investigator runs deep-dive read-only diagnostics on a CRITICAL finding
// before it surfaces to operators. The output (a short narrative summary
// plus structured observations) attaches to the Diagnostic/Finding and
// becomes part of the DriftReport spec, the Slack post, and any downstream
// AI-tier prompts.
//
// Two implementation strategies coexist:
//
//   - The OSS catalog ships a deterministic, rule-based Investigator that
//     pattern-matches the failure mode and runs a fixed set of follow-up
//     probes (DNS / HTTP / TLS / describe / events). No LLM cost; always
//     available; covers the most common failure patterns out of the box.
//
//   - The paid Srenix Enterprise binary ships an LLM-backed Investigator that picks
//     follow-up probes dynamically from the same closed Environment action
//     enum. Same interface; subject to the same RBAC ceiling.
//
// Contract:
//
//   - MUST NOT mutate cluster state — all tools in the Environment are
//     read-only.
//   - MUST respect ctx cancellation; a hard timeout is enforced by the
//     watcher (default ~20 s).
//   - SHOULD return (zero, nil) on its own failure; deterministic
//     diagnostics continue to flow regardless. Investigation is additive,
//     never blocking.
//   - The action surface is constrained to Environment — implementations
//     cannot invent commands or shell out.
type Investigator interface {
	Name() string

	// InvestigateDiagnostic runs the investigation on an analyzer Diagnostic
	// (Secret / ExternalSecret / Certificate-class findings). Implementations
	// should derive what to probe from d.Subject, d.Source, and d.Message.
	InvestigateDiagnostic(ctx context.Context, d diagnose.Diagnostic, env Environment) (InvestigationResult, error)

	// InvestigateFinding runs the investigation on a probe Finding (External
	// Endpoint failures, probe.Result failures). Implementations derive what
	// to probe from f.Component and f.Message.
	InvestigateFinding(ctx context.Context, f probe.Finding, env Environment) (InvestigationResult, error)
}

// InvestigationResult is the structured payload produced by an Investigator.
//
// The Summary is intended for human consumption — it appears in Slack and on
// the DriftReport. Observations are the raw tool outputs that fed the
// summary, kept for audit / replay / downstream AI consumption. Cost tracks
// resources used by the investigation itself, primarily for the LLM-backed
// tier (rule-based investigations have negligible cost).
type InvestigationResult struct {
	// Summary is a one-to-four-sentence human-readable narrative explaining
	// what the investigation concluded. Capped at MaxInvestigationSummaryChars.
	Summary string `json:"summary,omitempty"`

	// Observations is the ordered list of tool invocations the investigator
	// ran, each with its result. Renderers may surface a compact form;
	// audit/replay consumers get the full record.
	Observations []Observation `json:"observations,omitempty"`

	// Conclusion classifies what the investigation determined. Used by
	// downstream consumers (Slack severity routing, AI-tier prompt
	// selection) to decide how prominently to surface the finding.
	Conclusion Conclusion `json:"conclusion,omitempty"`

	// Cost tracks resources consumed by the investigation itself.
	Cost Cost `json:"cost,omitempty"`
}

// MaxInvestigationSummaryChars bounds Summary text length. Enforced by
// the validator before the result is attached to a Diagnostic/Finding.
const MaxInvestigationSummaryChars = 800

// Observation is one tool invocation captured for audit. The Tool field
// names which Environment method was called; Args records the inputs;
// Result is a free-form short description suitable for Slack rendering.
type Observation struct {
	Tool    string        `json:"tool"`
	Args    string        `json:"args,omitempty"`
	Result  string        `json:"result,omitempty"`
	Error   string        `json:"error,omitempty"`
	Elapsed time.Duration `json:"elapsed_ms,omitempty"`
}

// Conclusion is the investigator's high-level classification of the issue.
type Conclusion string

// Conclusion values. Renderers and downstream consumers may filter on these.
const (
	// ConclusionUnknown is the zero value — investigation ran but could not
	// classify. Treat as a regular finding without investigation enrichment.
	ConclusionUnknown Conclusion = ""

	// ConclusionConfirmedOutage means the investigation reproduced or
	// corroborated the failure. The original finding is real and actionable.
	ConclusionConfirmedOutage Conclusion = "confirmed_outage"

	// ConclusionLikelyTransient means the investigation found evidence
	// suggesting the failure was momentary (recovered on retry, intermittent
	// DNS, etc.). The watcher may down-rank or suppress.
	ConclusionLikelyTransient Conclusion = "likely_transient"

	// ConclusionRootCauseIdentified means the investigation surfaced the
	// underlying cause (e.g. expired cert, missing endpoint, OOMKilled pod).
	// The Summary should describe the root cause specifically.
	ConclusionRootCauseIdentified Conclusion = "root_cause_identified"

	// ConclusionInsufficientData means the investigation could not gather
	// enough signal (e.g. host had no DNS, no Ingress, no service). The
	// finding stands as-is.
	ConclusionInsufficientData Conclusion = "insufficient_data"
)

// IsValid reports whether c is a recognized Conclusion value.
func (c Conclusion) IsValid() bool {
	switch c {
	case ConclusionUnknown, ConclusionConfirmedOutage, ConclusionLikelyTransient,
		ConclusionRootCauseIdentified, ConclusionInsufficientData:
		return true
	}
	return false
}

// Cost captures resources spent on one investigation. For rule-based
// investigators most fields stay zero; LLM-backed implementations populate
// TokensIn/TokensOut for billing-side accounting.
type Cost struct {
	WallTime  time.Duration `json:"wall_ms,omitempty"`
	ToolCalls int           `json:"tool_calls,omitempty"`
	TokensIn  int           `json:"tokens_in,omitempty"`
	TokensOut int           `json:"tokens_out,omitempty"`
}
