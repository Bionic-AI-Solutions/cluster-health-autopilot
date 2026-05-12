// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package ai

import (
	"crypto/ed25519"
	"errors"
	"strings"
	"testing"
	"time"
)

func makeClaims(t *testing.T, exp time.Duration) TokenClaims {
	t.Helper()
	now := time.Now()
	return TokenClaims{
		Issuer:      "cha-com/approval-server",
		Audience:    "cha-com/executor",
		Subject:     "act-test-1",
		JTI:         "act-test-1",
		IssuedAt:    now.Unix(),
		ExpiresAt:   now.Add(exp).Unix(),
		Tier:        TierT1,
		ActionKind:  string(ActionDeletePod),
		Target:      ObjectRef{Kind: "Pod", Namespace: "default", Name: "demo-abc"},
		DiagSubject: "Pod/default/demo-abc",
	}
}

func TestSignAndVerify_HappyPath(t *testing.T) {
	pub, priv, err := GenerateSigningKey()
	if err != nil {
		t.Fatal(err)
	}
	claims := makeClaims(t, 15*time.Minute)

	tok, err := SignToken(priv, "kid-1", claims)
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.Split(tok, ".")
	if len(parts) != 3 {
		t.Fatalf("token must have 3 parts; got %d", len(parts))
	}

	out, err := VerifyToken(pub, tok)
	if err != nil {
		t.Fatal(err)
	}
	if out.Subject != claims.Subject {
		t.Errorf("subject mismatch: got %q want %q", out.Subject, claims.Subject)
	}
	if out.Tier != TierT1 {
		t.Errorf("tier mismatch: got %q want %q", out.Tier, TierT1)
	}
	if out.Target.Name != "demo-abc" {
		t.Errorf("target name mismatch: got %q", out.Target.Name)
	}
}

func TestVerify_WrongKey(t *testing.T) {
	_, privA, _ := GenerateSigningKey()
	pubB, _, _ := GenerateSigningKey()
	claims := makeClaims(t, 15*time.Minute)
	tok, _ := SignToken(privA, "kid-1", claims)
	if _, err := VerifyToken(pubB, tok); !errors.Is(err, ErrTokenInvalid) {
		t.Errorf("got %v; want ErrTokenInvalid for wrong public key", err)
	}
}

func TestVerify_Tampered(t *testing.T) {
	pub, priv, _ := GenerateSigningKey()
	claims := makeClaims(t, 15*time.Minute)
	tok, _ := SignToken(priv, "kid-1", claims)
	// Flip a character in the payload.
	parts := strings.Split(tok, ".")
	parts[1] = parts[1][:5] + "A" + parts[1][6:]
	tampered := strings.Join(parts, ".")
	if _, err := VerifyToken(pub, tampered); !errors.Is(err, ErrTokenInvalid) {
		t.Errorf("got %v; want ErrTokenInvalid for tampered payload", err)
	}
}

func TestVerify_Expired(t *testing.T) {
	pub, priv, _ := GenerateSigningKey()
	claims := makeClaims(t, -time.Minute) // already expired
	tok, _ := SignToken(priv, "kid-1", claims)
	if _, err := VerifyToken(pub, tok); !errors.Is(err, ErrTokenExpired) {
		t.Errorf("got %v; want ErrTokenExpired", err)
	}
}

func TestVerify_NotBefore(t *testing.T) {
	pub, priv, _ := GenerateSigningKey()
	claims := makeClaims(t, 15*time.Minute)
	claims.NotBefore = time.Now().Add(time.Hour).Unix() // not valid for an hour
	tok, _ := SignToken(priv, "kid-1", claims)
	if _, err := VerifyToken(pub, tok); !errors.Is(err, ErrTokenExpired) {
		t.Errorf("got %v; want ErrTokenExpired for nbf in future", err)
	}
}

func TestVerify_Malformed(t *testing.T) {
	pub, _, _ := GenerateSigningKey()
	cases := []string{
		"",
		"not-a-jwt",
		"only.two",
		"a.b.c.d", // four parts
	}
	for _, tok := range cases {
		if _, err := VerifyToken(pub, tok); !errors.Is(err, ErrTokenInvalid) {
			t.Errorf("token=%q: got %v; want ErrTokenInvalid", tok, err)
		}
	}
}

func TestVerify_InvalidKeySizes(t *testing.T) {
	if _, err := VerifyToken(ed25519.PublicKey([]byte{1, 2, 3}), "a.b.c"); err == nil {
		t.Error("expected error for short public key")
	}
	if _, err := SignToken(ed25519.PrivateKey([]byte{1, 2, 3}), "kid", TokenClaims{}); err == nil {
		t.Error("expected error for short private key")
	}
}

func TestSign_Determinism(t *testing.T) {
	// Ed25519 signatures are deterministic — same key + same input must
	// produce the same token. Useful for cache hit detection.
	_, priv, _ := GenerateSigningKey()
	claims := makeClaims(t, 15*time.Minute)
	t1, _ := SignToken(priv, "kid-1", claims)
	t2, _ := SignToken(priv, "kid-1", claims)
	if t1 != t2 {
		t.Errorf("deterministic signing produced different tokens:\n  t1=%s\n  t2=%s", t1, t2)
	}
}
