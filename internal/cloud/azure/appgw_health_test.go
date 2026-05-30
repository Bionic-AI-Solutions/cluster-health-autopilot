// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package azure

import (
	"context"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v6"
)

func srv(addr, health string) *armnetwork.ApplicationGatewayBackendHealthServer {
	s := &armnetwork.ApplicationGatewayBackendHealthServer{Address: to.Ptr(addr)}
	if health != "" {
		h := armnetwork.ApplicationGatewayBackendHealthServerHealth(health)
		s.Health = &h
	}
	return s
}

func pool(name string, settings ...[]*armnetwork.ApplicationGatewayBackendHealthServer) *armnetwork.ApplicationGatewayBackendHealthPool {
	p := &armnetwork.ApplicationGatewayBackendHealthPool{
		BackendAddressPool: &armnetwork.ApplicationGatewayBackendAddressPool{Name: to.Ptr(name)},
	}
	for _, s := range settings {
		p.BackendHTTPSettingsCollection = append(p.BackendHTTPSettingsCollection,
			&armnetwork.ApplicationGatewayBackendHealthHTTPSettings{Servers: s})
	}
	return p
}

func TestAggregateBackendHealth_BasicHealthCount(t *testing.T) {
	resp := armnetwork.ApplicationGatewayBackendHealth{
		BackendAddressPools: []*armnetwork.ApplicationGatewayBackendHealthPool{
			pool("api-pool",
				[]*armnetwork.ApplicationGatewayBackendHealthServer{
					srv("10.0.0.1", "Up"),
					srv("10.0.0.2", "Up"),
					srv("10.0.0.3", "Down"),
					srv("10.0.0.4", "Unknown"),
				},
			),
		},
	}
	got := aggregateBackendHealth(resp)
	ph, ok := got["api-pool"]
	if !ok {
		t.Fatalf("api-pool not in aggregated result: %+v", got)
	}
	if ph.Total != 4 || ph.Healthy != 2 {
		t.Errorf("api-pool: got total=%d healthy=%d; want 4/2", ph.Total, ph.Healthy)
	}
}

func TestAggregateBackendHealth_PartialCountsAsHealthy(t *testing.T) {
	// Partial means some probes Up; the instance is in rotation.
	resp := armnetwork.ApplicationGatewayBackendHealth{
		BackendAddressPools: []*armnetwork.ApplicationGatewayBackendHealthPool{
			pool("p",
				[]*armnetwork.ApplicationGatewayBackendHealthServer{
					srv("10.0.0.1", "Partial"),
					srv("10.0.0.2", "Draining"),
				},
			),
		},
	}
	ph := aggregateBackendHealth(resp)["p"]
	if ph.Healthy != 1 {
		t.Errorf("partial should count toward healthy; healthy=%d (want 1)", ph.Healthy)
	}
}

func TestAggregateBackendHealth_DedupSameAddressAcrossSettings(t *testing.T) {
	// The same backend address appears under two HTTP settings — should
	// count once, and the strongest health wins.
	resp := armnetwork.ApplicationGatewayBackendHealth{
		BackendAddressPools: []*armnetwork.ApplicationGatewayBackendHealthPool{
			pool("p",
				[]*armnetwork.ApplicationGatewayBackendHealthServer{srv("10.0.0.1", "Down")},
				[]*armnetwork.ApplicationGatewayBackendHealthServer{srv("10.0.0.1", "Up")},
			),
		},
	}
	ph := aggregateBackendHealth(resp)["p"]
	if ph.Total != 1 {
		t.Errorf("dup address should count once; total=%d (want 1)", ph.Total)
	}
	if ph.Healthy != 1 {
		t.Errorf("Up should win over Down (strongest health); healthy=%d (want 1)", ph.Healthy)
	}
}

func TestAggregateBackendHealth_NilPoolsSkipped(t *testing.T) {
	resp := armnetwork.ApplicationGatewayBackendHealth{
		BackendAddressPools: []*armnetwork.ApplicationGatewayBackendHealthPool{
			nil,
			{BackendAddressPool: nil},
			pool("p", []*armnetwork.ApplicationGatewayBackendHealthServer{srv("10.0.0.1", "Up")}),
		},
	}
	got := aggregateBackendHealth(resp)
	if len(got) != 1 {
		t.Errorf("nil-pool entries should be skipped; got %d entries: %+v", len(got), got)
	}
}

func TestBackendPoolName_ParsesID(t *testing.T) {
	cases := []struct {
		in   *armnetwork.ApplicationGatewayBackendAddressPool
		want string
	}{
		{nil, ""},
		{&armnetwork.ApplicationGatewayBackendAddressPool{Name: to.Ptr("by-name")}, "by-name"},
		{&armnetwork.ApplicationGatewayBackendAddressPool{
			ID: to.Ptr("/subscriptions/s/resourceGroups/r/providers/Microsoft.Network/applicationGateways/gw/backendAddressPools/from-id"),
		}, "from-id"},
		{&armnetwork.ApplicationGatewayBackendAddressPool{ID: to.Ptr("noslash")}, "noslash"},
	}
	for _, c := range cases {
		if got := backendPoolName(c.in); got != c.want {
			t.Errorf("backendPoolName(%+v) = %q; want %q", c.in, got, c.want)
		}
	}
}

func TestIsHealthy(t *testing.T) {
	for _, c := range []struct {
		h    string
		want bool
	}{
		{"Up", true}, {"Partial", true},
		{"Down", false}, {"Draining", false}, {"Unknown", false}, {"", false},
	} {
		if got := isHealthy(c.h); got != c.want {
			t.Errorf("isHealthy(%q) = %v; want %v", c.h, got, c.want)
		}
	}
}

func TestRankHealth_OrdersStrongestFirst(t *testing.T) {
	ordered := []string{"Up", "Partial", "Draining", "Down", "Unknown", ""}
	for i := 0; i < len(ordered)-1; i++ {
		if rankHealth(ordered[i]) <= rankHealth(ordered[i+1]) {
			t.Errorf("rankHealth(%q)=%d should be > rankHealth(%q)=%d",
				ordered[i], rankHealth(ordered[i]), ordered[i+1], rankHealth(ordered[i+1]))
		}
	}
}

// fakeBackendHealthClient stubs the LRO so the wiring path in
// ListAppGatewayBackends can be tested without a live Azure subscription.
// (The unit tests above cover aggregation; this just proves the interface
// contract.)
type fakeBackendHealthClient struct{}

func (fakeBackendHealthClient) BackendHealth(_ context.Context, _, _ string) (armnetwork.ApplicationGatewayBackendHealth, error) {
	return armnetwork.ApplicationGatewayBackendHealth{}, nil
}

func TestBackendHealthClientInterfaceContract(t *testing.T) {
	var _ backendHealthClient = fakeBackendHealthClient{}
}
