// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package fix

import (
	"context"

	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/internal/snapshot"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
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
