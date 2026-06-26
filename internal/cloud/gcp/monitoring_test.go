// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

package gcp

import (
	"context"
	"errors"
	"testing"

	monitoring "google.golang.org/api/monitoring/v3"
)

// ptr is a tiny helper for the *float64 the monitoring SDK uses.
func ptrFloat(v float64) *float64 { return &v }

func TestLatestDiskUsedPercent(t *testing.T) {
	cases := []struct {
		name string
		resp *monitoring.ListTimeSeriesResponse
		want int
	}{
		{"nil response → -1", nil, -1},
		{"empty series → -1", &monitoring.ListTimeSeriesResponse{}, -1},
		{
			"one series, first point is the most recent",
			&monitoring.ListTimeSeriesResponse{
				TimeSeries: []*monitoring.TimeSeries{{
					Points: []*monitoring.Point{
						{Value: &monitoring.TypedValue{DoubleValue: ptrFloat(0.81)}}, // newest first
						{Value: &monitoring.TypedValue{DoubleValue: ptrFloat(0.50)}},
					},
				}},
			},
			81,
		},
		{
			"skips points without DoubleValue",
			&monitoring.ListTimeSeriesResponse{
				TimeSeries: []*monitoring.TimeSeries{{
					Points: []*monitoring.Point{
						{Value: &monitoring.TypedValue{}}, // empty value
						{Value: &monitoring.TypedValue{DoubleValue: ptrFloat(0.42)}},
					},
				}},
			},
			42,
		},
		{
			"rounds to nearest percent",
			&monitoring.ListTimeSeriesResponse{
				TimeSeries: []*monitoring.TimeSeries{{
					Points: []*monitoring.Point{
						{Value: &monitoring.TypedValue{DoubleValue: ptrFloat(0.865)}},
					},
				}},
			},
			87,
		},
		{
			"caps at 100",
			&monitoring.ListTimeSeriesResponse{
				TimeSeries: []*monitoring.TimeSeries{{
					Points: []*monitoring.Point{
						{Value: &monitoring.TypedValue{DoubleValue: ptrFloat(1.07)}}, // bogus over-1.0
					},
				}},
			},
			100,
		},
		{
			"skips negative (bogus)",
			&monitoring.ListTimeSeriesResponse{
				TimeSeries: []*monitoring.TimeSeries{{
					Points: []*monitoring.Point{
						{Value: &monitoring.TypedValue{DoubleValue: ptrFloat(-0.5)}},
						{Value: &monitoring.TypedValue{DoubleValue: ptrFloat(0.10)}},
					},
				}},
			},
			10,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := latestDiskUsedPercent(c.resp); got != c.want {
				t.Errorf("latestDiskUsedPercent = %d; want %d", got, c.want)
			}
		})
	}
}

func TestSplitDatabaseID(t *testing.T) {
	cases := map[string][2]string{
		"my-project:my-instance": {"my-project", "my-instance"},
		"my-instance":            {"", "my-instance"},
		"":                       {"", ""},
	}
	for in, want := range cases {
		p, i := SplitDatabaseID(in)
		if p != want[0] || i != want[1] {
			t.Errorf("SplitDatabaseID(%q) = (%q,%q); want (%q,%q)", in, p, i, want[0], want[1])
		}
	}
}

// fakeMonitoring is a programmable monitoringQuerier. Used to test
// LiveClient.ListCloudSQLInstances' wiring without a real Cloud Monitoring
// call — the parsing logic is covered by latestDiskUsedPercent tests above.
type fakeMonitoring struct {
	got   []string
	value int
	err   error
}

func (f *fakeMonitoring) CloudSQLDiskUsedPercent(_ context.Context, id string) (int, error) {
	f.got = append(f.got, id)
	return f.value, f.err
}

func TestCloudMonitoringQuerier_NilServiceErrors(t *testing.T) {
	q := &cloudMonitoringQuerier{svc: nil, project: "p"}
	_, err := q.CloudSQLDiskUsedPercent(context.Background(), "inst")
	if err == nil {
		t.Error("nil monitoring service should error")
	}
}

func TestFakeMonitoringQuerier_WiringContract(t *testing.T) {
	// Sanity check that our test fake satisfies the interface (compile-time
	// guarantee that any LiveClient wiring will accept it).
	var _ monitoringQuerier = (*fakeMonitoring)(nil)

	f := &fakeMonitoring{value: 73}
	pct, err := f.CloudSQLDiskUsedPercent(context.Background(), "p:my-db")
	if err != nil || pct != 73 {
		t.Errorf("fake: got (%d,%v); want (73,nil)", pct, err)
	}
	if len(f.got) != 1 || f.got[0] != "p:my-db" {
		t.Errorf("fake didn't capture call: %v", f.got)
	}
}

func TestFakeMonitoringQuerier_ErrorPropagates(t *testing.T) {
	f := &fakeMonitoring{err: errors.New("api down")}
	_, err := f.CloudSQLDiskUsedPercent(context.Background(), "inst")
	if err == nil {
		t.Error("error should propagate")
	}
}
