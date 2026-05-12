// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package ai

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Approval token shape — a compact JWS (JWT) using EdDSA (Ed25519).
//
// Layout: base64url(header) "." base64url(payload) "." base64url(signature)
//
// Header (fixed): {"alg":"EdDSA","typ":"JWT","kid":"<key-id>"}
// Payload:        TokenClaims (see below)
//
// Implementation notes:
//   - We use crypto/ed25519 directly to avoid pulling in a JWT library.
//     The token has the JWT shape (interoperable), but verification is
//     a single ed25519.Verify call — minimum dep surface.
//   - jti (one-time-use enforcement) is the ActionID. The Verifier
//     side checks jti against a replay store (Redis/etcd) externally.
//   - exp is enforced inside VerifyToken.

// TokenClaims is the JWT payload for an approval token.
type TokenClaims struct {
	// Standard claims.
	Issuer    string `json:"iss"`           // "cha-com/approval-server"
	Audience  string `json:"aud"`           // "cha-com/executor"
	Subject   string `json:"sub"`           // ActionID or RunbookID
	JTI       string `json:"jti"`           // unique id (== Subject)
	IssuedAt  int64  `json:"iat"`           // unix seconds
	ExpiresAt int64  `json:"exp"`           // unix seconds
	NotBefore int64  `json:"nbf,omitempty"` // unix seconds (T3 only)

	// CHA-specific claims.
	Tier         Tier      `json:"tier"`
	ActionKind   string    `json:"action_kind,omitempty"`
	Target       ObjectRef `json:"target,omitempty"`
	RunbookID    string    `json:"runbook_id,omitempty"`
	ApprovalSlot int       `json:"slot,omitempty"` // 1 or 2 for T3
	DiagSubject  string    `json:"diag_subject,omitempty"`
}

// SignToken signs claims with priv. The returned string is a compact
// JWS suitable for embedding in an HTTPS URL.
func SignToken(priv ed25519.PrivateKey, kid string, claims TokenClaims) (string, error) {
	if len(priv) != ed25519.PrivateKeySize {
		return "", fmt.Errorf("ai: invalid ed25519 private key size")
	}
	header := map[string]string{"alg": "EdDSA", "typ": "JWT", "kid": kid}
	hBytes, err := json.Marshal(header)
	if err != nil {
		return "", err
	}
	pBytes, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	encH := b64.EncodeToString(hBytes)
	encP := b64.EncodeToString(pBytes)
	signingInput := encH + "." + encP
	sig := ed25519.Sign(priv, []byte(signingInput))
	encS := b64.EncodeToString(sig)
	return signingInput + "." + encS, nil
}

// VerifyToken parses token, verifies signature against pub, and checks
// exp/nbf. Returns the claims on success. Replay detection (jti uniqueness)
// is the caller's responsibility — VerifyToken does not maintain state.
func VerifyToken(pub ed25519.PublicKey, token string) (*TokenClaims, error) {
	if len(pub) != ed25519.PublicKeySize {
		return nil, errors.New("ai: invalid ed25519 public key size")
	}
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, ErrTokenInvalid
	}
	encH, encP, encS := parts[0], parts[1], parts[2]
	sig, err := b64.DecodeString(encS)
	if err != nil {
		return nil, ErrTokenInvalid
	}
	signingInput := encH + "." + encP
	if !ed25519.Verify(pub, []byte(signingInput), sig) {
		return nil, ErrTokenInvalid
	}
	pBytes, err := b64.DecodeString(encP)
	if err != nil {
		return nil, ErrTokenInvalid
	}
	var claims TokenClaims
	if err := json.Unmarshal(pBytes, &claims); err != nil {
		return nil, ErrTokenInvalid
	}
	now := time.Now().Unix()
	if claims.ExpiresAt > 0 && now > claims.ExpiresAt {
		return nil, ErrTokenExpired
	}
	if claims.NotBefore > 0 && now < claims.NotBefore {
		return nil, ErrTokenExpired // "not yet valid" is the same class
	}
	return &claims, nil
}

// b64 is the URL-safe base64 encoding without padding (JWS spec).
var b64 = base64.RawURLEncoding

// GenerateSigningKey produces a fresh Ed25519 keypair. The approval-server
// generates one at install time and stores the private half in a
// Kubernetes Secret; the public half is also stored and is consulted by
// any cha-com pod that needs to verify locally (e.g. for diagnostic
// purposes).
func GenerateSigningKey() (ed25519.PublicKey, ed25519.PrivateKey, error) {
	return ed25519.GenerateKey(rand.Reader)
}
