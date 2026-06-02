// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package ai_test

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/pkg/ai"
)

func mustGenKey(t *testing.T) ed25519.PrivateKey {
	t.Helper()
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("genkey: %v", err)
	}
	return priv
}

func TestNewSigner_RequiresValidKey(t *testing.T) {
	if _, err := ai.NewSigner(ai.SignerConfig{}); err == nil {
		t.Fatal("want error on empty key")
	}
	if _, err := ai.NewSigner(ai.SignerConfig{PrivateKey: []byte("too short")}); err == nil {
		t.Fatal("want error on wrong-size key")
	}
	if _, err := ai.NewSigner(ai.SignerConfig{PrivateKey: mustGenKey(t)}); err != nil {
		t.Errorf("valid key should construct: %v", err)
	}
}

func TestSignerImpl_Sign_HappyPath(t *testing.T) {
	signer, err := ai.NewSigner(ai.SignerConfig{PrivateKey: mustGenKey(t)})
	if err != nil {
		t.Fatalf("NewSigner: %v", err)
	}
	now := time.Now().UTC()
	prop := ai.AIProposedAction{
		ActionID:          "test-action-1",
		ActionKind:        ai.ActionApplyManifest,
		Target:            ai.ObjectRef{Kind: "NetworkPolicy", Namespace: "x", Name: "y"},
		Tier:              ai.TierT1,
		CreatedAt:         now,
		ExpiresAt:         now.Add(15 * time.Minute),
		DiagnosticSubject: "Namespace/cluster/x/missing-network-policy",
	}
	tok, err := signer.Sign(prop)
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
	parts := strings.Split(tok, ".")
	if len(parts) != 3 {
		t.Errorf("JWT should have 3 segments, got %d: %q", len(parts), tok)
	}
}

func TestSignerImpl_Sign_RefusesEmptyActionID(t *testing.T) {
	signer, _ := ai.NewSigner(ai.SignerConfig{PrivateKey: mustGenKey(t)})
	now := time.Now().UTC()
	if _, err := signer.Sign(ai.AIProposedAction{ExpiresAt: now}); err == nil {
		t.Fatal("want error on empty ActionID")
	}
}

func TestSignerImpl_Sign_RefusesZeroExpiry(t *testing.T) {
	signer, _ := ai.NewSigner(ai.SignerConfig{PrivateKey: mustGenKey(t)})
	if _, err := signer.Sign(ai.AIProposedAction{ActionID: "x"}); err == nil {
		t.Fatal("want error on zero ExpiresAt")
	}
}

func TestLoadSigningKey_ReadsBase64File(t *testing.T) {
	priv := mustGenKey(t)
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "key")
	if err := os.WriteFile(keyPath, []byte(base64.StdEncoding.EncodeToString(priv)), 0o600); err != nil {
		t.Fatalf("write key: %v", err)
	}
	loaded, err := ai.LoadSigningKey(keyPath)
	if err != nil {
		t.Fatalf("LoadSigningKey: %v", err)
	}
	if len(loaded) != ed25519.PrivateKeySize {
		t.Errorf("loaded key size: want %d got %d", ed25519.PrivateKeySize, len(loaded))
	}
}

func TestLoadSigningKey_StripsTrailingWhitespace(t *testing.T) {
	priv := mustGenKey(t)
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "key")
	// Many tools (cat, kubectl exec, base64) add a trailing newline.
	if err := os.WriteFile(keyPath, []byte(base64.StdEncoding.EncodeToString(priv)+"\n  \n"), 0o600); err != nil {
		t.Fatalf("write key: %v", err)
	}
	if _, err := ai.LoadSigningKey(keyPath); err != nil {
		t.Errorf("LoadSigningKey should tolerate trailing whitespace: %v", err)
	}
}

func TestLoadSigningKey_MissingReturnsErrSigningKeyMissing(t *testing.T) {
	_, err := ai.LoadSigningKey("/nonexistent/path/at/all")
	if !errors.Is(err, ai.ErrSigningKeyMissing) {
		t.Errorf("want ErrSigningKeyMissing, got %v", err)
	}
}

func TestLoadSigningKey_BadBase64Errors(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "key")
	if err := os.WriteFile(keyPath, []byte("not!valid!base64!"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	if _, err := ai.LoadSigningKey(keyPath); err == nil {
		t.Error("want error on bad base64")
	}
}

func TestLoadSigningKey_WrongSizeErrors(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "key")
	if err := os.WriteFile(keyPath, []byte(base64.StdEncoding.EncodeToString([]byte("short"))), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	if _, err := ai.LoadSigningKey(keyPath); err == nil {
		t.Error("want error on wrong-size key")
	}
}

func TestGenerateAndPersistSigningKey_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	privPath := filepath.Join(dir, "priv")
	pubPath := filepath.Join(dir, "pub")
	pub, err := ai.GenerateAndPersistSigningKey(privPath, pubPath)
	if err != nil {
		t.Fatalf("GenerateAndPersistSigningKey: %v", err)
	}
	if len(pub) != ed25519.PublicKeySize {
		t.Errorf("public key size: want %d got %d", ed25519.PublicKeySize, len(pub))
	}
	// LoadSigningKey should read back the private half.
	if _, err := ai.LoadSigningKey(privPath); err != nil {
		t.Errorf("LoadSigningKey round-trip: %v", err)
	}
}

func TestLoadSigningKey_FallsBackToEnvVar(t *testing.T) {
	priv := mustGenKey(t)
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "key")
	if err := os.WriteFile(keyPath, []byte(base64.StdEncoding.EncodeToString(priv)), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	t.Setenv(ai.EnvSigningKeyPath, keyPath)
	// path "" should pick up the env var.
	if _, err := ai.LoadSigningKey(""); err != nil {
		t.Errorf("env var fallback: %v", err)
	}
}
