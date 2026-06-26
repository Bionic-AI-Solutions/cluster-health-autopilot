// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

// Package vault is a thin alias-layer over pkg/vault, kept so existing
// OSS call-sites (internal/diagnose, cmd/srenix) continue to import
// internal/vault without churn. New code should import pkg/vault
// directly.
//
// The full Vault HTTP client implementation was promoted to pkg/vault
// in v1.6.1 to unblock paid-tier Vault analyzers in Srenix Enterprise (Go's
// `internal` rule blocks external modules from importing this package).
// See docs/design/2026-05-srenix-enterprise-publishing-gap.md G2.
package vault

import (
	pkgvault "github.com/srenix-ai/agentic-sre/pkg/vault"
)

// Client is re-exported from pkg/vault.
type Client = pkgvault.Client

// HTTPClient is re-exported from pkg/vault.
type HTTPClient = pkgvault.HTTPClient

// Config is re-exported from pkg/vault.
type Config = pkgvault.Config

// KubernetesAuthConfig is re-exported from pkg/vault.
type KubernetesAuthConfig = pkgvault.KubernetesAuthConfig

// ErrPathNotFound is re-exported from pkg/vault.
var ErrPathNotFound = pkgvault.ErrPathNotFound

// New delegates to pkg/vault.New.
func New(cfg Config) (*HTTPClient, error) { return pkgvault.New(cfg) }
