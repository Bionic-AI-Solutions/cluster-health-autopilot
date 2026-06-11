<!--
Thanks for contributing to Cluster Health Autopilot. Fill in the summary and
tick the release-discipline checklist below. The checklist items encode the
gates whose absence caused three same-week production regressions (v1.23.0
unwired-but-default-on triggers, v1.24.0 CRD pruning, CHA-com v1.20.0 panic).
-->

## Summary

<!-- What does this PR change and why? -->

## Release-discipline checklist

- [ ] New probes/analyzers/toggles default-OFF (or soak rationale recorded). New triggers ship default-off, soak ~7 days, then flip to default-on the next release. The `internal/chartgate` default-off golden gate fails on net-new/flip — update `internal/chartgate/testdata/toggle_defaults.golden` consciously.
- [ ] CRD schema regenerated if `api/v1alpha1` changed (run the CRD codegen; never hand-prune existing schema fields).
- [ ] CHANGELOG updated (and passes `scripts/changelog-lint.sh`: one ISO date per heading, descending semver order, no duplicate versions).
