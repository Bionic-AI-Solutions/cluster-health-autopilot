// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

package azure

import (
	"context"
	"errors"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/monitor/azquery"
)

func TestLatestStoragePercent(t *testing.T) {
	cases := []struct {
		name string
		resp azquery.Response
		want int
	}{
		{
			"empty response → -1",
			azquery.Response{},
			-1,
		},
		{
			"nil metric → -1",
			azquery.Response{Value: []*azquery.Metric{nil}},
			-1,
		},
		{
			"one series, newest point is last in Data — returns it",
			azquery.Response{Value: []*azquery.Metric{{
				TimeSeries: []*azquery.TimeSeriesElement{{
					Data: []*azquery.MetricValue{
						{Average: to.Ptr(40.0)},
						{Average: to.Ptr(45.0)},
						{Average: to.Ptr(83.0)}, // newest (oldest-first SDK convention)
					},
				}},
			}}},
			83,
		},
		{
			"skips trailing points without Average",
			azquery.Response{Value: []*azquery.Metric{{
				TimeSeries: []*azquery.TimeSeriesElement{{
					Data: []*azquery.MetricValue{
						{Average: to.Ptr(42.0)},
						{}, // missing Average
					},
				}},
			}}},
			42,
		},
		{
			"rounds to nearest percent",
			azquery.Response{Value: []*azquery.Metric{{
				TimeSeries: []*azquery.TimeSeriesElement{{
					Data: []*azquery.MetricValue{{Average: to.Ptr(86.5)}},
				}},
			}}},
			87,
		},
		{
			"caps at 100",
			azquery.Response{Value: []*azquery.Metric{{
				TimeSeries: []*azquery.TimeSeriesElement{{
					Data: []*azquery.MetricValue{{Average: to.Ptr(107.0)}},
				}},
			}}},
			100,
		},
		{
			"skips negative (bogus) — falls back to next",
			azquery.Response{Value: []*azquery.Metric{{
				TimeSeries: []*azquery.TimeSeriesElement{{
					Data: []*azquery.MetricValue{
						{Average: to.Ptr(10.0)},
						{Average: to.Ptr(-5.0)},
					},
				}},
			}}},
			10,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := latestStoragePercent(c.resp); got != c.want {
				t.Errorf("latestStoragePercent = %d; want %d", got, c.want)
			}
		})
	}
}

// fakeMetricsClient is a programmable metricsClient for testing the
// querier's request + parsing path without a live Azure subscription.
type fakeMetricsClient struct {
	gotURI string
	resp   azquery.Response
	err    error
}

func (f *fakeMetricsClient) QueryResource(_ context.Context, uri string, _ *azquery.MetricsClientQueryResourceOptions) (azquery.MetricsClientQueryResourceResponse, error) {
	f.gotURI = uri
	return azquery.MetricsClientQueryResourceResponse{Response: f.resp}, f.err
}

func TestAzureMonitoringQuerier_HappyPath(t *testing.T) {
	mc := &fakeMetricsClient{
		resp: azquery.Response{Value: []*azquery.Metric{{
			TimeSeries: []*azquery.TimeSeriesElement{{
				Data: []*azquery.MetricValue{{Average: to.Ptr(72.0)}},
			}},
		}}},
	}
	q := newAzureMonitoringQuerier(mc)
	pct, err := q.SQLDatabaseStoragePercent(context.Background(), "/subscriptions/s/resourceGroups/r/providers/Microsoft.Sql/servers/srv/databases/db")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pct != 72 {
		t.Errorf("pct = %d; want 72", pct)
	}
	if mc.gotURI == "" {
		t.Error("querier should pass through the resource URI")
	}
}

func TestAzureMonitoringQuerier_ErrorReturnsMinusOne(t *testing.T) {
	mc := &fakeMetricsClient{err: errors.New("monitor down")}
	q := newAzureMonitoringQuerier(mc)
	pct, err := q.SQLDatabaseStoragePercent(context.Background(), "/subscriptions/s/.../db")
	if err == nil {
		t.Error("expected error to propagate")
	}
	if pct != -1 {
		t.Errorf("pct on error = %d; want -1", pct)
	}
}

func TestAzureMonitoringQuerier_NilClientErrors(t *testing.T) {
	q := &azureMonitoringQuerier{client: nil}
	_, err := q.SQLDatabaseStoragePercent(context.Background(), "x")
	if err == nil {
		t.Error("nil metrics client should error")
	}
}

func TestAzureMetricsClientInterfaceContract(t *testing.T) {
	// Compile-time guarantee: the fake satisfies the metricsClient interface
	// the querier depends on.
	var _ metricsClient = (*fakeMetricsClient)(nil)
}
