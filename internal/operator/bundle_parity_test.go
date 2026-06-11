// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package operator

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"sigs.k8s.io/yaml"

	chav1alpha1 "github.com/Bionic-AI-Solutions/cluster-health-autopilot/api/v1alpha1"
)

// Phase 1c bundle/chart parity guard.
//
// The OLM ClusterServiceVersion at bundle/manifests/cha.clusterservice
// version.yaml ships the operator's RBAC + Deployment install strategy.
// The same install strategy is ALSO described in the helm chart at
// charts/.../templates/operator-deployment.yaml. If the two drift,
// `helm install` and `operator-sdk run bundle` produce structurally
// different operators — same image, different perms, hard-to-debug
// authorization failures.
//
// These tests don't lock down byte-equality (the helm template is
// dynamic, the CSV is not) — they catch the drift patterns that have
// actually bitten in the past: a new ClusterRole rule added to the
// chart that wasn't copied to the CSV.

const (
	csvPath          = "../../bundle/manifests/cha.clusterserviceversion.yaml"
	operatorTplPath  = "../../charts/cluster-health-autopilot/templates/operator-deployment.yaml"
	bundleDockerfile = "../../bundle.Dockerfile"
	chartTplDir      = "../../charts/cluster-health-autopilot/templates"
)

func TestBundle_CSVIsValidYAML(t *testing.T) {
	raw, err := os.ReadFile(csvPath)
	if err != nil {
		t.Fatalf("read CSV: %v", err)
	}
	var csv map[string]any
	if err := yaml.Unmarshal(raw, &csv); err != nil {
		t.Fatalf("CSV is not valid YAML: %v", err)
	}
	if got := csv["kind"]; got != "ClusterServiceVersion" {
		t.Errorf("CSV kind = %v; want ClusterServiceVersion", got)
	}
}

func TestBundle_CSVDeclaresAllShippedCRDs(t *testing.T) {
	// The CSV's spec.customresourcedefinitions.owned must list every
	// CRD shipped in bundle/manifests/. Drift here = an operator install
	// that's missing schema validation for a CRD the chart still installs.
	raw, _ := os.ReadFile(csvPath)
	var csv struct {
		Spec struct {
			CustomResourceDefinitions struct {
				Owned []struct {
					Name string `yaml:"name"`
				} `yaml:"owned"`
			} `yaml:"customresourcedefinitions"`
		} `yaml:"spec"`
	}
	if err := yaml.Unmarshal(raw, &csv); err != nil {
		t.Fatalf("decode CSV: %v", err)
	}
	declared := make(map[string]bool)
	for _, c := range csv.Spec.CustomResourceDefinitions.Owned {
		declared[c.Name] = true
	}

	// Discover what CRD manifests ship in the bundle (by metadata.name).
	entries, err := filepath.Glob(filepath.Dir(csvPath) + "/cha.bionicaisolutions.com_*.yaml")
	if err != nil {
		t.Fatalf("glob CRD manifests: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("no CRD manifests found in bundle/manifests/ — bundle is empty")
	}
	for _, p := range entries {
		raw, _ := os.ReadFile(p)
		var crd struct {
			Metadata struct {
				Name string `yaml:"name"`
			} `yaml:"metadata"`
		}
		if err := yaml.Unmarshal(raw, &crd); err != nil {
			t.Errorf("CRD %s: bad YAML: %v", p, err)
			continue
		}
		if !declared[crd.Metadata.Name] {
			t.Errorf("CRD %q ships in bundle/manifests/ but is NOT listed under CSV's customresourcedefinitions.owned",
				crd.Metadata.Name)
		}
	}
}

func TestBundle_CSVAndChartRBACInLockstep(t *testing.T) {
	// Both files declare the operator's own ClusterRole rules. If the
	// chart grows a new apiGroup/resources rule (e.g. for a new
	// reconciler subresource) and the CSV doesn't, the operator will
	// install via OLM but lack the perms it has under helm. Catch the
	// drift here as a low-effort parity guard.
	//
	// We compare the SET of (apiGroup, resource) keys present in each.
	// Verb-set comparison is fragile (chart uses YAML lists, CSV uses
	// CSV-list); the (group,resource) tuple is the strongest signal of
	// "did someone add a new rule and forget the other file?"
	chartRules := extractRuleKeys(t, operatorTplPath)
	csvRules := extractCSVRuleKeys(t, csvPath)

	for k := range chartRules {
		if !csvRules[k] {
			t.Errorf("chart operator-deployment.yaml has rule %s but CSV doesn't — add to CSV.spec.install.spec.clusterPermissions[0].rules", k)
		}
	}
	for k := range csvRules {
		if !chartRules[k] {
			t.Errorf("CSV has rule %s but chart operator-deployment.yaml doesn't — add to chart's ClusterRole rules", k)
		}
	}
}

// TestBundle_CRDSchemasMatchChart is the structural-schema parity gate
// between the OLM bundle CRDs and the helm chart CRDs.
//
// Both copies are hand-maintained. v1.24.0 shipped with the chart CRD
// carrying spec.watcher.triggers while the cluster's installed CRD
// didn't — schema-strict apiservers silently PRUNED the triggers block
// from applied CRs. The same class recurred: the bundle CRD lost the
// entire spec.externalDNS subtree relative to the chart CRD.
//
// This test walks spec.versions[*].schema.openAPIV3Schema of every
// bundle CRD and its chart counterpart recursively, collects the full
// set of property paths (descending into properties, items and
// additionalProperties) plus each path's declared type, and asserts
// the two sets are identical in BOTH directions. A path present in
// the chart but not the bundle = OLM installs prune it (the v1.24.0
// class); a path present in the bundle but not the chart = chart
// installs prune it.
func TestBundle_CRDSchemasMatchChart(t *testing.T) {
	chartCRDs := loadChartCRDs(t)

	bundleFiles, err := filepath.Glob(filepath.Dir(csvPath) + "/cha.bionicaisolutions.com_*.yaml")
	if err != nil {
		t.Fatalf("glob bundle CRD manifests: %v", err)
	}
	if len(bundleFiles) == 0 {
		t.Fatal("no CRD manifests found in bundle/manifests/")
	}

	seen := make(map[string]bool)
	for _, p := range bundleFiles {
		raw, err := os.ReadFile(p)
		if err != nil {
			t.Fatalf("read %s: %v", p, err)
		}
		bundle := parseCRDDoc(t, raw, p)
		chart, ok := chartCRDs[bundle.Metadata.Name]
		if !ok {
			t.Errorf("bundle CRD %q (%s) has no chart counterpart under %s", bundle.Metadata.Name, p, chartTplDir)
			continue
		}
		seen[bundle.Metadata.Name] = true

		chartVersions := make(map[string]map[string]any, len(chart.crd.Spec.Versions))
		for _, v := range chart.crd.Spec.Versions {
			chartVersions[v.Name] = v.Schema.OpenAPIV3Schema
		}
		for _, bv := range bundle.Spec.Versions {
			chartSchema, ok := chartVersions[bv.Name]
			if !ok {
				t.Errorf("CRD %s: bundle serves version %q but chart CRD %s doesn't", bundle.Metadata.Name, bv.Name, chart.path)
				continue
			}
			delete(chartVersions, bv.Name)

			bundlePaths := map[string]string{}
			collectSchemaPaths(bv.Schema.OpenAPIV3Schema, "", bundlePaths)
			chartPaths := map[string]string{}
			collectSchemaPaths(chartSchema, "", chartPaths)

			for _, path := range sortedKeys(chartPaths) {
				if _, ok := bundlePaths[path]; !ok {
					t.Errorf("CRD %s %s: schema path %q exists in chart CRD (%s) but is MISSING from the bundle CRD — schema-strict clusters will silently prune it on OLM installs; hand-port the subtree into (no CRD generation tooling exists, by design) %s",
						bundle.Metadata.Name, bv.Name, path, chart.path, p)
				}
			}
			for _, path := range sortedKeys(bundlePaths) {
				if _, ok := chartPaths[path]; !ok {
					t.Errorf("CRD %s %s: schema path %q exists in bundle CRD (%s) but is MISSING from the chart CRD — schema-strict clusters will silently prune it on helm installs; hand-port the subtree into (no CRD generation tooling exists, by design) %s",
						bundle.Metadata.Name, bv.Name, path, p, chart.path)
				}
			}
			for _, path := range sortedKeys(bundlePaths) {
				if ct, ok := chartPaths[path]; ok && ct != bundlePaths[path] {
					t.Errorf("CRD %s %s: schema path %q has type %q in the bundle CRD but %q in the chart CRD (%s vs %s)",
						bundle.Metadata.Name, bv.Name, path, bundlePaths[path], ct, p, chart.path)
				}
			}
		}
		for v := range chartVersions {
			t.Errorf("CRD %s: chart CRD %s serves version %q but the bundle CRD %s doesn't", bundle.Metadata.Name, chart.path, v, p)
		}
	}
	for name, c := range chartCRDs {
		if !seen[name] {
			t.Errorf("chart ships CRD %q (%s) with no bundle counterpart in bundle/manifests/", name, c.path)
		}
	}
}

// crdDoc is the slice of a CRD manifest the schema-parity test needs.
type crdDoc struct {
	Metadata struct {
		Name string `json:"name"`
	} `json:"metadata"`
	Spec struct {
		Versions []struct {
			Name   string `json:"name"`
			Schema struct {
				OpenAPIV3Schema map[string]any `json:"openAPIV3Schema"`
			} `json:"schema"`
		} `json:"versions"`
	} `json:"spec"`
}

type chartCRD struct {
	path string
	crd  crdDoc
}

// loadChartCRDs parses every crd-*.yaml helm template, stripping the
// {{ ... }} templating lines first (same approach as chartReaderRules
// in chart_rbac_parity_test.go — the CRD bodies are static YAML).
// Keyed by metadata.name.
func loadChartCRDs(t *testing.T) map[string]chartCRD {
	t.Helper()
	files, err := filepath.Glob(chartTplDir + "/crd-*.yaml")
	if err != nil {
		t.Fatalf("glob chart CRD templates: %v", err)
	}
	if len(files) == 0 {
		t.Fatalf("no crd-*.yaml templates found under %s", chartTplDir)
	}
	out := make(map[string]chartCRD, len(files))
	for _, p := range files {
		raw, err := os.ReadFile(p)
		if err != nil {
			t.Fatalf("read %s: %v", p, err)
		}
		crd := parseCRDDoc(t, stripHelmTemplating(raw), p)
		if crd.Metadata.Name == "" {
			t.Fatalf("chart CRD %s parsed without metadata.name — template shape changed; update loadChartCRDs()", p)
		}
		out[crd.Metadata.Name] = chartCRD{path: p, crd: crd}
	}
	return out
}

func parseCRDDoc(t *testing.T, raw []byte, path string) crdDoc {
	t.Helper()
	var crd crdDoc
	if err := yaml.Unmarshal(raw, &crd); err != nil {
		t.Fatalf("parse CRD %s: %v", path, err)
	}
	if len(crd.Spec.Versions) == 0 {
		t.Fatalf("CRD %s has no spec.versions — parse shape changed", path)
	}
	return crd
}

// stripHelmTemplating drops every line carrying a {{ ... }} action.
// The chart's CRD templates are static YAML apart from the install
// conditional + the cha.labels include, so the remainder parses.
func stripHelmTemplating(raw []byte) []byte {
	var b strings.Builder
	for _, line := range strings.Split(string(raw), "\n") {
		if strings.Contains(line, "{{") {
			continue
		}
		b.WriteString(line)
		b.WriteString("\n")
	}
	return []byte(b.String())
}

// collectSchemaPaths walks an openAPIV3Schema node recursively and
// records every property path with its declared type. Array item
// schemas are recorded under "<path>[]"; additionalProperties value
// schemas under "<path>{}".
func collectSchemaPaths(node map[string]any, prefix string, out map[string]string) {
	if prefix != "" {
		typ, _ := node["type"].(string)
		out[prefix] = typ
	}
	if props, ok := node["properties"].(map[string]any); ok {
		for k, v := range props {
			child, ok := v.(map[string]any)
			if !ok {
				continue
			}
			p := k
			if prefix != "" {
				p = prefix + "." + k
			}
			collectSchemaPaths(child, p, out)
		}
	}
	if items, ok := node["items"].(map[string]any); ok {
		collectSchemaPaths(items, prefix+"[]", out)
	}
	if ap, ok := node["additionalProperties"].(map[string]any); ok {
		collectSchemaPaths(ap, prefix+"{}", out)
	}
}

func sortedKeys(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// TestBundle_SampleCRsMatchGoTypes strict-decodes every smoke sample CR
// against the Go types. A typo'd field name in a sample would otherwise
// surface only as a far-away jsonpath failure inside the kind smoke job
// (or worse, silently weaken the pruning detection).
func TestBundle_SampleCRsMatchGoTypes(t *testing.T) {
	files, err := filepath.Glob("../../bundle/tests/sample-cr*.yaml")
	if err != nil {
		t.Fatalf("glob sample CRs: %v", err)
	}
	if len(files) < 3 {
		t.Fatalf("expected ≥3 sample CRs under bundle/tests/ (minimal, ai, full); found %d", len(files))
	}
	for _, p := range files {
		raw, err := os.ReadFile(p)
		if err != nil {
			t.Fatalf("read %s: %v", p, err)
		}
		var cr chav1alpha1.ClusterHealthAutopilot
		if err := yaml.UnmarshalStrict(raw, &cr); err != nil {
			t.Errorf("sample CR %s does not strict-decode into v1alpha1.ClusterHealthAutopilot: %v", p, err)
		}
	}
}

// TestBundle_FullSurfaceSampleCRCoversEverySpecPath pins the "full
// surface" property of bundle/tests/sample-cr-full.yaml: every spec
// property path the chart CRD schema declares must be SET in the
// sample. The bundle-smoke pruning detection only catches a missing
// CRD subtree if the sample actually exercises it — so a new spec
// field added to the CRDs without extending sample-cr-full.yaml fails
// here instead of silently shrinking the smoke's coverage.
func TestBundle_FullSurfaceSampleCRCoversEverySpecPath(t *testing.T) {
	chartCRDs := loadChartCRDs(t)
	cha, ok := chartCRDs["clusterhealthautopilots.cha.bionicaisolutions.com"]
	if !ok {
		t.Fatal("chart CRD for clusterhealthautopilots not found")
	}

	raw, err := os.ReadFile("../../bundle/tests/sample-cr-full.yaml")
	if err != nil {
		t.Fatalf("read sample-cr-full.yaml: %v", err)
	}
	var instance map[string]any
	if err := yaml.Unmarshal(raw, &instance); err != nil {
		t.Fatalf("parse sample-cr-full.yaml: %v", err)
	}
	set := map[string]bool{}
	collectInstancePaths(instance, "", set)

	for _, v := range cha.crd.Spec.Versions {
		schemaPaths := map[string]string{}
		collectSchemaPaths(v.Schema.OpenAPIV3Schema, "", schemaPaths)
		for _, path := range sortedKeys(schemaPaths) {
			if !strings.HasPrefix(path, "spec.") {
				continue // apiVersion/kind/metadata/status — not spec surface.
			}
			if set[path] {
				continue
			}
			// additionalProperties (map-valued) paths: the instance walk
			// records user-chosen keys, not the "{}" marker — covered as
			// long as the map itself is set.
			if i := strings.Index(path, "{}"); i >= 0 && set[path[:i]] {
				continue
			}
			t.Errorf("chart CRD %s declares spec path %q but bundle/tests/sample-cr-full.yaml does not set it — extend the full-surface sample so the bundle-smoke pruning detection keeps covering it",
				v.Name, path)
		}
	}
}

// collectInstancePaths walks a decoded YAML instance and records every
// path that is SET, using the same "[]" array notation as
// collectSchemaPaths (array elements are unioned).
func collectInstancePaths(v any, prefix string, out map[string]bool) {
	if prefix != "" {
		out[prefix] = true
	}
	switch n := v.(type) {
	case map[string]any:
		for k, c := range n {
			p := k
			if prefix != "" {
				p = prefix + "." + k
			}
			collectInstancePaths(c, p, out)
		}
	case []any:
		for _, c := range n {
			collectInstancePaths(c, prefix+"[]", out)
		}
	}
}

func TestBundle_DockerfileExists(t *testing.T) {
	if _, err := os.Stat(bundleDockerfile); err != nil {
		t.Errorf("bundle.Dockerfile missing or unreadable: %v", err)
	}
}

// extractRuleKeys walks a helm chart YAML and returns the set of
// (apiGroup,resource) keys that appear in `apiGroups: [...]` /
// `resources: [...]` pairs. It only looks at top-level rules (those
// with both fields adjacent under `rules:`), which is fine for the
// operator-deployment.yaml shape.
func extractRuleKeys(t *testing.T, path string) map[string]bool {
	t.Helper()
	out := make(map[string]bool)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	// Walk the file extracting (apiGroups,resources) pairs in order.
	lines := strings.Split(string(raw), "\n")
	var currentGroups []string
	for _, line := range lines {
		l := strings.TrimSpace(line)
		if strings.HasPrefix(l, "apiGroups:") {
			currentGroups = parseInlineList(l[len("apiGroups:"):])
		}
		if strings.HasPrefix(l, "resources:") && len(currentGroups) > 0 {
			for _, r := range parseInlineList(l[len("resources:"):]) {
				for _, g := range currentGroups {
					out[g+"/"+r] = true
				}
			}
			currentGroups = nil
		}
	}
	return out
}

// extractCSVRuleKeys is identical to extractRuleKeys but expects the
// CSV's indentation (the rules sit under
// spec.install.spec.clusterPermissions[0].rules). The parser is
// indentation-agnostic — same approach works.
func extractCSVRuleKeys(t *testing.T, path string) map[string]bool {
	return extractRuleKeys(t, path)
}

func parseInlineList(s string) []string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "[")
	s = strings.TrimSuffix(s, "]")
	if s == "" {
		return nil
	}
	var out []string
	for _, p := range strings.Split(s, ",") {
		p = strings.TrimSpace(p)
		p = strings.Trim(p, `"`)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
