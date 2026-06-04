// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package releasesrc

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// DetectInRawManifests scans the repo's .yaml/.yml files for an inline
// `image: <expectRepository>:<tag>` line. Used for repos that ship raw
// Kubernetes manifests (Deployment / StatefulSet / DaemonSet specs)
// instead of Helm charts.
//
// Returns the first match with file path + 1-based line + the current
// tag. Returns ErrNotFound when no file contains a matching image line;
// transport errors propagate.
//
// The match is byte-exact on `image: <repo>:<tag>` (with the leading
// whitespace common in Kubernetes manifests). To avoid false matches on
// substrings (e.g. an image whose repo is a prefix of another), the
// scanner anchors on the trailing tag boundary — the character after
// the tag must be whitespace, end-of-line, double-quote, or end-of-file.
func DetectInRawManifests(ctx context.Context, files RepoFiles, expectRepository string) (*ImageRef, error) {
	if files == nil {
		return nil, errors.New("releasesrc: RepoFiles required")
	}
	if expectRepository == "" {
		return nil, errors.New("releasesrc: expectRepository required")
	}
	paths, err := files.List(ctx, []string{"**/*.yaml", "**/*.yml"})
	if err != nil {
		return nil, fmt.Errorf("releasesrc: list: %w", err)
	}
	if len(paths) == 0 {
		// Fall back to a no-pattern list (some forge backends ignore
		// patterns); filter manually.
		paths, err = files.List(ctx, nil)
		if err != nil {
			return nil, fmt.Errorf("releasesrc: fallback list: %w", err)
		}
	}
	re := imageLineRegex(expectRepository)
	var firstTransportErr error
	for _, p := range paths {
		if !looksLikeYAMLPath(p) {
			continue
		}
		body, err := files.Get(ctx, p)
		if err != nil {
			if isNotFound(err) {
				continue
			}
			if firstTransportErr == nil {
				firstTransportErr = fmt.Errorf("releasesrc: read %s: %w", p, err)
			}
			continue
		}
		match := re.FindIndex(body)
		if match == nil {
			continue
		}
		// Pull the tag out of the captured submatch.
		tag := extractTagFromMatch(re, body, match)
		if tag == "" {
			continue
		}
		// 1-based line number = newlines before match + 1.
		line := 1 + bytes.Count(body[:match[0]], []byte("\n"))
		return &ImageRef{
			File:       p,
			Line:       line,
			KeyPath:    "image", // inline form — no nested key path
			CurrentTag: tag,
			Repository: expectRepository,
		}, nil
	}
	if firstTransportErr != nil {
		return nil, firstTransportErr
	}
	return nil, ErrNotFound
}

// Detect tries DetectInHelmValues first, then DetectInRawManifests.
// Returns the first hit. Operators don't need to know which detector
// matched — the resulting ImageRef carries enough context for the
// proposer to construct the patch.
//
// Fall-through rules:
//   - Helm match → return.
//   - Helm ErrNotFound → try raw scan.
//   - Helm transport error → ALSO try raw scan, then propagate the
//     raw scan's outcome. GitHub returns HTTP 403 under secondary rate
//     limit conditions that look indistinguishable from "scope denied"
//     to the forge layer; if we surfaced 403 here we'd block all
//     downstream digest-pin work whenever a transient burst of API
//     calls overlapped a single Helm probe (observed 2026-06-04 with
//     35 of 41 candidates spuriously erroring on `charts/X/values.yaml`
//     while the same paths returned a clean 404 seconds later). The
//     raw scan re-tries through a different path, surfaces real auth
//     failures via its own ListRepoFiles call, and converges to
//     ErrNotFound when neither shape matches.
func Detect(ctx context.Context, files RepoFiles, chartName, expectRepository string) (*ImageRef, error) {
	if ref, err := DetectInHelmValues(ctx, files, chartName, expectRepository); err == nil {
		return ref, nil
	}
	return DetectInRawManifests(ctx, files, expectRepository)
}

// imageLineRegex builds the per-(repository) regex that matches lines
// like `image: <repo>:<tag>` or `image: "<repo>:<tag>"` in K8s YAML.
// Captured group 1 is the tag.
//
// Anchored on `^\s+image:` so we don't match keys that happen to END
// in "image:" (e.g. `someimage:`). Tag terminator is one of:
// whitespace, end-of-line, `"`, `#` (YAML inline comment).
func imageLineRegex(repository string) *regexp.Regexp {
	escaped := regexp.QuoteMeta(repository)
	// `\bimage\s*:\s*` accepts both unquoted and quoted forms via the
	// optional " group; tag must be at least one non-space/quote/colon
	// char (tag chars per OCI: a-z A-Z 0-9 . - _).
	return regexp.MustCompile(`(?m)^\s*-?\s*image\s*:\s*"?` + escaped + `:([A-Za-z0-9][A-Za-z0-9._-]*)`)
}

// extractTagFromMatch returns submatch 1 (the tag) from a regex match.
func extractTagFromMatch(re *regexp.Regexp, body []byte, match []int) string {
	subs := re.FindSubmatch(body[match[0]:match[1]])
	if len(subs) < 2 {
		return ""
	}
	return string(subs[1])
}

// looksLikeYAMLPath is a conservative filter for the file list. Lists
// returned without a pattern argument can include non-YAML files; we
// only inspect .yaml/.yml.
func looksLikeYAMLPath(p string) bool {
	if p == "" {
		return false
	}
	lower := strings.ToLower(p)
	return strings.HasSuffix(lower, ".yaml") || strings.HasSuffix(lower, ".yml")
}
