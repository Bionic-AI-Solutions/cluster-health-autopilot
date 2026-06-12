// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package operator

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"sigs.k8s.io/yaml"
)

// Printer-column parity gate (O8 / K1).
//
// `kubectl get` output is part of the CRD contract: the K1
// investigation found `kubectl get silences` rendering UNTIL as
// `<invalid>` because the column was `type: date` — kubectl renders
// date columns as AGE-SINCE, which is negative for the future expiry
// every active Silence has by definition. The schema-parity gate
// (TestBundle_CRDSchemasMatchChart) walks openAPIV3Schema only, so
// printer columns could drift between chart / bundle / Go markers
// without any test noticing.
//
// Three assertions:
//  1. chart CRD columns == bundle CRD columns (name/type/JSONPath,
//     in order) for every CRD + version;
//  2. Go +kubebuilder:printcolumn markers == chart columns for types
//     that declare markers (hand-maintained, no controller-gen — the
//     markers are documentation that MUST match the shipped YAML);
//  3. no `type: date` column may point at a future-dated field
//     (until / expiry / deadline / notAfter) — that's the `<invalid>`
//     bug class itself.

type printColumn struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	JSONPath string `json:"jsonPath"`
}

func (c printColumn) String() string {
	return fmt.Sprintf("{name=%s type=%s jsonPath=%s}", c.Name, c.Type, c.JSONPath)
}

// crdColumnsDoc is the slice of a CRD manifest the printer-column gate
// needs (kept separate from crdDoc to avoid widening the schema gate's
// parse surface).
type crdColumnsDoc struct {
	Metadata struct {
		Name string `json:"name"`
	} `json:"metadata"`
	Spec struct {
		Versions []struct {
			Name                     string        `json:"name"`
			AdditionalPrinterColumns []printColumn `json:"additionalPrinterColumns"`
		} `json:"versions"`
	} `json:"spec"`
}

func parseCRDColumns(t *testing.T, raw []byte, path string) crdColumnsDoc {
	t.Helper()
	var crd crdColumnsDoc
	if err := yaml.Unmarshal(raw, &crd); err != nil {
		t.Fatalf("parse CRD %s: %v", path, err)
	}
	if crd.Metadata.Name == "" {
		t.Fatalf("CRD %s parsed without metadata.name — parse shape changed", path)
	}
	return crd
}

// versionColumns returns version → columns for a CRD doc.
func versionColumns(d crdColumnsDoc) map[string][]printColumn {
	out := map[string][]printColumn{}
	for _, v := range d.Spec.Versions {
		out[v.Name] = v.AdditionalPrinterColumns
	}
	return out
}

func TestCRD_PrinterColumnsChartBundleParity(t *testing.T) {
	chartFiles, err := filepath.Glob(chartTplDir + "/crd-*.yaml")
	if err != nil || len(chartFiles) == 0 {
		t.Fatalf("glob chart CRD templates: %v (%d files)", err, len(chartFiles))
	}
	chartCols := map[string]crdColumnsDoc{}
	for _, p := range chartFiles {
		raw, rerr := os.ReadFile(p)
		if rerr != nil {
			t.Fatalf("read %s: %v", p, rerr)
		}
		d := parseCRDColumns(t, stripHelmTemplating(raw), p)
		chartCols[d.Metadata.Name] = d
	}

	bundleFiles, err := filepath.Glob(filepath.Dir(csvPath) + "/cha.bionicaisolutions.com_*.yaml")
	if err != nil || len(bundleFiles) == 0 {
		t.Fatalf("glob bundle CRD manifests: %v (%d files)", err, len(bundleFiles))
	}
	for _, p := range bundleFiles {
		raw, rerr := os.ReadFile(p)
		if rerr != nil {
			t.Fatalf("read %s: %v", p, rerr)
		}
		bundle := parseCRDColumns(t, raw, p)
		chart, ok := chartCols[bundle.Metadata.Name]
		if !ok {
			continue // missing counterpart is the schema gate's finding
		}
		chartByVersion := versionColumns(chart)
		for v, bcols := range versionColumns(bundle) {
			ccols, ok := chartByVersion[v]
			if !ok {
				continue // version drift is the schema gate's finding
			}
			if len(bcols) != len(ccols) {
				t.Errorf("CRD %s %s: %d printer columns in bundle vs %d in chart — hand-port the change to both", bundle.Metadata.Name, v, len(bcols), len(ccols))
				continue
			}
			for i := range bcols {
				if bcols[i] != ccols[i] {
					t.Errorf("CRD %s %s: printer column %d diverged — bundle %v vs chart %v", bundle.Metadata.Name, v, i, bcols[i], ccols[i])
				}
			}
		}
	}
}

// markerRe matches the hand-maintained kubebuilder printcolumn markers
// in api/v1alpha1. No controller-gen runs in this repo, so the markers
// are documentation — this gate keeps them truthful against the
// shipped chart CRD.
var markerRe = regexp.MustCompile("\\+kubebuilder:printcolumn:name=\"([^\"]+)\",type=([a-z]+),JSONPath=`([^`]+)`")

// markerSources maps an api/v1alpha1 source file to the chart CRD it
// describes. Only files that DECLARE printcolumn markers belong here.
var markerSources = map[string]string{
	"../../api/v1alpha1/silence_types.go": "silences.cha.bionicaisolutions.com",
}

func TestCRD_PrinterColumnsMatchGoMarkers(t *testing.T) {
	for src, crdName := range markerSources {
		raw, err := os.ReadFile(src)
		if err != nil {
			t.Fatalf("read %s: %v", src, err)
		}
		var markers []printColumn
		for _, m := range markerRe.FindAllStringSubmatch(string(raw), -1) {
			markers = append(markers, printColumn{Name: m[1], Type: m[2], JSONPath: m[3]})
		}
		if len(markers) == 0 {
			t.Fatalf("%s declares no printcolumn markers — update markerSources", src)
		}

		chartPath := chartTplDir + "/crd-silence.yaml"
		rawChart, err := os.ReadFile(chartPath)
		if err != nil {
			t.Fatalf("read %s: %v", chartPath, err)
		}
		chart := parseCRDColumns(t, stripHelmTemplating(rawChart), chartPath)
		if chart.Metadata.Name != crdName {
			t.Fatalf("chart CRD %s is %q, expected %q", chartPath, chart.Metadata.Name, crdName)
		}
		for v, cols := range versionColumns(chart) {
			if len(cols) != len(markers) {
				t.Errorf("%s %s: %d printer columns in chart vs %d markers in %s", crdName, v, len(cols), len(markers), src)
				continue
			}
			for i := range cols {
				if cols[i] != markers[i] {
					t.Errorf("%s %s: printer column %d diverged — chart %v vs Go marker %v (%s)", crdName, v, i, cols[i], markers[i], src)
				}
			}
		}
	}
}

// futureFieldRe matches JSONPaths whose terminal field is, by naming
// convention, a FUTURE timestamp. `type: date` renders age-since —
// negative ("<invalid>") for future values — so such columns must be
// type: string (raw RFC3339). Past-event timestamps (lastObserved,
// recordedAt, creationTimestamp) are legitimately type: date.
var futureFieldRe = regexp.MustCompile(`(?i)(until|expir|deadline|notafter)`)

func TestCRD_NoDateColumnsOnFutureTimestamps(t *testing.T) {
	files, err := filepath.Glob(chartTplDir + "/crd-*.yaml")
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	more, err := filepath.Glob(filepath.Dir(csvPath) + "/cha.bionicaisolutions.com_*.yaml")
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	files = append(files, more...)
	for _, p := range files {
		raw, rerr := os.ReadFile(p)
		if rerr != nil {
			t.Fatalf("read %s: %v", p, rerr)
		}
		if strings.Contains(p, chartTplDir) {
			raw = stripHelmTemplating(raw)
		}
		d := parseCRDColumns(t, raw, p)
		for v, cols := range versionColumns(d) {
			for _, c := range cols {
				if c.Type == "date" && futureFieldRe.MatchString(c.JSONPath) {
					t.Errorf("%s (%s %s): printer column %v is type=date on a future-dated field — kubectl renders `<invalid>`; use type=string (the K1 UNTIL bug class)", p, d.Metadata.Name, v, c)
				}
			}
		}
	}
}
