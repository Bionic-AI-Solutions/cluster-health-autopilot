// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package diagnose

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/internal/snapshot"
	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/pkg/netpol"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// SecurityDrift surfaces audit-relevant gaps in cluster security
// posture. Each signal is observational — CHA does not enforce; it
// surfaces the gap so an operator can react.
//
// What's surfaced (v1.8 first cut):
//
//   - **Pod Security Standards posture gap** — user namespaces with
//     no `pod-security.kubernetes.io/enforce` label, or with
//     `enforce=privileged`. PSS is the modern replacement for Pod
//     Security Policies; absence of the label means K8s applies the
//     cluster-wide default (typically `privileged`). Warning when
//     missing, warning when explicitly `privileged`.
//
//   - **Mutable image tag (no digest pin)** — Pods whose containers
//     reference images by tag only (`<image>:v1.2.3`) rather than
//     digest (`<image>@sha256:...`). Mutable tags break the
//     attestation story image-signing relies on: even if you sign
//     `<image>:v1.2.3` today, the digest the tag points at can be
//     re-published tomorrow. The pod that ran "the signed image" is
//     not necessarily the pod running the same tag tomorrow.
//     Warning. Skipped for `latest` (the framework's other
//     pull-policy code paths already flag that).
//
//   - **NetworkPolicy coverage gap** — user namespaces running pods
//     with zero NetworkPolicies. K8s default networking is fully
//     permissive: without at least one NetworkPolicy, any pod can
//     reach any other pod cluster-wide. Warning per namespace.
//
// Out of scope (deliberately, for v1.8.x):
//
//   - True PSS-downgrade detection (was-restricted-is-now-baseline)
//     — requires label-history, not derivable from a snapshot.
//
//   - Active Cosign / Notation signature verification — admission-
//     time concern that needs the cluster's policy controller to
//     emit events; observational CHA is the wrong tool.
//
//   - NetworkPolicy egress-coverage analysis (a namespace has
//     NetworkPolicies but none set `policyTypes: [Egress]`) —
//     interesting but more nuanced than v1.8 needs; defer until
//     operator demand surfaces.
//
// Skip rules: kube-system / kube-public / kube-node-lease /
// cnpg-system / rook-ceph / vault / external-secrets — system
// namespaces whose posture is managed by their controllers rather
// than the cluster operator.
type SecurityDrift struct{}

// Name returns the analyzer's identifier. Pinned for metrics +
// dashboards.
func (SecurityDrift) Name() string { return "SecurityDrift" }

var (
	gvrNamespace = schema.GroupVersionResource{
		Group:    "",
		Version:  "v1",
		Resource: "namespaces",
	}
	gvrNetworkPolicy = schema.GroupVersionResource{
		Group:    "networking.k8s.io",
		Version:  "v1",
		Resource: "networkpolicies",
	}
)

// systemSecurityNamespaces are namespaces whose security posture is
// managed by the cluster operator at install time (or by the
// platform's own controllers); flagging them is noise.
var systemSecurityNamespaces = map[string]struct{}{
	"kube-system":      {},
	"kube-public":      {},
	"kube-node-lease":  {},
	"cnpg-system":      {},
	"rook-ceph":        {},
	"vault":            {},
	"external-secrets": {},
}

// pssEnforceLabel is the standard label apiserver-side PSS admission
// reads. See:
//
//	https://kubernetes.io/docs/concepts/security/pod-security-admission/
const pssEnforceLabel = "pod-security.kubernetes.io/enforce"

// Run walks namespaces + pods + networkpolicies and emits one
// Diagnostic per drift signal.
func (s SecurityDrift) Run(ctx context.Context, src snapshot.Source) []Diagnostic {
	var out []Diagnostic
	out = append(out, s.checkPSSPosture(ctx, src)...)
	out = append(out, s.checkMutableImageTags(ctx, src)...)
	out = append(out, s.checkNetworkPolicyCoverage(ctx, src)...)
	return out
}

// checkPSSPosture flags user namespaces with no PSS enforce label or
// with `enforce=privileged`.
func (s SecurityDrift) checkPSSPosture(ctx context.Context, src snapshot.Source) []Diagnostic {
	nsList, err := src.List(ctx, gvrNamespace, "")
	if err != nil || nsList == nil {
		logListFailure("namespaces", err, true) // silent when the CRD/resource is absent; logs Forbidden etc.
		return nil
	}
	var out []Diagnostic
	for i := range nsList.Items {
		ns := &nsList.Items[i]
		name := ns.GetName()
		if _, isSystem := systemSecurityNamespaces[name]; isSystem {
			continue
		}
		labels := ns.GetLabels()
		enforce, hasLabel := labels[pssEnforceLabel]
		switch {
		case !hasLabel:
			out = append(out, Diagnostic{
				Source:   "SecurityDrift",
				Subject:  fmt.Sprintf("Namespace/cluster/%s", name),
				Severity: "warning",
				Message: fmt.Sprintf(
					"Namespace %s has no pod-security.kubernetes.io/enforce label; "+
						"admission applies the cluster-wide default (typically privileged)",
					name),
				Remediation: fmt.Sprintf(
					"Label the namespace at the tightest profile its workloads tolerate. "+
						"Start with: `kubectl label namespace %s pod-security.kubernetes.io/enforce=baseline pod-security.kubernetes.io/warn=baseline`. "+
						"Tighten to `restricted` once you've confirmed pods aren't using HostPath / HostNetwork / privileged containers.",
					name),
			})
		case enforce == "privileged":
			out = append(out, Diagnostic{
				Source:   "SecurityDrift",
				Subject:  fmt.Sprintf("Namespace/cluster/%s", name),
				Severity: "warning",
				Message: fmt.Sprintf(
					"Namespace %s explicitly enforces PSS=privileged — the most-permissive profile",
					name),
				Remediation: fmt.Sprintf(
					"Confirm this namespace genuinely needs privileged escalation (host networking, "+
						"hostPath mounts, capabilities like SYS_ADMIN). If not, tighten with "+
						"`kubectl label namespace %s pod-security.kubernetes.io/enforce=baseline --overwrite`. "+
						"If yes, document the reason as a namespace annotation so audit reviewers don't re-flag it.",
					name),
			})
		}
	}
	return out
}

// trustedUpstreamRegistryPrefixes is the canonical list of upstream
// registries whose `:tag` references are conventionally trusted —
// pinning their images by digest is paranoia, not hygiene. On most
// clusters these account for 80%+ of digest-pin findings; flagging
// them at warning produces noise that drowns out real in-house gaps.
//
// v1.14.0 downgrades these to severity=info so the actionable
// in-house findings stand out. Operators can override the prefix
// list (and the default severity) via env vars:
//
//	CHA_DIGEST_PIN_TRUSTED_PREFIXES   — comma-separated, replaces defaults
//	CHA_DIGEST_PIN_UNTRUSTED_SEVERITY — "warning" (default) | "info"
//
// The list deliberately excludes docker.io/library/* (where most
// supply-chain compromises have landed historically) and
// docker.io/<random-user>/* (untrusted-by-default).
var trustedUpstreamRegistryPrefixes = []string{
	"quay.io/",
	"gcr.io/",
	"ghcr.io/",
	"registry.k8s.io/",
	"k8s.gcr.io/",
	"docker.io/postgres",
	"docker.io/redis",
	"docker.io/haproxy",
	"docker.io/envoyproxy/",
	"docker.io/mariadb",
	"docker.io/rabbitmq",
	"docker.io/nginx",
	"docker.io/busybox",
	"docker.io/alpine",
	"public.ecr.aws/",
	"mcr.microsoft.com/",
}

// classifyDigestPinSeverity returns "info" for trusted-upstream
// images and the configured untrusted-severity (default "warning")
// for everything else. Trust class lookup is a substring/prefix
// check — fast, no regex.
func classifyDigestPinSeverity(img string) string {
	// Normalize: docker.io is the implicit registry; "redis:7" →
	// "docker.io/library/redis:7" for matching purposes. But also
	// match bare "redis:7" against "docker.io/redis" prefix.
	imgLower := strings.ToLower(img)
	for _, prefix := range trustedUpstreamRegistryPrefixes {
		p := strings.ToLower(prefix)
		// Match either "quay.io/..." OR (for docker.io prefixes) the
		// implicit-registry short form like "redis:7" matching
		// "docker.io/redis".
		if strings.HasPrefix(imgLower, p) {
			return "info"
		}
		if strings.HasPrefix(p, "docker.io/") {
			short := strings.TrimPrefix(p, "docker.io/")
			if strings.HasPrefix(imgLower, short) {
				return "info"
			}
		}
	}
	if v := os.Getenv("CHA_DIGEST_PIN_UNTRUSTED_SEVERITY"); v == "info" {
		return "info"
	}
	return "warning"
}

// checkMutableImageTags flags Pods whose containers use a tag-only
// reference (no @sha256:... digest pin). One diagnostic per Pod,
// listing the affected container. Severity is per-image (trust-class)
// from v1.14.0 onward: upstream/trusted registries → info, in-house
// or unknown → warning.
func (s SecurityDrift) checkMutableImageTags(ctx context.Context, src snapshot.Source) []Diagnostic {
	pods, err := src.List(ctx, snapshot.GVRPod, "")
	if err != nil || pods == nil {
		logListFailure("pods", err, true) // silent when the CRD/resource is absent; logs Forbidden etc.
		return nil
	}
	// v1.25.0 — per-workload dedup. The same Deployment's N replica
	// Pods all share the same image set; emitting N identical
	// diagnostics flooded the Slack channel + made the seen-map
	// grow at the rate of replica churn (every rolling update
	// produced new Pod names → new entries → re-alert). Group by
	// (namespace, owner-controller-name, sorted-unpinned-image-set)
	// and emit ONE diagnostic per group. For DaemonSets that's one
	// alert per node-spread; for Deployments that's one per
	// ReplicaSet (which means at most 2 during a rolling update,
	// not N replicas per RS).
	type groupKey struct {
		ns        string
		ownerName string
		imageSet  string
	}
	type groupVal struct {
		samplePod *unstructured.Unstructured
		unpinned  []string
		severity  string
		podCount  int
	}
	groups := map[groupKey]*groupVal{}
	for i := range pods.Items {
		p := &pods.Items[i]
		ns := p.GetNamespace()
		if _, isSystem := systemSecurityNamespaces[ns]; isSystem {
			continue
		}

		containers, _, _ := unstructured.NestedSlice(p.Object, "spec", "containers")
		var unpinned []string
		podSeverity := "info"
		for _, c := range containers {
			ci, ok := c.(map[string]interface{})
			if !ok {
				continue
			}
			cn, _ := ci["name"].(string)
			img, _ := ci["image"].(string)
			if img == "" || strings.Contains(img, "@sha256:") {
				continue
			}
			if strings.HasSuffix(img, ":latest") || !strings.Contains(img, ":") {
				continue
			}
			unpinned = append(unpinned, fmt.Sprintf("%s=%s", cn, img))
			if classifyDigestPinSeverity(img) == "warning" {
				podSeverity = "warning"
			}
		}
		if len(unpinned) == 0 {
			continue
		}
		// Sort unpinned so different container declaration orders
		// hash to the same group.
		sorted := make([]string, len(unpinned))
		copy(sorted, unpinned)
		sort.Strings(sorted)
		owner := controllerOwnerName(p)
		if owner == "" {
			// Standalone Pod (no controller) — fall back to per-Pod
			// dedup so we still surface the finding.
			owner = p.GetName()
		}
		key := groupKey{ns: ns, ownerName: owner, imageSet: strings.Join(sorted, ",")}
		if g, ok := groups[key]; ok {
			g.podCount++
			// Upgrade severity if any pod in the group is warning.
			if podSeverity == "warning" {
				g.severity = "warning"
			}
			continue
		}
		groups[key] = &groupVal{
			samplePod: p,
			unpinned:  unpinned,
			severity:  podSeverity,
			podCount:  1,
		}
	}
	// Render one diagnostic per group, in deterministic order so the
	// seen-map fingerprint stays stable across cycles.
	keys := make([]groupKey, 0, len(groups))
	for k := range groups {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].ns != keys[j].ns {
			return keys[i].ns < keys[j].ns
		}
		return keys[i].ownerName < keys[j].ownerName
	})
	var out []Diagnostic
	for _, k := range keys {
		g := groups[k]
		// Subject = the workload-owner identity (ReplicaSet name for
		// Deployments, StatefulSet/DaemonSet name for those, Pod name
		// for standalone). The watcher's seen-map keys on Subject so
		// stable owner names stop the rolling-update re-alert.
		subject := fmt.Sprintf("Workload/%s/%s", k.ns, k.ownerName)
		msg := fmt.Sprintf(
			"Workload %s/%s mounts %d container image(s) without digest pin: %s",
			k.ns, k.ownerName, len(g.unpinned), strings.Join(g.unpinned, ", "))
		if g.podCount > 1 {
			msg = fmt.Sprintf("%s (across %d replica pods)", msg, g.podCount)
		}
		out = append(out, Diagnostic{
			Source:      "SecurityDrift",
			Subject:     subject,
			Severity:    g.severity,
			Message:     msg,
			Remediation: renderDigestPinRemediation(g.samplePod, g.unpinned),
		})
	}
	return out
}

// controllerOwnerName returns the name of the controller-managing
// OwnerReference on a Pod, or "" if none. ReplicaSets show up as the
// owner for Deployment-managed Pods; StatefulSet / DaemonSet show
// directly. The returned name is stable across the Pod's lifetime
// (unlike pod.Name which rotates on every rollout).
func controllerOwnerName(p *unstructured.Unstructured) string {
	for _, ref := range p.GetOwnerReferences() {
		if ref.Controller != nil && *ref.Controller {
			return ref.Name
		}
	}
	return ""
}

// renderDigestPinRemediation produces a remediation string with the
// actual observed digest substituted from each container's
// status.containerStatuses[].imageID — kubelet has already resolved the
// `:tag` to a digest at pull time, so we don't need the operator to run
// `crane digest` (which the AI tier cannot do).
//
// Fallback paths:
//   - status.containerStatuses missing or empty (Pod not yet scheduled,
//     image not pulled) → render a concrete `crane digest <repo>:<tag>`
//     command with the actual repo + tag substituted, NOT a literal
//     `<image>:<tag>` token.
//   - imageID for a specific container missing → fall back to
//     `crane digest` for that container only; other containers still get
//     the direct substitution.
func renderDigestPinRemediation(pod *unstructured.Unstructured, unpinned []string) string {
	imageIDs := podContainerImageIDs(pod)

	// unpinned entries have the shape "<containerName>=<image>:<tag>".
	var lines []string
	for _, u := range unpinned {
		eq := strings.IndexByte(u, '=')
		if eq < 0 {
			continue
		}
		cn := u[:eq]
		img := u[eq+1:]
		repo := img
		if c := strings.LastIndexByte(img, ':'); c >= 0 {
			repo = img[:c]
		}
		if iid, ok := imageIDs[cn]; ok && iid != "" {
			digest := extractDigestFromImageID(iid)
			if digest != "" {
				lines = append(lines, fmt.Sprintf(
					"  `%s` → replace `%s` with `%s@%s` in the workload manifest.",
					cn, img, repo, digest))
				continue
			}
		}
		// Fallback: kubelet hasn't recorded the digest yet. Render a
		// concrete `crane digest` invocation with the actual image:tag.
		lines = append(lines, fmt.Sprintf(
			"  `%s` → run `crane digest %s` to fetch the digest, then replace `%s` with `%s@<resulting-sha256>` in the workload manifest.",
			cn, img, img, repo))
	}
	body := "Replace each container's `:tag` reference with the resolved `@sha256:` digest so the runtime image is immutable + the image-attestation signature chain stays verifiable.\n" +
		strings.Join(lines, "\n")
	return body
}

// podContainerImageIDs maps container-name → status.containerStatuses[].imageID.
// Empty map when status.containerStatuses is missing (Pod not yet scheduled
// or image not pulled).
func podContainerImageIDs(pod *unstructured.Unstructured) map[string]string {
	out := map[string]string{}
	cs, _, _ := unstructured.NestedSlice(pod.Object, "status", "containerStatuses")
	for _, c := range cs {
		ci, ok := c.(map[string]interface{})
		if !ok {
			continue
		}
		name, _ := ci["name"].(string)
		iid, _ := ci["imageID"].(string)
		if name != "" && iid != "" {
			out[name] = iid
		}
	}
	return out
}

// extractDigestFromImageID pulls the `sha256:...` digest out of a
// kubelet imageID. Kubelet writes imageID as one of:
//   - `docker-pullable://<repo>@sha256:<hex>` (containerd-shim/docker)
//   - `<repo>@sha256:<hex>` (cri-o, modern containerd)
//   - `<repo>:<tag>` (rare — no digest recorded; returns empty)
func extractDigestFromImageID(imageID string) string {
	if i := strings.Index(imageID, "@sha256:"); i >= 0 {
		return imageID[i+1:] // drop the leading `@`, keep `sha256:...`
	}
	return ""
}

// checkNetworkPolicyCoverage flags user namespaces that run pods but
// have zero NetworkPolicies. Default K8s networking is fully open
// without at least one policy.
//
// v1.12.0: GATED on CNI enforcement. Clusters whose CNI doesn't
// actually enforce NetworkPolicy (Flannel-only is the canonical case)
// SKIP per-namespace warnings AND emit a single cluster-scope info
// finding explaining the gap — but ONLY when at least one user
// namespace would have been flagged under the old logic. Empty
// clusters / clusters where every user namespace already has a
// NetPol stay silent.
func (s SecurityDrift) checkNetworkPolicyCoverage(ctx context.Context, src snapshot.Source) []Diagnostic {
	nsList, err := src.List(ctx, gvrNamespace, "")
	if err != nil || nsList == nil {
		logListFailure("namespaces", err, true) // silent when the CRD/resource is absent; logs Forbidden etc.
		return nil
	}

	// Index NetworkPolicies by namespace once. List is cheap; the
	// per-namespace bool lookup avoids a second pass.
	netpolNamespaces := map[string]struct{}{}
	npList, _ := src.List(ctx, gvrNetworkPolicy, "")
	if npList != nil {
		for i := range npList.Items {
			netpolNamespaces[npList.Items[i].GetNamespace()] = struct{}{}
		}
	}

	// Index pod presence by namespace — namespaces with zero pods
	// don't pose a coverage gap (no traffic to govern). Saves
	// flagging brand-new empty namespaces.
	podNamespaces := map[string]struct{}{}
	pods, _ := src.List(ctx, snapshot.GVRPod, "")
	if pods != nil {
		for i := range pods.Items {
			podNamespaces[pods.Items[i].GetNamespace()] = struct{}{}
		}
	}

	// First pass: collect candidate namespaces (have pods, no NetPol,
	// not system). We need the count both to decide whether to emit the
	// CNI info finding AND to know what to flag if CNI enforces.
	var candidates []string
	for i := range nsList.Items {
		ns := &nsList.Items[i]
		name := ns.GetName()
		if _, isSystem := systemSecurityNamespaces[name]; isSystem {
			continue
		}
		if _, hasPods := podNamespaces[name]; !hasPods {
			continue
		}
		if _, hasPolicies := netpolNamespaces[name]; hasPolicies {
			continue
		}
		candidates = append(candidates, name)
	}
	if len(candidates) == 0 {
		return nil
	}

	// v1.12.0 CNI gate: if the runtime doesn't enforce NetPol, emit
	// ONE info-level finding explaining why we're not flagging the
	// per-namespace gaps as warnings. Applies on barebones k3s
	// (Flannel-only) and any cluster where CNI detection comes up
	// "unknown".
	cni := netpol.DetectCNI(ctx, src)
	if !cni.Enforces {
		return []Diagnostic{{
			Source:   "SecurityDrift",
			Subject:  "Cluster/cni-no-netpol-enforcement",
			Severity: "info",
			Message: fmt.Sprintf(
				"%d namespace(s) have no NetworkPolicy, but CNI %q does NOT enforce them. "+
					"%s. Adding NetworkPolicies here would be decorative-only.",
				len(candidates), cni.CNIName, cni.Evidence),
			Remediation: "For real zero-trust enforcement: add Calico-for-policy alongside Flannel, " +
				"or swap to Cilium. The CHA NetworkPolicy proposer (Phase 2d-β) only activates on " +
				"CNIs that enforce — barebones k3s with Flannel-only is intentionally left alone. " +
				"See docs/design/2026-06-rag-networkpolicy-proposer.md.",
		}}
	}

	// CNI enforces — emit the per-namespace warnings (original behaviour).
	var out []Diagnostic
	for _, name := range candidates {
		out = append(out, Diagnostic{
			Source:   "SecurityDrift",
			Subject:  fmt.Sprintf("Namespace/cluster/%s", name),
			Severity: "warning",
			Message: fmt.Sprintf(
				"Namespace %s runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide",
				name),
			Remediation: fmt.Sprintf(
				"Add at least a default-deny NetworkPolicy: "+
					"`kubectl apply -n %s -f - <<EOF\n"+
					"apiVersion: networking.k8s.io/v1\n"+
					"kind: NetworkPolicy\n"+
					"metadata:\n  name: default-deny-all\n"+
					"spec:\n  podSelector: {}\n  policyTypes: [Ingress, Egress]\n"+
					"EOF`\n"+
					"Then add explicit allow-rules for the inter-pod and egress paths the workloads need. "+
					"Helm-managed apps typically ship NetworkPolicies in-chart; missing here usually means the "+
					"`networkPolicy.enabled=true` value wasn't set at install.",
				name),
		})
	}
	return out
}
