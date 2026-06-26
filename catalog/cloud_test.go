// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

package catalog

import (
	"strings"
	"testing"

	gcpprobes "github.com/srenix-ai/agentic-sre/internal/cloud/gcp"
	"github.com/srenix-ai/agentic-sre/pkg/registry"
)

// cloudProbeNames returns the Name() set RegisterCloudOSS registers for
// the given provider switches under the current environment.
func cloudProbeNames(t *testing.T, aws, gcp, azure bool) map[string]bool {
	t.Helper()
	r := registry.New()
	RegisterCloudOSS(r, aws, gcp, azure)
	out := map[string]bool{}
	for _, p := range r.CloudProbes() {
		out[p.Name()] = true
	}
	return out
}

// cloudProbeToggles maps each per-cloud-probe opt-out env var to the
// probe Name()s it gates. The chart's cloud.<provider>.probes.* values
// render these envs (srenix.cloudProbeToggleEnv); the env names are the
// public contract. EKS / GKE / AKS each gate BOTH the control-plane and
// node-pool probes — same asset, one values key.
var cloudProbeToggles = map[string][]string{
	"SRENIX_CLOUD_PROBE_AWS_RDS": {"aws-rds"},
	"SRENIX_CLOUD_PROBE_AWS_EBS": {"aws-ebs"},
	"SRENIX_CLOUD_PROBE_AWS_EKS": {"aws-eks-control-plane", "aws-eks-nodegroups"},
	"SRENIX_CLOUD_PROBE_AWS_IAM": {"aws-iam-roles"},
	"SRENIX_CLOUD_PROBE_AWS_ALB": {"aws-alb-target-health"},
	"SRENIX_CLOUD_PROBE_AWS_ACM": {"aws-acm-cert-expiry"},
	"SRENIX_CLOUD_PROBE_AWS_KMS": {"aws-kms-keys"},
	"SRENIX_CLOUD_PROBE_AWS_S3":  {"aws-s3-bucket-public-access"},
	"SRENIX_CLOUD_PROBE_AWS_VPC": {"aws-vpc-subnets"},

	"SRENIX_CLOUD_PROBE_GCP_CLOUDSQL": {"gcp-cloudsql"},
	"SRENIX_CLOUD_PROBE_GCP_DISKS":    {"gcp-persistent-disks"},
	"SRENIX_CLOUD_PROBE_GCP_GKE":      {"gcp-gke-control-plane", "gcp-gke-nodepools"},
	"SRENIX_CLOUD_PROBE_GCP_IAM":      {"gcp-iam-serviceaccounts"},
	"SRENIX_CLOUD_PROBE_GCP_SUBNETS":  {"gcp-subnets"},
	"SRENIX_CLOUD_PROBE_GCP_LB":       {"gcp-lb-backends"},
	"SRENIX_CLOUD_PROBE_GCP_CERTS":    {"gcp-managed-certs"},
	"SRENIX_CLOUD_PROBE_GCP_GCS":      {"gcp-gcs-public-access"},
	"SRENIX_CLOUD_PROBE_GCP_KMS":      {"gcp-kms"},

	"SRENIX_CLOUD_PROBE_AZURE_SQL":        {"azure-sql"},
	"SRENIX_CLOUD_PROBE_AZURE_DISKS":      {"azure-disks"},
	"SRENIX_CLOUD_PROBE_AZURE_AKS":        {"azure-aks-control-plane", "azure-aks-nodepools"},
	"SRENIX_CLOUD_PROBE_AZURE_IDENTITIES": {"azure-managed-identities"},
	"SRENIX_CLOUD_PROBE_AZURE_SUBNETS":    {"azure-subnets"},
	"SRENIX_CLOUD_PROBE_AZURE_APPGW":      {"azure-appgw-backends"},
	"SRENIX_CLOUD_PROBE_AZURE_CERTS":      {"azure-certs"},
	"SRENIX_CLOUD_PROBE_AZURE_STORAGE":    {"azure-storage-public-access"},
	"SRENIX_CLOUD_PROBE_AZURE_KEYVAULTS":  {"azure-keyvaults"},
}

func TestCloudProbes_AllRegisteredByDefault(t *testing.T) {
	names := cloudProbeNames(t, true, true, true)
	if len(names) != 30 {
		t.Errorf("expected 30 cloud probes registered with all providers enabled, got %d", len(names))
	}
	for env, probes := range cloudProbeToggles {
		for _, p := range probes {
			if !names[p] {
				t.Errorf("cloud probe %q (gated by %s) must be registered by default", p, env)
			}
		}
	}
}

func TestCloudProbes_ProviderDisabledRegistersNothing(t *testing.T) {
	if names := cloudProbeNames(t, false, false, false); len(names) != 0 {
		t.Errorf("expected zero cloud probes with all providers disabled, got %v", names)
	}
	// Per-provider isolation: enabling only GCP must not register
	// AWS/Azure probes.
	names := cloudProbeNames(t, false, true, false)
	for n := range names {
		if !strings.HasPrefix(n, "gcp-") {
			t.Errorf("GCP-only registration leaked non-GCP probe %q", n)
		}
	}
}

func TestCloudProbes_SkippedWhenEnvOff(t *testing.T) {
	for env, probes := range cloudProbeToggles {
		t.Run(env, func(t *testing.T) {
			t.Setenv(env, "off")
			names := cloudProbeNames(t, true, true, true)
			for _, p := range probes {
				if names[p] {
					t.Errorf("%s=off must skip probe %q registration", env, p)
				}
			}
			// Sibling isolation: only the gated probe(s) drop out.
			want := 30 - len(probes)
			if len(names) != want {
				t.Errorf("%s=off: expected %d probes registered, got %d", env, want, len(names))
			}
		})
	}
}

// registeredGCPSubnets returns the gcp-subnets probe RegisterCloudOSS
// registered under the current environment.
func registeredGCPSubnets(t *testing.T) gcpprobes.Subnets {
	t.Helper()
	r := registry.New()
	RegisterCloudOSS(r, false, true, false)
	for _, p := range r.CloudProbes() {
		if s, ok := p.(gcpprobes.Subnets); ok {
			return s
		}
	}
	t.Fatal("gcp-subnets probe not registered")
	return gcpprobes.Subnets{}
}

// TestCloudProbes_GCPSubnetsSmallPrefixEnv pins the env→probe wiring:
// SRENIX_CLOUD_PROBE_GCP_SUBNETS_SMALL_PREFIX must reach
// gcpprobes.Subnets.SmallPrefixThreshold (the chart renders the env
// from cloud.gcp.subnetsSmallPrefixThreshold).
func TestCloudProbes_GCPSubnetsSmallPrefixEnv(t *testing.T) {
	t.Setenv("SRENIX_CLOUD_PROBE_GCP_SUBNETS_SMALL_PREFIX", "28")
	if got := registeredGCPSubnets(t).SmallPrefixThreshold; got != 28 {
		t.Errorf("SmallPrefixThreshold = %d, want 28 (from SRENIX_CLOUD_PROBE_GCP_SUBNETS_SMALL_PREFIX)", got)
	}
}

func TestCloudProbes_GCPSubnetsSmallPrefixEnv_UnsetOrInvalid(t *testing.T) {
	// Unset → 0 (probe's compiled-in /26 default).
	t.Setenv("SRENIX_CLOUD_PROBE_GCP_SUBNETS_SMALL_PREFIX", "")
	if got := registeredGCPSubnets(t).SmallPrefixThreshold; got != 0 {
		t.Errorf("unset env: SmallPrefixThreshold = %d, want 0 (probe default)", got)
	}
	// Garbage and non-positive values must fall back to the default,
	// never poison the probe.
	for _, bad := range []string{"not-a-number", "-3", "0"} {
		t.Setenv("SRENIX_CLOUD_PROBE_GCP_SUBNETS_SMALL_PREFIX", bad)
		if got := registeredGCPSubnets(t).SmallPrefixThreshold; got != 0 {
			t.Errorf("env=%q: SmallPrefixThreshold = %d, want 0 (probe default)", bad, got)
		}
	}
}

func TestCloudProbes_NonOffValueKeepsRegistration(t *testing.T) {
	t.Setenv("SRENIX_CLOUD_PROBE_AWS_RDS", "false") // only "off" disables — mirrors SRENIX_PROBE_*
	names := cloudProbeNames(t, true, false, false)
	if !names["aws-rds"] {
		t.Errorf("SRENIX_CLOUD_PROBE_AWS_RDS=false (non-off) must keep aws-rds registered")
	}
}
