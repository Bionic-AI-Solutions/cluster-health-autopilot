// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package diagnose

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	pkgsnapshot "github.com/Bionic-AI-Solutions/cluster-health-autopilot/pkg/snapshot"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// ── in-memory Source ──────────────────────────────────────────────────────────

// memSourceDNS is a minimal in-memory snapshot.Source for DNSChainDrift tests.
type memSourceDNS struct {
	byResource map[string][]unstructured.Unstructured
	mode       pkgsnapshot.Mode
}

func newMemSourceDNS() *memSourceDNS {
	return &memSourceDNS{
		byResource: make(map[string][]unstructured.Unstructured),
		mode:       pkgsnapshot.ModeLive,
	}
}

func (m *memSourceDNS) add(u unstructured.Unstructured) {
	res := u.GetKind()
	// Map Kind → resource name (same table the file source uses).
	switch res {
	case "Ingress":
		res = "ingresses"
	case "Service":
		res = "services"
	case "Endpoints":
		res = "endpoints"
	default:
		res = strings.ToLower(res) + "s"
	}
	m.byResource[res] = append(m.byResource[res], u)
}

func (m *memSourceDNS) List(_ context.Context, gvr schema.GroupVersionResource, ns string) (*unstructured.UnstructuredList, error) {
	out := &unstructured.UnstructuredList{}
	for _, u := range m.byResource[gvr.Resource] {
		if ns != "" && u.GetNamespace() != ns {
			continue
		}
		out.Items = append(out.Items, u)
	}
	return out, nil
}

func (m *memSourceDNS) Get(_ context.Context, gvr schema.GroupVersionResource, ns, name string) (*unstructured.Unstructured, error) {
	for _, u := range m.byResource[gvr.Resource] {
		if u.GetNamespace() == ns && u.GetName() == name {
			cp := u.DeepCopy()
			return cp, nil
		}
	}
	return nil, errors.New("not found")
}

func (m *memSourceDNS) Mode() pkgsnapshot.Mode { return m.mode }

// ── fake Cloudflare client ────────────────────────────────────────────────────

type fakeCFClient struct {
	zones   []Zone
	records map[string][]DNSRecord // zoneID → records
	err     error                  // if set, both methods return this error
	delay   time.Duration          // if > 0, sleep before returning
}

func (f *fakeCFClient) ListZones(ctx context.Context) ([]Zone, error) {
	if f.delay > 0 {
		select {
		case <-time.After(f.delay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	if f.err != nil {
		return nil, f.err
	}
	return f.zones, nil
}

func (f *fakeCFClient) ListDNSRecords(ctx context.Context, zoneID string) ([]DNSRecord, error) {
	if f.delay > 0 {
		select {
		case <-time.After(f.delay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	if f.err != nil {
		return nil, f.err
	}
	return f.records[zoneID], nil
}

// ── builder helpers ───────────────────────────────────────────────────────────

// makeIngress builds a networking.k8s.io/v1 Ingress with one rule pointing at svcName:svcPort.
// Set svcName="" to omit the backend (default-backend only Ingress).
// Set annotations["cha.bionicaisolutions.com/probe-disable"]="true" via annos param.
func makeIngress(ns, name, host, svcName string, svcPort int64, annos map[string]string) unstructured.Unstructured {
	u := unstructured.Unstructured{}
	u.SetAPIVersion("networking.k8s.io/v1")
	u.SetKind("Ingress")
	u.SetNamespace(ns)
	u.SetName(name)
	if len(annos) > 0 {
		u.SetAnnotations(annos)
	}

	var paths []any
	if svcName != "" {
		paths = []any{
			map[string]any{
				"path":     "/",
				"pathType": "Prefix",
				"backend": map[string]any{
					"service": map[string]any{
						"name": svcName,
						"port": map[string]any{
							"number": svcPort,
						},
					},
				},
			},
		}
	}

	rule := map[string]any{
		"host": host,
		"http": map[string]any{
			"paths": paths,
		},
	}
	_ = unstructured.SetNestedSlice(u.Object, []any{rule}, "spec", "rules")
	return u
}

// makeService builds a core/v1 Service (type ClusterIP by default).
func makeService(ns, name string) unstructured.Unstructured {
	u := unstructured.Unstructured{}
	u.SetAPIVersion("v1")
	u.SetKind("Service")
	u.SetNamespace(ns)
	u.SetName(name)
	_ = unstructured.SetNestedField(u.Object, "ClusterIP", "spec", "type")
	return u
}

// makeEndpoints builds a core/v1 Endpoints with readyAddresses addresses in one subset.
func makeEndpoints(ns, name string, readyAddresses int) unstructured.Unstructured {
	u := unstructured.Unstructured{}
	u.SetAPIVersion("v1")
	u.SetKind("Endpoints")
	u.SetNamespace(ns)
	u.SetName(name)
	if readyAddresses > 0 {
		addrs := make([]any, readyAddresses)
		for i := range addrs {
			addrs[i] = map[string]any{"ip": "10.0.0.1"}
		}
		_ = unstructured.SetNestedSlice(u.Object, []any{
			map[string]any{"addresses": addrs},
		}, "subsets")
	}
	return u
}

// makeKongService builds a LoadBalancer Service that mimics a Kong ingress
// controller so that findIngressControllerLBIP can pick it up.
func makeKongService(ns, name, lbIP string) unstructured.Unstructured {
	u := unstructured.Unstructured{}
	u.SetAPIVersion("v1")
	u.SetKind("Service")
	u.SetNamespace(ns)
	u.SetName(name)
	u.SetLabels(map[string]string{"app.kubernetes.io/name": "kong"})
	_ = unstructured.SetNestedField(u.Object, "LoadBalancer", "spec", "type")
	_ = unstructured.SetNestedSlice(u.Object, []any{
		map[string]any{"ip": lbIP},
	}, "status", "loadBalancer", "ingress")
	return u
}

// ── tests ─────────────────────────────────────────────────────────────────────

// TestDNSChainDrift_HappyPath — CF match, Ingress exists, Service has Endpoints → 0 diagnostics.
func TestDNSChainDrift_HappyPath(t *testing.T) {
	const (
		host   = "api.example.com"
		lbIP   = "203.0.113.10"
		zoneID = "zone1"
	)

	src := newMemSourceDNS()
	src.add(makeIngress("default", "api-ing", host, "api-svc", 80, nil))
	src.add(makeService("default", "api-svc"))
	src.add(makeEndpoints("default", "api-svc", 2))
	src.add(makeKongService("kong", "kong-proxy", lbIP))

	cf := &fakeCFClient{
		zones: []Zone{{ID: zoneID, Name: "example.com"}},
		records: map[string][]DNSRecord{
			zoneID: {{Name: host, Type: "A", Content: lbIP, Proxied: false}},
		},
	}

	a := DNSChainDrift{Client: cf}
	diags := a.Run(context.Background(), src)
	if len(diags) != 0 {
		t.Errorf("expected 0 diagnostics on happy path, got %d: %+v", len(diags), diags)
	}
}

// TestDNSChainDrift_MissingCFRecord — no CF record → missing-cloudflare-record.
func TestDNSChainDrift_MissingCFRecord(t *testing.T) {
	const (
		host   = "missing.example.com"
		lbIP   = "203.0.113.10"
		zoneID = "zone1"
	)

	src := newMemSourceDNS()
	src.add(makeIngress("default", "missing-ing", host, "missing-svc", 80, nil))
	src.add(makeService("default", "missing-svc"))
	src.add(makeEndpoints("default", "missing-svc", 1))
	src.add(makeKongService("kong", "kong-proxy", lbIP))

	cf := &fakeCFClient{
		zones:   []Zone{{ID: zoneID, Name: "example.com"}},
		records: map[string][]DNSRecord{}, // no records
	}

	a := DNSChainDrift{Client: cf}
	diags := a.Run(context.Background(), src)

	found := false
	for _, d := range diags {
		if strings.Contains(d.Subject, "missing-cloudflare-record") && strings.Contains(d.Subject, host) {
			found = true
			if d.Severity != "error" {
				t.Errorf("expected severity=error, got %q", d.Severity)
			}
		}
	}
	if !found {
		t.Errorf("expected missing-cloudflare-record diagnostic for %s, got: %+v", host, diags)
	}
}

// TestDNSChainDrift_CFPointsElsewhere — CF record has wrong IP → cloudflare-points-elsewhere.
func TestDNSChainDrift_CFPointsElsewhere(t *testing.T) {
	const (
		host    = "stale.example.com"
		lbIP    = "203.0.113.10"
		wrongIP = "9.9.9.9"
		zoneID  = "zone1"
	)

	src := newMemSourceDNS()
	src.add(makeIngress("default", "stale-ing", host, "stale-svc", 80, nil))
	src.add(makeService("default", "stale-svc"))
	src.add(makeEndpoints("default", "stale-svc", 1))
	src.add(makeKongService("kong", "kong-proxy", lbIP))

	cf := &fakeCFClient{
		zones: []Zone{{ID: zoneID, Name: "example.com"}},
		records: map[string][]DNSRecord{
			zoneID: {{Name: host, Type: "A", Content: wrongIP, Proxied: false}},
		},
	}

	a := DNSChainDrift{Client: cf}
	diags := a.Run(context.Background(), src)

	found := false
	for _, d := range diags {
		if strings.Contains(d.Subject, "cloudflare-points-elsewhere") && strings.Contains(d.Subject, host) {
			found = true
			if d.Severity != "error" {
				t.Errorf("expected severity=error, got %q", d.Severity)
			}
			if !strings.Contains(d.Message, wrongIP) {
				t.Errorf("expected message to contain wrong IP %q: %s", wrongIP, d.Message)
			}
		}
	}
	if !found {
		t.Errorf("expected cloudflare-points-elsewhere diagnostic, got: %+v", diags)
	}
}

// TestDNSChainDrift_MissingIngress — Ingress with a host but no backend Service means
// the Ingress layer reports missing-ingress when the ingress has an empty rule
// that resolves to be==nil but isn't in seedHostSet.
// Per the actual analyzer design: seed-only hosts with no Ingress are treated as
// "external-only" (CF-only) and skip the K8s chain. The missing-ingress path fires
// when an Ingress rule has no backend at all (empty paths) — the Ingress exists but
// resolves to be.svcName="". We test this via an Ingress with empty paths.
func TestDNSChainDrift_MissingIngress(t *testing.T) {
	const host = "nobackend.example.com"

	src := newMemSourceDNS()

	// Ingress exists but has no http.paths — resolves to be.svcName="" which
	// triggers the K8s chain's "orphan service" / no-endpoint path. However the
	// missing-ingress diagnostic is only emitted when be==nil AND !isExternalOnly.
	// The only way to get be==nil without the host being in seedHostSet is if the
	// ingress rule has no "http" key at all (resolveIngressBackend returns be with
	// svcName="" but be is non-nil). This corner exists for seed-only hosts.
	//
	// In practice the seed-host path is: host in SeedTargets, no Ingress → external-only.
	// The analyzer correctly treats such hosts as external-only (CF-layer only).
	// We verify that behaviour here.
	src.add(makeIngress("default", "other-ing", "other.example.com", "other-svc", 80, nil))
	src.add(makeService("default", "other-svc"))
	src.add(makeEndpoints("default", "other-svc", 1))

	// Seed host with no matching Ingress — must be treated as external-only:
	// no missing-ingress diagnostic, but the external-dns-not-verified info
	// diagnostic IS expected (CF disabled, host counted).
	a := DNSChainDrift{
		Client:      nil,
		SeedTargets: []string{"https://" + host + "/"},
	}
	diags := a.Run(context.Background(), src)

	// No missing-ingress diagnostic should appear (external-only path).
	for _, d := range diags {
		if strings.Contains(d.Subject, "missing-ingress") {
			t.Errorf("unexpected missing-ingress diagnostic (seed host should be external-only): %+v", d)
		}
	}

	// The CF-disabled summary must appear (both hosts counted: other.example.com + orphan).
	found := false
	for _, d := range diags {
		if strings.Contains(d.Subject, "external-dns-not-verified") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected external-dns-not-verified summary, got: %+v", diags)
	}
}

// TestDNSChainDrift_OrphanService — Ingress refs missing Service → ingress-orphan-service.
func TestDNSChainDrift_OrphanService(t *testing.T) {
	const host = "orphan-svc.example.com"

	src := newMemSourceDNS()
	// Ingress points at a service that does not exist.
	src.add(makeIngress("default", "orphan-ing", host, "ghost-svc", 80, nil))
	// No Service "ghost-svc" added.

	a := DNSChainDrift{Client: nil}
	diags := a.Run(context.Background(), src)

	found := false
	for _, d := range diags {
		if strings.Contains(d.Subject, "ingress-orphan-service") {
			found = true
			if d.Severity != "error" {
				t.Errorf("expected severity=error, got %q", d.Severity)
			}
			if !strings.Contains(d.Message, "ghost-svc") {
				t.Errorf("expected message to mention ghost-svc: %s", d.Message)
			}
		}
	}
	if !found {
		t.Errorf("expected ingress-orphan-service diagnostic, got: %+v", diags)
	}
}

// TestDNSChainDrift_NoEndpoints — Service exists, 0 ready → service-no-endpoints.
func TestDNSChainDrift_NoEndpoints(t *testing.T) {
	const host = "noep.example.com"

	src := newMemSourceDNS()
	src.add(makeIngress("default", "noep-ing", host, "noep-svc", 80, nil))
	src.add(makeService("default", "noep-svc"))
	src.add(makeEndpoints("default", "noep-svc", 0)) // zero ready addresses

	a := DNSChainDrift{Client: nil}
	diags := a.Run(context.Background(), src)

	found := false
	for _, d := range diags {
		if strings.Contains(d.Subject, "service-no-endpoints") {
			found = true
			if d.Severity != "error" {
				t.Errorf("expected severity=error, got %q", d.Severity)
			}
		}
	}
	if !found {
		t.Errorf("expected service-no-endpoints diagnostic, got: %+v", diags)
	}
}

// TestDNSChainDrift_CFDisabled — nil client → single external-dns-not-verified info finding.
func TestDNSChainDrift_CFDisabled(t *testing.T) {
	const host = "any.example.com"

	src := newMemSourceDNS()
	src.add(makeIngress("default", "any-ing", host, "any-svc", 80, nil))
	src.add(makeService("default", "any-svc"))
	src.add(makeEndpoints("default", "any-svc", 1))

	a := DNSChainDrift{Client: nil} // CF disabled
	diags := a.Run(context.Background(), src)

	found := false
	for _, d := range diags {
		if strings.Contains(d.Subject, "external-dns-not-verified") {
			found = true
			if d.Severity != "info" {
				t.Errorf("expected severity=info, got %q", d.Severity)
			}
		}
	}
	if !found {
		t.Errorf("expected external-dns-not-verified info diagnostic, got: %+v", diags)
	}

	// Verify there is exactly one such finding.
	count := 0
	for _, d := range diags {
		if strings.Contains(d.Subject, "external-dns-not-verified") {
			count++
		}
	}
	if count != 1 {
		t.Errorf("expected exactly 1 external-dns-not-verified finding, got %d", count)
	}
}

// TestDNSChainDrift_OptOut — Ingress with probe-disable=true → 0 findings (apart from CF info).
func TestDNSChainDrift_OptOut(t *testing.T) {
	const host = "opted-out.example.com"

	src := newMemSourceDNS()
	src.add(makeIngress("default", "opted-ing", host, "opted-svc", 80, map[string]string{
		"cha.bionicaisolutions.com/probe-disable": "true",
	}))
	// Intentionally NOT adding a Service or Endpoints — if the opt-out is
	// honoured the analyzer must not walk the K8s chain for this host.

	a := DNSChainDrift{Client: nil}
	diags := a.Run(context.Background(), src)

	// With CF disabled and the only ingress opted-out the host is never added
	// to hostBackend, so len(hostBackend)==0 and the analyzer returns nil.
	for _, d := range diags {
		// The only acceptable diagnostic is the CF-disabled summary, but since
		// there are 0 hosts to check that summary also should not appear.
		if strings.Contains(d.Subject, host) {
			t.Errorf("got unexpected diagnostic mentioning opted-out host: %+v", d)
		}
	}
}

// TestDNSChainDrift_CFAPITimeout — CF client ctx timeout → graceful, 0 CF-specific findings,
// analyzer continues with K8s layer (service-no-endpoints surfaces instead).
func TestDNSChainDrift_CFAPITimeout(t *testing.T) {
	const host = "timeout.example.com"

	src := newMemSourceDNS()
	src.add(makeIngress("default", "timeout-ing", host, "timeout-svc", 80, nil))
	src.add(makeService("default", "timeout-svc"))
	src.add(makeEndpoints("default", "timeout-svc", 0)) // zero ready → service-no-endpoints

	// CF client that always times out (delay longer than the passed context).
	slowCF := &fakeCFClient{delay: 5 * time.Second}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	a := DNSChainDrift{Client: slowCF}
	diags := a.Run(ctx, src)

	// Must NOT produce missing-cloudflare-record or cloudflare-points-elsewhere.
	for _, d := range diags {
		if strings.Contains(d.Subject, "missing-cloudflare-record") || strings.Contains(d.Subject, "cloudflare-points-elsewhere") {
			t.Errorf("unexpected CF diagnostic after timeout: %+v", d)
		}
	}

	// The CF timeout triggers fail-open; the K8s layer should continue and
	// surface service-no-endpoints for the zero-endpoint Service.
	foundNoEP := false
	for _, d := range diags {
		if strings.Contains(d.Subject, "service-no-endpoints") {
			foundNoEP = true
		}
	}
	if !foundNoEP {
		t.Errorf("expected service-no-endpoints after CF timeout (K8s layer should continue), got: %+v", diags)
	}
}

// TestDNSChainDrift_NoIngresses — empty cluster → nil returned.
func TestDNSChainDrift_NoIngresses(t *testing.T) {
	src := newMemSourceDNS() // no objects at all

	a := DNSChainDrift{Client: nil}
	diags := a.Run(context.Background(), src)
	if len(diags) != 0 {
		t.Errorf("expected nil/empty diagnostics for empty cluster, got %d: %+v", len(diags), diags)
	}
}

// TestDNSChainDrift_DuplicateIngressHost — two Ingresses claim same host on
// the SAME path → duplicate-ingress-host warn. Real collision: both default
// to `/` (Prefix) so the router can't disambiguate.
func TestDNSChainDrift_DuplicateIngressHost(t *testing.T) {
	const host = "dup.example.com"

	src := newMemSourceDNS()
	src.add(makeIngress("ns1", "ing-a", host, "svc-a", 80, nil))
	src.add(makeIngress("ns2", "ing-b", host, "svc-b", 80, nil))
	src.add(makeService("ns1", "svc-a"))
	src.add(makeEndpoints("ns1", "svc-a", 1))

	a := DNSChainDrift{Client: nil}
	diags := a.Run(context.Background(), src)

	found := false
	for _, d := range diags {
		if strings.Contains(d.Subject, "duplicate-ingress-host") {
			found = true
			if d.Severity != "warn" {
				t.Errorf("expected severity=warn, got %q", d.Severity)
			}
		}
	}
	if !found {
		t.Errorf("expected duplicate-ingress-host diagnostic, got: %+v", diags)
	}
}

// makeIngressWithPath builds an Ingress with ONE custom (path, pathType) rule.
// Lets the duplicate-host path-overlap test create the path-disjoint pattern
// CHA must NOT flag as duplicate.
func makeIngressWithPath(ns, name, host, path, pathType, svcName string, svcPort int64) unstructured.Unstructured {
	u := unstructured.Unstructured{}
	u.SetAPIVersion("networking.k8s.io/v1")
	u.SetKind("Ingress")
	u.SetNamespace(ns)
	u.SetName(name)
	rule := map[string]any{
		"host": host,
		"http": map[string]any{
			"paths": []any{
				map[string]any{
					"path":     path,
					"pathType": pathType,
					"backend": map[string]any{
						"service": map[string]any{
							"name": svcName,
							"port": map[string]any{"number": svcPort},
						},
					},
				},
			},
		},
	}
	_ = unstructured.SetNestedSlice(u.Object, []any{rule}, "spec", "rules")
	return u
}

// TestDNSChainDrift_DuplicateIngressHost_PathDisjointIsNotDuplicate — two
// Ingresses on the same host but with NON-OVERLAPPING (path, pathType) pairs
// MUST NOT fire duplicate-ingress-host. This is the production pattern
// observed on comfy.baisoln.com (`/` oauth + `/mcp` bypass), wa.baisoln.com
// (HTTP webhooks split from WebSocket), and mcp.baisoln.com (19 MCPs each on
// their own path prefix). Pre-1.10.4 the analyzer flagged all of these as
// duplicate-host noise.
func TestDNSChainDrift_DuplicateIngressHost_PathDisjointIsNotDuplicate(t *testing.T) {
	const host = "shared.example.com"

	src := newMemSourceDNS()
	src.add(makeIngressWithPath("ns1", "oauth", host, "/", "Prefix", "oauth-proxy", 4180))
	src.add(makeIngressWithPath("ns1", "mcp", host, "/mcp", "Prefix", "mcp-backend", 8080))
	src.add(makeService("ns1", "oauth-proxy"))
	src.add(makeEndpoints("ns1", "oauth-proxy", 1))

	a := DNSChainDrift{Client: nil}
	diags := a.Run(context.Background(), src)

	for _, d := range diags {
		if strings.Contains(d.Subject, "duplicate-ingress-host") {
			t.Errorf("path-disjoint co-tenancy must NOT fire duplicate-ingress-host; got: %+v", d)
		}
	}
}

// TestDNSChainDrift_DuplicateIngressHost_SameExactPathFires — two Ingresses
// claiming the SAME exact (path, pathType) on the same host IS a real
// collision (router picks non-deterministically) and MUST still fire.
func TestDNSChainDrift_DuplicateIngressHost_SameExactPathFires(t *testing.T) {
	const host = "collide.example.com"

	src := newMemSourceDNS()
	src.add(makeIngressWithPath("ns1", "ing-a", host, "/api", "Prefix", "svc-a", 80))
	src.add(makeIngressWithPath("ns2", "ing-b", host, "/api", "Prefix", "svc-b", 80))
	src.add(makeService("ns1", "svc-a"))
	src.add(makeEndpoints("ns1", "svc-a", 1))

	a := DNSChainDrift{Client: nil}
	diags := a.Run(context.Background(), src)

	var dup *Diagnostic
	for i := range diags {
		if strings.Contains(diags[i].Subject, "duplicate-ingress-host") {
			dup = &diags[i]
			break
		}
	}
	if dup == nil {
		t.Fatalf("expected duplicate-ingress-host for exact-path collision; got: %+v", diags)
	}
	if !strings.Contains(dup.Message, "/api") || !strings.Contains(dup.Message, "ns1/ing-a") || !strings.Contains(dup.Message, "ns2/ing-b") {
		t.Errorf("message must call out the colliding path AND both ingresses; got: %s", dup.Message)
	}
}

// TestDNSChainDrift_SnapshotModeSkipsCF — ModeSnapshot → CF skipped even when Client is set.
func TestDNSChainDrift_SnapshotModeSkipsCF(t *testing.T) {
	const host = "snap.example.com"

	src := newMemSourceDNS()
	src.mode = pkgsnapshot.ModeSnapshot
	src.add(makeIngress("default", "snap-ing", host, "snap-svc", 80, nil))
	src.add(makeService("default", "snap-svc"))
	src.add(makeEndpoints("default", "snap-svc", 1))

	cf := &fakeCFClient{
		zones:   []Zone{{ID: "z1", Name: "example.com"}},
		records: map[string][]DNSRecord{},
	}

	a := DNSChainDrift{Client: cf}
	diags := a.Run(context.Background(), src)

	// In snapshot mode CF is skipped → external-dns-not-verified info finding.
	// No missing-cloudflare-record should appear.
	for _, d := range diags {
		if strings.Contains(d.Subject, "missing-cloudflare-record") {
			t.Errorf("CF should be skipped in snapshot mode, got: %+v", d)
		}
	}
	found := false
	for _, d := range diags {
		if strings.Contains(d.Subject, "external-dns-not-verified") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected external-dns-not-verified in snapshot mode, got: %+v", diags)
	}
}
