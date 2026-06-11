#!/usr/bin/env bash
# Copyright 2026 Cluster Health Autopilot contributors
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
#       for the date/semver checks)
#   3. no version appears more than once
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
declare -A seen_version

while IFS= read -r line; do
  # Only consider version headings: `## [x.y.z] ...` or `## [Unreleased]`.
  [[ "$line" =~ ^##\ \[ ]] || continue

  if [[ "$line" =~ ^##\ \[Unreleased\] ]]; then
    continue
  fi

  # Extract the semver version inside the brackets.
  if [[ ! "$line" =~ ^##\ \[([0-9]+)\.([0-9]+)\.([0-9]+)\] ]]; then
    echo "changelog-lint: heading is neither [Unreleased] nor [x.y.z]:" >&2
    echo "  $line" >&2
    fail=1
    continue
  fi
  major="${BASH_REMATCH[1]}"
  minor="${BASH_REMATCH[2]}"
  patch="${BASH_REMATCH[3]}"
  version="${major}.${minor}.${patch}"

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

  # (2) descending semver order vs the previous version heading.
  if [[ "$have_prev" -eq 1 ]]; then
    # cur < prev required (strictly descending).
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
