// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package diagnose

import (
	"context"
	"fmt"
	"strings"

	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/internal/snapshot"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// FailingExternalSecrets ports diagnose_failing_externalsecrets from
// cluster-health-report.sh:496-520.
//
// Walks every ExternalSecret cluster-wide whose status.conditions[Ready] is
// False, fetches the controller's most recent UpdateFailed event (which
// carries the precise missing Vault property name), and emits one Diagnostic
// per failing ExternalSecret.
//
// Catches the silently-failing class where someone updates a Deployment
// template to consume new Secret keys but never seeds the corresponding
// Vault data — the new pod gets stuck while the ExternalSecret retries
// indefinitely, invisible to per-component dashboards.
type FailingExternalSecrets struct{}

// Name returns the analyzer's identifier.
func (FailingExternalSecrets) Name() string { return "FailingExternalSecrets" }

// Run walks every ExternalSecret and emits a diagnostic for each that is
// not Ready. Returns nil when the ESO CRD is not installed.
func (FailingExternalSecrets) Run(ctx context.Context, src snapshot.Source) []Diagnostic {
	list, err := src.List(ctx, snapshot.GVRExtSecret, "")
	if err != nil || len(list.Items) == 0 {
		return nil
	}

	var out []Diagnostic
	for i := range list.Items {
		eso := list.Items[i]
		readyStatus, readyMsg := readReadyCondition(eso)
		if readyStatus == "True" || readyStatus == "" {
			// True → healthy; "" → no Ready condition yet (just-created); skip both.
			continue
		}

		ns := eso.GetNamespace()
		name := eso.GetName()
		// Prefer the controller's most recent UpdateFailed event, which usually
		// carries the precise missing-property name. Fall back to the condition
		// message if no event was captured (snapshot mode without events).
		errMsg := latestUpdateFailedMessage(ctx, src, ns, name)
		if errMsg == "" {
			errMsg = readyMsg
		}
		if errMsg == "" {
			errMsg = "unknown error"
		}
		// Cap the line length for the Slack render.
		errMsg = truncate(strings.TrimSpace(errMsg), 200)

		out = append(out, Diagnostic{
			Subject: "ExternalSecret/" + ns + "/" + name,
			Message: fmt.Sprintf(
				"ExternalSecret `%s/%s` not Ready: %s. Check Vault path / property names.",
				ns, name, errMsg,
			),
		})
	}
	return out
}

// readReadyCondition returns (status, message) of the Ready condition on
// the ExternalSecret status, or empty strings if not present.
func readReadyCondition(eso unstructured.Unstructured) (string, string) {
	conds, _, _ := unstructured.NestedSlice(eso.Object, "status", "conditions")
	for _, c := range conds {
		cm, ok := c.(map[string]any)
		if !ok {
			continue
		}
		if t, _ := cm["type"].(string); t == "Ready" {
			status, _ := cm["status"].(string)
			msg, _ := cm["message"].(string)
			return status, msg
		}
	}
	return "", ""
}

// latestUpdateFailedMessage finds the most recent Event whose involvedObject
// is the named ExternalSecret and reason is UpdateFailed, and returns its
// message. Returns "" if the snapshot has no events for it.
//
// "Most recent" is approximated by lastTimestamp — sufficient since ESO
// generally reconciles every 5m and we only care about the current error.
func latestUpdateFailedMessage(ctx context.Context, src snapshot.Source, ns, name string) string {
	events, err := src.List(ctx, snapshot.GVREvent, ns)
	if err != nil {
		return ""
	}
	var best string
	var bestTS string
	for i := range events.Items {
		e := events.Items[i]
		reason, _, _ := unstructured.NestedString(e.Object, "reason")
		if reason != "UpdateFailed" {
			continue
		}
		objName, _, _ := unstructured.NestedString(e.Object, "involvedObject", "name")
		if objName != name {
			continue
		}
		ts, _, _ := unstructured.NestedString(e.Object, "lastTimestamp")
		if ts == "" {
			ts, _, _ = unstructured.NestedString(e.Object, "eventTime")
		}
		if best == "" || ts > bestTS {
			msg, _, _ := unstructured.NestedString(e.Object, "message")
			best = msg
			bestTS = ts
		}
	}
	return best
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
