// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

package diagnose

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/srenix-ai/agentic-sre/internal/snapshot"
)

// makeCertPEM returns a PEM-encoded x509 cert with the given notAfter.
func makeCertPEM(t *testing.T, host string, notAfter time.Time) string {
	t.Helper()
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("genkey: %v", err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: host},
		DNSNames:     []string{host},
		NotBefore:    notAfter.Add(-90 * 24 * time.Hour),
		NotAfter:     notAfter,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &priv.PublicKey, priv)
	if err != nil {
		t.Fatalf("create cert: %v", err)
	}
	pemBytes := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	// Secrets in snapshot/Live carry the value as base64-encoded; mirror that.
	return base64.StdEncoding.EncodeToString(pemBytes)
}

func TestTLSSecretMismatch_Detects(t *testing.T) {
	dir := t.TempDir()

	staleCert := makeCertPEM(t, "pg.example.com", time.Now().Add(-30*24*time.Hour))
	freshCert := makeCertPEM(t, "pg.example.com", time.Now().Add(60*24*time.Hour))

	files := map[string]string{
		"ingresses.json": `{
  "apiVersion":"networking.k8s.io/v1","kind":"IngressList","items":[{
    "apiVersion":"networking.k8s.io/v1","kind":"Ingress",
    "metadata":{"name":"pg-ing","namespace":"pg"},
    "spec":{"tls":[{"hosts":["pg.example.com"],"secretName":"pg-secret-old"}]}
  }]
}`,
		"secrets.json": fmt.Sprintf(`{
  "apiVersion":"v1","kind":"SecretList","items":[
    {
      "apiVersion":"v1","kind":"Secret","type":"kubernetes.io/tls",
      "metadata":{"name":"pg-secret-old","namespace":"pg"},
      "data":{"tls.crt":%q}
    },
    {
      "apiVersion":"v1","kind":"Secret","type":"kubernetes.io/tls",
      "metadata":{"name":"pg-tls","namespace":"pg"},
      "data":{"tls.crt":%q}
    }
  ]
}`, staleCert, freshCert),
		"certificates.json": `{
  "apiVersion":"cert-manager.io/v1","kind":"CertificateList","items":[{
    "apiVersion":"cert-manager.io/v1","kind":"Certificate",
    "metadata":{"name":"pg-cm","namespace":"pg"},
    "spec":{"secretName":"pg-tls","dnsNames":["pg.example.com"]},
    "status":{"conditions":[{"type":"Ready","status":"True"}]}
  }]
}`,
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	src, err := snapshot.LoadFile(dir)
	if err != nil {
		t.Fatalf("LoadFile: %v", err)
	}

	got := TLSSecretMismatch{}.Run(context.Background(), src)
	if len(got) != 1 {
		t.Fatalf("expected 1 diagnostic, got %d (%+v)", len(got), got)
	}
	d := got[0]
	if !strings.Contains(d.Subject, "tls-secret-mismatch/pg/pg-ing/0") {
		t.Errorf("subject mismatch: %s", d.Subject)
	}
	if !strings.Contains(d.Message, "pg-secret-old") || !strings.Contains(d.Message, "pg-tls") {
		t.Errorf("message should name both secrets:\n%s", d.Message)
	}
	if !strings.Contains(d.Remediation, "kubectl -n pg patch ingress pg-ing") {
		t.Errorf("remediation should include the precise patch command:\n%s", d.Remediation)
	}
}

func TestTLSSecretMismatch_FreshCert_NoDiagnostic(t *testing.T) {
	dir := t.TempDir()
	freshCert := makeCertPEM(t, "api.example.com", time.Now().Add(60*24*time.Hour))
	files := map[string]string{
		"ingresses.json": `{
  "apiVersion":"networking.k8s.io/v1","kind":"IngressList","items":[{
    "apiVersion":"networking.k8s.io/v1","kind":"Ingress",
    "metadata":{"name":"api","namespace":"api"},
    "spec":{"tls":[{"hosts":["api.example.com"],"secretName":"api-secret"}]}
  }]
}`,
		"secrets.json": fmt.Sprintf(`{
  "apiVersion":"v1","kind":"SecretList","items":[{
    "apiVersion":"v1","kind":"Secret","type":"kubernetes.io/tls",
    "metadata":{"name":"api-secret","namespace":"api"},
    "data":{"tls.crt":%q}
  }]
}`, freshCert),
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	src, err := snapshot.LoadFile(dir)
	if err != nil {
		t.Fatalf("LoadFile: %v", err)
	}
	got := TLSSecretMismatch{}.Run(context.Background(), src)
	if len(got) != 0 {
		t.Errorf("expected 0 diagnostics for fresh cert, got %d: %+v", len(got), got)
	}
}

func TestTLSSecretMismatch_NoBetterCert_NoDiagnostic(t *testing.T) {
	// Stale Secret exists, but there's no Certificate in the same namespace
	// pointing at a different name — the CertExpiry analyzer should surface
	// this, not us.
	dir := t.TempDir()
	staleCert := makeCertPEM(t, "old.example.com", time.Now().Add(-30*24*time.Hour))
	files := map[string]string{
		"ingresses.json": `{
  "apiVersion":"networking.k8s.io/v1","kind":"IngressList","items":[{
    "apiVersion":"networking.k8s.io/v1","kind":"Ingress",
    "metadata":{"name":"old","namespace":"old"},
    "spec":{"tls":[{"hosts":["old.example.com"],"secretName":"old-secret"}]}
  }]
}`,
		"secrets.json": fmt.Sprintf(`{
  "apiVersion":"v1","kind":"SecretList","items":[{
    "apiVersion":"v1","kind":"Secret","type":"kubernetes.io/tls",
    "metadata":{"name":"old-secret","namespace":"old"},
    "data":{"tls.crt":%q}
  }]
}`, staleCert),
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	src, err := snapshot.LoadFile(dir)
	if err != nil {
		t.Fatalf("LoadFile: %v", err)
	}
	got := TLSSecretMismatch{}.Run(context.Background(), src)
	if len(got) != 0 {
		t.Errorf("expected 0 diagnostics when no candidate Cert exists, got %d", len(got))
	}
}
