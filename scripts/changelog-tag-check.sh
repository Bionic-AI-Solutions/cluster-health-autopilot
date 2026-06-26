#!/usr/bin/env bash
# Copyright 2026 Agentic SRE contributors
# SPDX-License-Identifier: Apache-2.0
#
# changelog-tag-check.sh (O9) — CHANGELOG ↔ git-tag parity gate.
#
# Companion to changelog-lint.sh (which checks heading FORMAT). This
# gate checks that the CHANGELOG never claims a release that was not
# actually cut — the defect class where a `## [x.y.z] — DATE` heading
# lands on main but the `vx.y.z` tag is never pushed, so readers (and
# helm/values pins) reference a release that does not exist.
#
# Rules:
#   1. Every `## [x.y.z]` heading EXCEPT the topmost numbered one must
#      have a matching git tag: `vx.y.z` (binary release) or
#      `agentic-sre-x.y.z` (chart-releaser tag — the
#      1.8.x line shipped chart-only cuts under that form). The topmost
#      numbered heading is exempt because it is legitimately untagged
#      while the release PR that introduces it is in flight (the tag is
#      pushed after merge). If the topmost heading IS tagged, fine too.
#   2. The `[Unreleased]` section body must not present a version as
#      already shipped: no HEADING of any level naming a `[x.y.z]` —
#      dated (`### [x.y.z] — YYYY-MM-DD`, the shipped-header signature)
#      or bare. Both checks are anchored to the heading signature, so
#      prose that merely mentions a version/date (e.g. "Backport of
#      [1.25.1] fix from 2026-05-11") passes. Unreleased content belongs
#      under the topic `###` headings; it gets a version number only
#      when the release section is cut.
#
# Requires git tags to be present (CI: checkout with fetch-tags: true).
set -euo pipefail

CHANGELOG="${1:-$(dirname "$0")/../CHANGELOG.md}"
# Tags are checked against THIS repo regardless of where the changelog
# argument lives (lets tests feed fixture files from /tmp).
REPO_DIR="$(cd "$(dirname "$0")/.." && pwd)"

if [[ ! -f "$CHANGELOG" ]]; then
  echo "changelog-tag-check: file not found: $CHANGELOG" >&2
  exit 2
fi

if ! git -C "$REPO_DIR" rev-parse --git-dir >/dev/null 2>&1; then
  echo "changelog-tag-check: not inside a git repository — cannot check tags" >&2
  exit 2
fi

# Guard against a tagless checkout (shallow clone without fetch-tags),
# which would make every heading look untagged and fail noisily for the
# wrong reason.
if [[ -z "$(git -C "$REPO_DIR" tag --list 'v[0-9]*')" ]]; then
  echo "changelog-tag-check: no v* tags visible — fetch tags first (CI: actions/checkout with fetch-tags: true)" >&2
  exit 2
fi

fail=0
first_version_seen=0
in_unreleased=0
lineno=0

while IFS= read -r line; do
  lineno=$((lineno + 1))

  if [[ "$line" =~ ^##\ \[Unreleased\] ]]; then
    in_unreleased=1
    continue
  fi

  # Numbered release heading. The optional `-PRERELEASE` suffix (e.g.
  # `-alpha.1`) is part of the version and of the expected tag name
  # (`v0.1.0-alpha.1`), so capture the full semver including any suffix.
  if [[ "$line" =~ ^##\ \[([0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?)\] ]]; then
    in_unreleased=0
    version="${BASH_REMATCH[1]}"
    if [[ "$first_version_seen" -eq 0 ]]; then
      # Topmost numbered/pre-release heading — the section being released
      # right now (incl. the re-baseline cut); its tag may not exist yet.
      first_version_seen=1
      continue
    fi
    if ! git -C "$REPO_DIR" rev-parse -q --verify "refs/tags/v${version}" >/dev/null &&
       ! git -C "$REPO_DIR" rev-parse -q --verify "refs/tags/agentic-sre-${version}" >/dev/null; then
      echo "changelog-tag-check: line $lineno: heading [$version] has no git tag v${version} (or chart tag agentic-sre-${version}) — the CHANGELOG claims a release that was never cut:" >&2
      echo "  $line" >&2
      fail=1
    fi
    continue
  fi

  # Inside [Unreleased]: no HEADING may present a version as already
  # shipped. Both checks are anchored to the markdown heading signature
  # (`#`s + space, then the bracketed version) — prose that merely
  # MENTIONS a version and a date, e.g.
  # `- Backport of [1.25.1] fix from 2026-05-11`, is legitimate
  # cross-reference content and must pass (an unanchored version+date
  # regex false-positived on exactly that).
  if [[ "$in_unreleased" -eq 1 ]]; then
    # Heading whose bracketed version carries an ISO date — the
    # shipped-header signature (`### [x.y.z] — YYYY-MM-DD`); the date
    # separator is matched loosely since the dash glyph has varied.
    # (A two-`#` form is consumed by the release-heading branch above.)
    if [[ "$line" =~ ^#{1,6}\ +\[[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?\].*[0-9]{4}-[0-9]{2}-[0-9]{2} ]]; then
      echo "changelog-tag-check: line $lineno: [Unreleased] content presents a dated version heading as shipped:" >&2
      echo "  $line" >&2
      fail=1
    elif [[ "$line" =~ ^#{1,6}\ +\[[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?\] ]]; then
      echo "changelog-tag-check: line $lineno: [Unreleased] contains a version-numbered heading — cut a release section instead:" >&2
      echo "  $line" >&2
      fail=1
    fi
  fi
done <"$CHANGELOG"

if [[ "$fail" -ne 0 ]]; then
  echo "changelog-tag-check: FAILED" >&2
  exit 1
fi

echo "changelog-tag-check: OK ($CHANGELOG)"
