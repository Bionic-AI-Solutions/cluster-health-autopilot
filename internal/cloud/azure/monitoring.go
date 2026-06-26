// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

package azure

import (
	"context"
	"fmt"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/monitor/azquery"
)

// monitoringQuerier abstracts the Azure Monitor metrics queries the live
// wrapper needs. The interface keeps LiveClient's existing concrete-client
// pattern intact while making the monitoring path swappable in unit tests —
// no live-cloud integration required for CI coverage of the parsing logic.
type monitoringQuerier interface {
	// SQLDatabaseStoragePercent returns the most recent storage_percent
	// metric (0..100) for the named SQL database (resourceURI is the
	// full ARM ID), or (-1, nil) when no data point is available in the
	// recent window. Callers treat -1 the same as "not measured" and
	// skip the storage check rather than reporting 0%.
	SQLDatabaseStoragePercent(ctx context.Context, resourceURI string) (int, error)
}

// metricsClient is the small slice of *azquery.MetricsClient the production
// querier depends on. Defined as an interface so unit tests can stub it
// without spinning up the SDK transport. The real client satisfies it
// directly.
type metricsClient interface {
	QueryResource(ctx context.Context, resourceURI string, options *azquery.MetricsClientQueryResourceOptions) (azquery.MetricsClientQueryResourceResponse, error)
}

// azureMonitoringQuerier is the production monitoringQuerier. Auth flows
// through the same DefaultAzureCredential as the rest of LiveClient — the
// Helm chart's AAD Workload Identity binding needs the `Monitoring Reader`
// role to be granted on the subscription / resource group.
type azureMonitoringQuerier struct {
	client metricsClient
	// window controls how far back the metrics query reaches. Azure
	// Monitor ingestion lag for SQL DB metrics is typically <1 min; 5 min
	// is a tight bound that still catches steady-state values reliably.
	window time.Duration
}

// newAzureMonitoringQuerier constructs the production impl rooted at the
// caller-supplied metrics client. A nil client is a programming error.
func newAzureMonitoringQuerier(client metricsClient) *azureMonitoringQuerier {
	return &azureMonitoringQuerier{client: client, window: 5 * time.Minute}
}

// SQLDatabaseStoragePercent queries the storage_percent metric and returns
// the latest Average value as a 0..100 integer percent. A missing time-
// series (unmeasured / Monitoring Reader role not granted) is reported as
// -1 rather than an error — the probe treats -1 as "skip this check," the
// same fallback the schema-only path uses.
func (q *azureMonitoringQuerier) SQLDatabaseStoragePercent(ctx context.Context, resourceURI string) (int, error) {
	if q.client == nil {
		return -1, fmt.Errorf("azure: monitor metrics client not initialised")
	}
	now := time.Now().UTC()
	resp, err := q.client.QueryResource(ctx, resourceURI, &azquery.MetricsClientQueryResourceOptions{
		MetricNames: to.Ptr("storage_percent"),
		Aggregation: []*azquery.AggregationType{to.Ptr(azquery.AggregationTypeAverage)},
		Timespan:    to.Ptr(azquery.NewTimeInterval(now.Add(-q.window), now)),
		Interval:    to.Ptr(fmt.Sprintf("PT%dM", int(q.window.Minutes()))),
	})
	if err != nil {
		return -1, fmt.Errorf("azure: monitor query storage_percent: %w", err)
	}
	return latestStoragePercent(resp.Response), nil
}

// latestStoragePercent extracts the most recent Average value from a
// metrics response and returns it as a 0..100 integer percent, or -1 when
// the response carries no usable point. Pure parsing function — unit-
// testable without a live SDK.
func latestStoragePercent(resp azquery.Response) int {
	for _, m := range resp.Value {
		if m == nil {
			continue
		}
		for _, ts := range m.TimeSeries {
			if ts == nil {
				continue
			}
			// Walk newest-first; Azure returns oldest-first, so iterate in
			// reverse to surface the freshest observation.
			for i := len(ts.Data) - 1; i >= 0; i-- {
				p := ts.Data[i]
				if p == nil || p.Average == nil {
					continue
				}
				v := *p.Average
				if v < 0 {
					continue // defensive: bogus point
				}
				pct := int(v + 0.5) // round to nearest percent (Azure already returns 0..100)
				if pct > 100 {
					pct = 100
				}
				return pct
			}
		}
	}
	return -1
}
