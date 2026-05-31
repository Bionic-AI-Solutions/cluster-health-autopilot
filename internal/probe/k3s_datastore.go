// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package probe

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/internal/snapshot"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// K3sDatastore detects the k3s cluster's datastore mode (embedded etcd vs
// SQLite) and evaluates health signals appropriate to each mode.
//
// Design rationale:
//
//   - k3s runs etcd as a static pod in kube-system (labelled component=etcd),
//     identical to kubeadm, when the cluster is in HA embedded-etcd mode (3+
//     control-plane nodes). The existing ETCD probe therefore works correctly
//     on HA k3s clusters — but it emits a spurious "no etcd pods found" warning
//     on single-node k3s installs that use SQLite.
//
//   - Detection heuristic: any Node whose spec.providerID starts with "k3s://"
//     identifies this as a k3s cluster. If none match, the probe auto-skips so
//     it is safe to register default-on on any cluster.
//
//   - Embedded-etcd mode is confirmed when at least one static etcd pod is
//     found in kube-system (name prefix "etcd-" or label component=etcd). In
//     that mode the probe evaluates the same pod-readiness signals as the ETCD
//     probe and additionally checks for a recent etcd snapshot ConfigMap.
//
//   - SQLite mode (no etcd pods) is healthy-by-design on a single-node k3s
//     install. The probe emits a SeverityInfo advisory about the lack of HA
//     unless CHA_K3S_SINGLE_NODE_OK=true suppresses it.
//
// Opt-out: set CHA_PROBE_K3S_DATASTORE=off to disable entirely.
// Recommendation: set CHA_PROBE_ETCD=off when using this probe to avoid the
// redundant "no etcd pods" warning from the kubeadm ETCD probe.
type K3sDatastore struct {
	// SnapshotSLA caps how stale the newest etcd snapshot ConfigMap may be
	// before a warning is raised. Zero uses defaultK3sSnapshotSLA (26 h).
	SnapshotSLA time.Duration

	// Now returns the current time; overridable in tests.
	Now func() time.Time
}

const k3sDatastoreName = "K3s Datastore"

const defaultK3sSnapshotSLA = 26 * time.Hour

// k3sEtcdSnapshotPrefix is the ConfigMap name prefix that k3s writes on each
// scheduled etcd snapshot. k3s names these "k3s-etcd-snapshot-<timestamp>".
const k3sEtcdSnapshotPrefix = "k3s-etcd-snapshot"

// Name satisfies probe.Probe.
func (K3sDatastore) Name() string { return k3sDatastoreName }

// Run satisfies probe.Probe.
func (d K3sDatastore) Run(ctx context.Context, src snapshot.Source) Result {
	r := Result{Component: ComponentResult{Component: k3sDatastoreName}}

	now := d.Now
	if now == nil {
		now = time.Now
	}
	sla := d.SnapshotSLA
	if sla == 0 {
		sla = defaultK3sSnapshotSLA
	}

	// ── Step 1: List Nodes — detect k3s via providerID ────────────────────
	nodes, err := src.List(ctx, snapshot.GVRNode, "")
	if err != nil {
		r.Component.Status = "PROBE_FAILED"
		r.Component.Detail = "list nodes: " + err.Error()
		return r
	}

	isK3s := false
	for _, n := range nodes.Items {
		providerID, _, _ := unstructured.NestedString(n.Object, "spec", "providerID")
		if strings.HasPrefix(providerID, "k3s://") {
			isK3s = true
			break
		}
		// Fallback heuristic: any k3s.io/* annotation on the node.
		for k := range n.GetAnnotations() {
			if strings.HasPrefix(k, "k3s.io/") {
				isK3s = true
				break
			}
		}
		if isK3s {
			break
		}
	}

	if !isK3s {
		// Not a k3s cluster — probe is a no-op.
		r.Component.Status = "SKIPPED"
		r.Component.Detail = "No k3s node providerID or annotations found; probe only applies to k3s clusters"
		return r
	}

	// ── Step 2: List pods in kube-system — detect embedded etcd mode ──────
	pods, err := src.List(ctx, snapshot.GVRPod, "kube-system")
	if err != nil {
		r.Component.Status = "PROBE_FAILED"
		r.Component.Detail = "list pods in kube-system: " + err.Error()
		return r
	}

	// Collect etcd static pods (same heuristic as the ETCD probe).
	type etcdMember struct {
		name     string
		ready    bool
		restarts int64
	}
	var members []etcdMember
	for _, pod := range pods.Items {
		if !looksLikeEtcdPod(pod) {
			continue
		}
		m := etcdMember{name: pod.GetName()}
		conds, _, _ := getSliceField(pod.Object, "status", "conditions")
		for _, c := range conds {
			cm, ok := c.(map[string]any)
			if !ok {
				continue
			}
			if cm["type"] == "Ready" {
				m.ready = cm["status"] == "True"
			}
		}
		statuses, _, _ := getSliceField(pod.Object, "status", "containerStatuses")
		for _, s := range statuses {
			sm, ok := s.(map[string]any)
			if !ok {
				continue
			}
			if rc := asInt64(sm["restartCount"]); rc > m.restarts {
				m.restarts = rc
			}
		}
		members = append(members, m)
	}

	var findings []Finding

	if len(members) == 0 {
		// ── SQLite mode (single-node k3s, no etcd pods) ───────────────────
		singleNodeOK := os.Getenv("CHA_K3S_SINGLE_NODE_OK") == "true"
		if !singleNodeOK {
			findings = append(findings, Finding{
				Component: k3sDatastoreName,
				Severity:  SeverityInfo,
				Message:   "k3s cluster appears to use SQLite (single-node, no etcd static pods found); no HA for the datastore",
				Remediation: "For production deployments, run at least 3 control-plane nodes to enable embedded etcd HA. " +
					"See: https://docs.k3s.io/datastore/ha-embedded. " +
					"To suppress this advisory on an intentional single-node install, set CHA_K3S_SINGLE_NODE_OK=true.",
			})
		}

		// SQLite mode is otherwise healthy — no further checks needed.
		r.Component.Status = rollupComponentStatus(findings)
		if r.Component.Status == "HEALTHY" {
			r.Component.Detail = "k3s SQLite datastore (single-node); no etcd pods expected"
		} else {
			r.Component.Detail = "k3s SQLite datastore (single-node, no HA)"
		}
		r.Findings = findings
		return r
	}

	// ── Embedded etcd mode ────────────────────────────────────────────────
	// Evaluate pod readiness and restart counts — same logic as the ETCD probe.
	var notReady, restarted []string
	for _, m := range members {
		if !m.ready {
			notReady = append(notReady, m.name)
		}
		if m.restarts > 0 {
			restarted = append(restarted, fmt.Sprintf("%s(%d)", m.name, m.restarts))
		}
	}

	// ── Step 2b: Per-node etcd-member metadata ────────────────────────────
	// k3s labels each etcd-bearing node with `node-role.kubernetes.io/etcd`
	// and annotates with `etcd.k3s.cattle.io/local-snapshots-timestamp` on
	// every successful local snapshot. The two together tell us:
	//   1. How many etcd members the *cluster* believes it should have
	//      (count of nodes with the etcd role label).
	//   2. Whether each member is actively participating in the snapshot
	//      rotation (annotation present and recent).
	// Compare against the live etcd pod count → quorum risk detection.
	var etcdNodes []string
	memberSnapshotAges := make(map[string]time.Duration)
	for _, n := range nodes.Items {
		labels := n.GetLabels()
		if _, isEtcd := labels["node-role.kubernetes.io/etcd"]; !isEtcd {
			continue
		}
		etcdNodes = append(etcdNodes, n.GetName())
		annos := n.GetAnnotations()
		ts, ok := annos["etcd.k3s.cattle.io/local-snapshots-timestamp"]
		if !ok || ts == "" {
			memberSnapshotAges[n.GetName()] = -1 // sentinel: never snapshotted
			continue
		}
		// k3s timestamp format: 20060102150405 (sortable, no separators).
		parsed, err := time.Parse("20060102150405", ts)
		if err != nil {
			// Fall back to RFC3339; older k3s versions used that format.
			parsed, err = time.Parse(time.RFC3339, ts)
		}
		if err != nil {
			memberSnapshotAges[n.GetName()] = -2 // sentinel: unparseable
			continue
		}
		memberSnapshotAges[n.GetName()] = now().Sub(parsed)
	}

	// Quorum check: with N voting etcd members, the cluster tolerates
	// (N-1)/2 failures. If running pod count < (declared/2)+1, quorum
	// is at risk if even one more member goes down.
	declared := len(etcdNodes)
	running := len(members) - len(notReady)
	if declared >= 3 && running < (declared/2)+1 {
		findings = append(findings, Finding{
			Component: k3sDatastoreName,
			Severity:  SeverityCritical,
			Message: fmt.Sprintf(
				"k3s embedded etcd: only %d/%d members ready — below quorum threshold",
				running, declared,
			),
			Remediation: "etcd has lost quorum. The cluster is read-only. " +
				"Bring failed members back: " +
				"check k3s status on the affected node(s): `sudo systemctl status k3s`. " +
				"If a member is unrecoverable, see https://docs.k3s.io/datastore/ha-embedded#remove-a-failed-server.",
		})
	} else if declared == 2 {
		findings = append(findings, Finding{
			Component: k3sDatastoreName,
			Severity:  SeverityWarning,
			Message:   "k3s embedded etcd has exactly 2 voting members — no fault tolerance (a 2-node cluster cannot tolerate any failure)",
			Remediation: "Add a third control-plane node OR remove one to drop back to single-node. " +
				"See https://docs.k3s.io/datastore/ha-embedded.",
		})
	}

	// Member-count parity: declared etcd-role nodes vs running etcd pods.
	if declared > len(members) && len(members) > 0 {
		findings = append(findings, Finding{
			Component: k3sDatastoreName,
			Severity:  SeverityCritical,
			Message: fmt.Sprintf(
				"k3s embedded etcd: %d node(s) labelled node-role.kubernetes.io/etcd but only %d etcd pod(s) running — missing member(s)",
				declared, len(members),
			),
			Remediation: "An etcd-labelled node is not running its etcd member. " +
				"On the affected node: `sudo systemctl status k3s` and `journalctl -u k3s | grep etcd`. " +
				"If the node was removed permanently, drop the role label: " +
				"`kubectl label node <name> node-role.kubernetes.io/etcd-`.",
		})
	}

	// Per-member snapshot freshness. Compare each member's local
	// timestamp to the newest; a member that's >sla behind the newest
	// is not participating in the snapshot rotation.
	if len(memberSnapshotAges) > 0 {
		var newestAge time.Duration = 1<<62 - 1
		for _, age := range memberSnapshotAges {
			if age < 0 {
				continue
			}
			if age < newestAge {
				newestAge = age
			}
		}
		var stale []string
		var never []string
		for name, age := range memberSnapshotAges {
			switch {
			case age == -1:
				never = append(never, name)
			case age == -2:
				// Unparseable — skip silently; this is a forward-compat
				// guard, not a real failure mode.
			case newestAge >= 0 && age-newestAge > sla:
				stale = append(stale, fmt.Sprintf("%s(%s behind)", name, (age-newestAge).Round(time.Minute)))
			}
		}
		if len(never) > 0 {
			findings = append(findings, Finding{
				Component: k3sDatastoreName,
				Severity:  SeverityWarning,
				Message: fmt.Sprintf(
					"k3s embedded etcd member(s) never wrote a local snapshot: %s",
					strings.Join(never, ", "),
				),
				Remediation: "Member is etcd-labelled but never wrote etcd.k3s.cattle.io/local-snapshots-timestamp. " +
					"Either the member just joined (wait one snapshot cycle) or its k3s server can't write snapshots " +
					"(check `journalctl -u k3s | grep -i snapshot` on the node).",
			})
		}
		if len(stale) > 0 {
			findings = append(findings, Finding{
				Component: k3sDatastoreName,
				Severity:  SeverityWarning,
				Message: fmt.Sprintf(
					"k3s embedded etcd member(s) lag behind newest snapshot by >%s: %s",
					sla, strings.Join(stale, ", "),
				),
				Remediation: "Member is participating in the cluster but not the snapshot rotation. " +
					"Check `--etcd-snapshot-schedule-cron` on the affected node's k3s server config, " +
					"and inspect `journalctl -u k3s | grep snapshot` for write failures.",
			})
		}
	}

	if len(notReady) > 0 {
		findings = append(findings, Finding{
			Component: k3sDatastoreName,
			Severity:  SeverityCritical,
			Message:   fmt.Sprintf("k3s etcd member(s) not Ready: %s", strings.Join(notReady, ", ")),
			Remediation: "Inspect the etcd member: `kubectl describe pod <name> -n kube-system`. " +
				"On the control-plane node: `sudo systemctl status k3s`, " +
				"`journalctl -u k3s --since '10 minutes ago'`. " +
				"Check disk space and IO latency; etcd requires <100ms fsync.",
		})
	}
	if len(restarted) > 0 {
		findings = append(findings, Finding{
			Component: k3sDatastoreName,
			Severity:  SeverityWarning,
			Message:   fmt.Sprintf("k3s etcd member(s) restarted: %s", strings.Join(restarted, ", ")),
			Remediation: "Check etcd logs: `kubectl logs <pod> -n kube-system --previous`. " +
				"Common causes: disk IO latency, OOM (raise k3s memory limit), quorum loss.",
		})
	}

	// ── Step 3: Check etcd snapshot ConfigMaps ───────────────────────────
	// k3s writes a ConfigMap named "k3s-etcd-snapshot-<timestamp>" in
	// kube-system for each scheduled snapshot. The presence and age of the
	// most-recent one indicates whether the snapshot schedule is functioning.
	cms, err := src.List(ctx, snapshot.GVRConfigMap, "kube-system")
	if err != nil {
		// ConfigMap list failure is non-fatal — the etcd pod health check
		// above is the primary signal. Downgrade to a warning.
		findings = append(findings, Finding{
			Component:   k3sDatastoreName,
			Severity:    SeverityWarning,
			Message:     "Could not list kube-system ConfigMaps to check etcd snapshot age: " + err.Error(),
			Remediation: "Verify that the CHA service account has `get,list,watch` on ConfigMaps in kube-system.",
		})
	} else {
		var newestSnapshot time.Time
		var newestName string
		for _, cm := range cms.Items {
			if !strings.HasPrefix(cm.GetName(), k3sEtcdSnapshotPrefix) {
				continue
			}
			ts := cm.GetCreationTimestamp().Time
			if ts.After(newestSnapshot) {
				newestSnapshot = ts
				newestName = cm.GetName()
			}
		}

		t := now()
		if newestSnapshot.IsZero() {
			findings = append(findings, Finding{
				Component: k3sDatastoreName,
				Severity:  SeverityWarning,
				Message:   "k3s embedded etcd: no etcd snapshot ConfigMap found in kube-system",
				Remediation: "Check the k3s etcd-snapshot schedule: `kubectl get configmaps -n kube-system | grep k3s-etcd-snapshot`. " +
					"On the control-plane node: `k3s etcd-snapshot ls`. " +
					"Verify `--etcd-snapshot-schedule-cron` in the k3s server args.",
			})
		} else if t.Sub(newestSnapshot) > sla {
			age := t.Sub(newestSnapshot).Round(time.Minute)
			findings = append(findings, Finding{
				Component: k3sDatastoreName,
				Severity:  SeverityWarning,
				Message: fmt.Sprintf(
					"k3s embedded etcd: most recent snapshot ConfigMap %q is %s old (threshold: %s)",
					newestName, age, sla,
				),
				Remediation: "Check k3s etcd-snapshot schedule: `kubectl get configmaps -n kube-system -l 'k3s.io/etcd-snapshot-name'`. " +
					"On the control-plane node: `k3s etcd-snapshot ls`. " +
					"Verify `--etcd-snapshot-schedule-cron` in the k3s server args.",
			})
		}
	}

	r.Component.Status = rollupComponentStatus(findings)
	if r.Component.Status == "HEALTHY" {
		r.Component.Detail = fmt.Sprintf(
			"k3s embedded etcd: all %d member(s) ready, snapshot within SLA",
			len(members),
		)
	} else {
		parts := []string{fmt.Sprintf("%d etcd member(s)", len(members))}
		if len(notReady) > 0 {
			parts = append(parts, fmt.Sprintf("%d not ready", len(notReady)))
		}
		if len(restarted) > 0 {
			parts = append(parts, fmt.Sprintf("%d restarted", len(restarted)))
		}
		r.Component.Detail = strings.Join(parts, "; ")
	}
	r.Findings = findings
	return r
}
