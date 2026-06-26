// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

package diagnose

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/srenix-ai/agentic-sre/internal/snapshot"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// WorkloadStateDrift surfaces stateful-workload health signals that
// the basic Pod/StatefulSet probes miss. The basic Services probe
// only checks "replicas ready"; this analyzer goes deeper into the
// state-tier signal — replication lag, follower divergence, ordinal
// rollout stuckness — which is what pages oncall when the data tier
// is sick.
//
// What's surfaced (v1.7 first cut):
//
//   - **CloudNativePG (CNPG) cluster lag** — instance phase != healthy
//     OR readyInstances < instances past grace. CNPG publishes follower
//     lag indirectly via instance condition / readyInstances drift.
//   - **StatefulSet ordinal-zero stuck** — currentReplicas reports
//     a non-zero pod count but the cluster's pod-0 is missing or
//     not-ready past grace. This is the classic "Pod scheduled
//     but stuck on stale revision; StatefulSet won't roll forward
//     because ordinal 0 is unhealthy" signal that hides behind the
//     normal "X/Y ready" metric.
//
// Roadmap (v1.8+):
//   - Kafka ISR shrink (Strimzi KafkaTopic + Kafka resource state)
//   - etcd member-quorum drift via direct member health
//   - Redis Sentinel / Patroni replication-lag signals
//
// Reduced scope (deliberately):
//   - The Srenix Enterprise paid catalog has StatefulSetReplicaPressure which
//     covers PVC-bind-lag + replica-degraded for StatefulSets in
//     general. This OSS analyzer covers the *ordinal-zero stuck* case
//     specifically — that one falls through StatefulSetReplicaPressure
//     because it reports as "N/M ready" rather than "0 ready".
type WorkloadStateDrift struct {
	// GracePeriod is how long an instance may be off-ready before we
	// flag it. Zero defaults to 5 minutes. CNPG rolling upgrades
	// transit through "not ready" cleanly within this window.
	GracePeriod time.Duration

	// Now is the time-source; tests inject a fixed clock. Defaults to
	// time.Now.
	Now func() time.Time
}

// Name returns the analyzer's identifier. Pinned for metrics + dashboards.
func (WorkloadStateDrift) Name() string { return "WorkloadStateDrift" }

// Run walks CNPG clusters + StatefulSets and emits diagnostics for
// state-tier health drift past the GracePeriod.
func (w WorkloadStateDrift) Run(ctx context.Context, src snapshot.Source) []Diagnostic {
	grace := w.GracePeriod
	if grace == 0 {
		grace = 5 * time.Minute
	}
	now := time.Now
	if w.Now != nil {
		now = w.Now
	}

	var out []Diagnostic
	out = append(out, w.checkCNPGClusters(ctx, src, grace, now())...)
	out = append(out, w.checkStatefulSets(ctx, src, grace, now())...)
	return out
}

func (w WorkloadStateDrift) checkCNPGClusters(ctx context.Context, src snapshot.Source, grace time.Duration, now time.Time) []Diagnostic {
	clusters, err := src.List(ctx, snapshot.GVRCNPGCluster, "")
	if err != nil || clusters == nil || len(clusters.Items) == 0 {
		logListFailure("clusters.postgresql.cnpg.io", err, true) // optional CRD
		return nil
	}
	var out []Diagnostic
	for i := range clusters.Items {
		c := &clusters.Items[i]
		ns := c.GetNamespace()
		name := c.GetName()
		subject := fmt.Sprintf("CNPGCluster/%s/%s", ns, name)

		desired, _, _ := unstructured.NestedInt64(c.Object, "spec", "instances")
		ready, _, _ := unstructured.NestedInt64(c.Object, "status", "readyInstances")
		phase, _, _ := unstructured.NestedString(c.Object, "status", "phase")
		primary, _, _ := unstructured.NestedString(c.Object, "status", "currentPrimary")
		targetPrimary, _, _ := unstructured.NestedString(c.Object, "status", "targetPrimary")

		// CNPG cluster age — if the cluster itself is newer than the
		// grace window, suppress (operator just deployed it).
		created := c.GetCreationTimestamp().Time
		clusterAge := time.Duration(0)
		if !created.IsZero() {
			clusterAge = now.Sub(created)
		}
		if clusterAge > 0 && clusterAge < grace {
			continue
		}

		// Phase is the canonical health signal in CNPG. "Cluster in
		// healthy state" is the expected normal phase.
		if phase != "" && !strings.Contains(strings.ToLower(phase), "healthy") &&
			!strings.Contains(strings.ToLower(phase), "creating") {
			severity := "warning"
			if strings.Contains(strings.ToLower(phase), "failover") ||
				strings.Contains(strings.ToLower(phase), "failed") {
				severity = "critical"
			}
			out = append(out, Diagnostic{
				Source:   "WorkloadStateDrift",
				Subject:  subject,
				Severity: severity,
				Message: fmt.Sprintf(
					"CNPG cluster %s/%s phase=%q (%d/%d ready instances)",
					ns, name, phase, ready, desired),
				Remediation: fmt.Sprintf(
					"Inspect: `kubectl describe cluster.postgresql.cnpg.io -n %s %s` and `kubectl get pods -n %s -l cnpg.io/cluster=%s`. "+
						"Common causes: PVC bind failure on a follower, primary OOMKilled, replication slot drift.",
					ns, name, ns, name),
			})
		}

		// Followers degraded but cluster phase not yet flipped.
		if desired > 0 && ready < desired && phase != "" && strings.Contains(strings.ToLower(phase), "healthy") {
			out = append(out, Diagnostic{
				Source:   "WorkloadStateDrift",
				Subject:  subject,
				Severity: "warning",
				Message: fmt.Sprintf(
					"CNPG cluster %s/%s reports phase=Healthy but only %d/%d instances ready",
					ns, name, ready, desired),
				Remediation: fmt.Sprintf(
					"A follower is degraded while the cluster phase hasn't flipped yet (transient or about to). "+
						"List the cluster's pods with `kubectl get pods -n %s -l cnpg.io/cluster=%s`, "+
						"identify the non-Ready follower, then inspect its postgres container with "+
						"`kubectl -n %s logs <that-pod-name> -c postgres` (substitute the pod name from the prior list).",
					ns, name, ns),
			})
		}

		// Primary divergence — currentPrimary != targetPrimary means
		// CNPG is in the middle of a switchover. Past the grace window,
		// this is stuck.
		if primary != "" && targetPrimary != "" && primary != targetPrimary {
			out = append(out, Diagnostic{
				Source:   "WorkloadStateDrift",
				Subject:  subject,
				Severity: "critical",
				Message: fmt.Sprintf(
					"CNPG cluster %s/%s primary switchover stuck: currentPrimary=%q, targetPrimary=%q",
					ns, name, primary, targetPrimary),
				Remediation: fmt.Sprintf(
					"CNPG is mid-switchover but hasn't completed. "+
						"`kubectl describe cluster.postgresql.cnpg.io -n %s %s` shows the failover/switchover history. "+
						"The new primary may be unable to promote (pg_isready, wal lag, role mismatch).",
					ns, name),
			})
		}
	}
	return out
}

func (w WorkloadStateDrift) checkStatefulSets(ctx context.Context, src snapshot.Source, grace time.Duration, now time.Time) []Diagnostic {
	stsList, err := src.List(ctx, snapshot.GVRStatefulSet, "")
	if err != nil || stsList == nil || len(stsList.Items) == 0 {
		logListFailure("statefulsets", err, false)
		return nil
	}
	var out []Diagnostic

	// Optimization: only walk Pods when we have at least one suspect
	// StatefulSet — avoids a full Pod list on every cycle if all
	// StatefulSets look healthy.
	type susp struct {
		ns, name       string
		labelSelector  map[string]interface{}
		desired, ready int64
		ageOK          bool
	}
	var suspects []susp

	for i := range stsList.Items {
		ss := &stsList.Items[i]
		ns := ss.GetNamespace()
		name := ss.GetName()

		desired, _, _ := unstructured.NestedInt64(ss.Object, "spec", "replicas")
		ready, _, _ := unstructured.NestedInt64(ss.Object, "status", "readyReplicas")

		// Healthy StatefulSet — skip.
		if desired == 0 || ready == desired {
			continue
		}
		// Brand-new StatefulSet — skip (still scheduling).
		created := ss.GetCreationTimestamp().Time
		if created.IsZero() || now.Sub(created) < grace {
			continue
		}
		selector, _, _ := unstructured.NestedMap(ss.Object, "spec", "selector", "matchLabels")
		suspects = append(suspects, susp{
			ns: ns, name: name,
			labelSelector: selector,
			desired:       desired,
			ready:         ready,
			ageOK:         true,
		})
	}

	if len(suspects) == 0 {
		return out
	}

	// Fetch all Pods once; cross-reference.
	pods, err := src.List(ctx, snapshot.GVRPod, "")
	if err != nil || pods == nil {
		logListFailure("pods", err, false)
		return out
	}

	for _, s := range suspects {
		// Find pod-0 by name convention: <statefulset>-0.
		pod0Name := s.name + "-0"
		var pod0 *unstructured.Unstructured
		for i := range pods.Items {
			p := &pods.Items[i]
			if p.GetNamespace() == s.ns && p.GetName() == pod0Name {
				pod0 = p
				break
			}
		}
		if pod0 == nil {
			out = append(out, Diagnostic{
				Source:   "WorkloadStateDrift",
				Subject:  fmt.Sprintf("StatefulSet/%s/%s", s.ns, s.name),
				Severity: "critical",
				Message: fmt.Sprintf(
					"StatefulSet %s/%s missing pod-0 (ordinal-zero) — only %d/%d replicas ready",
					s.ns, s.name, s.ready, s.desired),
				Remediation: fmt.Sprintf(
					"Pod-0 is the canonical primary in a StatefulSet rollout. Without it, "+
						"the higher-ordinal replicas can't promote and the rollout is stuck. "+
						"`kubectl describe pod -n %s %s` "+
						"(if it exists in a Terminating state) or check the controller events for PVC bind failure.",
					s.ns, pod0Name),
			})
			continue
		}

		// Pod-0 exists — is it ready?
		ready := podIsReady(pod0)
		if !ready {
			out = append(out, Diagnostic{
				Source:   "WorkloadStateDrift",
				Subject:  fmt.Sprintf("StatefulSet/%s/%s", s.ns, s.name),
				Severity: "warning",
				Message: fmt.Sprintf(
					"StatefulSet %s/%s pod-0 (ordinal-zero) NOT ready — %d/%d replicas ready",
					s.ns, s.name, s.ready, s.desired),
				Remediation: fmt.Sprintf(
					"Pod-0 is unready while higher ordinals are. The StatefulSet rollout is blocked: "+
						"`kubectl get pod %s -n %s -o wide` and `kubectl describe pod %s -n %s`. "+
						"Common causes: PVC bind on pod-0, init container failing, image pull on the primary's container only.",
					pod0Name, s.ns, pod0Name, s.ns),
			})
		}
	}
	return out
}

// podIsReady walks pod.status.conditions and returns true iff
// type=Ready, status=True is present.
func podIsReady(pod *unstructured.Unstructured) bool {
	conds, _, _ := unstructured.NestedSlice(pod.Object, "status", "conditions")
	for _, c := range conds {
		cm, ok := c.(map[string]interface{})
		if !ok {
			continue
		}
		t, _ := cm["type"].(string)
		st, _ := cm["status"].(string)
		if t == "Ready" {
			return st == "True"
		}
	}
	return false
}
