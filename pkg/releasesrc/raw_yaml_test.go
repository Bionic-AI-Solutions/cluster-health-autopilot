// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package releasesrc_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/pkg/releasesrc"
)

// Real-shape WordPress Deployment that storethesoup-k8s ships.
const rawWordpressYAML = `apiVersion: apps/v1
kind: Deployment
metadata:
  name: wordpress
  namespace: storethesoup
spec:
  replicas: 1
  template:
    spec:
      containers:
      - name: wordpress
        image: docker4zerocool/storethesoup-wordpress:6.7-php8.2-wpcli-redis
        ports:
        - containerPort: 80
        env:
        - name: WORDPRESS_DB_HOST
          value: mariadb
`

// Quoted-image variant.
const quotedImageYAML = `apiVersion: apps/v1
kind: Deployment
metadata: {name: x}
spec:
  template:
    spec:
      containers:
        - name: c
          image: "docker4zerocool/some-app:v2.3.1"
`

// Multi-document YAML with multiple containers — must match the first.
const multiDocYAML = `# header
---
apiVersion: v1
kind: Service
metadata: {name: x}
spec:
  ports: [{port: 80}]
---
apiVersion: apps/v1
kind: Deployment
metadata: {name: y}
spec:
  template:
    spec:
      containers:
        - name: app
          image: docker4zerocool/myapp:1.0.0
        - name: sidecar
          image: docker4zerocool/sidecar:2.0.0
`

func TestDetectInRawManifests_HappyPath_StoreSoupShape(t *testing.T) {
	files := &memFiles{contents: map[string]string{
		"30-wordpress.yaml": rawWordpressYAML,
		"10-mariadb.yaml":   "apiVersion: apps/v1\nkind: StatefulSet\nspec: {}",
		"README.md":         "not yaml",
	}}
	ref, err := releasesrc.DetectInRawManifests(
		context.Background(), files, "docker4zerocool/storethesoup-wordpress")
	if err != nil {
		t.Fatalf("DetectInRawManifests: %v", err)
	}
	if ref.File != "30-wordpress.yaml" {
		t.Errorf("file: got %q want 30-wordpress.yaml", ref.File)
	}
	if ref.CurrentTag != "6.7-php8.2-wpcli-redis" {
		t.Errorf("tag: got %q want 6.7-php8.2-wpcli-redis", ref.CurrentTag)
	}
	if ref.Repository != "docker4zerocool/storethesoup-wordpress" {
		t.Errorf("repo: got %q", ref.Repository)
	}
	if ref.Line < 5 || ref.Line > 20 {
		t.Errorf("line: got %d, expected ~12 (the image: line)", ref.Line)
	}
	if ref.KeyPath != "image" {
		t.Errorf("keypath: got %q", ref.KeyPath)
	}
}

func TestDetectInRawManifests_QuotedImage(t *testing.T) {
	files := &memFiles{contents: map[string]string{
		"deployment.yaml": quotedImageYAML,
	}}
	ref, err := releasesrc.DetectInRawManifests(
		context.Background(), files, "docker4zerocool/some-app")
	if err != nil {
		t.Fatalf("DetectInRawManifests: %v", err)
	}
	if ref.CurrentTag != "v2.3.1" {
		t.Errorf("tag: got %q want v2.3.1", ref.CurrentTag)
	}
}

func TestDetectInRawManifests_MultiContainer_MatchesFirst(t *testing.T) {
	files := &memFiles{contents: map[string]string{
		"multi.yaml": multiDocYAML,
	}}
	ref, err := releasesrc.DetectInRawManifests(
		context.Background(), files, "docker4zerocool/myapp")
	if err != nil {
		t.Fatalf("DetectInRawManifests: %v", err)
	}
	if ref.CurrentTag != "1.0.0" {
		t.Errorf("expected first match (myapp:1.0.0); got %q", ref.CurrentTag)
	}
}

func TestDetectInRawManifests_NoMatch_ReturnsNotFound(t *testing.T) {
	files := &memFiles{contents: map[string]string{
		"30-wordpress.yaml": rawWordpressYAML,
	}}
	_, err := releasesrc.DetectInRawManifests(
		context.Background(), files, "different/repo")
	if !errors.Is(err, releasesrc.ErrNotFound) {
		t.Errorf("want ErrNotFound for missing repo; got %v", err)
	}
}

func TestDetectInRawManifests_EmptyRepo_ReturnsNotFound(t *testing.T) {
	files := &memFiles{contents: map[string]string{}}
	_, err := releasesrc.DetectInRawManifests(
		context.Background(), files, "docker4zerocool/wp")
	if !errors.Is(err, releasesrc.ErrNotFound) {
		t.Errorf("want ErrNotFound for empty repo; got %v", err)
	}
}

func TestDetectInRawManifests_RequiresArgs(t *testing.T) {
	_, err := releasesrc.DetectInRawManifests(context.Background(), nil, "x")
	if err == nil {
		t.Error("nil files should error")
	}
	files := &memFiles{contents: map[string]string{"x.yaml": rawWordpressYAML}}
	_, err = releasesrc.DetectInRawManifests(context.Background(), files, "")
	if err == nil {
		t.Error("empty repo should error")
	}
}

func TestDetectInRawManifests_SkipsNonYAMLFiles(t *testing.T) {
	// Files matching the image line but with non-yaml extensions must be ignored
	// (a `Dockerfile` containing `image: ...` is not a K8s manifest).
	files := &memFiles{contents: map[string]string{
		"README.md":     "image: docker4zerocool/wp:1.0",
		"Dockerfile":    "FROM docker4zerocool/wp:1.0",
		"manifest.yaml": rawWordpressYAML,
	}}
	ref, err := releasesrc.DetectInRawManifests(
		context.Background(), files, "docker4zerocool/storethesoup-wordpress")
	if err != nil {
		t.Fatalf("DetectInRawManifests: %v", err)
	}
	if ref.File != "manifest.yaml" {
		t.Errorf("expected manifest.yaml; got %q", ref.File)
	}
}

func TestDetect_HelmTakesPriority(t *testing.T) {
	// When BOTH a Helm values.yaml and a raw manifest match, Detect
	// prefers the Helm path (the proposer's preferred edit anchor).
	files := &memFiles{contents: map[string]string{
		"charts/myapp/values.yaml": `image:
  repository: docker4zerocool/myapp
  tag: "1.0.0"
`,
		"deployment.yaml": "apiVersion: apps/v1\nkind: Deployment\nspec:\n  template:\n    spec:\n      containers:\n        - image: docker4zerocool/myapp:1.0.0",
	}}
	ref, err := releasesrc.Detect(context.Background(), files, "myapp", "docker4zerocool/myapp")
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if !strings.HasSuffix(ref.File, "values.yaml") {
		t.Errorf("Helm path should win; got %q", ref.File)
	}
}

func TestDetect_FallsBackToRawWhenHelmMisses(t *testing.T) {
	// Repo has NO Helm chart — only raw manifests (storethesoup-k8s
	// pattern). Detect falls through to DetectInRawManifests.
	files := &memFiles{contents: map[string]string{
		"30-wordpress.yaml": rawWordpressYAML,
	}}
	ref, err := releasesrc.Detect(
		context.Background(), files, "storethesoup-wordpress", "docker4zerocool/storethesoup-wordpress")
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if ref.File != "30-wordpress.yaml" {
		t.Errorf("expected raw-manifest fallback; got %q", ref.File)
	}
	if ref.KeyPath != "image" {
		t.Errorf("raw-manifest KeyPath should be 'image'; got %q", ref.KeyPath)
	}
}

func TestDetect_PropagatesTransportError_FromHelm(t *testing.T) {
	// True transport error from the Helm probe must surface — Detect
	// must NOT silently paper over by falling to the raw scan.
	files := &memFiles{getErr: errors.New("connection refused")}
	_, err := releasesrc.Detect(context.Background(), files, "x", "docker4zerocool/y")
	if err == nil {
		t.Fatal("expected transport error to propagate")
	}
	if !strings.Contains(err.Error(), "connection refused") {
		t.Errorf("error should wrap transport detail; got %v", err)
	}
}
