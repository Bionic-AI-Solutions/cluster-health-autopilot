// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package catalog

import (
	awsprobes "github.com/Bionic-AI-Solutions/cluster-health-autopilot/internal/cloud/aws"
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
// M1 ships only the AWS RDS probe end-to-end; subsequent commits on
// feat/cloud-probes add EBS, EKS, IAM, ALB, ACM, KMS, S3, VPC, plus
// GCP and Azure in M2.
func RegisterCloudOSS(reg *registry.Registry, awsEnabled, gcpEnabled, azureEnabled bool) {
	if awsEnabled {
		reg.RegisterCloudProbe(awsprobes.RDS{})
		// M1 follow-up: EBSVolumes{}, EKSControlPlane{}, EKSNodeGroups{},
		// IAMRoles{}, ALBTargetHealth{}, ACMCertExpiry{}, KMSKeys{},
		// S3BucketPublicAccess{}, VPCSubnetCapacity{}
	}
	if gcpEnabled {
		// M2: CloudSQL, PersistentDisks, GKE control plane + node pools,
		// IAM service accounts, LB backends, managed certs, GCS public
		// access, KMS keys, subnet capacity
	}
	if azureEnabled {
		// M2: Azure SQL, ManagedDisks, AKS control plane + node pools,
		// ManagedIdentities, AppGatewayBackends, certs, storage public
		// access, KeyVault, VNet subnets
	}
}
