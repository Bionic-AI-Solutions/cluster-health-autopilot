#!/usr/bin/env bash
# Copyright 2026 Agentic SRE contributors
# SPDX-License-Identifier: Apache-2.0
#
# changelog-lint.sh (P3.3b) — CHANGELOG hygiene gate.
#
# Encodes the cheap-but-recurring defects that slipped into CHANGELOG.md
# (doubled date strings on a heading, version headings out of semver order,
# duplicate version headings). Run in CI so they cannot recur silently.
#
# Asserts, over every `## [x.y.z] — DATE` heading:
#   1. each version heading carries EXACTLY ONE ISO (YYYY-MM-DD) date
#   2. version headings appear in DESCENDING semver order
#      (the `[Unreleased]` heading, if present, must be first and is skipped
#       for the date/semver checks; the TOPMOST numbered heading is also
#       exempt from the order check — it is the in-flight release and may
#       be a deliberate version re-baseline, e.g. the 1.x → 0.x-alpha reset)
#   3. no version appears more than once
#
# Pre-release headings are supported: `## [x.y.z-alpha.N] — DATE` (and any
# other `-PRERELEASE` suffix). The numeric core (x.y.z) drives the order/dup
# checks; the full string (incl. suffix) is the dedup key.
#
# Exits non-zero with the offending lines on any violation.
#
# NOTE: this gate is VERSION-ORDER based, not date-order based. A date
# inversion where a heading's date predates an earlier (higher) version's
# date is NOT flagged here (see the [1.5.0] note in CHANGELOG.md) because
# historical release dates can be legitimately non-monotonic and cannot be
# safely auto-corrected.
set -euo pipefail

CHANGELOG="${1:-$(dirname "$0")/../CHANGELOG.md}"

if [[ ! -f "$CHANGELOG" ]]; then
  echo "changelog-lint: file not found: $CHANGELOG" >&2
  exit 2
fi

fail=0
iso_re='[0-9]{4}-[0-9]{2}-[0-9]{2}'

prev_major=-1
prev_minor=-1
prev_patch=-1
have_prev=0
version_count=0
declare -A seen_version

while IFS= read -r line; do
  # Only consider version headings: `## [x.y.z] ...` or `## [Unreleased]`.
  [[ "$line" =~ ^##\ \[ ]] || continue

  if [[ "$line" =~ ^##\ \[Unreleased\] ]]; then
    continue
  fi

  # Extract the semver version inside the brackets. The optional fourth
  # capture is a SemVer pre-release suffix (e.g. `-alpha.1`); the numeric
  # core (x.y.z) drives the order check, the full string is the dedup key.
  if [[ ! "$line" =~ ^##\ \[([0-9]+)\.([0-9]+)\.([0-9]+)(-[0-9A-Za-z.-]+)?\] ]]; then
    echo "changelog-lint: heading is neither [Unreleased] nor [x.y.z(-prerelease)]:" >&2
    echo "  $line" >&2
    fail=1
    continue
  fi
  major="${BASH_REMATCH[1]}"
  minor="${BASH_REMATCH[2]}"
  patch="${BASH_REMATCH[3]}"
  prerelease="${BASH_REMATCH[4]:-}"
  version="${major}.${minor}.${patch}${prerelease}"
  version_count=$((version_count + 1))

  # (1) exactly one ISO date on the heading.
  date_count=$(grep -oE "$iso_re" <<<"$line" | wc -l | tr -d ' ')
  if [[ "$date_count" -ne 1 ]]; then
    echo "changelog-lint: heading [$version] has $date_count ISO dates (expected exactly 1):" >&2
    echo "  $line" >&2
    fail=1
  fi

  # (3) no duplicate version heading.
  if [[ -n "${seen_version[$version]:-}" ]]; then
    echo "changelog-lint: duplicate version heading [$version]:" >&2
    echo "  $line" >&2
    fail=1
  fi
  seen_version[$version]=1

  # (2) descending semver order vs the previous version heading. The
  # TOPMOST numbered heading is exempt: it is the in-flight release and may
  # be a deliberate version re-baseline (the 1.x → 0.x-alpha reset places a
  # numerically-lower version on top by design). Order is enforced strictly
  # among every heading AFTER the topmost.
  # version_count==2 is the topmost→next pair, which straddles the
  # re-baseline boundary (0.x-alpha on top, 1.x below); skip just that one
  # comparison. Every later pair (1.x vs 1.x) is checked normally.
  if [[ "$have_prev" -eq 1 && "$version_count" -gt 2 ]]; then
    # cur < prev required (strictly descending), numeric core only.
    if (( major > prev_major )) || \
       (( major == prev_major && minor > prev_minor )) || \
       (( major == prev_major && minor == prev_minor && patch >= prev_patch )); then
      echo "changelog-lint: version order violation — [$version] must be strictly below [${prev_major}.${prev_minor}.${prev_patch}] (descending semver):" >&2
      echo "  $line" >&2
      fail=1
    fi
  fi
  prev_major="$major"; prev_minor="$minor"; prev_patch="$patch"; have_prev=1
done <"$CHANGELOG"

if [[ "$fail" -ne 0 ]]; then
  echo "changelog-lint: FAILED" >&2
  exit 1
fi

echo "changelog-lint: OK ($CHANGELOG)"
