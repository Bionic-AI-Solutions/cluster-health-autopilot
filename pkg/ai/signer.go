// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package ai

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"
)

// SignerImpl is the default Signer implementation. It holds an Ed25519
// private key and stamps JWTs for T1/T2 proposals and T3 runbook
// approvals.
//
// The approval-server process loads this key from disk via
// LoadSigningKey; the watcher pod loads the SAME key when it has
// `--approval-server-url` + `--signing-key-path` set so it can mint
// URLs in its own enrichDiagnostics path (avoiding the cross-pod
// hand-off that previously left URLs in aiwatch stdout but absent from
// Slack/OpenProject deliveries).
type SignerImpl struct {
	kid  string
	priv ed25519.PrivateKey
}

// SignerConfig configures a new SignerImpl.
type SignerConfig struct {
	// KeyID identifies the signing key. Stamped into the JWT header
	// (`kid`). Allows future key rotation. Default "default-1".
	KeyID string

	// PrivateKey is the Ed25519 private key. Required.
	PrivateKey ed25519.PrivateKey
}

// NewSigner constructs a SignerImpl. The private key MUST be the full
// 64-byte Ed25519 private key (seed + public half).
func NewSigner(cfg SignerConfig) (*SignerImpl, error) {
	if len(cfg.PrivateKey) != ed25519.PrivateKeySize {
		return nil, fmt.Errorf("ai: invalid private key size (got %d, want %d)",
			len(cfg.PrivateKey), ed25519.PrivateKeySize)
	}
	kid := cfg.KeyID
	if kid == "" {
		kid = "default-1"
	}
	return &SignerImpl{kid: kid, priv: cfg.PrivateKey}, nil
}

// Sign produces a JWT for a T1/T2 proposal. Satisfies the Signer
// interface.
func (s *SignerImpl) Sign(p AIProposedAction) (string, error) {
	if p.ActionID == "" {
		return "", errors.New("ai: proposal missing ActionID")
	}
	if p.ExpiresAt.IsZero() {
		return "", errors.New("ai: proposal missing ExpiresAt")
	}
	claims := TokenClaims{
		Issuer:      "cha/approval-server",
		Audience:    "cha/executor",
		Subject:     p.ActionID,
		JTI:         p.ActionID,
		IssuedAt:    time.Now().Unix(),
		ExpiresAt:   p.ExpiresAt.Unix(),
		Tier:        p.Tier,
		ActionKind:  string(p.ActionKind),
		Target:      p.Target,
		DiagSubject: p.DiagnosticSubject,
	}
	return SignToken(s.priv, s.kid, claims)
}

// SignRunbookApproval produces a JWT for a T3 runbook acknowledgement.
// slot is 1 for the first approver and 2 for the second; slot 2 has a
// NotBefore = first-approval-time + MinT3Delay enforced inside the
// claim payload itself.
func (s *SignerImpl) SignRunbookApproval(r VaultRunbook, slot int) (string, error) {
	if r.RunbookID == "" {
		return "", errors.New("ai: runbook missing RunbookID")
	}
	if slot != 1 && slot != 2 {
		return "", fmt.Errorf("ai: invalid approval slot %d", slot)
	}
	if r.ExpiresAt.IsZero() {
		return "", errors.New("ai: runbook missing ExpiresAt")
	}
	now := time.Now()
	nbf := now.Unix()
	if slot == 2 {
		nbf = now.Add(MinT3Delay).Unix()
	}
	jti := fmt.Sprintf("%s-slot%d", r.RunbookID, slot)
	claims := TokenClaims{
		Issuer:       "cha/approval-server",
		Audience:     "cha/executor",
		Subject:      r.RunbookID,
		JTI:          jti,
		IssuedAt:     now.Unix(),
		ExpiresAt:    r.ExpiresAt.Unix(),
		NotBefore:    nbf,
		Tier:         TierT3,
		RunbookID:    r.RunbookID,
		ApprovalSlot: slot,
	}
	return SignToken(s.priv, s.kid, claims)
}

// EnvSigningKeyPath is the env var the watcher / approval-server reads
// to find the signing key on disk.
const EnvSigningKeyPath = "CHA_SIGNING_KEY_PATH"

// DefaultSigningKeyPath is the conventional mount path for the signing
// key Secret. Deployments project the key here as a base64-encoded file
// in a tmpfs volume.
const DefaultSigningKeyPath = "/etc/cha/keys/signing.key"

// ErrSigningKeyMissing indicates the signing key file does not exist
// at the configured path. Callers should fall back to URL-less behavior
// (proposals still surface as text; just no click-to-fix link).
var ErrSigningKeyMissing = errors.New("ai: signing key file not found")

// LoadSigningKey reads the Ed25519 signing key from `path`. When `path`
// is empty, falls back to `$CHA_SIGNING_KEY_PATH`, then to
// DefaultSigningKeyPath. The key file is base64-encoded raw bytes (no
// PEM wrapping — minimizes deps).
//
// Returns ErrSigningKeyMissing when the file does not exist. Callers
// that want to fall back gracefully (continue without URL minting)
// should `errors.Is(err, ErrSigningKeyMissing)`.
func LoadSigningKey(path string) (ed25519.PrivateKey, error) {
	if path == "" {
		path = os.Getenv(EnvSigningKeyPath)
	}
	if path == "" {
		path = DefaultSigningKeyPath
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrSigningKeyMissing
		}
		return nil, err
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(data)))
	if err != nil {
		return nil, fmt.Errorf("ai: signing key not valid base64: %w", err)
	}
	if len(raw) != ed25519.PrivateKeySize {
		return nil, fmt.Errorf("ai: signing key wrong size (got %d, want %d)",
			len(raw), ed25519.PrivateKeySize)
	}
	return ed25519.PrivateKey(raw), nil
}

// GenerateAndPersistSigningKey produces a fresh Ed25519 keypair and
// writes the private half (base64-encoded) to privPath. The
// corresponding public key is also written next to it. Used by the
// install-time init job; not called from the watcher hot path.
func GenerateAndPersistSigningKey(privPath, pubPath string) (ed25519.PublicKey, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	privEnc := base64.StdEncoding.EncodeToString(priv)
	if err := os.WriteFile(privPath, []byte(privEnc), 0o600); err != nil {
		return nil, err
	}
	pubEnc := base64.StdEncoding.EncodeToString(pub)
	if err := os.WriteFile(pubPath, []byte(pubEnc), 0o644); err != nil {
		return nil, err
	}
	return pub, nil
}
