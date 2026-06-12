#!/usr/bin/env bash
# Copyright 2026 Cluster Health Autopilot contributors
# SPDX-License-Identifier: Apache-2.0
#
# Fixture tests for changelog-tag-check.sh — positive fixtures must
# PASS the gate, negative fixtures must FAIL it. Run from anywhere:
#   bash scripts/changelog-tag-check_test.sh
#
# Requires this repo's git tags to be visible (the gate itself bails
# with exit 2 on a tagless checkout; CI checks out with fetch-tags).
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
GATE="$SCRIPT_DIR/changelog-tag-check.sh"
FIXTURES="$(mktemp -d)"
trap 'rm -rf "$FIXTURES"' EXIT

fail=0

# expect <pass|fail> <name> <fixture-file>
expect() {
  local want="$1" name="$2" file="$3" rc=0
  bash "$GATE" "$file" >/dev/null 2>&1 || rc=$?
  case "$want" in
    pass)
      if [[ "$rc" -ne 0 ]]; then
        echo "FAIL [$name]: expected gate to PASS, got exit $rc" >&2
        bash "$GATE" "$file" 2>&1 | sed 's/^/    /' >&2 || true
        fail=1
      else
        echo "ok   [$name] (passes the gate, as expected)"
      fi
      ;;
    fail)
      if [[ "$rc" -ne 1 ]]; then
        echo "FAIL [$name]: expected gate to FAIL with exit 1, got exit $rc" >&2
        fail=1
      else
        echo "ok   [$name] (fails the gate, as expected)"
      fi
      ;;
  esac
}

# --- Positive fixtures (must PASS) ------------------------------------

# Prose inside [Unreleased] that MENTIONS a released version and a date
# is legitimate cross-reference content — the unanchored regex this
# pins against false-positived on exactly this line (the dated-version
# check must require the markdown heading signature, not fire on prose).
cat >"$FIXTURES/prose-version-date.md" <<'EOF'
# Changelog

## [Unreleased]

### Fixed — something

- Backport of [1.25.1] fix from 2026-05-11 into the trigger path.
- See the [1.24.0] release notes (shipped 2026-06-10) for context.

## [1.25.1] — 2026-06-11

### Fixed — goreleaser disk-OOM
EOF
expect pass "unreleased-prose-mentioning-version-and-date" "$FIXTURES/prose-version-date.md"

# Topmost numbered heading is exempt from the tag requirement (the
# in-flight release); older headings here are all tagged.
cat >"$FIXTURES/in-flight-release.md" <<'EOF'
# Changelog

## [Unreleased]

## [99.99.99] — 2026-06-12

### Added — the release being cut right now (tag lands after merge)

## [1.25.1] — 2026-06-11
EOF
expect pass "topmost-heading-untagged-is-exempt" "$FIXTURES/in-flight-release.md"

# The real CHANGELOG must pass its own gate.
expect pass "repo-changelog" "$SCRIPT_DIR/../CHANGELOG.md"

# --- Negative fixtures (must FAIL) ------------------------------------

# A dated version HEADING inside [Unreleased] presents the version as
# already shipped.
cat >"$FIXTURES/unreleased-dated-heading.md" <<'EOF'
# Changelog

## [Unreleased]

### [9.9.9] — 2026-01-01

- pretend this shipped

## [1.25.1] — 2026-06-11
EOF
expect fail "unreleased-dated-version-heading" "$FIXTURES/unreleased-dated-heading.md"

# A bare version-numbered heading inside [Unreleased] — cut a release
# section instead.
cat >"$FIXTURES/unreleased-bare-heading.md" <<'EOF'
# Changelog

## [Unreleased]

### [9.9.9]

- pretend this is a section

## [1.25.1] — 2026-06-11
EOF
expect fail "unreleased-bare-version-heading" "$FIXTURES/unreleased-bare-heading.md"

# A non-topmost release heading whose tag was never pushed — the
# claimed-but-never-cut release class the gate exists for.
cat >"$FIXTURES/untagged-release.md" <<'EOF'
# Changelog

## [Unreleased]

## [1.25.1] — 2026-06-11

### Fixed — goreleaser disk-OOM

## [99.99.99] — 2026-01-01

### Added — release that was never tagged
EOF
expect fail "non-topmost-heading-without-tag" "$FIXTURES/untagged-release.md"

if [[ "$fail" -ne 0 ]]; then
  echo "changelog-tag-check_test: FAILED" >&2
  exit 1
fi
echo "changelog-tag-check_test: OK"
