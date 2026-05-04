// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package diagnose

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/internal/snapshot"
	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/internal/vault"
)

// fakeVault returns a deterministic key set per path. ErrPathNotFound when
// path is in `notFound`; transport error when path is in `errs`.
type fakeVault struct {
	keys     map[string][]string
	notFound map[string]struct{}
	errs     map[string]error
	calls    []string
}

func (f *fakeVault) ListKeys(_ context.Context, p string) ([]string, error) {
	f.calls = append(f.calls, p)
	if _, ok := f.notFound[p]; ok {
		return nil, vault.ErrPathNotFound
	}
	if e, ok := f.errs[p]; ok {
		return nil, e
	}
	return f.keys[p], nil
}

// vaultSrc is a minimal Source that honors GVRExtSecret + GVRSecretStore +
// GVRClusterSecretStore. The two store collections are optional; when nil,
// the analyzer treats no-stores-discovered as the v0.2 "fall back, query
// every ESO" path (which most existing tests rely on).
type vaultSrc struct {
	esos                []unstructured.Unstructured
	secretStores        []unstructured.Unstructured
	clusterSecretStores []unstructured.Unstructured
	mode                snapshot.Mode
}

func (v *vaultSrc) List(_ context.Context, gvr schema.GroupVersionResource, _ string) (*unstructured.UnstructuredList, error) {
	switch gvr {
	case snapshot.GVRExtSecret:
		return &unstructured.UnstructuredList{Items: v.esos}, nil
	case snapshot.GVRSecretStore:
		return &unstructured.UnstructuredList{Items: v.secretStores}, nil
	case snapshot.GVRClusterSecretStore:
		return &unstructured.UnstructuredList{Items: v.clusterSecretStores}, nil
	}
	return &unstructured.UnstructuredList{}, nil
}
func (v *vaultSrc) Get(_ context.Context, _ schema.GroupVersionResource, _, _ string) (*unstructured.Unstructured, error) {
	return nil, fmt.Errorf("not implemented")
}
func (v *vaultSrc) Mode() snapshot.Mode { return v.mode }

func makeESO(ns, name string, dataRefs []map[string]string, dataFromKeys []string) unstructured.Unstructured {
	data := make([]any, 0, len(dataRefs))
	for _, ref := range dataRefs {
		data = append(data, map[string]any{
			"secretKey": ref["secretKey"],
			"remoteRef": map[string]any{
				"key":      ref["key"],
				"property": ref["property"],
			},
		})
	}
	dataFrom := make([]any, 0, len(dataFromKeys))
	for _, k := range dataFromKeys {
		dataFrom = append(dataFrom, map[string]any{
			"extract": map[string]any{"key": k},
		})
	}
	return unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "external-secrets.io/v1",
			"kind":       "ExternalSecret",
			"metadata":   map[string]any{"namespace": ns, "name": name},
			"spec":       map[string]any{"data": data, "dataFrom": dataFrom},
		},
	}
}

func TestVaultPathMissing_NoClient(t *testing.T) {
	src := &vaultSrc{mode: snapshot.ModeLive}
	out := VaultPathMissing{}.Run(context.Background(), src)
	if out != nil {
		t.Errorf("want nil when client unset, got %v", out)
	}
}

func TestVaultPathMissing_SnapshotMode(t *testing.T) {
	src := &vaultSrc{mode: snapshot.ModeSnapshot}
	out := VaultPathMissing{Client: &fakeVault{}}.Run(context.Background(), src)
	if out != nil {
		t.Errorf("want nil in snapshot mode, got %v", out)
	}
}

func TestVaultPathMissing_PathExistsAllKeys(t *testing.T) {
	eso := makeESO("livekit", "creds", []map[string]string{
		{"secretKey": "API_KEY", "key": "livekit/creds", "property": "API_KEY"},
		{"secretKey": "SECRET", "key": "livekit/creds", "property": "API_SECRET"},
	}, nil)
	src := &vaultSrc{esos: []unstructured.Unstructured{eso}, mode: snapshot.ModeLive}
	fv := &fakeVault{keys: map[string][]string{
		"livekit/creds": {"API_KEY", "API_SECRET"},
	}}
	out := VaultPathMissing{Client: fv}.Run(context.Background(), src)
	if len(out) != 0 {
		t.Errorf("want 0 diagnostics, got %d: %v", len(out), out)
	}
	if len(fv.calls) != 1 || fv.calls[0] != "livekit/creds" {
		t.Errorf("expected one call for livekit/creds, got %v", fv.calls)
	}
}

func TestVaultPathMissing_MissingKey(t *testing.T) {
	eso := makeESO("ns", "app", []map[string]string{
		{"secretKey": "DB_PASSWORD", "key": "team/db", "property": "DB_PASSWORD"},
		{"secretKey": "DB_USER", "key": "team/db", "property": "DB_USER"},
	}, nil)
	src := &vaultSrc{esos: []unstructured.Unstructured{eso}, mode: snapshot.ModeLive}
	fv := &fakeVault{keys: map[string][]string{
		"team/db": {"DB_USER"}, // DB_PASSWORD removed in Vault
	}}
	out := VaultPathMissing{Client: fv}.Run(context.Background(), src)
	if len(out) != 1 {
		t.Fatalf("want 1 diagnostic, got %d: %v", len(out), out)
	}
	if !strings.Contains(out[0].Message, "DB_PASSWORD") || !strings.Contains(out[0].Message, "team/db") {
		t.Errorf("unexpected message: %s", out[0].Message)
	}
	if !strings.Contains(out[0].Subject, "vault-missing-key/team/db/DB_PASSWORD") {
		t.Errorf("unexpected subject: %s", out[0].Subject)
	}
}

func TestVaultPathMissing_MissingPath(t *testing.T) {
	eso := makeESO("ns", "app", []map[string]string{
		{"secretKey": "K", "key": "deleted/path", "property": "K"},
	}, nil)
	src := &vaultSrc{esos: []unstructured.Unstructured{eso}, mode: snapshot.ModeLive}
	fv := &fakeVault{notFound: map[string]struct{}{"deleted/path": {}}}
	out := VaultPathMissing{Client: fv}.Run(context.Background(), src)
	if len(out) != 1 {
		t.Fatalf("want 1 diagnostic, got %d", len(out))
	}
	if !strings.Contains(out[0].Subject, "missing-vault-path/deleted/path") {
		t.Errorf("unexpected subject: %s", out[0].Subject)
	}
	if !strings.Contains(out[0].Message, "ExternalSecret/ns/app") {
		t.Errorf("expected consumer name in message, got: %s", out[0].Message)
	}
}

func TestVaultPathMissing_DataFrom_PathExists(t *testing.T) {
	eso := makeESO("ns", "app", nil, []string{"shared/all"})
	src := &vaultSrc{esos: []unstructured.Unstructured{eso}, mode: snapshot.ModeLive}
	fv := &fakeVault{keys: map[string][]string{"shared/all": {"a", "b", "c"}}}
	out := VaultPathMissing{Client: fv}.Run(context.Background(), src)
	if len(out) != 0 {
		t.Errorf("dataFrom path exists; want 0 diagnostics, got %d: %v", len(out), out)
	}
}

func TestVaultPathMissing_DataFrom_PathMissing(t *testing.T) {
	eso := makeESO("ns", "app", nil, []string{"shared/all"})
	src := &vaultSrc{esos: []unstructured.Unstructured{eso}, mode: snapshot.ModeLive}
	fv := &fakeVault{notFound: map[string]struct{}{"shared/all": {}}}
	out := VaultPathMissing{Client: fv}.Run(context.Background(), src)
	if len(out) != 1 || !strings.Contains(out[0].Subject, "missing-vault-path/shared/all") {
		t.Errorf("want missing-vault-path diagnostic, got: %v", out)
	}
}

func TestVaultPathMissing_TransportError(t *testing.T) {
	eso := makeESO("ns", "app", []map[string]string{
		{"secretKey": "K", "key": "path/x", "property": "K"},
	}, nil)
	src := &vaultSrc{esos: []unstructured.Unstructured{eso}, mode: snapshot.ModeLive}
	fv := &fakeVault{errs: map[string]error{"path/x": fmt.Errorf("connection refused")}}
	out := VaultPathMissing{Client: fv}.Run(context.Background(), src)
	if len(out) != 1 || !strings.Contains(out[0].Subject, "vault-error/path/x") {
		t.Errorf("want vault-error diagnostic, got: %v", out)
	}
}

func TestVaultPathMissing_DedupeAcrossESOs(t *testing.T) {
	// Two ExternalSecrets reference the same path with the same missing key;
	// expect a single diagnostic, but the consumer list should mention both.
	eso1 := makeESO("a", "first", []map[string]string{
		{"secretKey": "K", "key": "shared/path", "property": "MISSING"},
	}, nil)
	eso2 := makeESO("b", "second", []map[string]string{
		{"secretKey": "K", "key": "shared/path", "property": "MISSING"},
	}, nil)
	src := &vaultSrc{esos: []unstructured.Unstructured{eso1, eso2}, mode: snapshot.ModeLive}
	fv := &fakeVault{keys: map[string][]string{"shared/path": {"OTHER_KEY"}}}
	out := VaultPathMissing{Client: fv}.Run(context.Background(), src)
	if len(out) != 1 {
		t.Fatalf("want 1 deduped diagnostic, got %d: %v", len(out), out)
	}
	if !strings.Contains(out[0].Message, "ExternalSecret/a/first") || !strings.Contains(out[0].Message, "ExternalSecret/b/second") {
		t.Errorf("expected both consumers listed, got: %s", out[0].Message)
	}
	// Path queried once (cached per path via the requirements map).
	if len(fv.calls) != 1 {
		t.Errorf("want 1 vault call, got %d: %v", len(fv.calls), fv.calls)
	}
}

func TestVaultPathMissing_NoCRD(t *testing.T) {
	// Empty ESO list — analyzer should not call Vault at all.
	src := &vaultSrc{mode: snapshot.ModeLive}
	fv := &fakeVault{}
	out := VaultPathMissing{Client: fv}.Run(context.Background(), src)
	if len(out) != 0 {
		t.Errorf("want 0 diagnostics, got %d", len(out))
	}
	if len(fv.calls) != 0 {
		t.Errorf("expected no Vault calls, got %v", fv.calls)
	}
}

// makeStore builds a SecretStore (kind="SecretStore") or ClusterSecretStore
// (kind="ClusterSecretStore") with the named provider key under spec.provider.
// providerKey="vault" → vault-backed; "aws" → not vault-backed.
func makeStore(kind, ns, name, providerKey string) unstructured.Unstructured {
	meta := map[string]any{"name": name}
	if ns != "" {
		meta["namespace"] = ns
	}
	return unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "external-secrets.io/v1",
			"kind":       kind,
			"metadata":   meta,
			"spec": map[string]any{
				"provider": map[string]any{providerKey: map[string]any{}},
			},
		},
	}
}

// makeESOWithStore wraps makeESO to attach a secretStoreRef.
func makeESOWithStore(ns, name, storeKind, storeName string, dataRefs []map[string]string) unstructured.Unstructured {
	eso := makeESO(ns, name, dataRefs, nil)
	spec, _ := eso.Object["spec"].(map[string]any)
	spec["secretStoreRef"] = map[string]any{"kind": storeKind, "name": storeName}
	return eso
}

func TestVaultPathMissing_FilterNonVaultStore(t *testing.T) {
	// Mixed-provider cluster: AWS-backed ESO must be SKIPPED, Vault-backed
	// ESO must be queried. Without the filter (v0.2 behavior), Vault would
	// be queried for the AWS path and emit a phantom "missing" diagnostic.
	awsESO := makeESOWithStore("ns", "aws-app", "SecretStore", "aws-store",
		[]map[string]string{{"secretKey": "K", "key": "aws/path", "property": "K"}})
	vaultESO := makeESOWithStore("ns", "vault-app", "SecretStore", "vault-store",
		[]map[string]string{{"secretKey": "K", "key": "vault/path", "property": "K"}})

	src := &vaultSrc{
		esos: []unstructured.Unstructured{awsESO, vaultESO},
		secretStores: []unstructured.Unstructured{
			makeStore("SecretStore", "ns", "aws-store", "aws"),
			makeStore("SecretStore", "ns", "vault-store", "vault"),
		},
		mode: snapshot.ModeLive,
	}
	fv := &fakeVault{keys: map[string][]string{"vault/path": {"K"}}}
	out := VaultPathMissing{Client: fv}.Run(context.Background(), src)
	if len(out) != 0 {
		t.Errorf("want 0 diagnostics (vault-backed ESO is healthy), got %d: %v", len(out), out)
	}
	if len(fv.calls) != 1 || fv.calls[0] != "vault/path" {
		t.Errorf("expected only vault/path queried, got %v", fv.calls)
	}
}

func TestVaultPathMissing_FilterClusterSecretStore(t *testing.T) {
	// ClusterSecretStore variant — kind matters because the index is
	// keyed differently (cluster-scoped vs. namespaced).
	eso := makeESOWithStore("ns", "app", "ClusterSecretStore", "global-vault",
		[]map[string]string{{"secretKey": "K", "key": "p", "property": "K"}})
	src := &vaultSrc{
		esos:                []unstructured.Unstructured{eso},
		clusterSecretStores: []unstructured.Unstructured{makeStore("ClusterSecretStore", "", "global-vault", "vault")},
		mode:                snapshot.ModeLive,
	}
	fv := &fakeVault{keys: map[string][]string{"p": {"K"}}}
	out := VaultPathMissing{Client: fv}.Run(context.Background(), src)
	if len(out) != 0 {
		t.Errorf("want 0 diagnostics, got %d: %v", len(out), out)
	}
	if len(fv.calls) != 1 {
		t.Errorf("want 1 vault call, got %d: %v", len(fv.calls), fv.calls)
	}
}

func TestVaultPathMissing_FilterDegradesWithoutStores(t *testing.T) {
	// Operator hasn't given cha access to SecretStore CRDs (or they aren't
	// installed). hasAnyStore=false → fall back to v0.2 "treat all ESOs
	// as Vault-backed" so we don't silently drop real drift detection.
	eso := makeESOWithStore("ns", "app", "SecretStore", "any",
		[]map[string]string{{"secretKey": "K", "key": "p", "property": "K"}})
	src := &vaultSrc{esos: []unstructured.Unstructured{eso}, mode: snapshot.ModeLive}
	fv := &fakeVault{keys: map[string][]string{"p": {"K"}}}
	out := VaultPathMissing{Client: fv}.Run(context.Background(), src)
	if len(out) != 0 {
		t.Errorf("want 0 diagnostics, got %d: %v", len(out), out)
	}
	if len(fv.calls) != 1 {
		t.Errorf("want 1 vault call (degraded mode), got %d", len(fv.calls))
	}
}

func TestVaultPathMissing_OutageCollapsesAtThreshold(t *testing.T) {
	// 5 paths all failing with the same transport error → ONE summary
	// diagnostic, not five per-path noisy ones.
	dataRefs := []map[string]string{}
	for _, p := range []string{"a", "b", "c", "d", "e"} {
		dataRefs = append(dataRefs, map[string]string{"secretKey": "K", "key": p, "property": "K"})
	}
	eso := makeESO("ns", "app", dataRefs, nil)
	src := &vaultSrc{esos: []unstructured.Unstructured{eso}, mode: snapshot.ModeLive}
	fv := &fakeVault{errs: map[string]error{
		"a": fmt.Errorf("connection refused"),
		"b": fmt.Errorf("connection refused"),
		"c": fmt.Errorf("connection refused"),
		"d": fmt.Errorf("connection refused"),
		"e": fmt.Errorf("connection refused"),
	}}
	out := VaultPathMissing{Client: fv}.Run(context.Background(), src)
	if len(out) != 1 {
		t.Fatalf("want 1 summary diagnostic, got %d: %v", len(out), out)
	}
	if !strings.Contains(out[0].Subject, "vault-outage/") {
		t.Errorf("want vault-outage subject, got: %s", out[0].Subject)
	}
	if !strings.Contains(out[0].Message, "5 paths failed") {
		t.Errorf("want N-paths-failed message, got: %s", out[0].Message)
	}
	if !strings.Contains(out[0].Message, "+2 more") {
		t.Errorf("want sample-truncation indicator, got: %s", out[0].Message)
	}
}

func TestVaultPathMissing_OutageBelowThresholdIsPerPath(t *testing.T) {
	// 2 paths, same error → still per-path (operator-visible).
	dataRefs := []map[string]string{
		{"secretKey": "K", "key": "a", "property": "K"},
		{"secretKey": "K", "key": "b", "property": "K"},
	}
	eso := makeESO("ns", "app", dataRefs, nil)
	src := &vaultSrc{esos: []unstructured.Unstructured{eso}, mode: snapshot.ModeLive}
	fv := &fakeVault{errs: map[string]error{
		"a": fmt.Errorf("connection refused"),
		"b": fmt.Errorf("connection refused"),
	}}
	out := VaultPathMissing{Client: fv}.Run(context.Background(), src)
	if len(out) != 2 {
		t.Fatalf("want 2 per-path diagnostics, got %d", len(out))
	}
	for _, d := range out {
		if !strings.Contains(d.Subject, "vault-error/") {
			t.Errorf("want vault-error subject (below threshold), got: %s", d.Subject)
		}
	}
}

func TestVaultPathMissing_OutageGroupsByErrorString(t *testing.T) {
	// Different error shapes → separate groups; threshold applied per-group.
	dataRefs := []map[string]string{
		{"secretKey": "K", "key": "a1", "property": "K"},
		{"secretKey": "K", "key": "a2", "property": "K"},
		{"secretKey": "K", "key": "a3", "property": "K"},
		{"secretKey": "K", "key": "b1", "property": "K"},
	}
	eso := makeESO("ns", "app", dataRefs, nil)
	src := &vaultSrc{esos: []unstructured.Unstructured{eso}, mode: snapshot.ModeLive}
	fv := &fakeVault{errs: map[string]error{
		"a1": fmt.Errorf("connection refused"),
		"a2": fmt.Errorf("connection refused"),
		"a3": fmt.Errorf("connection refused"),
		"b1": fmt.Errorf("permission denied"),
	}}
	out := VaultPathMissing{Client: fv}.Run(context.Background(), src)
	// 1 summary for the 3-grouping + 1 per-path for the lone other error = 2.
	if len(out) != 2 {
		t.Fatalf("want 2 diagnostics (1 summary + 1 per-path), got %d: %v", len(out), out)
	}
}
