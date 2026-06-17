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

func makeSilenceClaims(exp time.Duration) SilenceTokenClaims {
	now := time.Now()
	return SilenceTokenClaims{
		Issuer:         "cha/approval-server",
		Audience:       "cha/silence",
		JTI:            "sil-subj-stalepods-pod-default-x-1",
		IssuedAt:       now.Unix(),
		ExpiresAt:      now.Add(exp).Unix(),
		Scope:          SilenceScopeSubject,
		Source:         "StaleErrorPods",
		Subject:        "Pod/default/legacy",
		MessagePattern: "",
		UntilUnix:      now.Add(24 * time.Hour).Unix(),
	}
}

func TestSilenceToken_SignVerifyRoundTrip(t *testing.T) {
	pub, priv, err := GenerateSigningKey()
	if err != nil {
		t.Fatal(err)
	}
	claims := makeSilenceClaims(time.Hour)
	tok, err := SignSilenceToken(priv, "kid-1", claims)
	if err != nil {
		t.Fatal(err)
	}
	if parts := strings.Split(tok, "."); len(parts) != 3 {
		t.Fatalf("token must have 3 parts; got %d", len(parts))
	}
	out, err := VerifySilenceToken(pub, tok)
	if err != nil {
		t.Fatal(err)
	}
	if out.Scope != SilenceScopeSubject {
		t.Errorf("scope mismatch: got %q", out.Scope)
	}
	if out.Source != claims.Source {
		t.Errorf("source mismatch: got %q", out.Source)
	}
	if out.Subject != claims.Subject {
		t.Errorf("subject mismatch: got %q", out.Subject)
	}
	if out.UntilUnix != claims.UntilUnix {
		t.Errorf("until mismatch: got %d want %d", out.UntilUnix, claims.UntilUnix)
	}
}

func TestSilenceToken_WrongKey(t *testing.T) {
	_, privA, _ := GenerateSigningKey()
	pubB, _, _ := GenerateSigningKey()
	tok, _ := SignSilenceToken(privA, "kid-1", makeSilenceClaims(time.Hour))
	if _, err := VerifySilenceToken(pubB, tok); !errors.Is(err, ErrTokenInvalid) {
		t.Errorf("got %v; want ErrTokenInvalid for wrong public key", err)
	}
}

// TestSilenceToken_TamperUntil pins the security model: an attacker who
// extends the silence window by editing the URL must fail verification.
func TestSilenceToken_TamperUntil(t *testing.T) {
	pub, priv, _ := GenerateSigningKey()
	claims := makeSilenceClaims(time.Hour)
	tok, _ := SignSilenceToken(priv, "kid-1", claims)
	parts := strings.Split(tok, ".")
	// Flip a byte in the payload (which carries UntilUnix).
	parts[1] = parts[1][:8] + flipChar(parts[1][8]) + parts[1][9:]
	tampered := strings.Join(parts, ".")
	if _, err := VerifySilenceToken(pub, tampered); !errors.Is(err, ErrTokenInvalid) {
		t.Errorf("got %v; want ErrTokenInvalid for tampered payload", err)
	}
}

func TestSilenceToken_TamperScopeAndSubject(t *testing.T) {
	// Re-signing a widened claim with a DIFFERENT key must not verify
	// against the legit pub key — proves no field can be widened post-hoc.
	pub, _, _ := GenerateSigningKey()
	_, attackerPriv, _ := GenerateSigningKey()
	widened := makeSilenceClaims(time.Hour)
	widened.Scope = SilenceScopeClass
	widened.Subject = ""
	widened.UntilUnix = time.Now().Add(90 * 24 * time.Hour).Unix()
	tok, _ := SignSilenceToken(attackerPriv, "kid-1", widened)
	if _, err := VerifySilenceToken(pub, tok); !errors.Is(err, ErrTokenInvalid) {
		t.Errorf("got %v; want ErrTokenInvalid for attacker-signed widened claim", err)
	}
}

func TestSilenceToken_Expired(t *testing.T) {
	pub, priv, _ := GenerateSigningKey()
	tok, _ := SignSilenceToken(priv, "kid-1", makeSilenceClaims(-time.Minute))
	if _, err := VerifySilenceToken(pub, tok); !errors.Is(err, ErrTokenExpired) {
		t.Errorf("got %v; want ErrTokenExpired", err)
	}
}

func TestSilenceToken_Malformed(t *testing.T) {
	pub, _, _ := GenerateSigningKey()
	for _, tok := range []string{"", "not-a-jwt", "only.two", "a.b.c.d"} {
		if _, err := VerifySilenceToken(pub, tok); !errors.Is(err, ErrTokenInvalid) {
			t.Errorf("token=%q: got %v; want ErrTokenInvalid", tok, err)
		}
	}
}

func TestSilenceToken_InvalidKeySizes(t *testing.T) {
	if _, err := VerifySilenceToken(ed25519.PublicKey([]byte{1, 2, 3}), "a.b.c"); err == nil {
		t.Error("expected error for short public key")
	}
	if _, err := SignSilenceToken(ed25519.PrivateKey([]byte{1, 2, 3}), "kid", SilenceTokenClaims{}); err == nil {
		t.Error("expected error for short private key")
	}
}

func flipChar(c byte) string {
	if c == 'A' {
		return "B"
	}
	return "A"
}
