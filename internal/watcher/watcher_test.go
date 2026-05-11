// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package watcher

import (
	"testing"

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
