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

// ImagePullAuth detects pods stuck in ImagePullBackOff or ErrImagePull due to
// registry authentication failures (wrong or missing imagePullSecret, expired
// credentials, private registry without pull-secret configured).
//
// It deliberately ignores non-auth pull failures such as "image not found" or
// "manifest unknown" — those are deployment errors, not credential issues.
// Only failures whose event messages contain auth-signal keywords are surfaced.
//
// Resolution path: verify the imagePullSecret referenced by the pod (or the
// default ServiceAccount's imagePullSecrets) is present and holds a valid
// .dockerconfigjson for the registry in question.
type ImagePullAuth struct{}

// Name returns the analyzer's identifier.
func (ImagePullAuth) Name() string { return "ImagePullAuth" }

// authSignals are substrings in kubelet pull-failure events that indicate a
// credential problem, as opposed to a missing-image problem.
var authSignals = []string{
	"unauthorized",
	"401",
	"denied",
	"authentication required",
	"no basic auth credentials",
	"pull access denied",
	"access forbidden",
	"403",
	"invalid username/password",
	"credential",
}

// notFoundSignals are substrings that DEFINITIVELY indicate the image tag or
// name does not exist at the registry, even when the event also contains an
// auth-signal keyword. We only include patterns that are unambiguous:
//
//   - "manifest unknown"  — image/tag genuinely absent; returned by most
//     compliant registries (GCR, ECR, GHCR) when the tag does not exist.
//   - "name unknown"      — GCR / Google Artifact Registry specific: the image
//     name/project does not exist at all.
//
// NOT included:
//   - "repository does not exist" — Docker Hub uses this in its combined
//     "repository does not exist or may require 'docker login'" message for
//     BOTH non-existent images AND private images with wrong/missing creds.
//     Including it would suppress real auth-failure diagnostics.
//   - "not found" / "does not exist" — too broad; appear in unrelated contexts.
var notFoundSignals = []string{
	"manifest unknown",
	"name unknown",
}

// Run walks every pod in the cluster. For each pod with a container waiting
// on ImagePullBackOff or ErrImagePull it checks the pod's Warning events for
// auth-signal keywords. Returns one Diagnostic per (namespace, pod, image)
// triple that shows an auth failure; skips non-auth pull errors.
func (ImagePullAuth) Run(ctx context.Context, src snapshot.Source) []Diagnostic {
	pods, err := src.List(ctx, snapshot.GVRPod, "")
	if err != nil || len(pods.Items) == 0 {
		logListFailure("pods", err, true) // silent when the CRD/resource is absent; logs Forbidden etc.
		return nil
	}

	var out []Diagnostic
	seen := map[string]bool{}

	for i := range pods.Items {
		pod := pods.Items[i]
		ns := pod.GetNamespace()
		name := pod.GetName()

		img, containerName := pullBackoffContainer(pod)
		if img == "" {
			continue
		}

		msg := pullAuthEvent(ctx, src, ns, name, img)
		if msg == "" {
			continue // pull failure is not auth-related
		}

		key := ns + "/" + name + "/" + img
		if seen[key] {
			continue
		}
		seen[key] = true

		out = append(out, Diagnostic{
			// Use Pod/ns/name so the investigator can Describe the Pod and
			// inspect its imagePullSecrets. The container name is captured in
			// the Message — it was previously in the Subject which broke the
			// Kind/namespace/name convention that all investigators expect.
			Subject: fmt.Sprintf("Pod/%s/%s", ns, name),
			// Critical: a container that cannot pull its image is hard-down —
			// the workload never runs. This matches the website/docs, which
			// label ImagePullAuth a Critical analyzer. (Previously emitted with
			// an empty severity, which normalizes to "warning" and routed away
			// from the human-action channel.)
			Severity: "critical",
			Message: fmt.Sprintf(
				"Pod `%s/%s` container %q cannot pull image %q: auth failure. "+
					"Check imagePullSecret in pod spec or ServiceAccount. Event: %s",
				ns, name, containerName, truncate(img, 80), truncate(msg, 300),
			),
		})
	}
	return out
}

// pullBackoffContainer returns (image, containerName) for the first container
// in the pod that is waiting on ImagePullBackOff or ErrImagePull, or ("","")
// if none.
func pullBackoffContainer(pod unstructured.Unstructured) (image, containerName string) {
	statuses, _, _ := unstructured.NestedSlice(pod.Object, "status", "containerStatuses")
	for _, raw := range statuses {
		cs, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		state, _ := cs["state"].(map[string]any)
		waiting, _ := state["waiting"].(map[string]any)
		reason, _ := waiting["reason"].(string)
		if reason != "ImagePullBackOff" && reason != "ErrImagePull" {
			continue
		}
		img, _ := cs["image"].(string)
		cname, _ := cs["name"].(string)
		return img, cname
	}
	// Also check initContainerStatuses.
	initStatuses, _, _ := unstructured.NestedSlice(pod.Object, "status", "initContainerStatuses")
	for _, raw := range initStatuses {
		cs, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		state, _ := cs["state"].(map[string]any)
		waiting, _ := state["waiting"].(map[string]any)
		reason, _ := waiting["reason"].(string)
		if reason != "ImagePullBackOff" && reason != "ErrImagePull" {
			continue
		}
		img, _ := cs["image"].(string)
		cname, _ := cs["name"].(string)
		return img, cname
	}
	return "", ""
}

// pullAuthEvent finds the most recent Failed event on the named pod whose
// message contains an auth-signal keyword. Returns the event message, or ""
// if no auth-related event is found (meaning the pull failure has a different
// cause and should not be reported by this analyzer).
func pullAuthEvent(ctx context.Context, src snapshot.Source, ns, podName, image string) string {
	events, err := src.List(ctx, snapshot.GVREvent, ns)
	if err != nil {
		return ""
	}
	var best string
	var bestTime string
	for _, e := range events.Items {
		kind, _, _ := unstructured.NestedString(e.Object, "involvedObject", "kind")
		oname, _, _ := unstructured.NestedString(e.Object, "involvedObject", "name")
		if kind != "Pod" || oname != podName {
			continue
		}
		reason, _, _ := unstructured.NestedString(e.Object, "reason")
		if reason != "Failed" {
			continue
		}
		msg, _, _ := unstructured.NestedString(e.Object, "message")
		msgLow := strings.ToLower(msg)
		// Docker Hub returns HTTP 401 for non-existent images to avoid
		// disclosing whether a repository is private. Prioritize the
		// not-found check so a missing image isn't misclassified as an auth
		// failure (which would lead operators to investigate pull secrets
		// instead of the image name).
		if containsAny(msgLow, notFoundSignals) {
			continue
		}
		if !containsAny(msgLow, authSignals) {
			continue
		}
		// Keep the most recent (lexicographic on lastTimestamp is good enough for ISO-8601).
		ts, _, _ := unstructured.NestedString(e.Object, "lastTimestamp")
		if ts > bestTime {
			bestTime = ts
			best = msg
		}
	}
	return best
}

func containsAny(s string, subs []string) bool {
	for _, sub := range subs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}
