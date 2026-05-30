// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package operator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"sigs.k8s.io/yaml"
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
