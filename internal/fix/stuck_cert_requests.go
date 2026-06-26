// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

package fix

import (
	"context"
	"fmt"

	"github.com/srenix-ai/agentic-sre/internal/snapshot"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// certManagerNamespace and certManagerDeployment identify the standard
// cert-manager controller installation. Non-standard installs (different
// namespace or Deployment name) will see the fixer fall through to its
// pre-1.5 behavior — see TestStuckCertRequests_ProceedsWhenCertManagerNotInSnapshot.
const (
	certManagerNamespace  = "cert-manager"
	certManagerDeployment = "cert-manager"
)

// StuckCertificateRequests deletes cert-manager CertificateRequest and ACME
// Order CRs that have permanently failed, allowing cert-manager to retry the
// issuance flow on its next reconcile.
//
// cert-manager owns the Certificate → CertificateRequest → Order lifecycle.
// When an ACME challenge fails (rate-limit, DNS not propagated, solver pod
// crash), the Order enters state=errored/invalid and the CertificateRequest
// is marked Ready=False/reason=Failed. cert-manager will NOT retry until the
// failed child resources are deleted — this fixer performs that deletion.
//
// Safety contract:
//   - Only touches CRs with an unambiguously terminal failure state.
//   - Never touches CRs still in progress (reason=Pending, state=pending/valid).
//   - Skips protected namespaces.
//   - cert-manager immediately recreates the deleted CR — this is idempotent.
//
// OWASP K8s Top-10 respected: K08 (Secrets Management Failures / TLS) —
// deleting a terminally-failed request lets cert-manager retry issuance; Srenix
// never writes the TLS Secret and cannot downgrade a live cert. See
// docs/OWASP_MAPPING.md and internal/fix/owasp_posture_test.go.
type StuckCertificateRequests struct{}

// Name returns the fixer's identifier.
func (StuckCertificateRequests) Name() string { return "StuckCertificateRequests" }

// Run deletes failed CertificateRequests and ACME Orders.
func (StuckCertificateRequests) Run(ctx context.Context, src snapshot.Source, m snapshot.Mutator) Result {
	r := Result{Fixer: "StuckCertificateRequests"}
	if m == nil {
		r.Refused = "snapshot mode — fixers require live cluster access"
		return r
	}

	// Health gate: if cert-manager's own controller Deployment is captured
	// in the snapshot and reports 0 ready replicas, refuse to delete CRs.
	// cert-manager cannot recreate them in this state; the deletion would
	// just nuke evidence of the failure with no retry. When the Deployment
	// is absent from the snapshot (non-standard install, incomplete
	// capture), fall through to the pre-1.5 behavior rather than block.
	if dep, err := src.Get(ctx, snapshot.GVRDeployment, certManagerNamespace, certManagerDeployment); err == nil && dep != nil {
		ready, _, _ := unstructured.NestedInt64(dep.Object, "status", "readyReplicas")
		desired, _, _ := unstructured.NestedInt64(dep.Object, "spec", "replicas")
		if ready == 0 {
			r.Skipped = append(r.Skipped, SkipReason{
				Object: "Deployment/" + certManagerNamespace + "/" + certManagerDeployment,
				Reason: fmt.Sprintf("cert-manager controller unhealthy: %d/%d ready — refusing to delete CRs (cert-manager cannot recreate them)",
					ready, desired),
			})
			return r
		}
	}

	r = deleteStaleCertificateRequests(ctx, src, m, r)
	r = deleteStaleOrders(ctx, src, m, r)
	return r
}

func deleteStaleCertificateRequests(ctx context.Context, src snapshot.Source, m snapshot.Mutator, r Result) Result {
	crs, err := src.List(ctx, snapshot.GVRCertificateRequest, "")
	if err != nil || len(crs.Items) == 0 {
		return r
	}
	for i := range crs.Items {
		cr := crs.Items[i]
		ns := cr.GetNamespace()
		name := cr.GetName()
		obj := "CertificateRequest/" + ns + "/" + name

		if IsProtectedNamespace(ns) {
			r.Skipped = append(r.Skipped, SkipReason{Object: obj, Reason: "protected namespace"})
			continue
		}
		if !certRequestFailed(cr) {
			continue
		}
		if err := m.Delete(ctx, snapshot.GVRCertificateRequest, ns, name); err != nil {
			r.Skipped = append(r.Skipped, SkipReason{Object: obj, Reason: "delete failed: " + err.Error()})
			continue
		}
		r.Actions = append(r.Actions, Action{
			Description: "Deleted failed CertificateRequest; cert-manager will retry issuance",
			Object:      obj,
		})
	}
	return r
}

func deleteStaleOrders(ctx context.Context, src snapshot.Source, m snapshot.Mutator, r Result) Result {
	orders, err := src.List(ctx, snapshot.GVRCertManagerOrder, "")
	if err != nil || len(orders.Items) == 0 {
		return r
	}
	for i := range orders.Items {
		order := orders.Items[i]
		ns := order.GetNamespace()
		name := order.GetName()
		obj := "Order/" + ns + "/" + name

		if IsProtectedNamespace(ns) {
			r.Skipped = append(r.Skipped, SkipReason{Object: obj, Reason: "protected namespace"})
			continue
		}
		if !orderFailed(order) {
			continue
		}
		if err := m.Delete(ctx, snapshot.GVRCertManagerOrder, ns, name); err != nil {
			r.Skipped = append(r.Skipped, SkipReason{Object: obj, Reason: "delete failed: " + err.Error()})
			continue
		}
		r.Actions = append(r.Actions, Action{
			Description: "Deleted failed ACME Order; cert-manager will retry the ACME challenge",
			Object:      obj,
		})
	}
	return r
}

// certRequestFailed returns true when a CertificateRequest has permanently
// failed: Ready=False with reason=Failed, or failureTime is set.
func certRequestFailed(cr unstructured.Unstructured) bool {
	// Check .status.failureTime — set only on terminal failure.
	ft, _, _ := unstructured.NestedString(cr.Object, "status", "failureTime")
	if ft != "" {
		return true
	}
	// Check .status.conditions for type=Ready, status=False, reason=Failed.
	conditions, _, _ := unstructured.NestedSlice(cr.Object, "status", "conditions")
	for _, c := range conditions {
		cm, ok := c.(map[string]any)
		if !ok {
			continue
		}
		if cm["type"] == "Ready" && cm["status"] == "False" && cm["reason"] == "Failed" {
			return true
		}
	}
	return false
}

// orderFailed returns true when an ACME Order is in a terminal failure state.
// Pending/valid/processing orders are not touched.
func orderFailed(order unstructured.Unstructured) bool {
	state, _, _ := unstructured.NestedString(order.Object, "status", "state")
	return state == "errored" || state == "invalid"
}
