// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

package report

import (
	"context"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// TestNormalizeSeverity is the regression test for the production bug where
// an analyzer emitted `severity: "warn"` and every DriftReport reconcile
// cycle failed CRD enum validation
// (`spec.severity: Unsupported value: "warn"`).
func TestNormalizeSeverity(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		// Enum values pass through untouched.
		{"info", "info"},
		{"warning", "warning"},
		{"critical", "critical"},
		// Known aliases map to the closest enum value.
		{"warn", "warning"},
		{"error", "critical"},
		{"err", "critical"},
		{"fatal", "critical"},
		{"crit", "critical"},
		// Case / whitespace robustness.
		{"WARN", "warning"},
		{"Error", "critical"},
		{" warning ", "warning"},
		// Empty: optional field, defaults to warning (matches the
		// AssembleEntries backwards-compat default).
		{"", "warning"},
		// Unknown values fail safe to warning so reconcile never breaks.
		{"sev1", "warning"},
		{"notice", "warning"},
	}
	for _, tc := range cases {
		if got := NormalizeSeverity(tc.in, "TestSource"); got != tc.want {
			t.Errorf("NormalizeSeverity(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// TestReconcile_NormalizesNonEnumSeverity is the defense-in-depth guard:
// even if a future emitter sneaks a non-enum severity past the source-level
// lint, Reconcile must never send a spec.severity the CRD enum rejects —
// on BOTH the create path and the per-cycle spec-refresh patch path.
func TestReconcile_NormalizesNonEnumSeverity(t *testing.T) {
	t.Run("create path maps warn→warning", func(t *testing.T) {
		src := &fakeSrc{}
		m := &fakeMutator{}
		entries := []DriftReportEntry{{Subject: "Host/x", Severity: "warn", Source: "DNSChainDrift", Category: "analyzer", Message: "m"}}
		if _, _, _, err := Reconcile(context.Background(), src, m, entries, "run-1"); err != nil {
			t.Fatalf("err: %v", err)
		}
		if len(m.created) != 1 {
			t.Fatalf("want 1 created CR, got %d", len(m.created))
		}
		sev, _, _ := unstructured.NestedString(m.created[0].Object, "spec", "severity")
		if sev != "warning" {
			t.Errorf("spec.severity = %q, want %q", sev, "warning")
		}
		lbl := m.created[0].GetLabels()["srenix.ai/severity"]
		if lbl != "warning" {
			t.Errorf("severity label = %q, want %q", lbl, "warning")
		}
	})

	t.Run("update path maps error→critical", func(t *testing.T) {
		const subj = "Host/y"
		src := &fakeSrc{existing: []unstructured.Unstructured{driftCR(subj)}}
		m := &fakeMutator{}
		entries := []DriftReportEntry{{Subject: subj, Severity: "error", Source: "DNSChainDrift", Category: "analyzer", Message: "m"}}
		if _, _, _, err := Reconcile(context.Background(), src, m, entries, "run-1"); err != nil {
			t.Fatalf("err: %v", err)
		}
		key := "Patch driftreports//" + nameForSubject(subj)
		body, ok := m.patchBodies[key]
		if !ok {
			t.Fatalf("no spec patch recorded for %s; calls=%v", key, m.calls)
		}
		patch := decodePatchBody(t, body)
		spec, _ := patch["spec"].(map[string]any)
		if spec == nil {
			t.Fatalf("patch has no spec: %s", string(body))
		}
		if spec["severity"] != "critical" {
			t.Errorf("patched spec.severity = %v, want %q", spec["severity"], "critical")
		}
	})
}

// TestSeverityLiteralsAreEnumValues statically walks every non-test Go file
// in the module's emitter/consumer trees (internal/, catalog/, pkg/, cmd/,
// api/) and asserts that every string literal assigned to a `Severity` or
// `severity` identifier (exported/unexported struct fields, local variables)
// is one of the DriftReport CRD enum values (info|warning|critical).
//
// This is the strongest feasible "walk all probes/analyzers" guard: probe
// severities are typed constants (pkg/probe.Severity), but analyzer
// Diagnostics carry free-form strings — a single `Severity: "warn"` literal
// in a new analyzer breaks every DriftReport reconcile cycle against the
// CRD enum. This test makes that a compile-adjacent failure instead of a
// production incident.
func TestSeverityLiteralsAreEnumValues(t *testing.T) {
	valid := map[string]bool{"info": true, "warning": true, "critical": true}

	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	// internal/report/severity_test.go → repo root is two dirs up.
	root := filepath.Dir(filepath.Dir(filepath.Dir(thisFile)))

	var violations []string
	fset := token.NewFileSet()
	for _, dir := range []string{"internal", "catalog", "pkg", "cmd", "api"} {
		base := filepath.Join(root, dir)
		err := filepath.WalkDir(base, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			f, perr := parser.ParseFile(fset, path, nil, 0)
			if perr != nil {
				return perr
			}
			ast.Inspect(f, func(n ast.Node) bool {
				switch x := n.(type) {
				case *ast.KeyValueExpr:
					// Severity: "..." / severity: "..." (unexported field)
					id, isIdent := x.Key.(*ast.Ident)
					if !isIdent || !isSeverityName(id.Name) {
						return true
					}
					checkSeverityLit(fset, x.Value, valid, &violations)
				case *ast.AssignStmt:
					// foo.Severity = "..." / severity := "..." / severity = "..."
					for i, lhs := range x.Lhs {
						name := ""
						switch l := lhs.(type) {
						case *ast.SelectorExpr:
							name = l.Sel.Name
						case *ast.Ident:
							name = l.Name
						}
						if !isSeverityName(name) || i >= len(x.Rhs) {
							continue
						}
						checkSeverityLit(fset, x.Rhs[i], valid, &violations)
					}
				}
				return true
			})
			return nil
		})
		if err != nil {
			t.Fatalf("walking %s: %v", base, err)
		}
	}

	if len(violations) > 0 {
		t.Errorf("found %d severity literal(s) outside the DriftReport CRD enum (info|warning|critical);\n"+
			"these break the watcher's driftreport reconcile (spec.severity: Unsupported value):\n  %s",
			len(violations), strings.Join(violations, "\n  "))
	}
}

// isSeverityName reports whether an identifier names a severity slot the
// lint must guard: the exported `Severity` field and the lowercase
// `severity` form (local vars like `severity = "warn"`, unexported struct
// fields like `severity: "..."`).
func isSeverityName(name string) bool {
	return name == "Severity" || name == "severity"
}

func checkSeverityLit(fset *token.FileSet, v ast.Expr, valid map[string]bool, violations *[]string) {
	lit, isLit := v.(*ast.BasicLit)
	if !isLit || lit.Kind != token.STRING {
		return
	}
	s, err := strconv.Unquote(lit.Value)
	if err != nil {
		return
	}
	// Empty is a documented optional value (defaulted downstream).
	if s == "" || valid[s] {
		return
	}
	*violations = append(*violations, fset.Position(lit.Pos()).String()+": Severity literal "+lit.Value)
}
