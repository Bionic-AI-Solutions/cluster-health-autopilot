// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package watcher

import (
	"testing"
	"time"

	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/internal/diagnose"
	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/internal/fix"
	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/internal/probe"
)

// TestDiffUsesPreFixStateWhenFixersAct verifies that when fixers delete
// resources in the same cycle they are detected, the Slack diff still
// reports them as "new" issues rather than silently posting only the
// "Fixes Applied" block with no context.
//
// Regression test for: watcher used post-fix buildCurrentState for the
// diff, so immediately-remediated pods never appeared in toPost.
func TestDiffUsesPreFixStateWhenFixersAct(t *testing.T) {
	w := &Watcher{
		cfg:  Config{PostOnResolved: true},
		seen: make(map[string]*seenEntry),
	}

	// Simulate pre-fix diagnose finding one stale pod diagnostic.
	preDiags := []diagnose.Diagnostic{
		{Subject: "stale-pod/demo-app/debug-probe", Message: "pod in Failed state"},
	}
	preFix := buildCurrentState([]probe.Result{}, preDiags)

	// Post-fix diagnose: pod is gone — empty state.
	postFix := buildCurrentState([]probe.Result{}, []diagnose.Diagnostic{})

	// Fixers had one action.
	fixResults := []fix.Result{{
		Fixer:   "StaleErrorPods",
		Actions: []fix.Action{{Description: "Deleted stale Failed pod", Object: "Pod/demo-app/debug-probe"}},
	}}

	// The watcher should use preFix for the diff when fixers acted.
	diffState := postFix
	if hasActions(fixResults) {
		diffState = preFix
	}

	w.mu.Lock()
	toPost, toResolve := w.diff(diffState)
	w.mu.Unlock()

	if len(toPost) != 1 {
		t.Errorf("toPost = %d, want 1 — pre-fix diagnostic must appear as new issue", len(toPost))
	}
	if toPost[0].subject != "stale-pod/demo-app/debug-probe" {
		t.Errorf("toPost[0].subject = %q, want stale-pod/demo-app/debug-probe", toPost[0].subject)
	}
	if len(toResolve) != 0 {
		t.Errorf("toResolve = %d, want 0 — issue not yet in seen so cannot resolve", len(toResolve))
	}
}

// TestDiffPostFixStatePersistedToSeen verifies that after a fix cycle,
// the seen map is updated from the POST-fix state, not the pre-fix state.
// This ensures the stale-pod subject does not linger in seen and trigger a
// spurious "resolved" post on the next cycle.
func TestDiffPostFixStatePersistedToSeen(t *testing.T) {
	w := &Watcher{
		cfg:  Config{PostOnResolved: true},
		seen: make(map[string]*seenEntry),
	}

	preDiags := []diagnose.Diagnostic{
		{Subject: "stale-pod/demo-app/debug-probe", Message: "pod in Failed state"},
	}
	preFix := buildCurrentState([]probe.Result{}, preDiags)
	postFix := buildCurrentState([]probe.Result{}, []diagnose.Diagnostic{})

	fixResults := []fix.Result{{
		Fixer:   "StaleErrorPods",
		Actions: []fix.Action{{Description: "Deleted stale Failed pod", Object: "Pod/demo-app/debug-probe"}},
	}}

	diffState := postFix
	if hasActions(fixResults) {
		diffState = preFix
	}

	w.mu.Lock()
	toPost, _ := w.diff(diffState)
	// Persist post-fix state to seen (not pre-fix).
	w.updateSeen(postFix, toPost)
	w.mu.Unlock()

	// Next cycle: no stale pods again.
	w.mu.Lock()
	_, toResolve2 := w.diff(postFix)
	w.mu.Unlock()

	if len(toResolve2) != 0 {
		t.Errorf("toResolve on next cycle = %d, want 0 — post-fix seen must not contain the fixed subject", len(toResolve2))
	}
}

// --- Sprint 4.1 — pure-logic dedup tests ---------------------------------

func TestFingerprint_Deterministic(t *testing.T) {
	a := fingerprint("Pod/ns/p", "critical", "x failed")
	b := fingerprint("Pod/ns/p", "critical", "x failed")
	if a != b {
		t.Errorf("fingerprint not deterministic: %s vs %s", a, b)
	}
}

func TestFingerprint_DifferentSeverity_DifferentFP(t *testing.T) {
	if fingerprint("S", "warning", "msg") == fingerprint("S", "critical", "msg") {
		t.Error("severity change must alter fingerprint")
	}
}

func TestFingerprint_DifferentMessage_DifferentFP(t *testing.T) {
	if fingerprint("S", "warning", "msg-1") == fingerprint("S", "warning", "msg-2") {
		t.Error("message change must alter fingerprint")
	}
}

func TestBuildCurrentState_KeyedBySubject(t *testing.T) {
	diags := []diagnose.Diagnostic{
		{Subject: "Pod/ns/a", Message: "broken", Severity: "critical"},
		{Subject: "Pod/ns/b", Message: "warning", Severity: "warning"},
	}
	got := buildCurrentState(nil, diags)
	if len(got) != 2 || got["Pod/ns/a"].severity != "critical" {
		t.Errorf("buildCurrentState: %+v", got)
	}
}

func TestBuildCurrentState_DefaultSeverityWarning(t *testing.T) {
	diags := []diagnose.Diagnostic{{Subject: "X/ns/y", Message: "z"}}
	got := buildCurrentState(nil, diags)
	if e := got["X/ns/y"]; e == nil || e.severity != "warning" {
		t.Errorf("expected default severity=warning, got %+v", e)
	}
}

func TestDiff_NewSubject_ToPost(t *testing.T) {
	w := &Watcher{seen: map[string]*seenEntry{}}
	cur := map[string]*seenEntry{
		"S": {subject: "S", fp: "fp-1", severity: "critical", message: "broken"},
	}
	toPost, toResolve := w.diff(cur)
	if len(toPost) != 1 || toPost[0].subject != "S" {
		t.Errorf("new subject should be in toPost; got %+v", toPost)
	}
	if len(toResolve) != 0 {
		t.Errorf("nothing to resolve; got %+v", toResolve)
	}
}

func TestDiff_IdenticalFingerprint_NoPost(t *testing.T) {
	w := &Watcher{seen: map[string]*seenEntry{
		"S": {subject: "S", fp: "fp-1", lastPosted: time.Now()},
	}}
	cur := map[string]*seenEntry{
		"S": {subject: "S", fp: "fp-1", severity: "critical", message: "broken"},
	}
	toPost, _ := w.diff(cur)
	if len(toPost) != 0 {
		t.Errorf("same fp should not re-post; got %+v", toPost)
	}
}

func TestDiff_ChangedFingerprint_ToPost(t *testing.T) {
	w := &Watcher{seen: map[string]*seenEntry{
		"S": {subject: "S", fp: "fp-old", lastPosted: time.Now()},
	}}
	cur := map[string]*seenEntry{
		"S": {subject: "S", fp: "fp-new", severity: "critical"},
	}
	toPost, _ := w.diff(cur)
	if len(toPost) != 1 {
		t.Errorf("changed fp should re-post; got %+v", toPost)
	}
}

func TestDiff_RepeatIntervalElapsed_ToPost(t *testing.T) {
	old := time.Now().Add(-2 * time.Hour)
	w := &Watcher{
		seen: map[string]*seenEntry{
			"S": {subject: "S", fp: "fp-1", lastPosted: old},
		},
		cfg: Config{RepeatInterval: time.Hour},
	}
	cur := map[string]*seenEntry{"S": {subject: "S", fp: "fp-1", severity: "critical"}}
	if toPost, _ := w.diff(cur); len(toPost) != 1 {
		t.Errorf("past RepeatInterval should re-post; got %+v", toPost)
	}
}

// Per-cycle delta render (Phase 1.E): the routing layer needs to know
// whether each posted entry is freshly-appeared OR a re-post of a
// stable finding. diff() must set isNewThisCycle accordingly.

func TestDiff_NewSubject_MarksIsNewThisCycle(t *testing.T) {
	w := &Watcher{seen: map[string]*seenEntry{}}
	cur := map[string]*seenEntry{
		"S": {subject: "S", fp: "fp-1", severity: "critical", message: "broken"},
	}
	toPost, _ := w.diff(cur)
	if len(toPost) != 1 {
		t.Fatalf("expected 1 toPost; got %d", len(toPost))
	}
	if !toPost[0].isNewThisCycle {
		t.Errorf("new subject must be marked isNewThisCycle=true")
	}
}

func TestDiff_ChangedFingerprint_MarksIsNewThisCycle(t *testing.T) {
	w := &Watcher{seen: map[string]*seenEntry{
		"S": {subject: "S", fp: "fp-old", lastPosted: time.Now()},
	}}
	cur := map[string]*seenEntry{
		"S": {subject: "S", fp: "fp-new", severity: "critical"},
	}
	toPost, _ := w.diff(cur)
	if len(toPost) != 1 || !toPost[0].isNewThisCycle {
		t.Errorf("changed-fp re-post must be marked isNewThisCycle=true; got %+v", toPost)
	}
}

func TestDiff_RepeatIntervalElapsed_NotIsNewThisCycle(t *testing.T) {
	old := time.Now().Add(-2 * time.Hour)
	w := &Watcher{
		seen: map[string]*seenEntry{
			"S": {subject: "S", fp: "fp-1", lastPosted: old},
		},
		cfg: Config{RepeatInterval: time.Hour},
	}
	cur := map[string]*seenEntry{"S": {subject: "S", fp: "fp-1", severity: "critical"}}
	toPost, _ := w.diff(cur)
	if len(toPost) != 1 {
		t.Fatalf("expected 1 toPost; got %d", len(toPost))
	}
	if toPost[0].isNewThisCycle {
		t.Errorf("repeat-interval re-post (same fp) must be marked isNewThisCycle=false; got true")
	}
}

// Per-severity repeat interval (v1.6.1):
// critical alerts re-post at the critical interval; warnings stay quiet
// until the (longer) regular interval elapses.
func TestDiff_CriticalRepeatIntervalOverride(t *testing.T) {
	// 2h since last post. Critical=1h should re-post; warning under 24h
	// should NOT.
	old := time.Now().Add(-2 * time.Hour)
	w := &Watcher{
		seen: map[string]*seenEntry{
			"crit": {subject: "crit", fp: "fp-c", severity: "critical", lastPosted: old},
			"warn": {subject: "warn", fp: "fp-w", severity: "warning", lastPosted: old},
		},
		cfg: Config{
			RepeatInterval:         24 * time.Hour,
			CriticalRepeatInterval: time.Hour,
		},
	}
	cur := map[string]*seenEntry{
		"crit": {subject: "crit", fp: "fp-c", severity: "critical"},
		"warn": {subject: "warn", fp: "fp-w", severity: "warning"},
	}
	toPost, _ := w.diff(cur)
	if len(toPost) != 1 {
		t.Fatalf("only critical should re-post; got %d: %+v", len(toPost), toPost)
	}
	if toPost[0].subject != "crit" {
		t.Errorf("expected critical to be re-posted; got %q", toPost[0].subject)
	}
}

// Backward compat: when CriticalRepeatInterval is 0 the per-severity
// helper falls back to RepeatInterval. Pre-v1.6.1 callers see no change.
func TestDiff_CriticalFallsBackToRepeatInterval(t *testing.T) {
	old := time.Now().Add(-2 * time.Hour)
	w := &Watcher{
		seen: map[string]*seenEntry{
			"crit": {subject: "crit", fp: "fp-c", severity: "critical", lastPosted: old},
		},
		cfg: Config{
			RepeatInterval:         time.Hour, // critical should still re-post via fallback
			CriticalRepeatInterval: 0,
		},
	}
	cur := map[string]*seenEntry{
		"crit": {subject: "crit", fp: "fp-c", severity: "critical"},
	}
	if toPost, _ := w.diff(cur); len(toPost) != 1 {
		t.Errorf("fallback to RepeatInterval should re-post critical; got %+v", toPost)
	}
}

func TestDiff_PostOnResolvedOnlyEmitsWhenEnabled(t *testing.T) {
	w := &Watcher{
		seen: map[string]*seenEntry{"old": {subject: "old", fp: "x"}},
		cfg:  Config{PostOnResolved: false},
	}
	if _, toResolve := w.diff(map[string]*seenEntry{}); len(toResolve) != 0 {
		t.Errorf("PostOnResolved=false should suppress; got %+v", toResolve)
	}
	w.cfg.PostOnResolved = true
	if _, toResolve := w.diff(map[string]*seenEntry{}); len(toResolve) != 1 {
		t.Errorf("PostOnResolved=true should emit; got %+v", toResolve)
	}
}

func TestUpdateSeen_RemovesAbsentSubjects(t *testing.T) {
	w := &Watcher{
		seen: map[string]*seenEntry{
			"S1": {subject: "S1", fp: "a"},
			"S2": {subject: "S2", fp: "b"},
		},
	}
	cur := map[string]*seenEntry{"S1": {subject: "S1", fp: "a"}}
	w.updateSeen(cur, nil)
	if _, ok := w.seen["S2"]; ok {
		t.Errorf("S2 should be removed; seen=%+v", w.seen)
	}
}

func TestUpdateSeen_PreservesLastPostedWhenUnPosted(t *testing.T) {
	earlier := time.Now().Add(-time.Hour)
	w := &Watcher{
		seen: map[string]*seenEntry{
			"S": {subject: "S", fp: "a", lastPosted: earlier},
		},
	}
	cur := map[string]*seenEntry{"S": {subject: "S", fp: "a"}}
	w.updateSeen(cur, nil)
	if !w.seen["S"].lastPosted.Equal(earlier) {
		t.Errorf("lastPosted should be preserved; got %v want %v",
			w.seen["S"].lastPosted, earlier)
	}
}

func TestUpdateSeen_RefreshesLastPostedForPosted(t *testing.T) {
	earlier := time.Now().Add(-time.Hour)
	w := &Watcher{
		seen: map[string]*seenEntry{
			"S": {subject: "S", fp: "old", lastPosted: earlier},
		},
	}
	cur := map[string]*seenEntry{"S": {subject: "S", fp: "new"}}
	w.updateSeen(cur, []*seenEntry{cur["S"]})
	if w.seen["S"].lastPosted.Equal(earlier) {
		t.Errorf("lastPosted should be refreshed; got %v (= earlier)", w.seen["S"].lastPosted)
	}
	if w.seen["S"].fp != "new" {
		t.Errorf("fp should be updated; got %q", w.seen["S"].fp)
	}
}

func TestHasActions_NilEmpty(t *testing.T) {
	if hasActions(nil) {
		t.Error("hasActions(nil) should be false")
	}
}

// TestSeenEntryToDeltaDiag_CarriesApprovalFields is the regression test
// for the silent Slack-button outage: prior to this fix, the Slack-bound
// mapping in runCycle dropped ProposedActionID + ApprovalURL while the
// Alertmanager-bound mapping kept them. The result was that every
// AI-tier proposal got an Alertmanager annotation but the Slack post
// rendered without the "✅ Approve · ❌ Deny" line — invisible to
// operators who use Slack as the primary surface. The collapsed helper
// guarantees every destination gets every field.
func TestSeenEntryToDeltaDiag_CarriesApprovalFields(t *testing.T) {
	e := &seenEntry{
		subject:          "Pod/ns/app-xyz",
		severity:         "warning",
		message:          "missing digest pin",
		remediation:      "pin via @sha256:...",
		investigation:    "rule R-IMG-01 flagged this",
		enrichment:       "LLM narrative addendum",
		proposedActionID: "Pod/ns/app-xyz:digest-pin:abcdef012345",
		approvalURL:      "https://cha-approve.example.com/approve?token=eyJ...",
	}
	got := seenEntryToDeltaDiag(e)
	if got.Subject != e.subject ||
		got.Severity != e.severity ||
		got.Message != e.message ||
		got.Remediation != e.remediation ||
		got.Investigation != e.investigation ||
		got.Enrichment != e.enrichment {
		t.Errorf("base fields not propagated: got %+v", got)
	}
	if got.ProposedActionID != e.proposedActionID {
		t.Errorf("ProposedActionID dropped: got %q want %q", got.ProposedActionID, e.proposedActionID)
	}
	if got.ApprovalURL != e.approvalURL {
		t.Errorf("ApprovalURL dropped: got %q want %q (this is the bug that silently disabled Slack Approve/Deny buttons)", got.ApprovalURL, e.approvalURL)
	}
}

// TestSeenEntryToDeltaDiag_AllowsEmptyAIFields confirms a non-AI-tier
// seenEntry (no Enricher / FixProposer registered) still renders cleanly:
// the AI-tier fields are zero values, not garbage.
func TestSeenEntryToDeltaDiag_AllowsEmptyAIFields(t *testing.T) {
	e := &seenEntry{
		subject:  "Pod/ns/legacy",
		severity: "critical",
		message:  "container OOMKilled",
	}
	got := seenEntryToDeltaDiag(e)
	if got.Enrichment != "" || got.ProposedActionID != "" || got.ApprovalURL != "" {
		t.Errorf("non-AI entry leaked AI fields: %+v", got)
	}
}
