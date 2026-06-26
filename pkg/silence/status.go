// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

package silence

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	chav1alpha1 "github.com/srenix-ai/agentic-sre/api/v1alpha1"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
)

// StatusPatcher is the narrow mutation surface the status writer needs
// — just /status patches. Satisfied by internal/snapshot.Mutator and
// pkg/snapshot.Mutator (the watcher passes snapshot.AsMutator(live)).
type StatusPatcher interface {
	PatchStatus(ctx context.Context, gvr schema.GroupVersionResource, ns, name string, patchType types.PatchType, patch []byte) error
}

// UpdateStatuses reconciles each Silence's status subresource against
// this cycle's observation, backing the ACTIVE / MATCHED printer
// columns and the spec's "status.active flips to false on expiry"
// contract:
//
//   - status.active     = spec.until > now (the window is open)
//   - status.matchCount += counts[ns/name] (running total — "did this
//     actually mute anything?")
//   - status.lastMatchAt = now when this cycle matched ≥ 1 diagnostic
//
// counts is CountMatches output (ns/name → matches this cycle).
// Writes are cheap by design: a Silence is patched ONLY when its
// active flag flipped or it matched this cycle — steady-state cycles
// with no matches and no expiries patch nothing.
//
// Patch failures are collected, not fatal: one unreachable Silence
// must not block status on the rest (same soft-fail posture as the
// filter itself). Returns the patched count and the joined errors.
func UpdateStatuses(ctx context.Context, p StatusPatcher, silences []chav1alpha1.Silence, counts map[string]int, now time.Time) (int, error) {
	if p == nil {
		return 0, nil
	}
	patched := 0
	var errs []error
	for i := range silences {
		s := &silences[i]
		key := s.Namespace + "/" + s.Name
		active := s.Spec.Until.After(now)
		matchedThisCycle := int64(counts[key])
		if s.Status.Active == active && matchedThisCycle == 0 {
			continue // no change — suppress the no-op patch
		}
		status := map[string]any{"active": active}
		if matchedThisCycle > 0 {
			status["matchCount"] = s.Status.MatchCount + matchedThisCycle
			status["lastMatchAt"] = now.UTC().Format(time.RFC3339)
		}
		body, err := json.Marshal(map[string]any{"status": status})
		if err != nil {
			errs = append(errs, fmt.Errorf("silence %s: marshal status: %w", key, err))
			continue
		}
		// Merge patch against /status — the CRD declares
		// subresources.status:{}, so a main-resource patch would
		// silently drop the status body (same gotcha as DriftReport
		// reconcile; see internal/report/driftreport.go).
		if err := p.PatchStatus(ctx, silenceGVR, s.Namespace, s.Name, types.MergePatchType, body); err != nil {
			errs = append(errs, fmt.Errorf("silence %s: patch status: %w", key, err))
			continue
		}
		patched++
	}
	return patched, errors.Join(errs...)
}
