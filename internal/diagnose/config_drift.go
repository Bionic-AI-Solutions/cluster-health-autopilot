// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package diagnose

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/internal/snapshot"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// ConfigDrift surfaces configuration-state drift the basic
// resource-health probes miss. Each signal here is "the spec
// progressed but the cluster didn't catch up" — silent until a
// downstream consumer hits the stale state.
//
// What's surfaced (v1.8 first cut):
//
//   - **CRD multi-storedVersions** — a CustomResourceDefinition's
//     `status.storedVersions` lists more than one apiserver storage
//     version. Operators are expected to run a storage migration
//     (`kubectl get <resource> -A -o yaml | kubectl replace -f -`,
//     or the storage-version-migrator controller) after a CRD
//     conversion. Until that runs, the older version is still in
//     etcd; future CRD upgrades that drop the old version will
//     fail. Flagged critical because the migration is one-shot and
//     drops out of view once the operator moves on.
//
//   - **Deployment rollout stuck** — `metadata.generation` is ahead
//     of `status.observedGeneration` past the grace window (the
//     controller hasn't observed the spec), or
//     `status.updatedReplicas` < `spec.replicas` past the grace
//     window (a new revision rolled out but at least one new pod
//     never went Ready, so the rollout is stuck mid-flight).
//     Default grace 15 minutes — long enough that a healthy rollout
//     of a slow-starting image (ML containers, JIT warmups) isn't
//     flagged. Warning unless every pod of the deployment is
//     unavailable, in which case critical.
//
//   - **Pods disagree on `checksum/config`** — common Helm pattern:
//     the workload's PodTemplate carries a `checksum/config`
//     annotation derived from the ConfigMaps + Secrets it mounts.
//     When pods of the same Deployment have differing values, the
//     rolling update from the last config change didn't propagate
//     to all replicas. Skipped when no pod of the workload carries
//     the annotation (the workload's Helm chart doesn't use the
//     pattern). Warning.
//
// Out of scope (deliberately, for v1.8.x):
//   - Helm release values vs cluster-live (parsing
//     `helm.sh/release.v1` secrets adds a non-trivial dependency;
//     defer to a follow-up).
//   - Open-Telemetry-collector / Prometheus-config rollouts (covered
//     by the rollout-stuck signal above when the controller emits a
//     Deployment; otherwise specialized).
//   - ConfigMap content schema drift (the data shape changed in a
//     way the consumer can't parse) — that's an application-level
//     concern, not a cluster-state probe.
type ConfigDrift struct {
	// GracePeriod is how long the rollout signals wait before
	// flagging — Deployments mid-flight on slow-starting images
	// shouldn't show up as drift. Zero uses defaultRolloutGrace.
	GracePeriod time.Duration

	// Now returns the current time; overridable in tests.
	Now func() time.Time
}

// Name returns the analyzer's identifier. Pinned for metrics +
// dashboards.
func (ConfigDrift) Name() string { return "ConfigDrift" }

const defaultRolloutGrace = 15 * time.Minute

var (
	gvrCRD = schema.GroupVersionResource{
		Group:    "apiextensions.k8s.io",
		Version:  "v1",
		Resource: "customresourcedefinitions",
	}
)

// systemConfigNamespaces are namespaces whose rollouts are expected
// to lag occasionally (control-plane upgrades, node-local DaemonSets
// rotating on cordon/drain). Flagging there is noisy and rarely
// useful to the end-user operator.
var systemConfigNamespaces = map[string]struct{}{
	"kube-system":     {},
	"kube-public":     {},
	"kube-node-lease": {},
}

// Run walks the CRD list + Deployments + Pods and emits one
// Diagnostic per drift signal.
func (c ConfigDrift) Run(ctx context.Context, src snapshot.Source) []Diagnostic {
	now := c.Now
	if now == nil {
		now = time.Now
	}
	grace := c.GracePeriod
	if grace == 0 {
		grace = defaultRolloutGrace
	}

	var out []Diagnostic
	out = append(out, c.checkCRDStoredVersions(ctx, src)...)
	out = append(out, c.checkDeploymentRollouts(ctx, src, now(), grace)...)
	out = append(out, c.checkConfigChecksumDrift(ctx, src)...)
	return out
}

// checkCRDStoredVersions flags any CRD whose status.storedVersions
// contains more than one apiserver version. After a CRD version
// bump, both versions remain in etcd until the operator runs a
// storage migration; future drops of the old version will fail
// until that's done.
func (c ConfigDrift) checkCRDStoredVersions(ctx context.Context, src snapshot.Source) []Diagnostic {
	list, err := src.List(ctx, gvrCRD, "")
	if err != nil || list == nil {
		logListFailure("customresourcedefinitions", err, true) // silent when the CRD/resource is absent; logs Forbidden etc.
		return nil
	}
	var out []Diagnostic
	for i := range list.Items {
		crd := &list.Items[i]
		stored, _, _ := unstructured.NestedStringSlice(crd.Object, "status", "storedVersions")
		if len(stored) <= 1 {
			continue
		}
		name := crd.GetName()
		out = append(out, Diagnostic{
			Source:   "ConfigDrift",
			Subject:  fmt.Sprintf("CustomResourceDefinition/cluster/%s", name),
			Severity: "critical",
			Message: fmt.Sprintf(
				"CRD %s has multiple storedVersions (%v); storage migration is pending and "+
					"future CRD upgrades that drop the old version will fail",
				name, stored),
			Remediation: fmt.Sprintf(
				"Run the storage version migrator: "+
					"`kubectl get %s -A -o yaml | kubectl replace -f -` (this re-writes each resource "+
					"in the current storage version). Alternative: deploy the storage-version-migrator "+
					"controller. Then re-run `kubectl get crd %s -o jsonpath='{.status.storedVersions}'` "+
					"and confirm only one version is listed.",
				name, name),
		})
	}
	return out
}

// checkDeploymentRollouts flags Deployments whose spec generation has
// outpaced observed generation past the grace window, or whose
// updatedReplicas trail spec.replicas past the grace window.
func (c ConfigDrift) checkDeploymentRollouts(ctx context.Context, src snapshot.Source, now time.Time, grace time.Duration) []Diagnostic {
	list, err := src.List(ctx, snapshot.GVRDeployment, "")
	if err != nil || list == nil {
		logListFailure("deployments", err, true) // silent when the CRD/resource is absent; logs Forbidden etc.
		return nil
	}
	var out []Diagnostic
	for i := range list.Items {
		d := &list.Items[i]
		ns := d.GetNamespace()
		if _, isSystem := systemConfigNamespaces[ns]; isSystem {
			continue
		}
		name := d.GetName()
		subject := fmt.Sprintf("Deployment/%s/%s", ns, name)

		// Has the spec advanced past the controller's observation?
		gen, _, _ := unstructured.NestedInt64(d.Object, "metadata", "generation")
		observed, _, _ := unstructured.NestedInt64(d.Object, "status", "observedGeneration")

		// Replica progress.
		replicas, _, _ := unstructured.NestedInt64(d.Object, "spec", "replicas")
		if replicas == 0 {
			// Scaled to zero is intentional state, not a stuck
			// rollout. Skip.
			continue
		}
		updated, _, _ := unstructured.NestedInt64(d.Object, "status", "updatedReplicas")
		available, _, _ := unstructured.NestedInt64(d.Object, "status", "availableReplicas")

		// Compute the age of the most recent spec change. We use
		// the Deployment's `lastUpdateTime` from a status condition
		// when present (Available / Progressing), else fall back to
		// metadata.creationTimestamp. If neither is parseable we
		// skip (can't reason about grace).
		mutatedAt, ok := deploymentMutatedAt(d)
		if !ok {
			continue
		}
		if now.Sub(mutatedAt) < grace {
			continue
		}

		// Generation skew — the controller never reconciled the latest
		// spec. Almost always a controller-pause or admission webhook
		// rejection.
		if gen > observed {
			out = append(out, Diagnostic{
				Source:   "ConfigDrift",
				Subject:  subject,
				Severity: "critical",
				Message: fmt.Sprintf(
					"Deployment %s/%s generation=%d but observedGeneration=%d for >%s; "+
						"the controller has not reconciled the latest spec",
					ns, name, gen, observed, grace),
				Remediation: "Check the deployment controller logs (`kubectl -n kube-system logs deploy/kube-controller-manager`) and any validating-admission webhooks that may be rejecting the new ReplicaSet. " +
					"`kubectl describe deploy " + name + " -n " + ns + "` will show the most recent Progressing/ReplicaFailure condition.",
			})
			continue
		}

		// Replicas progressed but new pods didn't go Ready.
		if updated < replicas {
			sev := "warning"
			if available == 0 {
				sev = "critical"
			}
			out = append(out, Diagnostic{
				Source:   "ConfigDrift",
				Subject:  subject,
				Severity: sev,
				Message: fmt.Sprintf(
					"Deployment %s/%s rollout stuck for >%s: updatedReplicas=%d/%d, availableReplicas=%d",
					ns, name, grace, updated, replicas, available),
				Remediation: renderRolloutStuckRemediation(d, ns, name),
			})
		}
	}
	return out
}

// checkConfigChecksumDrift flags Deployments whose Pods disagree on
// their `checksum/config` annotation. The annotation is a common
// Helm pattern: it's a sha256 of the ConfigMap / Secret content that
// the workload mounts, included in the PodTemplate so a config
// change triggers a rolling update. When live Pods carry mixed
// values, the rolling update never finished — half the replicas are
// still running on the old config.
func (c ConfigDrift) checkConfigChecksumDrift(ctx context.Context, src snapshot.Source) []Diagnostic {
	pods, err := src.List(ctx, snapshot.GVRPod, "")
	if err != nil || pods == nil {
		logListFailure("pods", err, true) // silent when the CRD/resource is absent; logs Forbidden etc.
		return nil
	}

	// Group pods by owner Deployment via owner-reference chain
	// Pod -> ReplicaSet -> Deployment. We map by "Deployment/ns/name"
	// because pods of two different revisions of the same Deployment
	// are exactly what we want to compare.
	rsToDeploy := map[string]string{} // ns+name -> Deployment/ns/name

	rsList, _ := src.List(ctx, snapshot.GVRReplicaSet, "")
	if rsList != nil {
		for i := range rsList.Items {
			rs := &rsList.Items[i]
			owners := rs.GetOwnerReferences()
			for _, o := range owners {
				if o.Kind == "Deployment" {
					rsToDeploy[rs.GetNamespace()+"/"+rs.GetName()] =
						"Deployment/" + rs.GetNamespace() + "/" + o.Name
					break
				}
			}
		}
	}

	// Collect distinct checksum values per workload.
	type seen struct {
		values  map[string]struct{}
		example string
	}
	byWorkload := map[string]*seen{}
	for i := range pods.Items {
		p := &pods.Items[i]
		ns := p.GetNamespace()
		if _, isSystem := systemConfigNamespaces[ns]; isSystem {
			continue
		}
		anns := p.GetAnnotations()
		checksum, ok := anns["checksum/config"]
		if !ok || checksum == "" {
			continue
		}
		// Walk ownerRefs to find the ReplicaSet, then map to Deployment.
		var workload string
		for _, o := range p.GetOwnerReferences() {
			if o.Kind == "ReplicaSet" {
				if d, found := rsToDeploy[ns+"/"+o.Name]; found {
					workload = d
				}
				break
			}
		}
		if workload == "" {
			continue
		}
		s, ok := byWorkload[workload]
		if !ok {
			s = &seen{values: map[string]struct{}{}, example: p.GetName()}
			byWorkload[workload] = s
		}
		s.values[checksum] = struct{}{}
	}

	var out []Diagnostic
	for workload, s := range byWorkload {
		if len(s.values) < 2 {
			continue
		}
		out = append(out, Diagnostic{
			Source:   "ConfigDrift",
			Subject:  workload,
			Severity: "warning",
			Message: fmt.Sprintf(
				"Pods of %s carry %d distinct checksum/config annotation values; "+
					"the rolling update from the last config change didn't propagate to all replicas",
				workload, len(s.values)),
			Remediation: "Force a fresh rollout to converge all replicas onto the current config: " +
				"`kubectl rollout restart " + workload + "`. If the rollout still doesn't converge, the issue is likely the same as a stuck-rollout signal — check the deployment's Progressing condition and any pod-level events.",
		})
	}
	return out
}

// renderRolloutStuckRemediation renders the per-pod kubectl flag using
// the Deployment's actual spec.selector.matchLabels rather than the
// literal `<selector>` placeholder. Operators can copy-paste the line
// directly; the AI tier surfacing this diagnostic also has a concrete
// command to execute via the kubectl proposer instead of a template.
//
// Falls back to a generic "kubectl get pods -n <ns>" hint when the
// Deployment has no matchLabels (rare — most workloads declare them).
func renderRolloutStuckRemediation(d *unstructured.Unstructured, ns, name string) string {
	labels, found, _ := unstructured.NestedStringMap(d.Object, "spec", "selector", "matchLabels")
	prefix := "The new revision's pods aren't going Ready. Check `kubectl describe deploy " + name + " -n " + ns + "` for the Progressing condition reason, "
	suffix := " for individual pod events (typically ImagePullBackOff, readiness-probe failure, or insufficient cluster capacity)."
	if !found || len(labels) == 0 {
		return prefix + "and `kubectl -n " + ns + " get pods` (Deployment has no spec.selector.matchLabels to filter by)" + suffix
	}
	// Sort keys so the rendered selector is stable across cycles — both
	// for diagnostic deduplication and for snapshot-based tests.
	keys := make([]string, 0, len(labels))
	for k := range labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, k+"="+labels[k])
	}
	sel := strings.Join(parts, ",")
	return prefix + "and `kubectl -n " + ns + " get pods -l " + sel + "`" + suffix
}

// deploymentMutatedAt returns the timestamp of the latest spec change
// for the Deployment, derived from its status conditions when
// present, falling back to metadata.creationTimestamp.
func deploymentMutatedAt(d *unstructured.Unstructured) (time.Time, bool) {
	// Prefer the Progressing condition's lastUpdateTime — that's
	// the time the controller last reacted to a spec change.
	conds, _, _ := unstructured.NestedSlice(d.Object, "status", "conditions")
	for _, c := range conds {
		cond, ok := c.(map[string]interface{})
		if !ok {
			continue
		}
		condType, _ := cond["type"].(string)
		if condType != "Progressing" {
			continue
		}
		lastUpdate, _ := cond["lastUpdateTime"].(string)
		if t, err := time.Parse(time.RFC3339, lastUpdate); err == nil {
			return t, true
		}
	}
	// Fall back to creationTimestamp.
	created := d.GetCreationTimestamp()
	if created.IsZero() {
		return time.Time{}, false
	}
	return created.Time, true
}
