// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

// Package releasesrc finds the file + key in a release-source repo
// (Helm chart, Argo CD Application, Kustomize overlay) that holds a
// workload's image tag. It is the keystone the paid-tier digest-pin
// proposer needs to construct a PR — without knowing which file +
// line sets the tag, the proposer can't propose a change.
//
// Today: Helm `values.yaml` detection (covers 80%+ of GitOps repos).
//
// Next slices (per docs/design/2026-06-rag-digest-pin-proposer.md):
//   - Argo CD Application probing (`spec.source.helm.parameters` /
//     `spec.source.kustomize.images`)
//   - Kustomize overlay (`images[].name + newTag`)
//   - Flux HelmRelease (`spec.values.image.tag`)
//
// The package defines a minimal `RepoFiles` interface so any forge
// client (the srenix-enterprise GitHub client, a Git over SSH driver, or an
// in-memory test fixture) can satisfy it without OSS importing a
// specific implementation.
package releasesrc

import (
	"context"
	"errors"
	"fmt"
	"path"
	"regexp"
	"strings"

	"sigs.k8s.io/yaml"
)

// RepoFiles is the minimal read-only surface releasesrc needs. The
// srenix-enterprise forge client implements this via a per-(owner, repo, ref)
// adapter; tests use an in-memory map.
type RepoFiles interface {
	// Get returns the raw bytes of `path` at the configured ref.
	// Returns os.ErrNotExist (or wraps it) when the file is absent;
	// callers Detect.* fail open on missing-file errors so a half-
	// scaffolded repo doesn't crash the proposer.
	Get(ctx context.Context, path string) ([]byte, error)

	// List returns file paths matching one of `patterns` (glob, matched
	// against the path) at the configured ref. Empty result is not an
	// error.
	List(ctx context.Context, patterns []string) ([]string, error)
}

// ImageRef pinpoints exactly where in a repo a workload's image tag is
// declared. The proposer uses this to construct a single-line patch
// that swaps `:tag` for `@sha256:<digest>` without disturbing the rest
// of the file.
//
// Line is 1-based to match `git blame` / editor conventions. KeyPath
// is a dot-separated YAML walk (e.g. "image.tag") so callers can
// render a friendly "this file's image.tag is X" message; the
// authoritative edit anchor is (File, Line).
type ImageRef struct {
	File       string // relative path in repo, e.g. "charts/srenix/values.yaml"
	Line       int    // 1-based line number of the tag key
	KeyPath    string // dot-separated path within the YAML, e.g. "image.tag"
	CurrentTag string // current tag value, e.g. "1.10.0"
	Repository string // current repository value, e.g. "docker4zerocool/srenix-enterprise"
}

// ErrNotFound is returned when no `image:` block holding the expected
// repository was found. Distinguish from transport errors so callers
// can fall back to a PR-template path instead of erroring out.
var ErrNotFound = errors.New("releasesrc: image tag not located")

// DetectInHelmValues looks for the workload's image tag in the
// conventional Helm chart values.yaml layout. It probes:
//
//  1. `charts/<chartName>/values.yaml`   (umbrella chart layout)
//  2. `values.yaml`                       (single-chart repo root)
//  3. `charts/<chartName>/Chart.yaml`     (appVersion fallback — only
//     when values.yaml omits tag)
//
// The detector skips files that don't parse as YAML or whose top-level
// shape isn't a map. It also requires the located `image.repository`
// to MATCH `expectRepository` — guards against false matches in
// umbrella charts that ship multiple subchart values blocks.
//
// Returns ErrNotFound when nothing matches. Transport errors from
// RepoFiles propagate unchanged.
func DetectInHelmValues(ctx context.Context, files RepoFiles, chartName, expectRepository string) (*ImageRef, error) {
	if files == nil {
		return nil, errors.New("releasesrc: RepoFiles required")
	}
	if expectRepository == "" {
		return nil, errors.New("releasesrc: expectRepository required")
	}
	candidates := candidatePaths(chartName)
	var firstTransportErr error
	for _, p := range candidates {
		body, err := files.Get(ctx, p)
		if err != nil {
			// Best-effort: missing file is normal (umbrella vs single-
			// chart shape differs per repo). Remember the first true
			// transport error so we can surface it if nothing matches.
			if isNotFound(err) {
				continue
			}
			if firstTransportErr == nil {
				firstTransportErr = fmt.Errorf("releasesrc: read %s: %w", p, err)
			}
			continue
		}
		if ref := findImageBlockInValues(body, p, expectRepository); ref != nil {
			return ref, nil
		}
	}
	if firstTransportErr != nil {
		return nil, firstTransportErr
	}
	return nil, ErrNotFound
}

// candidatePaths returns the values.yaml probe order. Order matters:
// umbrella-chart layout is more specific so try first.
func candidatePaths(chartName string) []string {
	if chartName == "" {
		return []string{"values.yaml"}
	}
	chartName = path.Base(strings.TrimSpace(chartName)) // defend against "../" inputs
	return []string{
		path.Join("charts", chartName, "values.yaml"),
		"values.yaml",
	}
}

// isNotFound recognises the various "file absent" error shapes that
// RepoFiles implementations may return. Conservative: only treats
// errors that EXPLICITLY say "not found" / "404" / wrap os.ErrNotExist
// as not-found; anything else is a transport error.
func isNotFound(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "not found") ||
		strings.Contains(msg, "404") ||
		strings.Contains(msg, "no such file") ||
		strings.Contains(msg, "does not exist")
}

// imageBlockShape mirrors the conventional Helm values.yaml shape:
//
//	image:
//	  repository: docker4zerocool/srenix-enterprise
//	  tag: "1.10.0"
//
// Only `repository` + `tag` are decoded — `pullPolicy`, `pullSecrets`,
// and digest variants are ignored at parse time but preserved if the
// caller edits the file. This struct exists only to coax sigs.k8s.io/yaml
// into giving us a typed view of the relevant fields.
type imageBlockShape struct {
	Image struct {
		Repository string `json:"repository,omitempty"`
		Tag        string `json:"tag,omitempty"`
	} `json:"image,omitempty"`
}

// findImageBlockInValues parses `body` as YAML, locates the top-level
// `image.{repository,tag}` block, and returns an ImageRef when the
// repository matches `expectRepository`. Returns nil when not found
// (parse error / shape mismatch / repo mismatch — all silent).
//
// Line lookup uses a regex scan of the raw bytes because sigs.k8s.io/yaml
// doesn't preserve positions. The fallback is conservative: only
// matches a `tag:` line that appears AFTER an `image:` line at lower
// or equal indent — same heuristic git diff itself uses.
func findImageBlockInValues(body []byte, file, expectRepository string) *ImageRef {
	var v imageBlockShape
	if err := yaml.Unmarshal(body, &v); err != nil {
		return nil
	}
	if v.Image.Repository == "" || v.Image.Tag == "" {
		return nil
	}
	if v.Image.Repository != expectRepository {
		return nil
	}
	line := locateTagLine(body)
	return &ImageRef{
		File:       file,
		Line:       line,
		KeyPath:    "image.tag",
		CurrentTag: v.Image.Tag,
		Repository: v.Image.Repository,
	}
}

// tagLineRe matches a `  tag: <value>` or `  tag: "<value>"` line at
// any indentation. We look for it AFTER the `image:` header to avoid
// false hits on `app.kubernetes.io/instance` etc.
var (
	imageHeaderRe = regexp.MustCompile(`(?m)^[\t ]*image[\t ]*:[\t ]*$`)
	tagLineRe     = regexp.MustCompile(`(?m)^[\t ]+tag[\t ]*:[\t ]*[^\n]+`)
)

// locateTagLine returns the 1-based line number of the `tag:` line
// inside the first `image:` block in `body`. Returns 0 when not
// findable — the caller stores 0 and falls back to a "search this
// file" hint in the PR description.
func locateTagLine(body []byte) int {
	headerLoc := imageHeaderRe.FindIndex(body)
	if headerLoc == nil {
		return 0
	}
	after := body[headerLoc[1]:]
	tagLoc := tagLineRe.FindIndex(after)
	if tagLoc == nil {
		return 0
	}
	// Count newlines from start to (headerLoc[1] + tagLoc[0]) to get
	// 1-based line number. headerLoc[1] is the byte AFTER the header
	// match; tagLoc[0] is the byte offset within `after`.
	absStart := headerLoc[1] + tagLoc[0]
	return 1 + strings.Count(string(body[:absStart]), "\n")
}
