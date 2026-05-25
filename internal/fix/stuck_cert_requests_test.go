// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package fix

import (
	"context"
	"errors"
	"testing"
)

const certRequestsFixture = `{
  "apiVersion": "v1",
  "kind": "List",
  "items": [
    {
      "apiVersion": "cert-manager.io/v1",
      "kind": "CertificateRequest",
      "metadata": {"name": "api-tls-1234", "namespace": "production"},
      "status": {
        "failureTime": "2026-05-01T00:00:00Z",
        "conditions": [
          {"type": "Ready", "status": "False", "reason": "Failed",
           "message": "The certificate request has failed to complete and will be retried"}
        ]
      }
    },
    {
      "apiVersion": "cert-manager.io/v1",
      "kind": "CertificateRequest",
      "metadata": {"name": "grafana-tls-5678", "namespace": "monitoring"},
      "status": {
        "conditions": [
          {"type": "Ready", "status": "False", "reason": "Pending",
           "message": "Waiting on certificate issuance from order"}
        ]
      }
    },
    {
      "apiVersion": "cert-manager.io/v1",
      "kind": "CertificateRequest",
      "metadata": {"name": "dashboard-tls-9abc", "namespace": "kube-system"},
      "status": {
        "failureTime": "2026-05-01T00:00:00Z",
        "conditions": [
          {"type": "Ready", "status": "False", "reason": "Failed"}
        ]
      }
    },
    {
      "apiVersion": "cert-manager.io/v1",
      "kind": "CertificateRequest",
      "metadata": {"name": "issued-tls-0001", "namespace": "production"},
      "status": {
        "conditions": [
          {"type": "Ready", "status": "True", "reason": "Issued"}
        ]
      }
    }
  ]
}`

const ordersFixture = `{
  "apiVersion": "v1",
  "kind": "List",
  "items": [
    {
      "apiVersion": "acme.cert-manager.io/v1",
      "kind": "Order",
      "metadata": {"name": "api-tls-order-abc", "namespace": "production"},
      "status": {"state": "errored", "reason": "Failed to create new order"}
    },
    {
      "apiVersion": "acme.cert-manager.io/v1",
      "kind": "Order",
      "metadata": {"name": "grafana-order-xyz", "namespace": "monitoring"},
      "status": {"state": "invalid", "reason": "CAA record prevents issuance"}
    },
    {
      "apiVersion": "acme.cert-manager.io/v1",
      "kind": "Order",
      "metadata": {"name": "pending-order-111", "namespace": "production"},
      "status": {"state": "pending"}
    },
    {
      "apiVersion": "acme.cert-manager.io/v1",
      "kind": "Order",
      "metadata": {"name": "vault-order-222", "namespace": "vault"},
      "status": {"state": "errored"}
    }
  ]
}`

func TestStuckCertificateRequests_RefusesOnSnapshot(t *testing.T) {
	src := loadSrc(t, map[string]string{
		"cert-manager.io-certificaterequests.json": certRequestsFixture,
	})
	r := StuckCertificateRequests{}.Run(context.Background(), src, nil)
	if r.Refused == "" {
		t.Error("expected Refused when Mutator is nil")
	}
	if len(r.Actions) != 0 {
		t.Errorf("expected no actions when refused, got %d", len(r.Actions))
	}
}

func TestStuckCertificateRequests_DeletesFailedCRs(t *testing.T) {
	src := loadSrc(t, map[string]string{
		"cert-manager.io-certificaterequests.json": certRequestsFixture,
	})
	m := newFakeMutator()
	r := StuckCertificateRequests{}.Run(context.Background(), src, m)

	if r.Refused != "" {
		t.Fatalf("unexpected Refused: %s", r.Refused)
	}
	// Only api-tls-1234 (production) should be deleted; kube-system is protected,
	// grafana-tls-5678 is still pending, issued-tls-0001 is ready.
	if got, want := len(r.Actions), 1; got != want {
		t.Fatalf("Actions = %d, want %d: %+v", got, want, r.Actions)
	}
	if got := m.calls[0]; got != "Delete certificaterequests/production/api-tls-1234" {
		t.Errorf("unexpected delete call: %s", got)
	}
}

func TestStuckCertificateRequests_SkipsPending(t *testing.T) {
	src := loadSrc(t, map[string]string{
		"cert-manager.io-certificaterequests.json": certRequestsFixture,
	})
	m := newFakeMutator()
	r := StuckCertificateRequests{}.Run(context.Background(), src, m)

	for _, s := range r.Skipped {
		if s.Object == "CertificateRequest/monitoring/grafana-tls-5678" {
			t.Errorf("pending CR should not appear in skipped (it should simply be ignored): %+v", s)
		}
	}
	for _, c := range m.calls {
		if c == "Delete certificaterequests/monitoring/grafana-tls-5678" {
			t.Error("pending CR must not be deleted")
		}
	}
}

func TestStuckCertificateRequests_SkipsProtectedNamespace(t *testing.T) {
	src := loadSrc(t, map[string]string{
		"cert-manager.io-certificaterequests.json": certRequestsFixture,
	})
	m := newFakeMutator()
	r := StuckCertificateRequests{}.Run(context.Background(), src, m)

	found := false
	for _, s := range r.Skipped {
		if s.Object == "CertificateRequest/kube-system/dashboard-tls-9abc" && s.Reason == "protected namespace" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected protected-namespace skip for kube-system CR; skipped=%+v", r.Skipped)
	}
}

func TestStuckCertificateRequests_DeletesFailedOrders(t *testing.T) {
	src := loadSrc(t, map[string]string{
		"acme.cert-manager.io-orders.json": ordersFixture,
	})
	m := newFakeMutator()
	r := StuckCertificateRequests{}.Run(context.Background(), src, m)

	// api-tls-order-abc (errored) + grafana-order-xyz (invalid) deleted;
	// pending-order-111 skipped; vault-order-222 in protected namespace.
	if got, want := len(r.Actions), 2; got != want {
		t.Fatalf("Actions = %d, want %d: %+v", got, want, r.Actions)
	}
	wantCalls := []string{
		"Delete orders/monitoring/grafana-order-xyz",
		"Delete orders/production/api-tls-order-abc",
	}
	if got := m.sortedCalls(); !equalStrings(got, wantCalls) {
		t.Errorf("calls = %v, want %v", got, wantCalls)
	}
}

func TestStuckCertificateRequests_DeleteError(t *testing.T) {
	src := loadSrc(t, map[string]string{
		"cert-manager.io-certificaterequests.json": certRequestsFixture,
	})
	m := newFakeMutator()
	m.returnErr["Delete certificaterequests/production/api-tls-1234"] = errors.New("forbidden")

	r := StuckCertificateRequests{}.Run(context.Background(), src, m)

	if len(r.Actions) != 0 {
		t.Errorf("expected 0 actions on delete error, got %d", len(r.Actions))
	}
	found := false
	for _, s := range r.Skipped {
		if s.Object == "CertificateRequest/production/api-tls-1234" && s.Reason == "delete failed: forbidden" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected delete-failure skip; skipped=%+v", r.Skipped)
	}
}

func TestStuckCertificateRequests_NothingToDo(t *testing.T) {
	src := loadSrc(t, map[string]string{})
	m := newFakeMutator()
	r := StuckCertificateRequests{}.Run(context.Background(), src, m)
	if len(r.Actions) != 0 || len(m.calls) != 0 {
		t.Errorf("expected no-op on empty cluster: actions=%d calls=%v", len(r.Actions), m.calls)
	}
}

// certManagerHealthy is the standard installation Deployment with 2/2
// replicas ready. Provided so the fixer's health gate sees a working
// controller and proceeds with deletion.
const certManagerHealthy = `{
  "apiVersion": "apps/v1", "kind": "Deployment",
  "metadata": {"name": "cert-manager", "namespace": "cert-manager"},
  "spec": {"replicas": 2},
  "status": {"replicas": 2, "readyReplicas": 2, "availableReplicas": 2}
}`

// certManagerDown has all replicas crashed. The fixer must NOT delete
// failed CRs in this state — cert-manager can't recreate them, and the
// deletion would simply nuke evidence of the failure without retry.
const certManagerDown = `{
  "apiVersion": "apps/v1", "kind": "Deployment",
  "metadata": {"name": "cert-manager", "namespace": "cert-manager"},
  "spec": {"replicas": 2},
  "status": {"replicas": 2, "readyReplicas": 0, "availableReplicas": 0}
}`

func TestStuckCertificateRequests_SkipsWhenCertManagerUnhealthy(t *testing.T) {
	src := loadSrc(t, map[string]string{
		"cert-manager.io-certificaterequests.json": certRequestsFixture,
		"deployments.json": certManagerDown,
	})
	m := newFakeMutator()
	r := StuckCertificateRequests{}.Run(context.Background(), src, m)

	if len(m.calls) != 0 {
		t.Errorf("must NOT delete any CRs when cert-manager itself is down, got: %v", m.calls)
	}
	if len(r.Actions) != 0 {
		t.Errorf("expected 0 Actions when cert-manager is unhealthy, got %d", len(r.Actions))
	}
	foundHealthGate := false
	for _, s := range r.Skipped {
		if contains(s.Reason, "cert-manager") && contains(s.Reason, "0 ready") {
			foundHealthGate = true
		}
	}
	if !foundHealthGate {
		t.Errorf("expected a 'cert-manager … 0 ready' skip entry; got: %+v", r.Skipped)
	}
}

func TestStuckCertificateRequests_ProceedsWhenCertManagerHealthy(t *testing.T) {
	src := loadSrc(t, map[string]string{
		"cert-manager.io-certificaterequests.json": certRequestsFixture,
		"deployments.json": certManagerHealthy,
	})
	m := newFakeMutator()
	r := StuckCertificateRequests{}.Run(context.Background(), src, m)

	if len(r.Actions) == 0 {
		t.Errorf("expected the failed CR in 'production' to be deleted; calls=%v skipped=%+v",
			m.calls, r.Skipped)
	}
}

// TestStuckCertRequests_ProceedsWhenCertManagerNotInSnapshot — defensive:
// if cert-manager's Deployment isn't captured (incomplete snapshot, or a
// non-standard installation namespace), the fixer cannot evaluate health.
// Fall back to today's behavior (proceed) rather than over-blocking.
// Operators who want strict gating can pin the cert-manager namespace
// into the snapshot via Helm values.
func TestStuckCertRequests_ProceedsWhenCertManagerNotInSnapshot(t *testing.T) {
	src := loadSrc(t, map[string]string{
		"cert-manager.io-certificaterequests.json": certRequestsFixture,
	})
	m := newFakeMutator()
	r := StuckCertificateRequests{}.Run(context.Background(), src, m)

	if len(r.Actions) == 0 {
		t.Errorf("expected deletion to proceed when cert-manager Deployment is absent from snapshot; calls=%v skipped=%+v",
			m.calls, r.Skipped)
	}
}
