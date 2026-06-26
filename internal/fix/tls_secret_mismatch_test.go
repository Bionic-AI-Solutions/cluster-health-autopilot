// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

package fix

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
	"strings"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
)

// makeCertB64 returns a base64(PEM) self-signed x509 cert with the given
// notAfter, mirroring how Secrets carry data values in snapshots. Generated
// in-code — no binary fixtures, no real keys.
func makeCertB64(t *testing.T, host string, notAfter time.Time) string {
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
	return base64.StdEncoding.EncodeToString(pemBytes)
}

// patchRecorder wraps fakeMutator and additionally captures the EXACT
// patch bytes per call, so tests can assert the JSONPatch produced
// against Ingress.spec.tls byte-for-byte.
type patchRecorder struct {
	*fakeMutator
	patches [][]byte
}

func newPatchRecorder() *patchRecorder { return &patchRecorder{fakeMutator: newFakeMutator()} }

func (p *patchRecorder) Patch(ctx context.Context, gvr schema.GroupVersionResource, ns, name string, pt types.PatchType, patch []byte) error {
	p.patches = append(p.patches, append([]byte(nil), patch...))
	return p.fakeMutator.Patch(ctx, gvr, ns, name, pt, patch)
}

// ingressJSON renders a one-Ingress list with arbitrary tls blocks.
func ingressJSON(ns, name, tlsBlocks string) string {
	return fmt.Sprintf(`{
  "apiVersion":"networking.k8s.io/v1","kind":"IngressList","items":[{
    "apiVersion":"networking.k8s.io/v1","kind":"Ingress",
    "metadata":{"name":%q,"namespace":%q},
    "spec":{"tls":[%s]}
  }]
}`, name, ns, tlsBlocks)
}

func tlsSecretsJSON(ns string, certs map[string]string) string {
	var items []string
	for name, crt := range certs {
		items = append(items, fmt.Sprintf(`{
      "apiVersion":"v1","kind":"Secret","type":"kubernetes.io/tls",
      "metadata":{"name":%q,"namespace":%q},
      "data":{"tls.crt":%q}
    }`, name, ns, crt))
	}
	return `{"apiVersion":"v1","kind":"SecretList","items":[` + strings.Join(items, ",") + `]}`
}

func certificateJSON(ns, name, secretName, host, readyStatus string) string {
	return fmt.Sprintf(`{
  "apiVersion":"cert-manager.io/v1","kind":"CertificateList","items":[{
    "apiVersion":"cert-manager.io/v1","kind":"Certificate",
    "metadata":{"name":%q,"namespace":%q},
    "spec":{"secretName":%q,"dnsNames":[%q]},
    "status":{"conditions":[{"type":"Ready","status":%q}]}
  }]
}`, name, ns, secretName, host, readyStatus)
}

// TestTLSSecretMismatchFix is the fixture-driven table over the four
// canonical states. Each case asserts the EXACT JSONPatch bytes emitted
// (or that none were).
func TestTLSSecretMismatchFix(t *testing.T) {
	const host = "pg.example.com"
	expired := makeCertB64(t, host, time.Now().Add(-30*24*time.Hour))
	expiringSoon := makeCertB64(t, host, time.Now().Add(3*24*time.Hour)) // inside 14d window
	fresh := makeCertB64(t, host, time.Now().Add(60*24*time.Hour))

	cases := []struct {
		name  string
		files map[string]string

		wantPatches []string // exact JSONPatch bytes, in order
		wantActions int
	}{
		{
			// Hosts mismatch: Ingress pins the stale Secret while a
			// Ready cert-manager Certificate for the SAME host targets
			// a different Secret — repoint tls[0].
			name: "hosts mismatch with expired secret patches tls[0]",
			files: map[string]string{
				"ingresses.json": ingressJSON("pg", "pg-ing",
					`{"hosts":["pg.example.com"],"secretName":"pg-secret-old"}`),
				"secrets.json": tlsSecretsJSON("pg", map[string]string{
					"pg-secret-old": expired,
					"pg-tls":        fresh,
				}),
				"certificates.json": certificateJSON("pg", "pg-cm", "pg-tls", host, "True"),
			},
			wantPatches: []string{
				`[{"op":"replace","path":"/spec/tls/0/secretName","value":"pg-tls"}]`,
			},
			wantActions: 1,
		},
		{
			// Same mismatch but the stale block is tls[1] — the patch
			// path must carry the right index.
			name: "expiring-soon secret in second tls block patches tls[1]",
			files: map[string]string{
				"ingresses.json": ingressJSON("pg", "pg-ing",
					`{"hosts":["other.example.com"],"secretName":"other-tls"},
					 {"hosts":["pg.example.com"],"secretName":"pg-secret-old"}`),
				"secrets.json": tlsSecretsJSON("pg", map[string]string{
					"other-tls":     fresh, // healthy → tls[0] is skipped
					"pg-secret-old": expiringSoon,
					"pg-tls":        fresh,
				}),
				"certificates.json": certificateJSON("pg", "pg-cm", "pg-tls", host, "True"),
			},
			wantPatches: []string{
				`[{"op":"replace","path":"/spec/tls/1/secretName","value":"pg-tls"}]`,
			},
			wantActions: 1,
		},
		{
			// Candidate Certificate exists but is not Ready — repointing
			// would swap a stale cert for a nonexistent/incomplete one.
			name: "cert not ready emits no patch",
			files: map[string]string{
				"ingresses.json": ingressJSON("pg", "pg-ing",
					`{"hosts":["pg.example.com"],"secretName":"pg-secret-old"}`),
				"secrets.json": tlsSecretsJSON("pg", map[string]string{
					"pg-secret-old": expired,
				}),
				"certificates.json": certificateJSON("pg", "pg-cm", "pg-tls", host, "False"),
			},
			wantPatches: nil,
			wantActions: 0,
		},
		{
			// Healthy no-op: the pinned Secret's cert is fresh, nothing
			// to do even though a Ready Certificate exists elsewhere.
			name: "healthy fresh cert is a no-op",
			files: map[string]string{
				"ingresses.json": ingressJSON("pg", "pg-ing",
					`{"hosts":["pg.example.com"],"secretName":"pg-tls-current"}`),
				"secrets.json": tlsSecretsJSON("pg", map[string]string{
					"pg-tls-current": fresh,
					"pg-tls-next":    fresh,
				}),
				"certificates.json": certificateJSON("pg", "pg-cm", "pg-tls-next", host, "True"),
			},
			wantPatches: nil,
			wantActions: 0,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			src := loadSrc(t, tc.files)
			m := newPatchRecorder()
			r := TLSSecretMismatch{}.Run(context.Background(), src, m)

			if r.Refused != "" {
				t.Fatalf("unexpected Refused: %q", r.Refused)
			}
			if got, want := len(m.patches), len(tc.wantPatches); got != want {
				t.Fatalf("patch calls = %d, want %d; calls: %v", got, want, m.calls)
			}
			for i, want := range tc.wantPatches {
				if got := string(m.patches[i]); got != want {
					t.Errorf("patch[%d] bytes:\n got %s\nwant %s", i, got, want)
				}
			}
			if len(tc.wantPatches) > 0 {
				// Patch must target the Ingress GVR with a JSON patch.
				wantCall := "Patch ingresses/pg/pg-ing [" + string(types.JSONPatchType) + "]"
				if m.calls[0] != wantCall {
					t.Errorf("mutator call = %q, want %q", m.calls[0], wantCall)
				}
			}
			if got := len(r.Actions); got != tc.wantActions {
				t.Errorf("Actions = %d, want %d (%+v)", got, tc.wantActions, r.Actions)
			}
		})
	}
}

// TestTLSSecretMismatchFix_RefusesOnSnapshot — nil Mutator → Refused, no I/O.
func TestTLSSecretMismatchFix_RefusesOnSnapshot(t *testing.T) {
	src := loadSrc(t, map[string]string{
		"ingresses.json": ingressJSON("pg", "pg-ing",
			`{"hosts":["pg.example.com"],"secretName":"s"}`),
	})
	r := TLSSecretMismatch{}.Run(context.Background(), src, nil)
	if r.Refused == "" {
		t.Error("expected Refused when Mutator is nil")
	}
	if got := (TLSSecretMismatch{}).Name(); got != "TLSSecretMismatch" {
		t.Errorf("Name() = %q", got)
	}
	if len(r.Actions) != 0 {
		t.Errorf("expected no actions when refused, got %d", len(r.Actions))
	}
}

// TestTLSSecretMismatchFix_SkipsGitOpsAndProtected — safety contract:
// GitOps-managed Ingresses and protected namespaces are skipped with a
// recorded reason, never patched.
func TestTLSSecretMismatchFix_SkipsGitOpsAndProtected(t *testing.T) {
	const host = "argo.example.com"
	expired := makeCertB64(t, host, time.Now().Add(-time.Hour))

	src := loadSrc(t, map[string]string{
		"ingresses.json": fmt.Sprintf(`{
  "apiVersion":"networking.k8s.io/v1","kind":"IngressList","items":[
    {
      "apiVersion":"networking.k8s.io/v1","kind":"Ingress",
      "metadata":{"name":"argo-ing","namespace":"apps",
        "annotations":{"argocd.argoproj.io/tracking-id":"apps:networking.k8s.io/Ingress:apps/argo-ing"}},
      "spec":{"tls":[{"hosts":[%q],"secretName":"stale"}]}
    },
    {
      "apiVersion":"networking.k8s.io/v1","kind":"Ingress",
      "metadata":{"name":"sys-ing","namespace":"kube-system"},
      "spec":{"tls":[{"hosts":[%q],"secretName":"stale"}]}
    }
  ]
}`, host, host),
		"secrets.json":      tlsSecretsJSON("apps", map[string]string{"stale": expired}),
		"certificates.json": certificateJSON("apps", "cm", "better", host, "True"),
	})

	m := newPatchRecorder()
	r := TLSSecretMismatch{}.Run(context.Background(), src, m)
	if len(m.patches) != 0 {
		t.Fatalf("must not patch GitOps-managed or protected Ingresses; got %v", m.calls)
	}
	if len(r.Skipped) != 2 {
		t.Fatalf("Skipped = %d, want 2 (%+v)", len(r.Skipped), r.Skipped)
	}
}

// ---- helper-level tests ---------------------------------------------------

func TestTLSHostsFix(t *testing.T) {
	cases := []struct {
		name string
		in   map[string]any
		want []string
	}{
		{"lowercases and drops empties", map[string]any{"hosts": []any{"PG.Example.Com", "", "api.example.com"}}, []string{"pg.example.com", "api.example.com"}},
		{"missing hosts", map[string]any{}, nil},
		{"wrong type", map[string]any{"hosts": "pg.example.com"}, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := tlsHostsFix(tc.in)
			if len(got) != len(tc.want) {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Errorf("got %v, want %v", got, tc.want)
				}
			}
		})
	}
}

func TestSecretCertExpiryFix_RawPEM(t *testing.T) {
	// snapshot.File keeps the base64 form; Live decodes to raw PEM. The
	// parser must accept BOTH. Decode the b64 helper output to exercise
	// the raw-PEM branch.
	notAfter := time.Now().Add(48 * time.Hour).Truncate(time.Second)
	b64 := makeCertB64(t, "raw.example.com", notAfter)
	rawPEM, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		t.Fatal(err)
	}
	src := loadSrc(t, map[string]string{
		"secrets.json": fmt.Sprintf(`{
  "apiVersion":"v1","kind":"SecretList","items":[{
    "apiVersion":"v1","kind":"Secret","type":"kubernetes.io/tls",
    "metadata":{"name":"raw","namespace":"ns"},
    "data":{"tls.crt":%q}
  }]
}`, string(rawPEM)),
	})
	got, ok := secretCertExpiryFix(context.Background(), src, "ns", "raw")
	if !ok {
		t.Fatal("expected raw-PEM tls.crt to parse")
	}
	if !got.Equal(notAfter) {
		t.Errorf("NotAfter = %v, want %v", got, notAfter)
	}
}

func TestSecretCertExpiryFix_MissingOrGarbage(t *testing.T) {
	src := loadSrc(t, map[string]string{
		"secrets.json": `{
  "apiVersion":"v1","kind":"SecretList","items":[{
    "apiVersion":"v1","kind":"Secret",
    "metadata":{"name":"junk","namespace":"ns"},
    "data":{"tls.crt":"bm90LWEtY2VydA=="}
  }]
}`,
	})
	if _, ok := secretCertExpiryFix(context.Background(), src, "ns", "junk"); ok {
		t.Error("garbage tls.crt must not parse")
	}
	if _, ok := secretCertExpiryFix(context.Background(), src, "ns", "absent"); ok {
		t.Error("missing Secret must not parse")
	}
}

func TestCertReadyFix(t *testing.T) {
	mk := func(status string) string {
		return certificateJSON("ns", "c", "s", "h.example.com", status)
	}
	for _, tc := range []struct {
		name   string
		status string
		want   bool
	}{
		{"ready true", "True", true},
		{"ready false", "False", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			src := loadSrc(t, map[string]string{"certificates.json": mk(tc.status)})
			gvr := schema.GroupVersionResource{Group: "cert-manager.io", Version: "v1", Resource: "certificates"}
			list, err := src.List(context.Background(), gvr, "ns")
			if err != nil || len(list.Items) != 1 {
				t.Fatalf("fixture list: %v / %d items", err, len(list.Items))
			}
			if got := certReadyFix(list.Items[0]); got != tc.want {
				t.Errorf("certReadyFix = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestFindMismatchedCertFix_SkipsSameSecretAndWrongHost(t *testing.T) {
	src := loadSrc(t, map[string]string{
		"certificates.json": `{
  "apiVersion":"cert-manager.io/v1","kind":"CertificateList","items":[
    {
      "apiVersion":"cert-manager.io/v1","kind":"Certificate",
      "metadata":{"name":"same-secret","namespace":"ns"},
      "spec":{"secretName":"current","dnsNames":["a.example.com"]},
      "status":{"conditions":[{"type":"Ready","status":"True"}]}
    },
    {
      "apiVersion":"cert-manager.io/v1","kind":"Certificate",
      "metadata":{"name":"wrong-host","namespace":"ns"},
      "spec":{"secretName":"other","dnsNames":["b.example.com"]},
      "status":{"conditions":[{"type":"Ready","status":"True"}]}
    }
  ]
}`,
	})
	if got := findMismatchedCertFix(context.Background(), src, "ns", []string{"a.example.com"}, "current"); got != "" {
		t.Errorf("expected no candidate (same secret / wrong host), got %q", got)
	}
	if got := findMismatchedCertFix(context.Background(), src, "ns", []string{"b.example.com"}, "current"); got != "other" {
		t.Errorf("expected candidate %q, got %q", "other", got)
	}
}
