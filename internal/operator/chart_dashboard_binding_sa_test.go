// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

package operator

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// The dashboard (P6.6) runs under its OWN ServiceAccount
// (<fullname>-dashboard), distinct from the watcher SA. This is the
// dashboard analog of TestChartWatcherBindings_UseServiceAccountHelper:
// the dashboard ClusterRoleBinding must bind the dashboard SA, NOT the
// watcher SA (srenix.serviceAccountName) and NOT a bare srenix.fullname — the
// silence-binding bug (a binding pointing at a non-existent SA so the
// grant silently never lands) must not recur in a copy-paste.
//
// It also asserts the read-only RBAC contract structurally: the
// dashboard ClusterRole must carry ONLY get/list/watch and must NOT
// grant any mutate verb or touch Secrets — the dashboard is read-only by
// construction, and an accidental mutate verb here would be a privilege
// escalation no other gate catches (the chart↔operator RBAC parity test
// excludes the srenix.ai group).
func TestChartDashboardBinding_BindsDashboardSA(t *testing.T) {
	path := filepath.Join("..", "..", "charts", "agentic-sre", "templates", "dashboard-rbac.yaml")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read dashboard-rbac.yaml: %v", err)
	}
	src := string(b)

	// 1. Subject SA must be <fullname>-dashboard. The watcher guard
	//    requires srenix.serviceAccountName; the dashboard is the opposite
	//    case — its OWN SA, so it must NOT route through that helper.
	idx := strings.Index(src, "kind: ServiceAccount")
	if idx < 0 {
		t.Fatalf("dashboard-rbac.yaml: no ServiceAccount subject found — template shape changed; update this guard")
	}
	rest := src[idx:]
	saLine := regexp.MustCompile(`name:\s*(.+)`).FindStringSubmatch(rest)
	if saLine == nil {
		t.Fatalf("dashboard-rbac.yaml: could not parse the subject SA-name line — update this guard")
	}
	subj := strings.TrimSpace(saLine[1])
	if !strings.Contains(subj, `include "srenix.fullname" .`) || !strings.Contains(subj, "-dashboard") {
		t.Errorf("dashboard binding subject is %q, want the dashboard SA `{{ include \"srenix.fullname\" . }}-dashboard`. "+
			"The dashboard runs under its own SA; binding the wrong SA either silently drops the grant "+
			"(the silence-binding bug class) or over-grants the watcher SA.", subj)
	}
	if strings.Contains(subj, `include "srenix.serviceAccountName"`) {
		t.Errorf("dashboard binding subject uses srenix.serviceAccountName (the WATCHER SA); the dashboard must bind its OWN <fullname>-dashboard SA.")
	}

	// 2. Read-only RBAC contract: only get/list/watch verbs; no mutate
	//    verbs; no Secret access.
	forbiddenVerbs := []string{"create", "update", "patch", "delete", "deletecollection", "*"}
	for _, v := range forbiddenVerbs {
		if regexp.MustCompile(`"` + regexp.QuoteMeta(v) + `"`).MatchString(verbsBlock(src)) {
			t.Errorf("dashboard ClusterRole grants the mutate verb %q — the dashboard is read-only and must carry ONLY get/list/watch.", v)
		}
	}
	if strings.Contains(src, "secrets") {
		t.Errorf("dashboard ClusterRole references `secrets` — the dashboard must hold NO Secret access (no signing key).")
	}
	for _, want := range []string{`"get"`, `"list"`, `"watch"`} {
		if !strings.Contains(src, want) {
			t.Errorf("dashboard ClusterRole missing expected read verb %s", want)
		}
	}
	if !strings.Contains(src, "driftreports") {
		t.Errorf("dashboard ClusterRole must grant read on driftreports")
	}
}

// verbsBlock returns the substring of the rules region so the
// forbidden-verb scan ignores comment prose elsewhere in the file.
func verbsBlock(src string) string {
	start := strings.Index(src, "rules:")
	end := strings.Index(src, "kind: ClusterRoleBinding")
	if start < 0 || end < 0 || end < start {
		return src
	}
	return src[start:end]
}
