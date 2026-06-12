// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package catalog

import (
	"testing"

	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/pkg/registry"
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
// render these envs (cha.cloudProbeToggleEnv); the env names are the
// public contract. EKS / GKE / AKS each gate BOTH the control-plane and
// node-pool probes — same asset, one values key.
var cloudProbeToggles = map[string][]string{
	"CHA_CLOUD_PROBE_AWS_RDS": {"aws-rds"},
	"CHA_CLOUD_PROBE_AWS_EBS": {"aws-ebs"},
	"CHA_CLOUD_PROBE_AWS_EKS": {"aws-eks-control-plane", "aws-eks-nodegroups"},
	"CHA_CLOUD_PROBE_AWS_IAM": {"aws-iam-roles"},
	"CHA_CLOUD_PROBE_AWS_ALB": {"aws-alb-target-health"},
	"CHA_CLOUD_PROBE_AWS_ACM": {"aws-acm-cert-expiry"},
	"CHA_CLOUD_PROBE_AWS_KMS": {"aws-kms-keys"},
	"CHA_CLOUD_PROBE_AWS_S3":  {"aws-s3-bucket-public-access"},
	"CHA_CLOUD_PROBE_AWS_VPC": {"aws-vpc-subnets"},

	"CHA_CLOUD_PROBE_GCP_CLOUDSQL": {"gcp-cloudsql"},
	"CHA_CLOUD_PROBE_GCP_DISKS":    {"gcp-persistent-disks"},
	"CHA_CLOUD_PROBE_GCP_GKE":      {"gcp-gke-control-plane", "gcp-gke-nodepools"},
	"CHA_CLOUD_PROBE_GCP_IAM":      {"gcp-iam-serviceaccounts"},
	"CHA_CLOUD_PROBE_GCP_SUBNETS":  {"gcp-subnets"},
	"CHA_CLOUD_PROBE_GCP_LB":       {"gcp-lb-backends"},
	"CHA_CLOUD_PROBE_GCP_CERTS":    {"gcp-managed-certs"},
	"CHA_CLOUD_PROBE_GCP_GCS":      {"gcp-gcs-public-access"},
	"CHA_CLOUD_PROBE_GCP_KMS":      {"gcp-kms"},

	"CHA_CLOUD_PROBE_AZURE_SQL":        {"azure-sql"},
	"CHA_CLOUD_PROBE_AZURE_DISKS":      {"azure-disks"},
	"CHA_CLOUD_PROBE_AZURE_AKS":        {"azure-aks-control-plane", "azure-aks-nodepools"},
	"CHA_CLOUD_PROBE_AZURE_IDENTITIES": {"azure-managed-identities"},
	"CHA_CLOUD_PROBE_AZURE_SUBNETS":    {"azure-subnets"},
	"CHA_CLOUD_PROBE_AZURE_APPGW":      {"azure-appgw-backends"},
	"CHA_CLOUD_PROBE_AZURE_CERTS":      {"azure-certs"},
	"CHA_CLOUD_PROBE_AZURE_STORAGE":    {"azure-storage-public-access"},
	"CHA_CLOUD_PROBE_AZURE_KEYVAULTS":  {"azure-keyvaults"},
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
		if n[:4] != "gcp-" {
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

func TestCloudProbes_NonOffValueKeepsRegistration(t *testing.T) {
	t.Setenv("CHA_CLOUD_PROBE_AWS_RDS", "false") // only "off" disables — mirrors CHA_PROBE_*
	names := cloudProbeNames(t, true, false, false)
	if !names["aws-rds"] {
		t.Errorf("CHA_CLOUD_PROBE_AWS_RDS=false (non-off) must keep aws-rds registered")
	}
}
