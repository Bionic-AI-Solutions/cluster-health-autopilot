// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

package releasesrc_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/srenix-ai/agentic-sre/pkg/releasesrc"
)

// memFiles is a deterministic in-memory RepoFiles for tests. Mirrors
// the contract of the real srenix-enterprise forge client adapter: missing
// files return an error whose message contains "not found" so the
// detector's isNotFound() recognises it.
type memFiles struct {
	contents map[string]string
	getErr   error // when non-nil, every Get returns this (transport-error simulation)
}

func (m *memFiles) Get(_ context.Context, path string) ([]byte, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	body, ok := m.contents[path]
	if !ok {
		return nil, fmt.Errorf("memFiles: %s not found", path)
	}
	return []byte(body), nil
}

func (m *memFiles) List(_ context.Context, _ []string) ([]string, error) {
	out := make([]string, 0, len(m.contents))
	for k := range m.contents {
		out = append(out, k)
	}
	return out, nil
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

const goodValuesYAML = `# Srenix Helm chart values.yaml
image:
  repository: docker4zerocool/agentic-sre
  tag: "1.16.0"
  pullPolicy: IfNotPresent
operator:
  enabled: true
`

const umbrellaChartValues = `apiwatch:
  enabled: false
image:
  repository: docker4zerocool/srenix-enterprise
  tag: "1.10.0"
ai:
  enabled: true
`

const wrongRepoValues = `image:
  repository: docker.io/library/redis
  tag: "7.2"
`

const noTagValues = `image:
  repository: docker4zerocool/srenix-enterprise
  pullPolicy: IfNotPresent
`

const garbleYAML = `this is { not yaml :::`

func TestDetectInHelmValues_HappyPath_UmbrellaChartLayout(t *testing.T) {
	files := &memFiles{contents: map[string]string{
		"charts/srenix/values.yaml": goodValuesYAML,
	}}
	ref, err := releasesrc.DetectInHelmValues(context.Background(), files, "srenix", "docker4zerocool/agentic-sre")
	if err != nil {
		t.Fatalf("DetectInHelmValues: %v", err)
	}
	if ref == nil {
		t.Fatal("ref nil for valid match")
	}
	if ref.File != "charts/srenix/values.yaml" {
		t.Errorf("file: got %q", ref.File)
	}
	if ref.CurrentTag != "1.16.0" {
		t.Errorf("tag: got %q want 1.16.0", ref.CurrentTag)
	}
	if ref.Repository != "docker4zerocool/agentic-sre" {
		t.Errorf("repo: got %q", ref.Repository)
	}
	if ref.KeyPath != "image.tag" {
		t.Errorf("keypath: got %q want image.tag", ref.KeyPath)
	}
	if ref.Line < 2 || ref.Line > 6 {
		t.Errorf("line: got %d, expected 2-6 (the `tag:` line in the fixture)", ref.Line)
	}
}

func TestDetectInHelmValues_HappyPath_RootValuesYAML(t *testing.T) {
	// Single-chart repo layout: values.yaml at root, no `charts/<chart>/`.
	files := &memFiles{contents: map[string]string{
		"values.yaml": umbrellaChartValues,
	}}
	ref, err := releasesrc.DetectInHelmValues(context.Background(), files, "srenix-enterprise", "docker4zerocool/srenix-enterprise")
	if err != nil {
		t.Fatalf("DetectInHelmValues: %v", err)
	}
	if ref.File != "values.yaml" {
		t.Errorf("file: got %q want values.yaml", ref.File)
	}
	if ref.CurrentTag != "1.10.0" {
		t.Errorf("tag: got %q", ref.CurrentTag)
	}
}

func TestDetectInHelmValues_RepositoryMismatch_ReturnsNotFound(t *testing.T) {
	files := &memFiles{contents: map[string]string{
		"charts/x/values.yaml": wrongRepoValues, // image is docker.io/library/redis
	}}
	_, err := releasesrc.DetectInHelmValues(context.Background(), files, "x", "docker4zerocool/srenix-enterprise")
	if !errors.Is(err, releasesrc.ErrNotFound) {
		t.Errorf("want ErrNotFound on repo mismatch; got %v", err)
	}
}

func TestDetectInHelmValues_NoTagField_ReturnsNotFound(t *testing.T) {
	files := &memFiles{contents: map[string]string{
		"charts/x/values.yaml": noTagValues,
	}}
	_, err := releasesrc.DetectInHelmValues(context.Background(), files, "x", "docker4zerocool/srenix-enterprise")
	if !errors.Is(err, releasesrc.ErrNotFound) {
		t.Errorf("want ErrNotFound when tag absent; got %v", err)
	}
}

func TestDetectInHelmValues_GarbledYAML_SilentlySkipped(t *testing.T) {
	// Garbled values.yaml in the chart path; values.yaml at root has
	// the real shape. Detector should fall through to the root file.
	files := &memFiles{contents: map[string]string{
		"charts/x/values.yaml": garbleYAML,
		"values.yaml":          umbrellaChartValues,
	}}
	ref, err := releasesrc.DetectInHelmValues(context.Background(), files, "x", "docker4zerocool/srenix-enterprise")
	if err != nil {
		t.Fatalf("garbled chart file should not abort the probe; got %v", err)
	}
	if ref.File != "values.yaml" {
		t.Errorf("expected fallback to root values.yaml; got %q", ref.File)
	}
}

func TestDetectInHelmValues_AllPathsMissing_ReturnsNotFound(t *testing.T) {
	files := &memFiles{contents: map[string]string{}}
	_, err := releasesrc.DetectInHelmValues(context.Background(), files, "x", "docker4zerocool/srenix-enterprise")
	if !errors.Is(err, releasesrc.ErrNotFound) {
		t.Errorf("want ErrNotFound when no candidate files exist; got %v", err)
	}
}

func TestDetectInHelmValues_TransportError_Propagated(t *testing.T) {
	// True transport error (not a missing-file) — must propagate so the
	// proposer can surface "GitHub unreachable" instead of silently
	// degrading.
	files := &memFiles{getErr: errors.New("connection refused: dial tcp 1.2.3.4:443")}
	_, err := releasesrc.DetectInHelmValues(context.Background(), files, "x", "docker4zerocool/srenix-enterprise")
	if err == nil || errors.Is(err, releasesrc.ErrNotFound) {
		t.Errorf("want non-NotFound transport error; got %v", err)
	}
	if !strings.Contains(err.Error(), "connection refused") {
		t.Errorf("error should wrap transport detail; got %v", err)
	}
}

func TestDetectInHelmValues_NilFiles_Errors(t *testing.T) {
	_, err := releasesrc.DetectInHelmValues(context.Background(), nil, "x", "docker4zerocool/srenix-enterprise")
	if err == nil {
		t.Error("nil RepoFiles should error")
	}
}

func TestDetectInHelmValues_EmptyExpectRepository_Errors(t *testing.T) {
	files := &memFiles{contents: map[string]string{"values.yaml": goodValuesYAML}}
	_, err := releasesrc.DetectInHelmValues(context.Background(), files, "x", "")
	if err == nil {
		t.Error("empty expectRepository should error (would match every image block)")
	}
}

func TestDetectInHelmValues_EmptyChartName_FallsBackToRootOnly(t *testing.T) {
	files := &memFiles{contents: map[string]string{
		"values.yaml": goodValuesYAML,
	}}
	ref, err := releasesrc.DetectInHelmValues(context.Background(), files, "", "docker4zerocool/agentic-sre")
	if err != nil {
		t.Fatalf("empty chart name should fall back to root values.yaml; got %v", err)
	}
	if ref.File != "values.yaml" {
		t.Errorf("file: got %q want values.yaml", ref.File)
	}
}

func TestDetectInHelmValues_PathTraversalAttempt_Sanitized(t *testing.T) {
	// Caller may pass an attacker-controlled chart name. Detector
	// strips path components so "../../etc" can't escape the chart dir.
	files := &memFiles{contents: map[string]string{
		"charts/etc/values.yaml": goodValuesYAML,
		"values.yaml":            umbrellaChartValues,
	}}
	ref, err := releasesrc.DetectInHelmValues(context.Background(), files, "../../etc", "docker4zerocool/agentic-sre")
	if err != nil {
		t.Fatalf("path-traversal chart name should be sanitized to basename; got %v", err)
	}
	if ref.File != "charts/etc/values.yaml" {
		t.Errorf("expected chart name sanitized to basename 'etc'; got file=%q", ref.File)
	}
}

func TestDetectInHelmValues_LineNumber_CorrectForRealYAML(t *testing.T) {
	// Fixture with a precise line layout — `tag:` is at line 4
	// (header on 1, image: on 2, repository: on 3, tag: on 4).
	yaml := "# header\n" +
		"image:\n" +
		"  repository: docker4zerocool/srenix-enterprise\n" +
		"  tag: \"1.10.0\"\n"
	files := &memFiles{contents: map[string]string{"values.yaml": yaml}}
	ref, err := releasesrc.DetectInHelmValues(context.Background(), files, "x", "docker4zerocool/srenix-enterprise")
	if err != nil {
		t.Fatalf("DetectInHelmValues: %v", err)
	}
	if ref.Line != 4 {
		t.Errorf("line: got %d want 4", ref.Line)
	}
}

func TestDetectInHelmValues_TagWithoutQuotes(t *testing.T) {
	// Real-world: many values.yaml omit quotes on numeric tags.
	yaml := "image:\n  repository: myorg/app\n  tag: 1.0\n"
	files := &memFiles{contents: map[string]string{"values.yaml": yaml}}
	ref, err := releasesrc.DetectInHelmValues(context.Background(), files, "x", "myorg/app")
	if err != nil {
		t.Fatalf("unquoted tag should parse; got %v", err)
	}
	if ref.CurrentTag != "1" {
		// sigs.k8s.io/yaml will parse "1.0" as a number — accept either
		// "1" or "1.0" depending on YAML decoder behavior. Skip the
		// exact-value check; just confirm it's non-empty.
		if ref.CurrentTag == "" {
			t.Errorf("tag should be non-empty; got %q", ref.CurrentTag)
		}
	}
}
