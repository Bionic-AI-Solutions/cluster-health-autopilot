// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

// Package rag is the read/write contract for Srenix's per-cluster learnt
// knowledge layer.
//
// The OSS build links only NoopReader / NoopWriter — these literally never
// reach a store. The paid AI tier (srenix-enterprise) links a Qdrant-backed
// implementation that persists across cycles and learns from SRE outcomes.
//
// Why split it this way: every probe/analyzer that ingests learnt state takes
// a rag.Reader by value, so the OSS path is type-identical to the paid path
// at compile time. No build tags, no interface assertions at boot, no
// conditional wiring. Swap the Reader at construction; everything else
// stays the same.
//
// Design doc: docs/design/2026-06-rag-cluster-knowledge.md (Phase 2d).
package rag

import "time"

// EntryKind enumerates the kinds of cluster-learnt knowledge Srenix stores.
// Stored as the partition key inside Qdrant; treated as a stable string —
// renames need a migration.
type EntryKind string

const (
	// KindApexDomain is a public-facing hostname (apex domain, Cloudflare-
	// only host, externally-hosted endpoint) that the cluster should be
	// monitoring even though no Ingress lists it. Seeded by walking
	// Cloudflare zones; refined by importance + SRE click-history.
	KindApexDomain EntryKind = "apex_domain"

	// KindWorkload is a Deployment / StatefulSet / DaemonSet / Job /
	// CronJob observed in the cluster. Stores label hints (tier=critical,
	// priorityClassName, ownerRefs) and traffic stats; replayed as
	// ServiceTargets with criticality driving severity.
	KindWorkload EntryKind = "workload"

	// KindFindingOutcome is one SRE-action signal: the result of a click
	// on an Approve / Deny / Silence URL, or an analyzer-self-resolution.
	// Keyed by (subject_pattern, finding_class). Drives auto-silence and
	// AI proposer ranking.
	KindFindingOutcome EntryKind = "finding_outcome"

	// KindBaseline is per-(workload, metric) rolling statistics — restart
	// distributions, OOM frequency, RPS p50/p95. Drives "is this churn
	// abnormal for THIS cluster?" instead of universal thresholds.
	KindBaseline EntryKind = "baseline"

	// KindSilencedClass mirrors active Silence CRs so the AI tier can
	// reason about long-suppressed classes without re-querying the K8s
	// API every cycle. Auto-populated by the watcher when Silence CRs
	// are created or expire.
	KindSilencedClass EntryKind = "silenced_class"
)

// Entry is one learnt fact about a single cluster. Fields are intentionally
// loose — the schema is owned by each EntryKind and is interpreted by the
// probe/analyzer that consumes it. Features carries kind-specific payload.
type Entry struct {
	// ClusterID identifies the cluster this entry belongs to. Sourced from
	// spec.alerting.alertmanager.clusterName on the AgenticSRE
	// CR. Entries from different clusters are isolated and never federate.
	ClusterID string

	Kind EntryKind

	// Key is the natural identifier within the (ClusterID, Kind) namespace.
	// For KindApexDomain: the hostname ("api.example.com").
	// For KindWorkload:   "<namespace>/<name>".
	// For KindFindingOutcome: "<finding_class>:<subject_pattern>".
	Key string

	// FirstSeen / LastSeen anchor the entry's lifetime in the cluster.
	// LastSeen drives decay — entries unseen for > halflife×N are demoted
	// in importance and eventually GC'd.
	FirstSeen time.Time
	LastSeen  time.Time

	// Observations counts how many analyzer cycles have re-confirmed this
	// entry. Used as a denominator in importance smoothing.
	Observations int

	// Importance is the learned weight in [0,1]. Reader implementations
	// support filtering by ImportanceMin; entries below the configured
	// threshold are not replayed into the probe/analyzer pipeline.
	// Initial value: 0.5 (neutral). Adjusted by:
	//   - +observations × tiny_step           (decays via LastSeen)
	//   - +signal_history events (approved)   (strong positive)
	//   - −signal_history events (denied/silenced for entry's own class)
	Importance float64

	// Sources tracks WHERE this entry was learnt from (multi-source is
	// possible). E.g. an apex_domain seen in both Cloudflare zone listing
	// AND auto-discovered Ingress gets ["cloudflare_zone", "ingress_host"].
	// Useful for explaining "why is Srenix probing this?" to operators.
	Sources []string

	// Features is the kind-specific payload. Example shapes:
	//   KindWorkload: {"replicas": 3, "tier": "critical", "ownerKind": "Helm"}
	//   KindBaseline: {"metric": "restart_count", "p50": 0.0, "p95": 1.2}
	// Empty for entries that don't need it.
	Features map[string]any

	// SignalHistory is the append-only log of SRE click-throughs that
	// touched this entry's subject. Bounded internally by the writer
	// (e.g. last 100 events) to keep entries from growing unboundedly.
	SignalHistory []SignalEvent
}

// SignalEvent is one row of SRE click-history.
type SignalEvent struct {
	Timestamp time.Time

	// Action ∈ {"approved", "denied", "silenced", "opened", "resolved"}.
	// "approved" / "denied" come from srenix-enterprise #17 symmetric one-shot
	// approval-server clicks. "silenced" comes from Silence-CR creation
	// events watched by the watcher. "opened" / "resolved" come from the
	// analyzer cycle itself.
	Action string

	// FindingClass is the analyzer subject-class that fired (e.g.
	// "DNSChainDrift/duplicate-ingress-host", "HPAScaling"). Lets the
	// outcome reasoner distinguish "user denied a CrashLoopBackOff fix"
	// from "user denied a digest-pin warning".
	FindingClass string

	// Actor is the JWT subject claim from the approval-server token, or
	// "system" for analyzer-self-resolutions. Empty when no auth was in
	// the path.
	Actor string
}

// Query is a read-side filter for Reader.List.
//
// Zero-value semantics: every field optional; an empty Query returns
// everything in the (ClusterID, Reader) scope. Most call sites only set
// Kind and ImportanceMin.
type Query struct {
	// Kind constrains results to a single kind. Empty = all kinds.
	Kind EntryKind

	// ImportanceMin filters out entries below this learned-weight floor.
	// Zero accepts everything (including importance=0). Use 0.3 as the
	// default for "include unless explicitly de-ranked".
	ImportanceMin float64

	// Limit caps the result set. Zero = no limit. Implementations should
	// return the top-Limit by Importance descending when set.
	Limit int

	// Source, when set, requires the entry's Sources slice to contain
	// this string. Useful for "give me only the Cloudflare-zone-learnt
	// apex domains" filters.
	Source string
}
