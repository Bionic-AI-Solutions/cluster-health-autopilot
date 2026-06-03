// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

// Package feeder houses the snapshot-driven RAG entry writers — components
// that observe the live cluster and persist learned facts into the
// rag.Writer so analyzers / probes / paid-tier proposers can recall them
// across cycles.
//
// Today: workload feeder (kind=workload).
//
// Next slices (per docs/design/2026-06-rag-digest-pin-proposer.md):
//   - release-source detection enrichment (helm/argocd/kustomize file
//     that holds the workload's image tag) — adds "release_source"
//     features to the workload entries this feeder writes.
//   - baseline feeder (kind=baseline) — restart/OOM histograms per
//     workload, drives "is this churn abnormal for THIS cluster?".
package feeder

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/pkg/rag"
	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/pkg/snapshot"
)

// GVRs the feeder needs. Defined locally so pkg/feeder doesn't depend
// on internal/snapshot — pkg/ packages can't import internal/. These
// match the canonical Kubernetes GVRs.
var (
	gvrPod         = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}
	gvrDeployment  = schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}
	gvrStatefulSet = schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "statefulsets"}
	gvrDaemonSet   = schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "daemonsets"}
)

// defaultWorkloadImportance seeds every fresh workload entry. Above the
// OSS default ImportanceMin (0.3) so the workload-aware analyzers /
// proposers see it immediately; the paid-tier outcome loop later
// adjusts based on SRE approve/deny signals.
const defaultWorkloadImportance = 0.5

// workloadGVRs are the controllers the feeder observes. Order matters
// only for logging; entries collide on (kind, key) anyway.
var workloadGVRs = []struct {
	gvr  schema.GroupVersionResource
	kind string
}{
	{gvrDeployment, "Deployment"},
	{gvrStatefulSet, "StatefulSet"},
	{gvrDaemonSet, "DaemonSet"},
}

// systemNamespaces are infrastructure namespaces whose workloads are
// managed by their own controllers — pinning their images is the
// controller's job, not the SRE's. Matches the digest-pin analyzer's
// `systemSecurityNamespaces` so the feeder and analyzer agree on
// "is this workload operator-relevant".
var systemNamespaces = map[string]struct{}{
	"kube-system":        {},
	"kube-public":        {},
	"kube-node-lease":    {},
	"cnpg-system":        {},
	"rook-ceph":          {},
	"vault":              {},
	"external-secrets":   {},
	"calico-system":      {},
	"tigera-operator":    {},
	"calico-apiserver":   {},
	"local-path-storage": {},
}

// WorkloadFeeder upserts a rag.Entry per observed Deployment / StatefulSet /
// DaemonSet, carrying enough features (image, image_digest, owner annotations)
// for the paid-tier digest-pin proposer to construct a PR without a second
// snapshot pass.
//
// One RunOnce per watch cycle. Idempotent: rag.Writer.Upsert preserves
// FirstSeen and bumps LastSeen so importance decay computes correctly
// across restarts.
//
// Fail-open: a single workload's parse failure does not abort the sweep.
// The contract matches the cha-com CloudflareFeeder: best-effort
// observation, never block the watcher.
type WorkloadFeeder struct {
	Source    snapshot.Source
	Writer    rag.Writer
	ClusterID string

	// MinImportance seeds every fresh entry. Override for tests; 0.5
	// is the production default.
	MinImportance float64
}

// NewWorkloadFeeder constructs a feeder with production defaults.
func NewWorkloadFeeder(src snapshot.Source, w rag.Writer, clusterID string) *WorkloadFeeder {
	return &WorkloadFeeder{
		Source:        src,
		Writer:        w,
		ClusterID:     clusterID,
		MinImportance: defaultWorkloadImportance,
	}
}

// Result summarises one sweep. observed = workloads walked across all
// controller kinds; upserts = rag.Entry writes that succeeded.
type Result struct {
	Observed int
	Upserts  int
}

// RunOnce walks Deployments, StatefulSets, and DaemonSets, derives one
// rag.Entry per workload, and upserts via the Writer. Returns counts
// plus a wrapped error when the Source / Writer is missing; per-item
// failures (parse, upsert) are silently skipped so one bad workload
// can't stall the sweep.
func (f *WorkloadFeeder) RunOnce(ctx context.Context) (Result, error) {
	if f == nil || f.Source == nil || f.Writer == nil {
		return Result{}, errors.New("feeder: Source and Writer required")
	}
	now := time.Now().UTC()
	importance := f.MinImportance
	if importance == 0 {
		importance = defaultWorkloadImportance
	}

	// Pod imageID lookup, populated once per sweep — every workload
	// reuses the same map. Keyed by (namespace, container-name)
	// across all pods of the owning controller. We pick any ready
	// pod's imageID for a container; replicas of one Deployment
	// run identical container images so the choice is deterministic.
	digestIdx := f.buildDigestIndex(ctx)

	var res Result
	for _, gk := range workloadGVRs {
		list, err := f.Source.List(ctx, gk.gvr, "")
		if err != nil || list == nil {
			continue
		}
		for i := range list.Items {
			obj := &list.Items[i]
			ns := obj.GetNamespace()
			if _, isSystem := systemNamespaces[ns]; isSystem {
				continue
			}
			res.Observed++
			entry := f.buildEntry(obj, gk.kind, now, importance, digestIdx)
			if entry == nil {
				continue
			}
			if uerr := f.Writer.Upsert(ctx, *entry); uerr == nil {
				res.Upserts++
			}
		}
	}
	return res, nil
}

// containerInfo is the per-container shape stamped into workload
// entry features. Kept private; consumers read via the typed map.
type containerInfo struct {
	Name        string `json:"name"`
	Image       string `json:"image"`
	ImageDigest string `json:"image_digest,omitempty"`
}

// digestKey is the (namespace, container) pair used to look up the
// kubelet-resolved digest from a Pod's containerStatuses.
type digestKey struct {
	ns        string
	container string
}

// buildDigestIndex builds the (namespace, container) → digest map by
// reading status.containerStatuses[].imageID across every Pod the
// snapshot Source can see. kubelet writes the resolved digest there
// after a successful pull; mutable :tag references resolve to a
// concrete sha256.
//
// Pods that haven't pulled (ImagePullBackOff / pending) contribute
// nothing — the entry's image_digest field is omitted for those
// containers, which is the correct signal for a downstream proposer
// ("we observed the workload but couldn't resolve a digest; come back
// next cycle"). Two pods of one controller may report different
// digests if mid-rollout; the feeder takes the first one observed and
// leaves drift detection to the next slice.
func (f *WorkloadFeeder) buildDigestIndex(ctx context.Context) map[digestKey]string {
	idx := map[digestKey]string{}
	pods, err := f.Source.List(ctx, gvrPod, "")
	if err != nil || pods == nil {
		return idx
	}
	for i := range pods.Items {
		p := &pods.Items[i]
		ns := p.GetNamespace()
		if _, isSystem := systemNamespaces[ns]; isSystem {
			continue
		}
		statuses, _, _ := unstructured.NestedSlice(p.Object, "status", "containerStatuses")
		for _, raw := range statuses {
			cs, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			cn, _ := cs["name"].(string)
			imgID, _ := cs["imageID"].(string)
			if cn == "" || imgID == "" {
				continue
			}
			digest := extractDigest(imgID)
			if digest == "" {
				continue
			}
			k := digestKey{ns: ns, container: cn}
			if _, seen := idx[k]; !seen {
				idx[k] = digest
			}
		}
	}
	return idx
}

// extractDigest pulls the sha256:... portion of an imageID. kubelet
// writes imageID as either:
//
//	docker.io/library/redis@sha256:abc...
//	docker-pullable://registry.example.com/foo@sha256:def...
//	registry.example.com/foo@sha256:abc...
//
// Returns "" when no @sha256: marker is found (image not pulled yet,
// or kubelet running with an unfamiliar driver).
func extractDigest(imageID string) string {
	idx := strings.Index(imageID, "@sha256:")
	if idx < 0 {
		return ""
	}
	return imageID[idx+1:] // includes the leading "sha256:"
}

// buildEntry constructs one rag.Entry from a workload object. Returns
// nil when the workload has no containers (degenerate manifest).
func (f *WorkloadFeeder) buildEntry(obj *unstructured.Unstructured, kind string, now time.Time, importance float64, digestIdx map[digestKey]string) *rag.Entry {
	ns := obj.GetNamespace()
	name := obj.GetName()
	if ns == "" || name == "" {
		return nil
	}
	containers, _, _ := unstructured.NestedSlice(obj.Object, "spec", "template", "spec", "containers")
	if len(containers) == 0 {
		return nil
	}
	var cinfo []containerInfo
	for _, raw := range containers {
		cm, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		cn, _ := cm["name"].(string)
		img, _ := cm["image"].(string)
		if cn == "" || img == "" {
			continue
		}
		ci := containerInfo{Name: cn, Image: img}
		if d, ok := digestIdx[digestKey{ns: ns, container: cn}]; ok {
			ci.ImageDigest = d
		}
		cinfo = append(cinfo, ci)
	}
	if len(cinfo) == 0 {
		return nil
	}

	// Materialise containers as []any so map-walkers (Qdrant payload,
	// JSON unmarshal) don't see a typed-slice surprise.
	containerPayload := make([]any, 0, len(cinfo))
	for _, ci := range cinfo {
		m := map[string]any{"name": ci.Name, "image": ci.Image}
		if ci.ImageDigest != "" {
			m["image_digest"] = ci.ImageDigest
		}
		containerPayload = append(containerPayload, m)
	}

	features := map[string]any{
		"kind":       kind,
		"namespace":  ns,
		"name":       name,
		"containers": containerPayload,
	}
	if replicas, found, _ := unstructured.NestedInt64(obj.Object, "spec", "replicas"); found {
		features["replicas"] = replicas
	}
	if owner := detectOwner(obj); owner != nil {
		features["owner_kind"] = owner.Kind
		if owner.ChartName != "" {
			features["owner_chart"] = owner.ChartName
		}
		if owner.ReleaseName != "" {
			features["owner_release"] = owner.ReleaseName
		}
		if owner.ReleaseNamespace != "" {
			features["owner_release_namespace"] = owner.ReleaseNamespace
		}
	}

	return &rag.Entry{
		ClusterID:    f.ClusterID,
		Kind:         rag.KindWorkload,
		Key:          fmt.Sprintf("%s/%s", ns, name),
		FirstSeen:    now,
		LastSeen:     now,
		Observations: 1,
		Importance:   importance,
		Sources:      []string{"k8s_apiserver"},
		Features:     features,
	}
}

// ownerInfo summarises the workload's release source (Helm or Argo CD)
// derived from standard annotations. Best-effort: empty struct means
// "unknown source-of-truth"; the digest-pin proposer falls back to a
// PR-template path in that case.
type ownerInfo struct {
	Kind             string // "Helm" | "ArgoCD" | ""
	ReleaseName      string
	ReleaseNamespace string
	ChartName        string
}

// detectOwner reads the conventional release annotations:
//
//	meta.helm.sh/release-name        (Helm)
//	meta.helm.sh/release-namespace
//	argocd.argoproj.io/instance      (Argo CD; "<namespace>_<name>" form)
//
// Returns nil when neither is set. The proposer slice consumes this to
// pick the right release pipeline; future enrichment will probe the
// repo / Application CR for the exact image.tag file path.
func detectOwner(obj *unstructured.Unstructured) *ownerInfo {
	anns := obj.GetAnnotations()
	if anns == nil {
		return nil
	}
	if rel := anns["meta.helm.sh/release-name"]; rel != "" {
		o := &ownerInfo{Kind: "Helm", ReleaseName: rel}
		if relNS := anns["meta.helm.sh/release-namespace"]; relNS != "" {
			o.ReleaseNamespace = relNS
		}
		// Chart name lands in the standard label, not an annotation;
		// labels on the workload are an acceptable hint.
		if labels := obj.GetLabels(); labels != nil {
			if c := labels["helm.sh/chart"]; c != "" {
				// chart label is "<chart>-<version>"; strip the trailing
				// version so the proposer matches the chart in the repo.
				if i := strings.LastIndex(c, "-"); i > 0 {
					o.ChartName = c[:i]
				} else {
					o.ChartName = c
				}
			} else if c := labels["app.kubernetes.io/name"]; c != "" {
				o.ChartName = c
			}
		}
		return o
	}
	if inst := anns["argocd.argoproj.io/instance"]; inst != "" {
		// Argo CD writes "<application-namespace>_<application-name>".
		o := &ownerInfo{Kind: "ArgoCD"}
		if i := strings.Index(inst, "_"); i > 0 {
			o.ReleaseNamespace = inst[:i]
			o.ReleaseName = inst[i+1:]
		} else {
			o.ReleaseName = inst
		}
		return o
	}
	return nil
}
