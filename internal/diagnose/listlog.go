// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

package diagnose

import (
	"log"
	"strings"
	"sync"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

// Once-per-(resource, error-reason) list-failure logs. Lifted from
// disruption_drift.go (P1.2) so every analyzer can use it (P1.7).
//
// Analyzers keep their fail-open contract — a List failure must never
// abort the diagnose cycle — but before this helper most analyzers
// swallowed the error with a bare `return nil`, so a persistent RBAC
// denial (Forbidden) made the analyzer silently dead forever. The log
// is keyed by error reason (not a bare sync.Once) so a transient
// startup blip does not swallow the log line for a *different*
// persistent failure (e.g. a later RBAC regression). Cardinality is
// bounded by resource count × apierrors reason values.
var (
	logListFailMu   sync.Mutex
	logListFailSeen = map[string]bool{}
)

// logListFailure emits the once-per-(resource, reason) degradation line
// for a failed List call.
//
// Two knobs:
//   - resource: the listed resource's plural name, used as the log key.
//   - optional: pass true when the GVR is an optional CRD (cert-manager,
//     Argo, Flux, CNPG, ESO, …). NotFound then means "not installed" —
//     the expected case — and stays silent. Every other reason
//     (Forbidden, timeouts, …) logs regardless of the flag.
func logListFailure(resource string, err error, optional bool) {
	if err == nil {
		return
	}
	if optional && isMissingResourceErr(err) {
		return
	}
	key := resource + "/" + string(apierrors.ReasonForError(err))
	logListFailMu.Lock()
	defer logListFailMu.Unlock()
	if logListFailSeen[key] {
		return
	}
	logListFailSeen[key] = true
	log.Printf("diagnose: list %s failed (analyzer degraded until this resolves): %v", resource, err)
}

// isMissingResourceErr reports whether err means the GVR itself does not
// exist on the cluster (CRD not installed). Covers both the apiserver's
// 404 StatusError and the RESTMapper's NoKindMatchError, which is not an
// apierrors NotFound (mirrors watcher.isNotFound).
func isMissingResourceErr(err error) bool {
	if apierrors.IsNotFound(err) {
		return true
	}
	msg := err.Error()
	return strings.Contains(msg, "the server could not find the requested resource") ||
		strings.Contains(msg, "no matches for kind")
}
