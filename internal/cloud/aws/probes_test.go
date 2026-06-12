// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0
//
// Tests for the 9 non-RDS probes. RDS lives in rds_probe_test.go.
// All tests use the shared fakeAWS / fakeSource from fake_test.go.
package aws

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	pkgaws "github.com/Bionic-AI-Solutions/cluster-health-autopilot/pkg/cloud/aws"
)

// --- EBS ---------------------------------------------------------------

func TestEBS_SkippedWithoutAWS(t *testing.T) {
	r := (EBSVolumes{}).Run(context.Background(), &fakeSource{})
	if r.Component.Status != "SKIPPED" {
		t.Errorf("Status=%q want SKIPPED", r.Component.Status)
	}
}

func TestEBS_DetachedTooLongWarns(t *testing.T) {
	src := &fakeSource{aws: &fakeAWS{
		region:  "us-east-1",
		volumes: []pkgaws.Volume{{VolumeID: "vol-1", State: "available", SizeGB: 100, DetachedDuration: 14 * 24 * time.Hour}},
	}}
	r := (EBSVolumes{}).Run(context.Background(), src)
	if r.Component.Status != "DEGRADED" {
		t.Errorf("Status=%q want DEGRADED for 14d detached", r.Component.Status)
	}
	if len(r.Findings) != 1 {
		t.Fatalf("want 1 finding got %d", len(r.Findings))
	}
}

func TestEBS_ErrorStateCritical(t *testing.T) {
	src := &fakeSource{aws: &fakeAWS{
		region:  "us-east-1",
		volumes: []pkgaws.Volume{{VolumeID: "vol-bad", State: "error", SizeGB: 100}},
	}}
	r := (EBSVolumes{}).Run(context.Background(), src)
	if r.Component.Status != "CRITICAL" {
		t.Errorf("Status=%q want CRITICAL", r.Component.Status)
	}
}

func TestEBS_HealthyInUseAndAvailable(t *testing.T) {
	src := &fakeSource{aws: &fakeAWS{
		region: "us-east-1",
		volumes: []pkgaws.Volume{
			{VolumeID: "vol-1", State: "in-use", AttachedToEC2: "i-x"},
			{VolumeID: "vol-2", State: "available", DetachedDuration: 1 * time.Hour},
		},
	}}
	r := (EBSVolumes{}).Run(context.Background(), src)
	if r.Component.Status != "HEALTHY" {
		t.Errorf("Status=%q want HEALTHY", r.Component.Status)
	}
}

// --- EKS Control Plane -------------------------------------------------

func TestEKSControlPlane_SkippedWithoutClusterEnv(t *testing.T) {
	os.Unsetenv(eksClusterEnv)
	src := &fakeSource{aws: &fakeAWS{region: "us-east-1"}}
	r := (EKSControlPlane{}).Run(context.Background(), src)
	if r.Component.Status != "SKIPPED" {
		t.Errorf("Status=%q want SKIPPED without env", r.Component.Status)
	}
}

func TestEKSControlPlane_NotFoundCritical(t *testing.T) {
	t.Setenv(eksClusterEnv, "missing-cluster")
	src := &fakeSource{aws: &fakeAWS{region: "us-east-1"}} // no clusters
	r := (EKSControlPlane{}).Run(context.Background(), src)
	if r.Component.Status != "CRITICAL" {
		t.Errorf("Status=%q want CRITICAL when cluster missing", r.Component.Status)
	}
}

func TestEKSControlPlane_ActiveHealthy(t *testing.T) {
	t.Setenv(eksClusterEnv, "prod")
	src := &fakeSource{aws: &fakeAWS{
		region:   "us-east-1",
		clusters: []pkgaws.EKSCluster{{Name: "prod", Status: "ACTIVE", Version: "1.30"}},
	}}
	r := (EKSControlPlane{}).Run(context.Background(), src)
	if r.Component.Status != "HEALTHY" {
		t.Errorf("Status=%q want HEALTHY", r.Component.Status)
	}
}

func TestEKSControlPlane_FailedStateCritical(t *testing.T) {
	t.Setenv(eksClusterEnv, "broken")
	src := &fakeSource{aws: &fakeAWS{
		region:   "us-east-1",
		clusters: []pkgaws.EKSCluster{{Name: "broken", Status: "FAILED", Version: "1.30"}},
	}}
	r := (EKSControlPlane{}).Run(context.Background(), src)
	if r.Component.Status != "CRITICAL" {
		t.Errorf("Status=%q want CRITICAL for FAILED", r.Component.Status)
	}
}

// --- EKS Node Groups ---------------------------------------------------

func TestEKSNodeGroups_DegradedCritical(t *testing.T) {
	t.Setenv(eksClusterEnv, "prod")
	src := &fakeSource{aws: &fakeAWS{
		region: "us-east-1",
		nodeGroups: []pkgaws.EKSNodeGroup{
			{ClusterName: "prod", Name: "ng-1", Status: "DEGRADED", HealthIssues: []string{"InstanceLimitExceeded"}},
		},
	}}
	r := (EKSNodeGroups{}).Run(context.Background(), src)
	if r.Component.Status != "CRITICAL" {
		t.Errorf("Status=%q want CRITICAL for DEGRADED", r.Component.Status)
	}
}

func TestEKSNodeGroups_ActiveHealthyButHealthIssuesWarn(t *testing.T) {
	t.Setenv(eksClusterEnv, "prod")
	src := &fakeSource{aws: &fakeAWS{
		region: "us-east-1",
		nodeGroups: []pkgaws.EKSNodeGroup{
			{ClusterName: "prod", Name: "ng-1", Status: "ACTIVE", HealthIssues: []string{"AsgInstanceLaunchFailures"}},
		},
	}}
	r := (EKSNodeGroups{}).Run(context.Background(), src)
	if r.Component.Status != "DEGRADED" {
		t.Errorf("Status=%q want DEGRADED for ACTIVE-with-issues", r.Component.Status)
	}
}

// --- IAM ---------------------------------------------------------------

func TestIAM_SkippedWithoutRolesEnv(t *testing.T) {
	os.Unsetenv(iamRolesEnv)
	src := &fakeSource{aws: &fakeAWS{region: "us-east-1"}}
	r := (IAMRoles{}).Run(context.Background(), src)
	if r.Component.Status != "SKIPPED" {
		t.Errorf("Status=%q want SKIPPED without env", r.Component.Status)
	}
}

func TestIAM_MissingRoleCritical(t *testing.T) {
	t.Setenv(iamRolesEnv, "arn:aws:iam::123:role/cha-irsa")
	src := &fakeSource{aws: &fakeAWS{region: "us-east-1"}} // no roles configured
	r := (IAMRoles{}).Run(context.Background(), src)
	if r.Component.Status != "CRITICAL" {
		t.Errorf("Status=%q want CRITICAL for missing role", r.Component.Status)
	}
}

func TestIAM_PresentHealthy(t *testing.T) {
	t.Setenv(iamRolesEnv, "arn:aws:iam::123:role/cha-irsa")
	src := &fakeSource{aws: &fakeAWS{
		region: "us-east-1",
		roles: []pkgaws.IAMRole{
			{ARN: "arn:aws:iam::123:role/cha-irsa", Exists: true, HasTrustPolicy: true},
		},
	}}
	r := (IAMRoles{}).Run(context.Background(), src)
	if r.Component.Status != "HEALTHY" {
		t.Errorf("Status=%q want HEALTHY", r.Component.Status)
	}
}

// --- ALB ---------------------------------------------------------------

func TestALB_ZeroHealthyCritical(t *testing.T) {
	src := &fakeSource{aws: &fakeAWS{
		region: "us-east-1",
		targetGroups: []pkgaws.ALBTargetGroup{
			{Name: "tg-1", ARN: "arn:...", Protocol: "HTTP", Port: 80, HealthyCount: 0, UnhealthyCount: 3},
		},
	}}
	r := (ALBTargetHealth{}).Run(context.Background(), src)
	if r.Component.Status != "CRITICAL" {
		t.Errorf("Status=%q want CRITICAL for 0-healthy-3-unhealthy", r.Component.Status)
	}
}

func TestALB_EmptyTargetGroupWarn(t *testing.T) {
	src := &fakeSource{aws: &fakeAWS{
		region: "us-east-1",
		targetGroups: []pkgaws.ALBTargetGroup{
			{Name: "tg-empty", HealthyCount: 0, UnhealthyCount: 0, InitialCount: 0},
		},
	}}
	r := (ALBTargetHealth{}).Run(context.Background(), src)
	if r.Component.Status != "DEGRADED" {
		t.Errorf("Status=%q want DEGRADED for empty TG", r.Component.Status)
	}
}

// CHA-com RCA join contract (ai/cloudcontext): the 0-healthy message
// carries a " (lb: <LB DNS name>)" suffix when the live wrapper
// resolved the owning load balancer. See internal/cloud/contract_test.go.
func TestALB_ZeroHealthyMessageCarriesLBJoinKey(t *testing.T) {
	src := &fakeSource{aws: &fakeAWS{
		region: "us-east-1",
		targetGroups: []pkgaws.ALBTargetGroup{
			{Name: "tg-1", ARN: "arn:...", Protocol: "HTTP", Port: 80, HealthyCount: 0, UnhealthyCount: 3,
				LoadBalancerDNS: "my-alb-123.us-east-1.elb.amazonaws.com"},
		},
	}}
	r := (ALBTargetHealth{}).Run(context.Background(), src)
	if len(r.Findings) != 1 {
		t.Fatalf("want 1 finding got %d", len(r.Findings))
	}
	want := "Target group tg-1 has 0 healthy targets (3 unhealthy) on HTTP:80 (lb: my-alb-123.us-east-1.elb.amazonaws.com)"
	if r.Findings[0].Message != want {
		t.Errorf("Message=%q want %q", r.Findings[0].Message, want)
	}
}

// Backward compat: no LoadBalancerDNS (old snapshot files, unresolvable
// LB) → the pre-enrichment message, no empty "(lb: )" suffix.
func TestALB_ZeroHealthyMessageUnsuffixedWithoutLBDNS(t *testing.T) {
	src := &fakeSource{aws: &fakeAWS{
		region: "us-east-1",
		targetGroups: []pkgaws.ALBTargetGroup{
			{Name: "tg-1", ARN: "arn:...", Protocol: "HTTP", Port: 80, HealthyCount: 0, UnhealthyCount: 3},
		},
	}}
	r := (ALBTargetHealth{}).Run(context.Background(), src)
	if len(r.Findings) != 1 {
		t.Fatalf("want 1 finding got %d", len(r.Findings))
	}
	want := "Target group tg-1 has 0 healthy targets (3 unhealthy) on HTTP:80"
	if r.Findings[0].Message != want {
		t.Errorf("Message=%q want %q", r.Findings[0].Message, want)
	}
}

func TestALB_HealthyHasHealthy(t *testing.T) {
	src := &fakeSource{aws: &fakeAWS{
		region: "us-east-1",
		targetGroups: []pkgaws.ALBTargetGroup{
			{Name: "tg-ok", HealthyCount: 3, UnhealthyCount: 0},
		},
	}}
	r := (ALBTargetHealth{}).Run(context.Background(), src)
	if r.Component.Status != "HEALTHY" {
		t.Errorf("Status=%q want HEALTHY", r.Component.Status)
	}
}

// --- ACM ---------------------------------------------------------------

func TestACM_ExpiredCritical(t *testing.T) {
	src := &fakeSource{aws: &fakeAWS{
		region: "us-east-1",
		certs: []pkgaws.ACMCertificate{
			{ARN: "arn:...", DomainName: "old.example", Status: "ISSUED", NotAfter: time.Now().Add(-1 * time.Hour)},
		},
	}}
	r := (ACMCertExpiry{}).Run(context.Background(), src)
	if r.Component.Status != "CRITICAL" {
		t.Errorf("Status=%q want CRITICAL for expired cert", r.Component.Status)
	}
}

func TestACM_NearExpiryWarn(t *testing.T) {
	src := &fakeSource{aws: &fakeAWS{
		region: "us-east-1",
		certs: []pkgaws.ACMCertificate{
			{ARN: "arn:...", DomainName: "soon.example", Status: "ISSUED", NotAfter: time.Now().Add(7 * 24 * time.Hour)},
		},
	}}
	r := (ACMCertExpiry{}).Run(context.Background(), src)
	if r.Component.Status != "DEGRADED" {
		t.Errorf("Status=%q want DEGRADED for 7d-to-expiry", r.Component.Status)
	}
}

func TestACM_FailedStateCritical(t *testing.T) {
	src := &fakeSource{aws: &fakeAWS{
		region: "us-east-1",
		certs: []pkgaws.ACMCertificate{
			{ARN: "arn:...", DomainName: "fail.example", Status: "FAILED"},
		},
	}}
	r := (ACMCertExpiry{}).Run(context.Background(), src)
	if r.Component.Status != "CRITICAL" {
		t.Errorf("Status=%q want CRITICAL for FAILED", r.Component.Status)
	}
}

func TestACM_HealthyIssuedFarOut(t *testing.T) {
	src := &fakeSource{aws: &fakeAWS{
		region: "us-east-1",
		certs: []pkgaws.ACMCertificate{
			{ARN: "arn:...", DomainName: "good.example", Status: "ISSUED", NotAfter: time.Now().Add(60 * 24 * time.Hour)},
		},
	}}
	r := (ACMCertExpiry{}).Run(context.Background(), src)
	if r.Component.Status != "HEALTHY" {
		t.Errorf("Status=%q want HEALTHY", r.Component.Status)
	}
}

// --- KMS ---------------------------------------------------------------

func TestKMS_PendingDeletionImmCritical(t *testing.T) {
	src := &fakeSource{aws: &fakeAWS{
		region: "us-east-1",
		keys: []pkgaws.KMSKey{
			{KeyID: "k1", State: "PendingDeletion", DeletionDate: time.Now().Add(48 * time.Hour)},
		},
	}}
	r := (KMSKeys{}).Run(context.Background(), src)
	if r.Component.Status != "CRITICAL" {
		t.Errorf("Status=%q want CRITICAL for deletion <7d", r.Component.Status)
	}
}

func TestKMS_DisabledWarn(t *testing.T) {
	src := &fakeSource{aws: &fakeAWS{
		region: "us-east-1",
		keys: []pkgaws.KMSKey{
			{KeyID: "k2", State: "Disabled"},
		},
	}}
	r := (KMSKeys{}).Run(context.Background(), src)
	if r.Component.Status != "DEGRADED" {
		t.Errorf("Status=%q want DEGRADED for Disabled", r.Component.Status)
	}
}

func TestKMS_EnabledHealthy(t *testing.T) {
	src := &fakeSource{aws: &fakeAWS{
		region: "us-east-1",
		keys: []pkgaws.KMSKey{
			{KeyID: "k3", State: "Enabled", Enabled: true},
		},
	}}
	r := (KMSKeys{}).Run(context.Background(), src)
	if r.Component.Status != "HEALTHY" {
		t.Errorf("Status=%q want HEALTHY", r.Component.Status)
	}
}

// --- S3 ----------------------------------------------------------------

func TestS3_NoPABCritical(t *testing.T) {
	src := &fakeSource{aws: &fakeAWS{
		region: "us-east-1",
		bucketPAB: []pkgaws.S3BucketPAB{
			{Bucket: "unprotected", HasPolicyError: true},
		},
	}}
	r := (S3BucketPublicAccess{}).Run(context.Background(), src)
	if r.Component.Status != "CRITICAL" {
		t.Errorf("Status=%q want CRITICAL for missing PAB", r.Component.Status)
	}
}

func TestS3_PartialPABCritical(t *testing.T) {
	src := &fakeSource{aws: &fakeAWS{
		region: "us-east-1",
		bucketPAB: []pkgaws.S3BucketPAB{
			{Bucket: "partial", BlockPublicAcls: true, IgnorePublicAcls: false, BlockPublicPolicy: true, RestrictPublicBuckets: true},
		},
	}}
	r := (S3BucketPublicAccess{}).Run(context.Background(), src)
	if r.Component.Status != "CRITICAL" {
		t.Errorf("Status=%q want CRITICAL for partial PAB", r.Component.Status)
	}
}

func TestS3_FullPABHealthy(t *testing.T) {
	src := &fakeSource{aws: &fakeAWS{
		region: "us-east-1",
		bucketPAB: []pkgaws.S3BucketPAB{
			{Bucket: "locked", BlockPublicAcls: true, IgnorePublicAcls: true, BlockPublicPolicy: true, RestrictPublicBuckets: true},
		},
	}}
	r := (S3BucketPublicAccess{}).Run(context.Background(), src)
	if r.Component.Status != "HEALTHY" {
		t.Errorf("Status=%q want HEALTHY for full PAB", r.Component.Status)
	}
}

// --- VPC subnets -------------------------------------------------------

func TestVPC_NearExhaustionCritical(t *testing.T) {
	src := &fakeSource{aws: &fakeAWS{
		region: "us-east-1",
		subnets: []pkgaws.VPCSubnet{
			{SubnetID: "subnet-1", VPCID: "vpc-1", CIDRBlock: "10.0.0.0/28", AvailabilityZone: "us-east-1a", AvailableIPv4AddressCount: 2},
		},
	}}
	r := (VPCSubnets{}).Run(context.Background(), src)
	if r.Component.Status != "CRITICAL" {
		t.Errorf("Status=%q want CRITICAL for <3 IPs", r.Component.Status)
	}
}

func TestVPC_LowIPsWarn(t *testing.T) {
	src := &fakeSource{aws: &fakeAWS{
		region: "us-east-1",
		subnets: []pkgaws.VPCSubnet{
			{SubnetID: "subnet-1", AvailableIPv4AddressCount: 7},
		},
	}}
	r := (VPCSubnets{}).Run(context.Background(), src)
	if r.Component.Status != "DEGRADED" {
		t.Errorf("Status=%q want DEGRADED for 7 IPs", r.Component.Status)
	}
}

func TestVPC_AmpleIPsHealthy(t *testing.T) {
	src := &fakeSource{aws: &fakeAWS{
		region: "us-east-1",
		subnets: []pkgaws.VPCSubnet{
			{SubnetID: "subnet-1", AvailableIPv4AddressCount: 200},
		},
	}}
	r := (VPCSubnets{}).Run(context.Background(), src)
	if r.Component.Status != "HEALTHY" {
		t.Errorf("Status=%q want HEALTHY for 200 IPs", r.Component.Status)
	}
}

// --- generic probeFailed-on-error coverage -----------------------------

func TestProbe_ErrorYieldsProbeFailed(t *testing.T) {
	apiErr := errors.New("network unreachable")
	cases := []struct {
		name  string
		probe interface {
			Name() string
		}
		setup func(*fakeAWS)
	}{
		{"EBS", EBSVolumes{}, func(f *fakeAWS) { f.volumesErr = apiErr }},
		{"ALB", ALBTargetHealth{}, func(f *fakeAWS) { f.targetGroupsErr = apiErr }},
		{"ACM", ACMCertExpiry{}, func(f *fakeAWS) { f.certsErr = apiErr }},
		{"KMS", KMSKeys{}, func(f *fakeAWS) { f.keysErr = apiErr }},
		{"S3", S3BucketPublicAccess{}, func(f *fakeAWS) { f.bucketPABErr = apiErr }},
		{"VPC", VPCSubnets{}, func(f *fakeAWS) { f.subnetsErr = apiErr }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fa := &fakeAWS{region: "us-east-1"}
			tc.setup(fa)
			src := &fakeSource{aws: fa}
			// All these probes implement the same cloudprobe.Probe interface.
			type runner interface {
				Run(ctx context.Context, src interface {
					AWS() pkgaws.Client
				}) interface { /* probe.Result */
				}
			}
			_ = runner(nil) // type-assertion only; cast at the call site
			switch p := tc.probe.(type) {
			case EBSVolumes:
				if r := p.Run(context.Background(), src); r.Component.Status != "PROBE_FAILED" {
					t.Errorf("%s: status=%q want PROBE_FAILED", tc.name, r.Component.Status)
				}
			case ALBTargetHealth:
				if r := p.Run(context.Background(), src); r.Component.Status != "PROBE_FAILED" {
					t.Errorf("%s: status=%q want PROBE_FAILED", tc.name, r.Component.Status)
				}
			case ACMCertExpiry:
				if r := p.Run(context.Background(), src); r.Component.Status != "PROBE_FAILED" {
					t.Errorf("%s: status=%q want PROBE_FAILED", tc.name, r.Component.Status)
				}
			case KMSKeys:
				if r := p.Run(context.Background(), src); r.Component.Status != "PROBE_FAILED" {
					t.Errorf("%s: status=%q want PROBE_FAILED", tc.name, r.Component.Status)
				}
			case S3BucketPublicAccess:
				if r := p.Run(context.Background(), src); r.Component.Status != "PROBE_FAILED" {
					t.Errorf("%s: status=%q want PROBE_FAILED", tc.name, r.Component.Status)
				}
			case VPCSubnets:
				if r := p.Run(context.Background(), src); r.Component.Status != "PROBE_FAILED" {
					t.Errorf("%s: status=%q want PROBE_FAILED", tc.name, r.Component.Status)
				}
			}
		})
	}
}
