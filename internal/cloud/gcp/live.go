// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package gcp

import (
	"context"
	"fmt"
	"strings"
	"time"

	pkggcp "github.com/Bionic-AI-Solutions/cluster-health-autopilot/pkg/cloud/gcp"
	kms "google.golang.org/api/cloudkms/v1"
	compute "google.golang.org/api/compute/v1"
	container "google.golang.org/api/container/v1"
	iam "google.golang.org/api/iam/v1"
	monitoring "google.golang.org/api/monitoring/v3"
	sqladmin "google.golang.org/api/sqladmin/v1"
	storage "google.golang.org/api/storage/v1"
)

// LiveClient implements pkggcp.Client against the google.golang.org/api
// generated clients. Auth flows through Application Default
// Credentials (ADC): for the in-cluster CHA Deployment that's GKE
// Workload Identity (the Helm chart annotates the KSA with the GSA);
// locally it's `gcloud auth application-default login` or
// GOOGLE_APPLICATION_CREDENTIALS.
//
// All calls are READ-ONLY. The clients are constructed once in
// NewLiveClient and reused.
//
// NOTE ON VERIFICATION: this wrapper compiles against the real
// google.golang.org/api surface (so the API usage is correct), but it
// is not integration-tested against a live GCP project in CI — that
// requires credentials. The probe evaluation logic it feeds is unit-
// tested independently (see *_probe_test.go).
type LiveClient struct {
	project string
	region  string

	sql       *sqladmin.Service
	compute   *compute.Service
	container *container.Service
	iam       *iam.Service
	storage   *storage.Service
	kms       *kms.Service

	// monitor populates time-series-only signals (Cloud SQL disk usage
	// etc.) that the SQL Admin / Compute APIs don't expose directly. nil
	// is acceptable — callers fall back to the "not measured" sentinel
	// (DiskUsedPercent = -1), preserving the v1.8.x posture.
	monitor monitoringQuerier
}

// NewLiveClient constructs a Live GCP client bound to project+region.
// It initialises one generated-API service per resource family using
// ADC. region may be empty for project-global resources.
func NewLiveClient(ctx context.Context, project, region string) (*LiveClient, error) {
	if project == "" {
		return nil, fmt.Errorf("gcp: project is required")
	}
	sqlSvc, err := sqladmin.NewService(ctx)
	if err != nil {
		return nil, fmt.Errorf("gcp: sqladmin: %w", err)
	}
	computeSvc, err := compute.NewService(ctx)
	if err != nil {
		return nil, fmt.Errorf("gcp: compute: %w", err)
	}
	containerSvc, err := container.NewService(ctx)
	if err != nil {
		return nil, fmt.Errorf("gcp: container: %w", err)
	}
	iamSvc, err := iam.NewService(ctx)
	if err != nil {
		return nil, fmt.Errorf("gcp: iam: %w", err)
	}
	storageSvc, err := storage.NewService(ctx)
	if err != nil {
		return nil, fmt.Errorf("gcp: storage: %w", err)
	}
	kmsSvc, err := kms.NewService(ctx)
	if err != nil {
		return nil, fmt.Errorf("gcp: kms: %w", err)
	}
	// Cloud Monitoring is best-effort: if NewService fails (no Monitoring
	// Viewer role, network blocked, etc.) we proceed without it and fall
	// back to the "not measured" sentinel for time-series-only signals.
	// This preserves install-success on partial credential grants.
	var monitor monitoringQuerier
	if monSvc, mErr := monitoring.NewService(ctx); mErr == nil {
		monitor = newCloudMonitoringQuerier(monSvc, project)
	}
	return &LiveClient{
		project:   project,
		region:    region,
		sql:       sqlSvc,
		compute:   computeSvc,
		container: containerSvc,
		iam:       iamSvc,
		storage:   storageSvc,
		kms:       kmsSvc,
		monitor:   monitor,
	}, nil
}

// Project satisfies pkggcp.Client.
func (l *LiveClient) Project() string { return l.project }

// Region satisfies pkggcp.Client.
func (l *LiveClient) Region() string { return l.region }

// ListCloudSQLInstances satisfies pkggcp.Client.
func (l *LiveClient) ListCloudSQLInstances(ctx context.Context) ([]pkggcp.CloudSQLInstance, error) {
	resp, err := l.sql.Instances.List(l.project).Context(ctx).Do()
	if err != nil {
		return nil, err
	}
	out := make([]pkggcp.CloudSQLInstance, 0, len(resp.Items))
	for _, in := range resp.Items {
		inst := pkggcp.CloudSQLInstance{
			Name:            in.Name,
			DatabaseVersion: in.DatabaseVersion,
			State:           in.State,
			Region:          in.Region,
			ConnectionName:  in.ConnectionName,
			// DiskUsedPercent comes from Cloud Monitoring time-series
			// (the SQL Admin response doesn't carry it). G9: queried
			// here from monitoring.googleapis.com/cloudsql.googleapis
			// .com/database/disk/utilization. The querier returns -1
			// when no recent point exists (fresh instance / lag /
			// Monitoring API not reachable) — the probe then skips
			// the storage check rather than treating it as 0%.
			DiskUsedPercent: -1,
		}
		if l.monitor != nil {
			if pct, mErr := l.monitor.CloudSQLDiskUsedPercent(ctx, in.Name); mErr == nil {
				inst.DiskUsedPercent = pct
			}
			// Monitoring errors are swallowed: a query failure leaves
			// DiskUsedPercent at -1 (skip the check), same posture as
			// "not measured." A noisy probe is worse than a quiet one
			// for a non-critical signal.
		}
		if in.Settings != nil {
			inst.Tier = in.Settings.Tier
			inst.DiskSizeGB = in.Settings.DataDiskSizeGb
			inst.StorageAutoResize = in.Settings.StorageAutoResize != nil && *in.Settings.StorageAutoResize
			inst.AvailabilityType = in.Settings.AvailabilityType
			if in.Settings.BackupConfiguration != nil {
				inst.BackupConfigured = in.Settings.BackupConfiguration.Enabled
			}
		}
		if t, perr := time.Parse(time.RFC3339, in.CreateTime); perr == nil {
			inst.CreatedAt = t
		}
		out = append(out, inst)
	}
	return out, nil
}

// ListPersistentDisks satisfies pkggcp.Client. Uses AggregatedList so
// a single call covers all zones/regions in the project.
func (l *LiveClient) ListPersistentDisks(ctx context.Context) ([]pkggcp.PersistentDisk, error) {
	var out []pkggcp.PersistentDisk
	err := l.compute.Disks.AggregatedList(l.project).Pages(ctx, func(page *compute.DiskAggregatedList) error {
		for _, scoped := range page.Items {
			for _, d := range scoped.Disks {
				disk := pkggcp.PersistentDisk{
					Name:   d.Name,
					Status: d.Status,
					SizeGB: d.SizeGb,
					Type:   lastPathSegment(d.Type),
					Zone:   lastPathSegment(d.Zone),
					Region: lastPathSegment(d.Region),
				}
				if len(d.Users) > 0 {
					disk.AttachedToVM = lastPathSegment(d.Users[0])
				}
				if t, perr := time.Parse(time.RFC3339, d.CreationTimestamp); perr == nil {
					disk.CreatedAt = t
				}
				out = append(out, disk)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// GetGKECluster satisfies pkggcp.Client. name is the short cluster
// name; the location is taken from the bound region (falls back to a
// "-" wildcard list when region is empty).
func (l *LiveClient) GetGKECluster(ctx context.Context, name string) (*pkggcp.GKECluster, error) {
	location := l.region
	if location == "" {
		location = "-" // wildcard: search all locations
	}
	// projects/*/locations/*/clusters/* resource path.
	path := fmt.Sprintf("projects/%s/locations/%s/clusters/%s", l.project, location, name)
	c, err := l.container.Projects.Locations.Clusters.Get(path).Context(ctx).Do()
	if err != nil {
		if isNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	out := &pkggcp.GKECluster{
		Name:           c.Name,
		Status:         c.Status,
		Location:       c.Location,
		CurrentVersion: c.CurrentMasterVersion,
		NodeCount:      c.CurrentNodeCount,
		Endpoint:       c.Endpoint,
	}
	if t, perr := time.Parse(time.RFC3339, c.CreateTime); perr == nil {
		out.CreatedAt = t
	}
	return out, nil
}

// ListGKENodePools satisfies pkggcp.Client.
func (l *LiveClient) ListGKENodePools(ctx context.Context, clusterName string) ([]pkggcp.GKENodePool, error) {
	location := l.region
	if location == "" {
		location = "-"
	}
	parent := fmt.Sprintf("projects/%s/locations/%s/clusters/%s", l.project, location, clusterName)
	resp, err := l.container.Projects.Locations.Clusters.NodePools.List(parent).Context(ctx).Do()
	if err != nil {
		if isNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	out := make([]pkggcp.GKENodePool, 0, len(resp.NodePools))
	for _, np := range resp.NodePools {
		pool := pkggcp.GKENodePool{
			Name:        np.Name,
			ClusterName: clusterName,
			Status:      np.Status,
			NodeCount:   np.InitialNodeCount,
			Version:     np.Version,
		}
		if np.Autoscaling != nil {
			pool.Autoscaling = np.Autoscaling.Enabled
		}
		out = append(out, pool)
	}
	return out, nil
}

// ListServiceAccounts satisfies pkggcp.Client. Counts user-managed
// keys per SA via a second call (the List response doesn't inline
// keys).
func (l *LiveClient) ListServiceAccounts(ctx context.Context) ([]pkggcp.ServiceAccount, error) {
	parent := "projects/" + l.project
	var out []pkggcp.ServiceAccount
	err := l.iam.Projects.ServiceAccounts.List(parent).Pages(ctx, func(page *iam.ListServiceAccountsResponse) error {
		for _, sa := range page.Accounts {
			rec := pkggcp.ServiceAccount{
				Email:       sa.Email,
				DisplayName: sa.DisplayName,
				Disabled:    sa.Disabled,
			}
			// Count user-managed keys. Tolerate per-SA errors (a
			// deleted SA mid-iteration shouldn't fail the whole probe).
			keyResp, kerr := l.iam.Projects.ServiceAccounts.Keys.List(sa.Name).
				KeyTypes("USER_MANAGED").Context(ctx).Do()
			if kerr == nil {
				rec.KeyCount = len(keyResp.Keys)
			}
			out = append(out, rec)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// ListSubnets satisfies pkggcp.Client.
//
// LIMITATION: the Compute API Subnetwork resource does not expose a
// free-IP count. Accurate utilization needs the Monitoring API metric
// compute.googleapis.com/subnetwork/... — out of scope for this
// wrapper. We populate TotalIPCount from the primary CIDR mask and set
// AvailableIPCount = -1 ("not measured") so the IP-exhaustion probe
// SKIPS the utilization check rather than treating the subnet as 100%
// free, which would silently never fire in live mode. A follow-up
// (v1.9) can wire the Monitoring API for real utilization.
func (l *LiveClient) ListSubnets(ctx context.Context) ([]pkggcp.Subnet, error) {
	var out []pkggcp.Subnet
	err := l.compute.Subnetworks.AggregatedList(l.project).Pages(ctx, func(page *compute.SubnetworkAggregatedList) error {
		for _, scoped := range page.Items {
			for _, s := range scoped.Subnetworks {
				total := usableIPsFromCIDR(s.IpCidrRange)
				out = append(out, pkggcp.Subnet{
					Name:             s.Name,
					Network:          lastPathSegment(s.Network),
					Region:           lastPathSegment(s.Region),
					IPCIDRRange:      s.IpCidrRange,
					TotalIPCount:     total,
					AvailableIPCount: -1, // not measured — see LIMITATION above
				})
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// ListBackendServices satisfies pkggcp.Client. Backend health requires
// a per-backend GetHealth call; we aggregate the per-group health into
// the inlined healthy/unhealthy counts.
func (l *LiveClient) ListBackendServices(ctx context.Context) ([]pkggcp.BackendService, error) {
	// One ForwardingRules aggregated list per probe cycle builds the
	// backend-service → forwarding-rule (IP or name) map that feeds
	// ForwardingRule — the CHA-com "(lb: ...)" RCA join key.
	// Best-effort: a failure leaves the map empty (message enrichment
	// is never worth failing the probe over) and the probe falls back
	// to the backend-service name.
	frByBackendService := l.listForwardingRuleIndex(ctx)
	var out []pkggcp.BackendService
	err := l.compute.BackendServices.AggregatedList(l.project).Pages(ctx, func(page *compute.BackendServiceAggregatedList) error {
		for _, scoped := range page.Items {
			for _, bs := range scoped.BackendServices {
				rec := pkggcp.BackendService{
					Name:           bs.Name,
					Protocol:       bs.Protocol,
					TotalBackends:  len(bs.Backends),
					ForwardingRule: frByBackendService[bs.Name],
				}
				for _, b := range bs.Backends {
					h, herr := l.compute.BackendServices.GetHealth(
						l.project, bs.Name,
						&compute.ResourceGroupReference{Group: b.Group},
					).Context(ctx).Do()
					if herr != nil {
						continue
					}
					for _, hs := range h.HealthStatus {
						if hs.HealthState == "HEALTHY" {
							rec.HealthyCount++
						} else {
							rec.UnhealthyCount++
						}
					}
				}
				out = append(out, rec)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// listForwardingRuleIndex fetches every forwarding rule once
// (aggregated across regions; includes the global scope) and returns
// the backend-service-name → forwarding-rule-value index built by
// forwardingRuleIndex. Best-effort enrichment helper: returns nil
// (nil-map reads are safe; callers fall back to name / omit the suffix).
func (l *LiveClient) listForwardingRuleIndex(ctx context.Context) map[string]string {
	var rules []*compute.ForwardingRule
	err := l.compute.ForwardingRules.AggregatedList(l.project).Pages(ctx, func(page *compute.ForwardingRuleAggregatedList) error {
		for _, scoped := range page.Items {
			rules = append(rules, scoped.ForwardingRules...)
		}
		return nil
	})
	if err != nil {
		return nil
	}
	return forwardingRuleIndex(rules)
}

// forwardingRuleIndex maps backend-service name → forwarding-rule IP
// (preferred) or name. Only passthrough-LB rules carry a direct
// BackendService reference; proxy-based rules (rule.Target) would need
// a target-proxy + URL-map walk to resolve and are deliberately left
// unmapped — the probe falls back to the backend-service name.
func forwardingRuleIndex(rules []*compute.ForwardingRule) map[string]string {
	out := make(map[string]string, len(rules))
	for _, r := range rules {
		if r == nil || r.BackendService == "" {
			continue
		}
		value := r.IPAddress
		if value == "" {
			value = r.Name
		}
		if value == "" {
			continue
		}
		out[lastPathSegment(r.BackendService)] = value
	}
	return out
}

// ListManagedCertificates satisfies pkggcp.Client. Maps Compute SSL
// certificates of type MANAGED.
func (l *LiveClient) ListManagedCertificates(ctx context.Context) ([]pkggcp.ManagedCertificate, error) {
	var out []pkggcp.ManagedCertificate
	err := l.compute.SslCertificates.AggregatedList(l.project).Pages(ctx, func(page *compute.SslCertificateAggregatedList) error {
		for _, scoped := range page.Items {
			for _, c := range scoped.SslCertificates {
				if c.Managed == nil {
					continue // skip self-managed certs; they have no GCP-tracked status
				}
				rec := pkggcp.ManagedCertificate{
					Name:   c.Name,
					Status: c.Managed.Status,
				}
				if len(c.Managed.Domains) > 0 {
					rec.DomainName = c.Managed.Domains[0]
				}
				if t, perr := time.Parse(time.RFC3339, c.ExpireTime); perr == nil {
					rec.NotAfter = t
				}
				out = append(out, rec)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// ListBuckets satisfies pkggcp.Client. Detects an allUsers /
// allAuthenticatedUsers IAM binding via the bucket IAM policy.
func (l *LiveClient) ListBuckets(ctx context.Context) ([]pkggcp.Bucket, error) {
	resp, err := l.storage.Buckets.List(l.project).Context(ctx).Do()
	if err != nil {
		return nil, err
	}
	out := make([]pkggcp.Bucket, 0, len(resp.Items))
	for _, b := range resp.Items {
		rec := pkggcp.Bucket{Name: b.Name}
		if b.IamConfiguration != nil {
			if b.IamConfiguration.PublicAccessPrevention != "" {
				rec.PublicAccessPrevention = b.IamConfiguration.PublicAccessPrevention
			}
			if b.IamConfiguration.UniformBucketLevelAccess != nil {
				rec.UniformBucketLevelAccess = b.IamConfiguration.UniformBucketLevelAccess.Enabled
			}
		}
		// Inspect the IAM policy for public members.
		if pol, perr := l.storage.Buckets.GetIamPolicy(b.Name).Context(ctx).Do(); perr == nil {
			for _, binding := range pol.Bindings {
				for _, m := range binding.Members {
					if m == "allUsers" || m == "allAuthenticatedUsers" {
						rec.HasAllUsersBinding = true
					}
				}
			}
		}
		out = append(out, rec)
	}
	return out, nil
}

// ListKMSKeys satisfies pkggcp.Client. Walks key rings across the
// bound location (or global) and reports each crypto key's primary
// version state + rotation config.
func (l *LiveClient) ListKMSKeys(ctx context.Context) ([]pkggcp.KMSKey, error) {
	location := l.region
	if location == "" {
		location = "global"
	}
	parent := fmt.Sprintf("projects/%s/locations/%s", l.project, location)
	var out []pkggcp.KMSKey
	ringErr := l.kms.Projects.Locations.KeyRings.List(parent).Pages(ctx, func(rings *kms.ListKeyRingsResponse) error {
		for _, ring := range rings.KeyRings {
			keyErr := l.kms.Projects.Locations.KeyRings.CryptoKeys.List(ring.Name).Pages(ctx, func(keys *kms.ListCryptoKeysResponse) error {
				for _, k := range keys.CryptoKeys {
					rec := pkggcp.KMSKey{
						Name:              lastPathSegment(k.Name),
						Purpose:           k.Purpose,
						RotationScheduled: k.RotationPeriod != "" || k.NextRotationTime != "",
					}
					if k.Primary != nil {
						rec.PrimaryState = k.Primary.State
					}
					if t, perr := time.Parse(time.RFC3339, k.NextRotationTime); perr == nil {
						rec.NextRotation = t
					}
					out = append(out, rec)
				}
				return nil
			})
			if keyErr != nil {
				return keyErr
			}
		}
		return nil
	})
	if ringErr != nil {
		return nil, ringErr
	}
	return out, nil
}

// lastPathSegment returns the final segment of a slash-delimited GCP
// resource URL (e.g. ".../zones/us-central1-a" → "us-central1-a").
func lastPathSegment(s string) string {
	if s == "" {
		return ""
	}
	if i := strings.LastIndex(s, "/"); i >= 0 {
		return s[i+1:]
	}
	return s
}

// usableIPsFromCIDR returns the count of usable host addresses in an
// IPv4 CIDR. GCP reserves 4 addresses per subnet (network, default
// gateway, second-to-last, broadcast), so usable = 2^(32-mask) - 4.
// Returns 0 for unparseable / non-IPv4 ranges so the probe skips them.
func usableIPsFromCIDR(cidr string) int64 {
	i := strings.LastIndex(cidr, "/")
	if i < 0 {
		return 0
	}
	var mask int
	if _, err := fmt.Sscanf(cidr[i+1:], "%d", &mask); err != nil || mask < 0 || mask > 32 {
		return 0
	}
	hostBits := 32 - mask
	if hostBits <= 2 {
		return 0
	}
	total := int64(1) << uint(hostBits)
	usable := total - 4 // GCP reserves 4 per subnet
	if usable < 0 {
		return 0
	}
	return usable
}

// isNotFound reports whether err is a googleapi 404.
func isNotFound(err error) bool {
	return err != nil && strings.Contains(err.Error(), "404")
}
