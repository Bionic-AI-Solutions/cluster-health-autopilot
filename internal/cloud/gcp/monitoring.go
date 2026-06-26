// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

package gcp

import (
	"context"
	"fmt"
	"strings"
	"time"

	monitoring "google.golang.org/api/monitoring/v3"
)

// monitoringQuerier abstracts the Cloud Monitoring time-series queries the
// live wrapper needs. The interface keeps LiveClient's existing concrete-
// service pattern intact while making the new monitoring path swappable in
// unit tests — no live-cloud integration required for CI coverage of the
// parsing logic.
type monitoringQuerier interface {
	// CloudSQLDiskUsedPercent returns the most recent disk utilization
	// (0..100, percent) for instanceID, or (-1, nil) when no data point is
	// available in the recent window (Cloud Monitoring has a few-minute
	// lag; the metric is also briefly absent for a freshly-created
	// instance). Callers treat a -1 result the same as "not measured" and
	// skip the storage-utilization check rather than reporting 0%.
	CloudSQLDiskUsedPercent(ctx context.Context, instanceID string) (int, error)
}

// cloudMonitoringQuerier is the production monitoringQuerier — it queries
// the Cloud Monitoring v3 TimeSeries API. Auth flows through the same ADC
// the rest of LiveClient uses (Workload Identity in-cluster; ADC locally),
// so the Helm chart annotation that grants Cloud SQL Admin already covers
// this when extended with the Monitoring Viewer role.
type cloudMonitoringQuerier struct {
	svc     *monitoring.Service
	project string
	// window controls how far back the timeSeries query reaches. Cloud
	// Monitoring's ingestion lag for SQL metrics is up to ~3 min; 5 min is
	// a tight bound that still catches steady-state values reliably.
	window time.Duration
}

// newCloudMonitoringQuerier constructs the production impl rooted at the
// given project. A nil svc is a programming error (caller didn't wire the
// monitoring.Service via NewService).
func newCloudMonitoringQuerier(svc *monitoring.Service, project string) *cloudMonitoringQuerier {
	return &cloudMonitoringQuerier{svc: svc, project: project, window: 5 * time.Minute}
}

// CloudSQLDiskUsedPercent queries
//
//	metric.type = "cloudsql.googleapis.com/database/disk/utilization"
//
// for the named instance and returns the most recent value as an integer
// percent. The metric is a 0..1 fraction; we round to nearest percent. A
// missing time-series (instance unmeasured / monitoring not enabled) is
// reported as -1 rather than an error — the probe treats -1 as "skip this
// check," which is the same fallback the schema-only path uses.
func (c *cloudMonitoringQuerier) CloudSQLDiskUsedPercent(ctx context.Context, instanceID string) (int, error) {
	if c.svc == nil {
		return -1, fmt.Errorf("gcp: monitoring service not initialised")
	}
	now := time.Now().UTC()
	filter := fmt.Sprintf(
		`metric.type="cloudsql.googleapis.com/database/disk/utilization" AND resource.labels.database_id="%s:%s"`,
		c.project, instanceID,
	)
	resp, err := c.svc.Projects.TimeSeries.List("projects/" + c.project).
		Filter(filter).
		IntervalStartTime(now.Add(-c.window).Format(time.RFC3339)).
		IntervalEndTime(now.Format(time.RFC3339)).
		AggregationAlignmentPeriod(fmt.Sprintf("%ds", int(c.window.Seconds()))).
		AggregationPerSeriesAligner("ALIGN_MEAN").
		Context(ctx).
		Do()
	if err != nil {
		return -1, fmt.Errorf("gcp: cloud monitoring timeSeries.list: %w", err)
	}
	return latestDiskUsedPercent(resp), nil
}

// latestDiskUsedPercent extracts the most recent disk utilization fraction
// from a TimeSeries response, returns it as a 0..100 integer percent, or -1
// when the response carries no usable point. Exposed (lowercase but
// callable from tests in the same package) so the response-parsing logic
// can be unit-tested without a live SDK.
func latestDiskUsedPercent(resp *monitoring.ListTimeSeriesResponse) int {
	if resp == nil || len(resp.TimeSeries) == 0 {
		return -1
	}
	// One series per database_id; the first carries the readings. Points
	// are time-ordered newest-first per the API contract.
	for _, ts := range resp.TimeSeries {
		for _, p := range ts.Points {
			if p == nil || p.Value == nil || p.Value.DoubleValue == nil {
				continue
			}
			frac := *p.Value.DoubleValue
			if frac < 0 {
				continue // defensive: bogus point
			}
			pct := int(frac*100 + 0.5) // round to nearest percent
			if pct > 100 {
				pct = 100 // cap stray over-100 values
			}
			return pct
		}
	}
	return -1
}

// SplitDatabaseID parses a Cloud Monitoring resource label
// "<project>:<instance>" back into its parts. Returns ("", id) when no
// colon is present (legacy / partial inputs).
func SplitDatabaseID(id string) (project, instance string) {
	if i := strings.IndexByte(id, ':'); i >= 0 {
		return id[:i], id[i+1:]
	}
	return "", id
}
