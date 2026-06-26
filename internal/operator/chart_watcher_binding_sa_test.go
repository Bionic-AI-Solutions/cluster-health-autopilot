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

// Every chart ClusterRoleBinding that grants the WATCHER ServiceAccount a
// read role must reference it via the `srenix.serviceAccountName` helper — the
// actual SA the watcher Deployment runs as (`<fullname>-sa`). A binding that
// uses `srenix.fullname` (no `-sa`) points at a non-existent SA, so the grant
// silently never reaches the watcher. This regressed on clusterrole-silence
// (caught only by live RBAC enforcement: `silences ... is forbidden`, silence
// filtering skipped every cycle) because the chart↔operator RBAC parity test
// excludes the srenix.ai group. This file is the cheap guard.
func TestChartWatcherBindings_UseServiceAccountHelper(t *testing.T) {
	// Binding templates that bind the watcher SA to a watcher read role.
	watcherBindingFiles := []string{
		"clusterrole-reader.yaml",
		"clusterrole-driftreport.yaml",
		"clusterrole-resolutionrecord.yaml",
		"clusterrole-silence.yaml",
	}
	dir := filepath.Join("..", "..", "charts", "agentic-sre", "templates")
	// Subject SA-name line inside a `kind: ServiceAccount` block.
	saName := regexp.MustCompile(`name:\s*\{\{\s*include\s+"srenix\.(\w+)"`)
	for _, f := range watcherBindingFiles {
		path := filepath.Join(dir, f)
		b, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", f, err)
		}
		src := string(b)
		idx := strings.Index(src, "kind: ServiceAccount")
		if idx < 0 {
			t.Fatalf("%s: no ServiceAccount subject found — template shape changed; update this guard", f)
		}
		// The SA name line follows the kind line within the subjects block.
		rest := src[idx:]
		m := saName.FindStringSubmatch(rest)
		if m == nil {
			t.Fatalf("%s: could not parse the subject SA-name helper — update this guard", f)
		}
		if m[1] != "serviceAccountName" {
			t.Errorf("%s: watcher binding references the SA via srenix.%s, but the watcher runs as srenix.serviceAccountName (<fullname>-sa). "+
				"srenix.fullname points at a non-existent SA, so the grant silently never reaches the watcher (live symptom: \"... is forbidden\"). "+
				"Use {{ include \"srenix.serviceAccountName\" . }}.", f, m[1])
		}
	}
}
