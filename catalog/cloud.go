// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package catalog

import (
	awsprobes "github.com/Bionic-AI-Solutions/cluster-health-autopilot/internal/cloud/aws"
	azureprobes "github.com/Bionic-AI-Solutions/cluster-health-autopilot/internal/cloud/azure"
	gcpprobes "github.com/Bionic-AI-Solutions/cluster-health-autopilot/internal/cloud/gcp"
	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/pkg/registry"
)

// RegisterCloudOSS registers the OSS cloud-resource probe set. Each
// per-provider group of probes is gated on whether that provider was
// configured at process start — when awsEnabled is false the AWS probes
// are NOT registered and pay zero overhead per cycle.
//
// Cloud probes share the report.AssembleEntries pipeline with K8s
// probes via the cloudprobe.Probe contract; downstream rendering
// (Slack / Alertmanager / DriftReport / ticketing) is unchanged.
//
// M1 shipped the AWS probe set (10 probes); M2 begins with the GCP
// Cloud SQL + Persistent Disk probes (this PR) and grows via
// follow-ups on feat/gcp-cloud-probes. Azure follows on
// feat/azure-cloud-probes.
func RegisterCloudOSS(reg *registry.Registry, awsEnabled, gcpEnabled, azureEnabled bool) {
	if awsEnabled {
		reg.RegisterCloudProbe(
			awsprobes.RDS{},
			awsprobes.EBSVolumes{},
			awsprobes.EKSControlPlane{},
			awsprobes.EKSNodeGroups{},
			awsprobes.IAMRoles{},
			awsprobes.ALBTargetHealth{},
			awsprobes.ACMCertExpiry{},
			awsprobes.KMSKeys{},
			awsprobes.S3BucketPublicAccess{},
			awsprobes.VPCSubnets{},
		)
	}
	if gcpEnabled {
		reg.RegisterCloudProbe(
			gcpprobes.CloudSQL{},
			gcpprobes.PersistentDisks{},
			gcpprobes.GKEControlPlane{},
			gcpprobes.GKENodePools{},
			gcpprobes.IAMServiceAccounts{},
			gcpprobes.Subnets{},
		)
	}
	if azureEnabled {
		reg.RegisterCloudProbe(
			azureprobes.SQLDatabases{},
			azureprobes.Disks{},
			azureprobes.AKSControlPlane{},
			azureprobes.AKSNodePools{},
			azureprobes.ManagedIdentities{},
			azureprobes.Subnets{},
		)
	}
}
