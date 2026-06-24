// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package diagnose

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/internal/snapshot"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// SecretKeyMissing ports diagnose_secret_key_missing from
// cluster-health-report.sh:443-490.
//
// Walks every pod in CreateContainerConfigError, extracts the missing
// Secret key + Secret namespace/name from the kubelet event, identifies
// the consuming Deployment (Pod → ReplicaSet → Deployment) and the owning
// ExternalSecret (in the same namespace targeting the same Secret name),
// and emits a single precise Diagnostic per (Secret, key) pair.
//
// Resolution requires a Vault / ExternalSecret / Deployment edit via git;
// the analyzer never auto-applies anything.
type SecretKeyMissing struct{}

// Name returns the analyzer's identifier.
func (SecretKeyMissing) Name() string { return "SecretKeyMissing" }

// Match the kubelet-event substrings:
//
//	couldn't find key LIVEKIT_API_KEY in Secret livekit/livekit-api-keys
//	couldnt find key OPENPROJECT_URL in Secret mcp/mcp-openproject-secrets   (apostrophe-stripped variant)
var (
	reCouldntFindKey = regexp.MustCompile(`couldn'?t find key ([A-Za-z0-9_.\-]+)`)
	reInSecret       = regexp.MustCompile(`in Secret ([a-z0-9-]+)/([a-z0-9.\-]+)`)
)

// Run produces the diagnostic set.
func (SecretKeyMissing) Run(ctx context.Context, src snapshot.Source) []Diagnostic {
	pods, err := src.List(ctx, snapshot.GVRPod, "")
	if err != nil || len(pods.Items) == 0 {
		logListFailure("pods", err, true) // silent when the CRD/resource is absent; logs Forbidden etc.
		return nil
	}

	seen := map[string]struct{}{}
	var out []Diagnostic

	for i := range pods.Items {
		pod := pods.Items[i]
		if !podHasCCE(pod) {
			continue
		}
		evt := collectErrorMessages(ctx, src, pod)
		if evt == "" {
			continue
		}
		mKey := reCouldntFindKey.FindStringSubmatch(evt)
		if len(mKey) != 2 {
			continue
		}
		mSecret := reInSecret.FindStringSubmatch(evt)
		if len(mSecret) != 3 {
			continue
		}
		missingKey := mKey[1]
		secretNS := mSecret[1]
		secretName := mSecret[2]
		dedupe := secretNS + "/" + secretName + "/" + missingKey
		if _, dup := seen[dedupe]; dup {
			continue
		}
		seen[dedupe] = struct{}{}

		parent := podConsumerDeployment(ctx, src, pod)
		esoName := findOwningExternalSecret(ctx, src, secretNS, secretName)

		hint := fmt.Sprintf("Secret `%s/%s` missing key `%s`", secretNS, secretName, missingKey)
		if parent != "" {
			hint += " (referenced by " + parent + " in ns " + pod.GetNamespace() + ")"
		} else {
			hint += " (referenced by " + pod.GetNamespace() + "/" + pod.GetName() + ")"
		}
		hint += "."
		if esoName != "" {
			hint += fmt.Sprintf(" Owning ExternalSecret: `%s/%s` — add data/template entry exposing `%s`, or remove the env reference if unused.", secretNS, esoName, missingKey)
		} else {
			hint += " No owning ExternalSecret — Secret is hand-managed."
		}
		out = append(out, Diagnostic{
			Subject: "Secret/" + dedupe,
			// Critical: a pod stuck in CreateContainerConfigError on a missing
			// Secret key never starts — the workload is hard-down. Matches the
			// website/docs, which label SecretKeyMissing a Critical analyzer.
			// (Previously emitted with empty severity → normalized to "warning".)
			Severity: "critical",
			Message:  hint,
		})
	}
	return out
}

// podHasCCE returns true if any container's waiting reason is
// CreateContainerConfigError. Mirrors the bash awk on the STATUS column,
// but works on the structured pod state instead of `kubectl get` text.
func podHasCCE(pod unstructured.Unstructured) bool {
	statuses, _, _ := unstructured.NestedSlice(pod.Object, "status", "containerStatuses")
	for _, s := range statuses {
		sm, ok := s.(map[string]any)
		if !ok {
			continue
		}
		state, _ := sm["state"].(map[string]any)
		waiting, _ := state["waiting"].(map[string]any)
		if reason, _ := waiting["reason"].(string); reason == "CreateContainerConfigError" {
			return true
		}
	}
	return false
}

// collectErrorMessages joins every "couldn't find key …" message we can find
// for a pod, drawing from container waiting messages plus events involving
// the pod. Snapshot mode typically only has events if the user captured them;
// containerStatuses messages are present in any pod export.
func collectErrorMessages(ctx context.Context, src snapshot.Source, pod unstructured.Unstructured) string {
	var parts []string
	statuses, _, _ := unstructured.NestedSlice(pod.Object, "status", "containerStatuses")
	for _, s := range statuses {
		sm, ok := s.(map[string]any)
		if !ok {
			continue
		}
		state, _ := sm["state"].(map[string]any)
		waiting, _ := state["waiting"].(map[string]any)
		if msg, _ := waiting["message"].(string); msg != "" {
			parts = append(parts, msg)
		}
	}
	events, err := src.List(ctx, snapshot.GVREvent, pod.GetNamespace())
	if err == nil {
		for _, e := range events.Items {
			involved, _, _ := unstructured.NestedMap(e.Object, "involvedObject")
			if involved["name"] != pod.GetName() {
				continue
			}
			msg, _, _ := unstructured.NestedString(e.Object, "message")
			if msg != "" {
				parts = append(parts, msg)
			}
		}
	}
	return strings.Join(parts, "\n")
}

// podConsumerDeployment walks Pod → ReplicaSet → Deployment owner chain and
// returns "Deployment/<name>" when found, or empty string. Pods owned by a
// Job (or unowned) return empty since the bash version reported the kind/name
// directly for those cases.
func podConsumerDeployment(ctx context.Context, src snapshot.Source, pod unstructured.Unstructured) string {
	owners := pod.GetOwnerReferences()
	if len(owners) == 0 {
		return ""
	}
	owner := owners[0]
	switch owner.Kind {
	case "ReplicaSet":
		rs, err := src.Get(ctx, snapshot.GVRReplicaSet, pod.GetNamespace(), owner.Name)
		if err != nil {
			return "ReplicaSet/" + owner.Name
		}
		rsOwners := rs.GetOwnerReferences()
		if len(rsOwners) > 0 && rsOwners[0].Kind == "Deployment" {
			return "Deployment/" + rsOwners[0].Name
		}
		return "ReplicaSet/" + owner.Name
	case "Job":
		return "Job/" + owner.Name
	default:
		return owner.Kind + "/" + owner.Name
	}
}

// findOwningExternalSecret returns the name of the ExternalSecret in the
// given namespace whose .spec.target.name matches the Secret name, or empty
// string if none / the CRD is not installed.
func findOwningExternalSecret(ctx context.Context, src snapshot.Source, ns, secretName string) string {
	list, err := src.List(ctx, snapshot.GVRExtSecret, ns)
	if err != nil {
		return ""
	}
	for _, eso := range list.Items {
		target, _, _ := unstructured.NestedString(eso.Object, "spec", "target", "name")
		if target == secretName {
			return eso.GetName()
		}
		// Some templates omit spec.target.name; ESO defaults the target name
		// to the ExternalSecret's metadata.name in that case.
		if target == "" && eso.GetName() == secretName {
			return eso.GetName()
		}
	}
	return ""
}
