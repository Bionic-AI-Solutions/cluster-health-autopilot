// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package ai

import (
	"crypto/ed25519"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Silence token — a signed one-click "mute this finding" link.
//
// Security model
// ==============
// The silence token is a compact JWS (EdDSA / Ed25519), structurally
// identical to the approval TokenClaims JWT (see jwt.go). It is embedded
// in a Slack link of the form:
//
//	<approval-base-url>/silence?token=<compact-jws>
//
// When an operator clicks the link, the CHA-com approval-server's
// `/silence` handler:
//
//  1. VerifySilenceToken(pub, token) — rejects tampered / wrong-key /
//     expired tokens BEFORE any field is trusted. The signature covers
//     the ENTIRE payload, so EVERY claim below is tamper-proof.
//  2. Translates the verified claims into a real OSS Silence CR
//     (api/v1alpha1) and creates it.
//
// Why the duration (UntilUnix) is SIGNED, not derived server-side:
//   - The whole point of a one-click link is that the server does no
//     policy decision — it just materializes exactly what the link
//     promised. If `until` were a server default, the link's label
//     ("Silence 24h" vs "Silence class 90d") could silently diverge from
//     what actually got created. Signing the precise expiry makes the
//     link self-describing and un-forgeable: an attacker cannot extend a
//     24h subject snooze into a 90d cluster-wide class mute by editing
//     the URL, because that flips the signature.
//   - `Scope`, `Subject`, `Source`, and `MessagePattern` are likewise
//     signed: the matcher the CR ends up with is precisely the matcher
//     the minting watcher chose. No field can be widened post-hoc.
//
// exp vs UntilUnix:
//   - ExpiresAt (`exp`) bounds how long the LINK is clickable (mirrors
//     the approve-token exp policy — a link left in stale Slack history
//     stops working). It is enforced inside VerifySilenceToken.
//   - UntilUnix is the silence WINDOW written to spec.until of the
//     resulting CR. They are independent: a short-lived link can mint a
//     long (90d) silence. UntilUnix is NOT enforced by the token verify;
//     it is data carried INTO the CR.
//
// JTI is unique per link (one link == one mint). The approval-server is
// expected to enforce one-time-use against its replay store, exactly as
// it does for approval tokens.

// SilenceScope is the breadth of a silence link.
type SilenceScope string

const (
	// SilenceScopeSubject snoozes one specific finding (Source +
	// Subject exact match). Short-lived by convention (e.g. 24h).
	SilenceScopeSubject SilenceScope = "subject"

	// SilenceScopeClass mutes an entire finding class (Source match,
	// no Subject — every subject under that analyzer). Long-lived by
	// convention (e.g. 90d).
	SilenceScopeClass SilenceScope = "class"
)

// SilenceTokenClaims is the JWT payload for a one-click silence link.
// Every field is covered by the signature.
type SilenceTokenClaims struct {
	// Standard claims.
	Issuer    string `json:"iss"` // "cha/approval-server"
	Audience  string `json:"aud"` // "cha/silence"
	JTI       string `json:"jti"` // unique id (one link == one mint)
	IssuedAt  int64  `json:"iat"` // unix seconds
	ExpiresAt int64  `json:"exp"` // unix seconds — bounds link clickability

	// Silence claims — translated into the Silence CR matcher + window.
	Scope          SilenceScope `json:"scope"`         // "subject" | "class"
	Source         string       `json:"source"`        // matcher.source (analyzer name)
	Subject        string       `json:"subject"`       // matcher.subject (subject-scoped only)
	MessagePattern string       `json:"msg,omitempty"` // matcher.messagePattern (optional)
	UntilUnix      int64        `json:"until"`         // spec.until (silence WINDOW expiry)
}

// SignSilenceToken signs claims with priv. Mirrors SignToken (jwt.go):
// same EdDSA compact-JWS layout, same kid header, single ed25519.Sign.
func SignSilenceToken(priv ed25519.PrivateKey, kid string, claims SilenceTokenClaims) (string, error) {
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

// VerifySilenceToken parses token, verifies the signature against pub,
// and checks exp. Returns the claims on success. Mirrors VerifyToken:
// verify-before-unmarshal — the payload is NEVER decoded into the claims
// struct until the signature has been validated, so a caller can trust
// every returned field. Replay detection (jti uniqueness) is the
// caller's responsibility.
func VerifySilenceToken(pub ed25519.PublicKey, token string) (*SilenceTokenClaims, error) {
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
	var claims SilenceTokenClaims
	if err := json.Unmarshal(pBytes, &claims); err != nil {
		return nil, ErrTokenInvalid
	}
	now := time.Now().Unix()
	if claims.ExpiresAt > 0 && now > claims.ExpiresAt {
		return nil, ErrTokenExpired
	}
	return &claims, nil
}
