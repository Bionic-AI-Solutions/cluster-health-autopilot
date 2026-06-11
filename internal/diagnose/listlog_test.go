// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package diagnose

import (
	"bytes"
	"errors"
	"log"
	"strings"
	"testing"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// captureLog redirects the stdlib logger for the duration of fn and
// returns everything written.
func captureLog(t *testing.T, fn func()) string {
	t.Helper()
	var buf bytes.Buffer
	prev := log.Writer()
	log.SetOutput(&buf)
	defer log.SetOutput(prev)
	fn()
	return buf.String()
}

func TestLogListFailure_OncePerResourceReason(t *testing.T) {
	forbidden := apierrors.NewForbidden(
		schema.GroupResource{Resource: "llt-widgets"}, "", errors.New("rbac says no"))
	out := captureLog(t, func() {
		logListFailure("llt-widgets", forbidden, false)
		logListFailure("llt-widgets", forbidden, false) // deduped
	})
	if got := strings.Count(out, "list llt-widgets failed"); got != 1 {
		t.Errorf("expected exactly 1 log line, got %d:\n%s", got, out)
	}
}

func TestLogListFailure_DistinctReasonsBothLog(t *testing.T) {
	forbidden := apierrors.NewForbidden(
		schema.GroupResource{Resource: "llt-gadgets"}, "", errors.New("rbac says no"))
	timeout := apierrors.NewServerTimeout(
		schema.GroupResource{Resource: "llt-gadgets"}, "list", 1)
	out := captureLog(t, func() {
		logListFailure("llt-gadgets", forbidden, false)
		logListFailure("llt-gadgets", timeout, false) // different reason → logs again
	})
	if got := strings.Count(out, "list llt-gadgets failed"); got != 2 {
		t.Errorf("expected 2 log lines (one per reason), got %d:\n%s", got, out)
	}
}

func TestLogListFailure_OptionalSilentOnNotFound(t *testing.T) {
	notFound := apierrors.NewNotFound(schema.GroupResource{Resource: "llt-crds"}, "")
	noMatch := errors.New(`no matches for kind "Widget" in version "x.io/v1"`)
	out := captureLog(t, func() {
		logListFailure("llt-crds", notFound, true)
		logListFailure("llt-crds", noMatch, true)
		logListFailure("llt-crds", nil, true) // nil err is always a no-op
	})
	if out != "" {
		t.Errorf("optional NotFound/no-match must stay silent; got:\n%s", out)
	}
}

func TestLogListFailure_OptionalStillLogsForbidden(t *testing.T) {
	forbidden := apierrors.NewForbidden(
		schema.GroupResource{Resource: "llt-optional"}, "", errors.New("rbac says no"))
	out := captureLog(t, func() {
		logListFailure("llt-optional", forbidden, true)
	})
	if !strings.Contains(out, "list llt-optional failed") {
		t.Errorf("optional=true must still log Forbidden; got:\n%s", out)
	}
}

func TestLogListFailure_NonOptionalLogsNotFound(t *testing.T) {
	notFound := apierrors.NewNotFound(schema.GroupResource{Resource: "llt-core"}, "")
	out := captureLog(t, func() {
		logListFailure("llt-core", notFound, false)
	})
	if !strings.Contains(out, "list llt-core failed") {
		t.Errorf("core resources must log even NotFound; got:\n%s", out)
	}
}
