// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

package diagnose

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/srenix-ai/agentic-sre/internal/snapshot"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// certSrc is a minimal snapshot.Source for CertExpiry tests.
type certSrc struct {
	certs []map[string]any
}

func (s *certSrc) List(_ context.Context, gvr schema.GroupVersionResource, _ string) (*unstructured.UnstructuredList, error) {
	if gvr != snapshot.GVRCertificate {
		return &unstructured.UnstructuredList{}, nil
	}
	list := &unstructured.UnstructuredList{}
	for _, m := range s.certs {
		list.Items = append(list.Items, unstructured.Unstructured{Object: m})
	}
	return list, nil
}

func (s *certSrc) Get(_ context.Context, _ schema.GroupVersionResource, _, _ string) (*unstructured.Unstructured, error) {
	return nil, nil
}

func (s *certSrc) Mode() snapshot.Mode { return snapshot.ModeSnapshot }

func mustParseCert(t *testing.T, raw string) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	return m
}

func certJSON(ns, name, readyStatus, readyMsg, notAfter string) string {
	cond := ""
	if readyStatus != "" {
		cond = fmt.Sprintf(`[{"type":"Ready","status":%q,"message":%q}]`, readyStatus, readyMsg)
	} else {
		cond = `[]`
	}
	naField := ""
	if notAfter != "" {
		naField = fmt.Sprintf(`,"notAfter":%q`, notAfter)
	}
	return fmt.Sprintf(`{
		"metadata":{"namespace":%q,"name":%q},
		"status":{"conditions":%s%s}
	}`, ns, name, cond, naField)
}

func TestCertExpiry_NoCertManager(t *testing.T) {
	// Empty list — CRD not installed.
	src := &certSrc{}
	got := CertExpiry{}.Run(context.Background(), src)
	if len(got) != 0 {
		t.Errorf("expected no diagnostics when no certs, got %d", len(got))
	}
}

func TestCertExpiry_HealthyCert(t *testing.T) {
	future := time.Now().UTC().Add(60 * 24 * time.Hour).Format(time.RFC3339)
	src := &certSrc{
		certs: []map[string]any{
			mustParseCert(t, certJSON("infra", "wildcard", "True", "", future)),
		},
	}
	got := CertExpiry{}.Run(context.Background(), src)
	if len(got) != 0 {
		t.Errorf("healthy cert should produce no diagnostic, got %d", len(got))
	}
}

func TestCertExpiry_NotReady(t *testing.T) {
	src := &certSrc{
		certs: []map[string]any{
			mustParseCert(t, certJSON("infra", "api-tls", "False", "ACME challenge failed: DNS record not found", "")),
		},
	}
	got := CertExpiry{}.Run(context.Background(), src)
	if len(got) != 1 {
		t.Fatalf("not-Ready cert should produce 1 diagnostic, got %d", len(got))
	}
	if got[0].Subject != "Certificate/infra/api-tls" {
		t.Errorf("unexpected subject: %s", got[0].Subject)
	}
	if !strings.Contains(got[0].Message, "not Ready") {
		t.Errorf("message should say 'not Ready', got: %s", got[0].Message)
	}
	if !strings.Contains(got[0].Message, "ACME") {
		t.Errorf("message should include the condition message, got: %s", got[0].Message)
	}
}

func TestCertExpiry_Expired(t *testing.T) {
	past := time.Now().UTC().Add(-24 * time.Hour).Format(time.RFC3339)
	src := &certSrc{
		certs: []map[string]any{
			mustParseCert(t, certJSON("prod", "db-tls", "True", "", past)),
		},
	}
	got := CertExpiry{}.Run(context.Background(), src)
	if len(got) != 1 {
		t.Fatalf("expired cert should produce 1 diagnostic, got %d", len(got))
	}
	if !strings.Contains(got[0].Message, "EXPIRED") {
		t.Errorf("message should say EXPIRED, got: %s", got[0].Message)
	}
}

func TestCertExpiry_ExpiringWithinWindow(t *testing.T) {
	soon := time.Now().UTC().Add(5 * 24 * time.Hour).Format(time.RFC3339)
	src := &certSrc{
		certs: []map[string]any{
			mustParseCert(t, certJSON("prod", "ingress-tls", "True", "", soon)),
		},
	}
	got := CertExpiry{WarnWindow: 14 * 24 * time.Hour}.Run(context.Background(), src)
	if len(got) != 1 {
		t.Fatalf("cert expiring in 5d (window 14d) should produce 1 diagnostic, got %d", len(got))
	}
	if !strings.Contains(got[0].Message, "day") {
		t.Errorf("message should mention days remaining, got: %s", got[0].Message)
	}
}

func TestCertExpiry_ExpiringOutsideWindow(t *testing.T) {
	later := time.Now().UTC().Add(30 * 24 * time.Hour).Format(time.RFC3339)
	src := &certSrc{
		certs: []map[string]any{
			mustParseCert(t, certJSON("prod", "ingress-tls", "True", "", later)),
		},
	}
	got := CertExpiry{WarnWindow: 14 * 24 * time.Hour}.Run(context.Background(), src)
	if len(got) != 0 {
		t.Errorf("cert expiring in 30d (window 14d) should not trigger, got %d", len(got))
	}
}

func TestCertExpiry_NotReadyTakesPriorityOverExpiry(t *testing.T) {
	// Even if notAfter is past, the not-Ready condition produces the diagnostic
	// and we don't emit a second one for expiry on the same cert.
	past := time.Now().UTC().Add(-24 * time.Hour).Format(time.RFC3339)
	src := &certSrc{
		certs: []map[string]any{
			mustParseCert(t, certJSON("ops", "old-cert", "False", "renewal failed", past)),
		},
	}
	got := CertExpiry{}.Run(context.Background(), src)
	if len(got) != 1 {
		t.Fatalf("expected exactly 1 diagnostic (not-Ready wins), got %d", len(got))
	}
	if !strings.Contains(got[0].Message, "not Ready") {
		t.Errorf("not-Ready message should take priority, got: %s", got[0].Message)
	}
}

func TestCertExpiry_MultipleCerts(t *testing.T) {
	future := time.Now().UTC().Add(60 * 24 * time.Hour).Format(time.RFC3339)
	past := time.Now().UTC().Add(-2 * 24 * time.Hour).Format(time.RFC3339)
	soon := time.Now().UTC().Add(3 * 24 * time.Hour).Format(time.RFC3339)
	src := &certSrc{
		certs: []map[string]any{
			mustParseCert(t, certJSON("ns1", "ok-cert", "True", "", future)),
			mustParseCert(t, certJSON("ns2", "expired-cert", "True", "", past)),
			mustParseCert(t, certJSON("ns3", "soon-cert", "True", "", soon)),
			mustParseCert(t, certJSON("ns4", "broken-cert", "False", "issuer error", "")),
		},
	}
	got := CertExpiry{WarnWindow: 14 * 24 * time.Hour}.Run(context.Background(), src)
	if len(got) != 3 {
		t.Fatalf("expected 3 diagnostics (expired+soon+notReady), got %d: %v", len(got), got)
	}
}
