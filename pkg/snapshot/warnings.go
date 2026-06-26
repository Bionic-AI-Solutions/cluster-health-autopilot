// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

package snapshot

import (
	"context"
	"os"
	"strings"

	"k8s.io/client-go/rest"
)

// endpointsDeprecationFragment matches the API server's deprecation warning
// for the legacy core/v1 Endpoints API, e.g.
//
//	v1 Endpoints is deprecated in v1.33+; use discovery.k8s.io/v1 EndpointSlice
//
// Matching on the stable prefix (not the exact version string) keeps the
// filter working as the server bumps the "in v1.NN+" part across upgrades.
const endpointsDeprecationFragment = "v1 Endpoints is deprecated"

// suppressingWarningHandler drops the core/v1 Endpoints deprecation warning
// and forwards every other server warning to next. Srenix keeps a handful of
// deliberate legacy-Endpoints fallback reads (DNSChainDrift + KongRoutes
// when EndpointSlices are unavailable, snapshot capture); without this
// filter the deployed watcher/aiwatch printed the same deprecation line for
// every such call (~2.5 log lines/sec), drowning real signal.
type suppressingWarningHandler struct {
	next rest.WarningHandler
}

// HandleWarningHeader implements rest.WarningHandler.
func (h suppressingWarningHandler) HandleWarningHeader(code int, agent string, text string) {
	if strings.Contains(text, endpointsDeprecationFragment) {
		return
	}
	if h.next != nil {
		h.next.HandleWarningHeader(code, agent, text)
	}
}

// suppressingWarningHandlerWithContext is the context-aware twin of
// suppressingWarningHandler, wrapping a caller-installed
// rest.WarningHandlerWithContext (which client-go prefers over the legacy
// WarningHandler field when both are set).
type suppressingWarningHandlerWithContext struct {
	next rest.WarningHandlerWithContext
}

// HandleWarningHeaderWithContext implements rest.WarningHandlerWithContext.
func (h suppressingWarningHandlerWithContext) HandleWarningHeaderWithContext(ctx context.Context, code int, agent string, text string) {
	if strings.Contains(text, endpointsDeprecationFragment) {
		return
	}
	if h.next != nil {
		h.next.HandleWarningHeaderWithContext(ctx, code, agent, text)
	}
}

// newSuppressingWarningHandler builds the warning filter around next:
// Endpoints deprecation warnings are dropped entirely; all other server
// warnings pass through to next. When next is nil the production default is
// used — a DEDUPLICATING stderr writer (each unique warning prints once per
// process instead of once per call).
func newSuppressingWarningHandler(next rest.WarningHandler) rest.WarningHandler {
	if next == nil {
		next = rest.NewWarningWriter(os.Stderr, rest.WarningWriterOptions{Deduplicate: true})
	}
	return suppressingWarningHandler{next: next}
}

// SuppressEndpointsDeprecationWarnings installs the Endpoints-deprecation
// warning filter on cfg (see suppressingWarningHandler): the core/v1
// Endpoints deprecation line is dropped, every other API-server warning
// passes through. The function is self-composable — a caller-installed
// handler is wrapped, not replaced:
//
//   - cfg.WarningHandlerWithContext set → it becomes the filter's `next`
//     (client-go prefers this field, so it is the one that must be wrapped);
//   - else cfg.WarningHandler set → it becomes the filter's `next`;
//   - neither set → non-suppressed warnings go to a deduplicating stderr
//     writer (once per unique message per process).
//
// Returns cfg for call-site chaining. Safe on a nil config (no-op). Note cfg
// itself is mutated; callers that must not touch a shared config should pass
// rest.CopyConfig(cfg) (as NewLiveSource does).
func SuppressEndpointsDeprecationWarnings(cfg *rest.Config) *rest.Config {
	if cfg == nil {
		return nil
	}
	if cfg.WarningHandlerWithContext != nil {
		cfg.WarningHandlerWithContext = suppressingWarningHandlerWithContext{next: cfg.WarningHandlerWithContext}
		return cfg
	}
	cfg.WarningHandler = newSuppressingWarningHandler(cfg.WarningHandler)
	return cfg
}
