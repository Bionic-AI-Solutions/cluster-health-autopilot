// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package catalog

import (
	"log"
	"os"
	"strconv"

	awsprobes "github.com/Bionic-AI-Solutions/cluster-health-autopilot/internal/cloud/aws"
	azureprobes "github.com/Bionic-AI-Solutions/cluster-health-autopilot/internal/cloud/azure"
	gcpprobes "github.com/Bionic-AI-Solutions/cluster-health-autopilot/internal/cloud/gcp"
	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/pkg/registry"
)

// RegisterCloudOSS registers the OSS cloud-resource probe set. Each
// per-provider group of probes is gated on whether that provider was
// configured at process start — when awsEnabled is false the AWS probes
// are NOT registered and pay zero overhead per cycle. Within an enabled
// provider, each probe is independently disablable via
// CHA_CLOUD_PROBE_<PROVIDER>_<NAME>=off (default ON) — the same opt-out
// pattern as the K8s CHA_PROBE_* gates. The chart's
// cloud.<provider>.probes.* values render these envs (see
// cha.cloudProbeToggleEnv in _helpers.tpl). The control-plane toggles
// (EKS / GKE / AKS) each gate BOTH the control-plane and the node-pool
// probe — they watch the same asset and share one values key.
//
// Cloud probes share the report.AssembleEntries pipeline with K8s
// probes via the cloudprobe.Probe contract; downstream rendering
// (Slack / Alertmanager / DriftReport / ticketing) is unchanged.
//
// All three provider probe sets shipped in v1.8: M1 = AWS (10 probes),
// M2 = GCP (10) + Azure (10). Live-mode signal coverage today: AWS
// fetches every signal live. GCP Cloud SQL storage-% and Azure SQL
// storage-% / App Gateway backend health are fetched via the cloud
// Monitoring APIs (best-effort — "not measured" when the metric is
// unavailable). GCP subnet IP utilization is CAPACITY-ONLY in live
// mode: GCP exposes no cheap used-IP count (Network Analyzer insights
// live behind the Recommender API); the probe instead flags
// small-capacity subnets (threshold configurable via
// CHA_CLOUD_PROBE_GCP_SUBNETS_SMALL_PREFIX / the chart's
// cloud.gcp.subnetsSmallPrefixThreshold). Azure subnet utilization is
// MEASURED live: used IPs are counted from every subnet-attached
// resource (NIC IP configurations, App Gateway IP configs, IP-config
// profiles, private endpoints), so available = total - used. See
// internal/cloud/{gcp,azure}/live.go for the exact set.
func RegisterCloudOSS(reg *registry.Registry, awsEnabled, gcpEnabled, azureEnabled bool) {
	// NOTE: the literal `os.Getenv("CHA_...") != "off"` form below is
	// load-bearing — the chartgate toggle-drift/default-off regex
	// scanners match it verbatim; don't refactor it into a helper.
	if awsEnabled {
		if os.Getenv("CHA_CLOUD_PROBE_AWS_RDS") != "off" {
			reg.RegisterCloudProbe(awsprobes.RDS{})
		}
		if os.Getenv("CHA_CLOUD_PROBE_AWS_EBS") != "off" {
			reg.RegisterCloudProbe(awsprobes.EBSVolumes{})
		}
		if os.Getenv("CHA_CLOUD_PROBE_AWS_EKS") != "off" {
			reg.RegisterCloudProbe(awsprobes.EKSControlPlane{}, awsprobes.EKSNodeGroups{})
		}
		if os.Getenv("CHA_CLOUD_PROBE_AWS_IAM") != "off" {
			reg.RegisterCloudProbe(awsprobes.IAMRoles{})
		}
		if os.Getenv("CHA_CLOUD_PROBE_AWS_ALB") != "off" {
			reg.RegisterCloudProbe(awsprobes.ALBTargetHealth{})
		}
		if os.Getenv("CHA_CLOUD_PROBE_AWS_ACM") != "off" {
			reg.RegisterCloudProbe(awsprobes.ACMCertExpiry{})
		}
		if os.Getenv("CHA_CLOUD_PROBE_AWS_KMS") != "off" {
			reg.RegisterCloudProbe(awsprobes.KMSKeys{})
		}
		if os.Getenv("CHA_CLOUD_PROBE_AWS_S3") != "off" {
			reg.RegisterCloudProbe(awsprobes.S3BucketPublicAccess{})
		}
		if os.Getenv("CHA_CLOUD_PROBE_AWS_VPC") != "off" {
			reg.RegisterCloudProbe(awsprobes.VPCSubnets{})
		}
	}
	if gcpEnabled {
		if os.Getenv("CHA_CLOUD_PROBE_GCP_CLOUDSQL") != "off" {
			reg.RegisterCloudProbe(gcpprobes.CloudSQL{})
		}
		if os.Getenv("CHA_CLOUD_PROBE_GCP_DISKS") != "off" {
			reg.RegisterCloudProbe(gcpprobes.PersistentDisks{})
		}
		if os.Getenv("CHA_CLOUD_PROBE_GCP_GKE") != "off" {
			reg.RegisterCloudProbe(gcpprobes.GKEControlPlane{}, gcpprobes.GKENodePools{})
		}
		if os.Getenv("CHA_CLOUD_PROBE_GCP_IAM") != "off" {
			reg.RegisterCloudProbe(gcpprobes.IAMServiceAccounts{})
		}
		if os.Getenv("CHA_CLOUD_PROBE_GCP_SUBNETS") != "off" {
			reg.RegisterCloudProbe(gcpprobes.Subnets{SmallPrefixThreshold: gcpSubnetSmallPrefix()})
		}
		if os.Getenv("CHA_CLOUD_PROBE_GCP_LB") != "off" {
			reg.RegisterCloudProbe(gcpprobes.LoadBalancerBackends{})
		}
		if os.Getenv("CHA_CLOUD_PROBE_GCP_CERTS") != "off" {
			reg.RegisterCloudProbe(gcpprobes.ManagedCertificates{})
		}
		if os.Getenv("CHA_CLOUD_PROBE_GCP_GCS") != "off" {
			reg.RegisterCloudProbe(gcpprobes.GCSPublicAccess{})
		}
		if os.Getenv("CHA_CLOUD_PROBE_GCP_KMS") != "off" {
			reg.RegisterCloudProbe(gcpprobes.KMSKeys{})
		}
	}
	if azureEnabled {
		if os.Getenv("CHA_CLOUD_PROBE_AZURE_SQL") != "off" {
			reg.RegisterCloudProbe(azureprobes.SQLDatabases{})
		}
		if os.Getenv("CHA_CLOUD_PROBE_AZURE_DISKS") != "off" {
			reg.RegisterCloudProbe(azureprobes.Disks{})
		}
		if os.Getenv("CHA_CLOUD_PROBE_AZURE_AKS") != "off" {
			reg.RegisterCloudProbe(azureprobes.AKSControlPlane{}, azureprobes.AKSNodePools{})
		}
		if os.Getenv("CHA_CLOUD_PROBE_AZURE_IDENTITIES") != "off" {
			reg.RegisterCloudProbe(azureprobes.ManagedIdentities{})
		}
		if os.Getenv("CHA_CLOUD_PROBE_AZURE_SUBNETS") != "off" {
			reg.RegisterCloudProbe(azureprobes.Subnets{})
		}
		if os.Getenv("CHA_CLOUD_PROBE_AZURE_APPGW") != "off" {
			reg.RegisterCloudProbe(azureprobes.AppGatewayBackends{})
		}
		if os.Getenv("CHA_CLOUD_PROBE_AZURE_CERTS") != "off" {
			reg.RegisterCloudProbe(azureprobes.Certificates{})
		}
		if os.Getenv("CHA_CLOUD_PROBE_AZURE_STORAGE") != "off" {
			reg.RegisterCloudProbe(azureprobes.StoragePublicAccess{})
		}
		if os.Getenv("CHA_CLOUD_PROBE_AZURE_KEYVAULTS") != "off" {
			reg.RegisterCloudProbe(azureprobes.KeyVaults{})
		}
	}
}

// gcpSubnetSmallPrefix reads CHA_CLOUD_PROBE_GCP_SUBNETS_SMALL_PREFIX
// — the capacity-only GCP Subnets probe's small-prefix threshold (an
// unmeasured subnet whose primary CIDR is smaller than /<threshold>
// is flagged; see gcpprobes.Subnets.SmallPrefixThreshold). Rendered by
// the chart from cloud.gcp.subnetsSmallPrefixThreshold (a tuning knob
// next to the cha.cloudProbeToggleEnv opt-outs, NOT a registration
// gate). Unset, non-numeric, or non-positive values return 0 = the
// probe's compiled-in default (/26).
func gcpSubnetSmallPrefix() int {
	raw := os.Getenv("CHA_CLOUD_PROBE_GCP_SUBNETS_SMALL_PREFIX")
	if raw == "" {
		return 0
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		log.Printf("catalog: invalid CHA_CLOUD_PROBE_GCP_SUBNETS_SMALL_PREFIX=%q (want a positive integer prefix length); using the probe's compiled-in /26 default", raw)
		return 0
	}
	return n
}
