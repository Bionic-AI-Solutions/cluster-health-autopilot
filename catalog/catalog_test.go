// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

package catalog

import (
	"testing"

	"github.com/srenix-ai/agentic-sre/pkg/registry"
)

// probeNames returns the Name() set of all probes RegisterOSS registers
// under the current environment.
func probeNames(t *testing.T) map[string]bool {
	t.Helper()
	r := registry.New()
	RegisterOSS(r)
	out := map[string]bool{}
	for _, p := range r.Probes() {
		out[p.Name()] = true
	}
	return out
}

// analyzerNames returns the Name() set of all analyzers RegisterOSS
// registers under the current environment.
func analyzerNames(t *testing.T) map[string]bool {
	t.Helper()
	r := registry.New()
	RegisterOSS(r)
	out := map[string]bool{}
	for _, a := range r.Analyzers() {
		out[a.Name()] = true
	}
	return out
}

// baseProbeToggles maps each base-probe opt-out env var to the probe
// Name() it gates. These are the original six probes that were
// registered unconditionally before the toggles were added — docs
// promised "each probe independently togglable", so the env names here
// are the documented public contract (CRITICAL_WORKLOADS maps to the
// Critical Services probe by design).
var baseProbeToggles = map[string]string{
	"SRENIX_PROBE_CEPH":               "Ceph Storage",
	"SRENIX_PROBE_NODES":              "Cluster Nodes",
	"SRENIX_PROBE_POSTGRES":           "PostgreSQL",
	"SRENIX_PROBE_PVCS":               "Storage Claims",
	"SRENIX_PROBE_CRITICAL_WORKLOADS": "Critical Services",
	"SRENIX_PROBE_ENDPOINTS":          "External Endpoints",
}

// coreAnalyzerToggles maps each core-analyzer opt-out env var to the
// analyzer Name() it gates. These seven (secret-chain + cert + image
// auth) are the product's core value — they MUST default ON; the env
// gate exists only so the docs' "disable any analyzer" promise holds.
var coreAnalyzerToggles = map[string]string{
	"SRENIX_ANALYZER_SECRET_KEY_MISSING":         "SecretKeyMissing",
	"SRENIX_ANALYZER_FAILING_EXTERNAL_SECRETS":   "FailingExternalSecrets",
	"SRENIX_ANALYZER_PROACTIVE_SECRET_KEY_CHECK": "ProactiveSecretKeyCheck",
	"SRENIX_ANALYZER_UNPROVISIONED_SECRET":       "UnprovisionedSecret",
	"SRENIX_ANALYZER_IMAGE_PULL_AUTH":            "ImagePullAuth",
	"SRENIX_ANALYZER_CERT_EXPIRY":                "CertExpiry",
	"SRENIX_ANALYZER_TLS_SECRET_MISMATCH":        "TLSSecretMismatch",
}

func TestBaseProbes_RegisteredByDefault(t *testing.T) {
	names := probeNames(t)
	for env, probe := range baseProbeToggles {
		if !names[probe] {
			t.Errorf("probe %q (gated by %s) must be registered by default — defaults flipped OFF", probe, env)
		}
	}
}

func TestBaseProbes_SkippedWhenEnvOff(t *testing.T) {
	for env, probe := range baseProbeToggles {
		t.Run(env, func(t *testing.T) {
			t.Setenv(env, "off")
			names := probeNames(t)
			if names[probe] {
				t.Errorf("probe %q still registered with %s=off", probe, env)
			}
			// Disabling one probe must not disable any sibling.
			for otherEnv, other := range baseProbeToggles {
				if otherEnv != env && !names[other] {
					t.Errorf("%s=off also dropped sibling probe %q (gated by %s)", env, other, otherEnv)
				}
			}
		})
	}
}

func TestBaseProbes_NonOffValuesKeepProbeOn(t *testing.T) {
	// The contract is "=off disables"; any other value (true/on/garbage)
	// must leave the probe registered — matching the 15 pre-existing gates.
	for _, v := range []string{"on", "true", "false", "OFF", "0"} {
		t.Run(v, func(t *testing.T) {
			t.Setenv("SRENIX_PROBE_CEPH", v)
			if !probeNames(t)["Ceph Storage"] {
				t.Errorf("SRENIX_PROBE_CEPH=%q must NOT disable the probe (only exactly \"off\" does)", v)
			}
		})
	}
}

func TestCoreAnalyzers_RegisteredByDefault(t *testing.T) {
	names := analyzerNames(t)
	for env, an := range coreAnalyzerToggles {
		if !names[an] {
			t.Errorf("analyzer %q (gated by %s) must be registered by default — these are the secret-chain core value, defaults flipped OFF", an, env)
		}
	}
}

func TestCoreAnalyzers_SkippedWhenEnvOff(t *testing.T) {
	for env, an := range coreAnalyzerToggles {
		t.Run(env, func(t *testing.T) {
			t.Setenv(env, "off")
			names := analyzerNames(t)
			if names[an] {
				t.Errorf("analyzer %q still registered with %s=off", an, env)
			}
			for otherEnv, other := range coreAnalyzerToggles {
				if otherEnv != env && !names[other] {
					t.Errorf("%s=off also dropped sibling analyzer %q (gated by %s)", env, other, otherEnv)
				}
			}
		})
	}
}

// TestExistingGatedProbes_StillToggle pins the pre-existing gate
// behavior so a refactor of RegisterOSS cannot regress the original 15.
func TestExistingGatedProbes_StillToggle(t *testing.T) {
	t.Setenv("SRENIX_PROBE_GPU_NODES", "off")
	if probeNames(t)["GPU Nodes"] {
		t.Error("SRENIX_PROBE_GPU_NODES=off must skip the GPU Nodes probe")
	}
}
