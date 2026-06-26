// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

package report

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"

	"github.com/srenix-ai/agentic-sre/internal/snapshot"
	"github.com/srenix-ai/agentic-sre/pkg/ticketing"
)

// tixSource returns a fixed UnstructuredList for any List call. Other
// Source methods panic — RouteTickets only calls List.
type tixSource struct {
	items []unstructured.Unstructured
}

func (f *tixSource) List(_ context.Context, _ schema.GroupVersionResource, _ string) (*unstructured.UnstructuredList, error) {
	return &unstructured.UnstructuredList{Items: f.items}, nil
}

func (f *tixSource) Get(_ context.Context, _ schema.GroupVersionResource, _, _ string) (*unstructured.Unstructured, error) {
	panic("Get not used by RouteTickets")
}

func (f *tixSource) Mode() snapshot.Mode { return snapshot.ModeLive }

// tixMutator captures Patch calls so tests can assert ref persistence.
// Other methods are no-ops.
type tixMutator struct {
	patches []patchCall
}

type patchCall struct {
	gvr   schema.GroupVersionResource
	name  string
	body  []byte
	pType types.PatchType
	sub   string // "" for main resource, "status" for /status subresource
}

func (m *tixMutator) Patch(_ context.Context, gvr schema.GroupVersionResource, _ string, name string, p types.PatchType, body []byte) error {
	m.patches = append(m.patches, patchCall{gvr: gvr, name: name, body: body, pType: p, sub: ""})
	return nil
}
func (m *tixMutator) PatchStatus(_ context.Context, gvr schema.GroupVersionResource, _ string, name string, p types.PatchType, body []byte) error {
	m.patches = append(m.patches, patchCall{gvr: gvr, name: name, body: body, pType: p, sub: "status"})
	return nil
}
func (m *tixMutator) Delete(_ context.Context, _ schema.GroupVersionResource, _, _ string) error {
	return nil
}
func (m *tixMutator) Create(_ context.Context, _ schema.GroupVersionResource, _ string, _ *unstructured.Unstructured) error {
	return nil
}

// recordingSink captures all calls to the ticketing.Sink methods.
type recordingSink struct {
	upserts []ticketing.Ticket
	resolve []ticketing.TicketRef
	comment []commentCall
	ref     ticketing.TicketRef
	err     error
}

type commentCall struct {
	ref  ticketing.TicketRef
	body string
}

func (r *recordingSink) Provider() string { return "openproject" }

func (r *recordingSink) Upsert(_ context.Context, t ticketing.Ticket) (ticketing.TicketRef, error) {
	r.upserts = append(r.upserts, t)
	if r.err != nil {
		return ticketing.TicketRef{}, r.err
	}
	return r.ref, nil
}

func (r *recordingSink) Resolve(_ context.Context, ref ticketing.TicketRef, _ string) error {
	r.resolve = append(r.resolve, ref)
	return nil
}

func (r *recordingSink) Comment(_ context.Context, ref ticketing.TicketRef, body string) error {
	r.comment = append(r.comment, commentCall{ref: ref, body: body})
	return nil
}

func makeDriftReport(subject, name string, ticket map[string]any) unstructured.Unstructured {
	obj := map[string]any{
		"apiVersion": "srenix.ai/v1alpha1",
		"kind":       "DriftReport",
		"metadata": map[string]any{
			"name": name,
		},
		"spec": map[string]any{
			"subject": subject,
		},
	}
	if ticket != nil {
		obj["status"] = map[string]any{"ticket": ticket}
	}
	return unstructured.Unstructured{Object: obj}
}

func TestTicketFromDeltaCarriesAllFields(t *testing.T) {
	delta := DeltaDiag{
		Subject:       "Secret/mcp/openproject-secrets/openproject-url",
		Severity:      ticketing.SeverityCritical,
		Message:       "key missing",
		Remediation:   "restore via Vault",
		Investigation: "ESO refresh hadn't run since 09:00 UTC",
	}
	cfg := TicketingConfig{
		Cluster: "gpu-cluster",
		Labels:  []string{"srenix", "auto-filed"},
	}
	tk := ticketFromDelta(delta, cfg, "run-42")

	if tk.Fingerprint != ticketing.Fingerprint(delta.Subject, "gpu-cluster") {
		t.Errorf("fingerprint mismatch: %q", tk.Fingerprint)
	}
	if tk.Severity != ticketing.SeverityCritical {
		t.Errorf("severity=%q want critical", tk.Severity)
	}
	if !strings.Contains(tk.Body, "key missing") {
		t.Errorf("body missing diagnostic message: %q", tk.Body)
	}
	if !strings.Contains(tk.Body, "restore via Vault") {
		t.Errorf("body missing remediation: %q", tk.Body)
	}
	if !strings.Contains(tk.Body, "ESO refresh") {
		t.Errorf("body missing investigation: %q", tk.Body)
	}
	if !strings.Contains(tk.Body, "gpu-cluster") || !strings.Contains(tk.Body, "run-42") {
		t.Errorf("body missing cluster/runID footer: %q", tk.Body)
	}
	if len(tk.Labels) != 2 || tk.Labels[0] != "srenix" {
		t.Errorf("labels=%v want config labels propagated", tk.Labels)
	}
}

func TestReadTicketRefRoundTrip(t *testing.T) {
	cr := makeDriftReport("Pod/default/x", "drift-x", map[string]any{
		"provider": "openproject",
		"key":      "WP-1287",
		"url":      "https://op.example/wp/1287",
	})
	ref, ok := readTicketRef(&cr)
	if !ok {
		t.Fatal("expected ticket ref to be readable")
	}
	if ref.Provider != "openproject" || ref.Key != "WP-1287" || ref.URL != "https://op.example/wp/1287" {
		t.Errorf("ref=%+v want openproject/WP-1287/url", ref)
	}
}

func TestReadTicketRefAbsent(t *testing.T) {
	cr := makeDriftReport("Pod/default/x", "drift-x", nil)
	if _, ok := readTicketRef(&cr); ok {
		t.Fatal("expected absent ticket ref to return ok=false")
	}
}

func TestRouteTicketsUpsertsForUnfixableWithoutExistingRef(t *testing.T) {
	src := &tixSource{items: []unstructured.Unstructured{
		makeDriftReport("Pod/default/broken", "drift-broken", nil),
	}}
	mut := &tixMutator{}
	sink := &recordingSink{ref: ticketing.TicketRef{Provider: "openproject", Key: "WP-1", URL: "u"}}

	RouteTickets(
		context.Background(),
		TicketingConfig{Sink: sink, Cluster: "gpu", Labels: []string{"srenix"}},
		src,
		mut,
		map[string]bool{"Pod/default/broken": true},
		[]DeltaDiag{{Subject: "Pod/default/broken", Severity: "critical", Message: "down"}},
		"run-1",
	)

	if len(sink.upserts) != 1 {
		t.Fatalf("upserts=%d want 1", len(sink.upserts))
	}
	if sink.upserts[0].Subject != "Pod/default/broken" {
		t.Errorf("upsert subject=%q", sink.upserts[0].Subject)
	}
	if len(mut.patches) != 1 {
		t.Fatalf("patches=%d want 1", len(mut.patches))
	}
	if mut.patches[0].name != "drift-broken" {
		t.Errorf("patch target=%q want drift-broken", mut.patches[0].name)
	}
	if mut.patches[0].sub != "status" {
		t.Errorf("patch must target /status subresource (CRD declares subresources.status:{}), got sub=%q", mut.patches[0].sub)
	}
	// Patch body should contain provider+key.
	var body map[string]any
	if err := json.Unmarshal(mut.patches[0].body, &body); err != nil {
		t.Fatalf("unmarshal patch body: %v", err)
	}
	status, _ := body["status"].(map[string]any)
	ticket, _ := status["ticket"].(map[string]any)
	if ticket["key"] != "WP-1" || ticket["provider"] != "openproject" {
		t.Errorf("patch ticket=%v want key WP-1 provider openproject", ticket)
	}
}

func TestRouteTicketsSkipsFixedSubjects(t *testing.T) {
	src := &tixSource{items: []unstructured.Unstructured{
		makeDriftReport("Pod/default/transient", "drift-transient", nil),
	}}
	mut := &tixMutator{}
	sink := &recordingSink{ref: ticketing.TicketRef{Key: "WP-9"}}

	// postFixSubjects empty → all diagnostics were fixed this cycle.
	RouteTickets(
		context.Background(),
		TicketingConfig{Sink: sink, Cluster: "gpu"},
		src,
		mut,
		map[string]bool{}, // none unfixable
		[]DeltaDiag{{Subject: "Pod/default/transient", Severity: "warning"}},
		"run-2",
	)

	if len(sink.upserts) != 0 {
		t.Errorf("expected no upserts for fixed subjects, got %d", len(sink.upserts))
	}
}

func TestRouteTicketsSkipsAlreadyTicketed(t *testing.T) {
	src := &tixSource{items: []unstructured.Unstructured{
		makeDriftReport("Pod/default/old", "drift-old", map[string]any{
			"provider": "openproject",
			"key":      "WP-42",
			"url":      "u",
		}),
	}}
	mut := &tixMutator{}
	sink := &recordingSink{ref: ticketing.TicketRef{Key: "WP-NEW"}}

	RouteTickets(
		context.Background(),
		TicketingConfig{Sink: sink, Cluster: "gpu"},
		src,
		mut,
		map[string]bool{"Pod/default/old": true},
		[]DeltaDiag{{Subject: "Pod/default/old", Severity: "critical"}},
		"run-3",
	)

	if len(sink.upserts) != 0 {
		t.Errorf("expected no upsert for already-ticketed subject, got %d", len(sink.upserts))
	}
	if len(mut.patches) != 0 {
		t.Errorf("expected no patch for already-ticketed subject, got %d", len(mut.patches))
	}
}

func TestRouteTicketsNoOpWithNilSink(t *testing.T) {
	src := &tixSource{}
	mut := &tixMutator{}
	RouteTickets(
		context.Background(),
		TicketingConfig{}, // Sink: nil
		src,
		mut,
		map[string]bool{"x": true},
		[]DeltaDiag{{Subject: "x"}},
		"run-4",
	)
	if len(mut.patches) != 0 {
		t.Error("nil sink should be a complete no-op")
	}
}

func TestRouteTicketsHandlesUpsertError(t *testing.T) {
	src := &tixSource{items: []unstructured.Unstructured{
		makeDriftReport("Pod/default/sad", "drift-sad", nil),
	}}
	mut := &tixMutator{}
	sink := &recordingSink{err: context.DeadlineExceeded}

	// Must not panic, must not patch when upsert fails.
	RouteTickets(
		context.Background(),
		TicketingConfig{Sink: sink, Cluster: "gpu"},
		src,
		mut,
		map[string]bool{"Pod/default/sad": true},
		[]DeltaDiag{{Subject: "Pod/default/sad", Severity: "critical"}},
		"run-5",
	)
	if len(mut.patches) != 0 {
		t.Errorf("expected no patch on upsert failure, got %d", len(mut.patches))
	}
}

// ensure our fake satisfies the Source/Mutator interfaces at compile time
var _ snapshot.Source = (*tixSource)(nil)
var _ snapshot.Mutator = (*tixMutator)(nil)

func TestRouteTickets_BelowFloorNotTicketed(t *testing.T) {
	// Default floor = critical. A warning finding must NOT open a ticket.
	src := &tixSource{items: []unstructured.Unstructured{
		makeDriftReport("Namespace/cluster/x/missing-network-policy", "drift-w", nil),
	}}
	mut := &tixMutator{}
	sink := &recordingSink{ref: ticketing.TicketRef{Key: "WP-1"}}
	RouteTickets(context.Background(),
		TicketingConfig{Sink: sink, Cluster: "gpu"}, // MinSeverity empty → critical
		src, mut,
		map[string]bool{"Namespace/cluster/x/missing-network-policy": true},
		[]DeltaDiag{{Subject: "Namespace/cluster/x/missing-network-policy", Severity: "warning"}},
		"run-1",
	)
	if len(sink.upserts) != 0 {
		t.Fatalf("warning finding must not be ticketed at critical floor; upserts=%d", len(sink.upserts))
	}
}

func TestRouteTickets_WidenedFloorTicketsWarning(t *testing.T) {
	src := &tixSource{items: []unstructured.Unstructured{
		makeDriftReport("Pod/default/warn", "drift-warn", nil),
	}}
	mut := &tixMutator{}
	sink := &recordingSink{ref: ticketing.TicketRef{Provider: "openproject", Key: "WP-2"}}
	RouteTickets(context.Background(),
		TicketingConfig{Sink: sink, Cluster: "gpu", MinSeverity: "warning"},
		src, mut,
		map[string]bool{"Pod/default/warn": true},
		[]DeltaDiag{{Subject: "Pod/default/warn", Severity: "warning"}},
		"run-2",
	)
	if len(sink.upserts) != 1 {
		t.Fatalf("warning floor should ticket a warning; upserts=%d", len(sink.upserts))
	}
}

func TestRouteTicketCleanup_ClosesBelowFloorTickets(t *testing.T) {
	// A warning finding that already has an OPEN ticket (filed under an old
	// loose policy) must be auto-closed when the floor is critical.
	dr := makeDriftReport("Namespace/cluster/x/missing-network-policy", "drift-old",
		map[string]any{"provider": "openproject", "key": "WP-7", "resolved": false})
	_ = unstructured.SetNestedField(dr.Object, "warning", "spec", "severity")
	// A critical finding with an open ticket must be LEFT open.
	dr2 := makeDriftReport("Pod/default/crit", "drift-crit",
		map[string]any{"provider": "openproject", "key": "WP-8", "resolved": false})
	_ = unstructured.SetNestedField(dr2.Object, "critical", "spec", "severity")

	src := &tixSource{items: []unstructured.Unstructured{dr, dr2}}
	mut := &tixMutator{}
	sink := &recordingSink{}
	RouteTicketCleanup(context.Background(),
		TicketingConfig{Sink: sink, Cluster: "gpu", ResolveOnClear: true}, // critical floor
		src, mut, fixedNowTix(),
	)
	if len(sink.resolve) != 1 || sink.resolve[0].Key != "WP-7" {
		t.Fatalf("expected only WP-7 (warning) closed; got %+v", sink.resolve)
	}
}

func fixedNowTix() time.Time { return time.Date(2026, 6, 24, 0, 0, 0, 0, time.UTC) }
