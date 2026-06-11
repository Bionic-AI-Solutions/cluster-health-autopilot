// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package operator

import (
	"reflect"
	"strings"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	chav1alpha1 "github.com/Bionic-AI-Solutions/cluster-health-autopilot/api/v1alpha1"
)

// Go-types ↔ CRD schema parity gate — the ROOT CAUSE check for the
// v1.24.0 pruning class.
//
// The chart + bundle CRDs are hand-maintained (the repo deliberately
// has no controller-gen / `make manifests` — see
// docs/design/2026-05-v1.8-operator-phase-1.md "Why hand-written
// DeepCopy"). That makes the failure mode structural: a field added to
// api/v1alpha1/clusterhealthautopilot_types.go without hand-porting it
// into the CRD schemas is silently PRUNED by schema-strict apiservers
// — exactly how v1.24.0 shipped with spec.watcher.triggers dropped on
// apply, and how the bundle CRD later lost spec.externalDNS.
//
// This test derives the EXPECTED property path set from the Go structs
// via reflection over json tags (recursing structs / slices / maps,
// honoring `json:",inline"` and `json:"-"`) and compares it against
// the chart CRD's openAPIV3Schema in BOTH directions:
//
//   - Go field without a CRD path  → FAIL (the v1.24.0 class: the
//     apiserver prunes the field on apply).
//   - CRD path without a Go field  → FAIL (dead schema: the CRD
//     validates a field no code consumes — usually a leftover after a
//     rename, or a typo'd port).
//
// The bundle CRD is transitively covered: TestBundle_CRDSchemasMatchChart
// pins bundle == chart, and this test pins chart == Go types.
//
// Legitimate exceptions go in crdTypesParityAllowlist below — per-path,
// with a documented reason. Keep it EMPTY unless apiextensions itself
// forces a divergence.

// crdTypesParityAllowlist maps "<direction>:<path>" → reason. Direction
// is "go-only" (Go field intentionally not in the CRD schema) or
// "crd-only" (CRD path with no Go field). Currently empty — every past
// divergence has been a real bug; think hard before adding entries.
var crdTypesParityAllowlist = map[string]string{}

func TestCRD_ChartSchemaMatchesGoTypes(t *testing.T) {
	chartCRDs := loadChartCRDs(t)
	cha, ok := chartCRDs["clusterhealthautopilots.cha.bionicaisolutions.com"]
	if !ok {
		t.Fatal("chart CRD for clusterhealthautopilots not found")
	}

	// Expected paths from the Go types. The CRD's root-level
	// apiVersion / kind / metadata properties are apiextensions
	// boilerplate (TypeMeta is inline, ObjectMeta's schema is owned by
	// the apiserver) — compare the spec and status subtrees, which are
	// the surfaces this repo hand-maintains.
	goPaths := map[string]string{}
	collectGoTypePaths(reflect.TypeOf(chav1alpha1.ClusterHealthAutopilotSpec{}), "spec", goPaths)
	collectGoTypePaths(reflect.TypeOf(chav1alpha1.ClusterHealthAutopilotStatus{}), "status", goPaths)
	goPaths["spec"] = "object"
	goPaths["status"] = "object"

	for _, v := range cha.crd.Spec.Versions {
		crdPaths := map[string]string{}
		collectSchemaPaths(v.Schema.OpenAPIV3Schema, "", crdPaths)
		// Root boilerplate properties — out of scope (see above).
		delete(crdPaths, "apiVersion")
		delete(crdPaths, "kind")
		delete(crdPaths, "metadata")

		for _, path := range sortedKeys(goPaths) {
			if _, ok := crdPaths[path]; ok {
				continue
			}
			if reason, ok := crdTypesParityAllowlist["go-only:"+path]; ok {
				t.Logf("allowlisted go-only path %q: %s", path, reason)
				continue
			}
			t.Errorf("Go types declare %q but the chart CRD (%s, version %s) has no matching schema path — schema-strict clusters will SILENTLY PRUNE the field on apply (the v1.24.0 class); hand-port the subtree into the chart CRD AND bundle/manifests/cha.bionicaisolutions.com_clusterhealthautopilots.yaml",
				path, cha.path, v.Name)
		}
		for _, path := range sortedKeys(crdPaths) {
			if _, ok := goPaths[path]; ok {
				continue
			}
			if reason, ok := crdTypesParityAllowlist["crd-only:"+path]; ok {
				t.Logf("allowlisted crd-only path %q: %s", path, reason)
				continue
			}
			t.Errorf("chart CRD (%s, version %s) declares schema path %q but the Go types in api/v1alpha1 have no matching field — dead schema; remove it from the CRD (and the bundle CRD) or add the field to the types",
				cha.path, v.Name, path)
		}
		// Type agreement on shared paths.
		for _, path := range sortedKeys(goPaths) {
			ct, ok := crdPaths[path]
			if !ok || ct == goPaths[path] {
				continue
			}
			t.Errorf("schema path %q: Go types imply OpenAPI type %q but the chart CRD declares %q", path, goPaths[path], ct)
		}
	}
}

// collectGoTypePaths walks a Go struct type and records every json
// property path with its OpenAPI type, using the same notation as
// collectSchemaPaths ("[]" for slice elements, "{}" for map values).
func collectGoTypePaths(t reflect.Type, prefix string, out map[string]string) {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	// Types that marshal to a JSON scalar (metav1.Time → RFC3339
	// string) must not be recursed into.
	if t == reflect.TypeOf(metav1.Time{}) {
		out[prefix] = "string"
		return
	}
	switch t.Kind() {
	case reflect.Struct:
		out[prefix] = "object"
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			if f.PkgPath != "" { // unexported
				continue
			}
			tag := f.Tag.Get("json")
			name, _, _ := strings.Cut(tag, ",")
			if name == "-" {
				continue
			}
			if name == "" {
				if strings.Contains(tag, ",inline") || f.Anonymous {
					collectGoTypePaths(f.Type, prefix, out)
					continue
				}
				name = f.Name
			}
			collectGoTypePaths(f.Type, prefix+"."+name, out)
		}
	case reflect.Slice, reflect.Array:
		out[prefix] = "array"
		collectGoTypePaths(t.Elem(), prefix+"[]", out)
	case reflect.Map:
		out[prefix] = "object"
		collectGoTypePaths(t.Elem(), prefix+"{}", out)
	case reflect.String:
		out[prefix] = "string"
	case reflect.Bool:
		out[prefix] = "boolean"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		out[prefix] = "integer"
	case reflect.Float32, reflect.Float64:
		out[prefix] = "number"
	default:
		out[prefix] = "unknown:" + t.Kind().String()
	}
	// "object" entries above overwrite struct re-walks consistently;
	// nothing else to do.
}
