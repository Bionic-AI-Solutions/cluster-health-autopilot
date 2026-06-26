// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

package azure

import (
	"context"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v6"
)

// backendHealthClient abstracts the AppGW BackendHealth LRO so tests can
// stub the per-gateway result without spinning up the SDK poller. The
// production impl wraps ApplicationGatewaysClient.BeginBackendHealth +
// PollUntilDone.
type backendHealthClient interface {
	BackendHealth(ctx context.Context, resourceGroup, gatewayName string) (armnetwork.ApplicationGatewayBackendHealth, error)
}

// liveBackendHealthClient is the production impl, backed by the
// armnetwork.ApplicationGatewaysClient already owned by LiveClient.
//
// The LRO can take seconds-to-tens-of-seconds per gateway. We poll with the
// SDK default frequency and a fixed cap so a single misbehaving gateway
// can't stretch a probe cycle indefinitely.
type liveBackendHealthClient struct {
	gws     *armnetwork.ApplicationGatewaysClient
	pollCap time.Duration
}

func newLiveBackendHealthClient(gws *armnetwork.ApplicationGatewaysClient) *liveBackendHealthClient {
	return &liveBackendHealthClient{gws: gws, pollCap: 60 * time.Second}
}

func (c *liveBackendHealthClient) BackendHealth(ctx context.Context, rg, gw string) (armnetwork.ApplicationGatewayBackendHealth, error) {
	poller, err := c.gws.BeginBackendHealth(ctx, rg, gw, nil)
	if err != nil {
		return armnetwork.ApplicationGatewayBackendHealth{}, err
	}
	pollCtx, cancel := context.WithTimeout(ctx, c.pollCap)
	defer cancel()
	resp, err := poller.PollUntilDone(pollCtx, nil)
	if err != nil {
		return armnetwork.ApplicationGatewayBackendHealth{}, err
	}
	return resp.ApplicationGatewayBackendHealth, nil
}

// poolHealth is the per-pool aggregated result.
type poolHealth struct {
	Healthy int
	Total   int
}

// aggregateBackendHealth flattens an LRO response into a per-pool map.
//
//	resp.BackendAddressPools[i]
//	  .BackendHTTPSettingsCollection[j]
//	    .Servers[k] → { Address, Health }
//
// Each pool may serve multiple HTTP settings; servers can repeat across
// settings. We count UNIQUE backend addresses per pool, treating "Up" or
// "Partial" as healthy (Partial == some probes Up, some Down — the
// instance is in rotation, so it counts toward HealthyCount). Down /
// Draining / Unknown are not counted as healthy.
//
// Returns map keyed by pool name (the AppGW pool resource name parsed from
// BackendAddressPool.ID).
func aggregateBackendHealth(resp armnetwork.ApplicationGatewayBackendHealth) map[string]poolHealth {
	out := map[string]poolHealth{}
	for _, p := range resp.BackendAddressPools {
		if p == nil || p.BackendAddressPool == nil {
			continue
		}
		name := backendPoolName(p.BackendAddressPool)
		if name == "" {
			continue
		}
		seen := map[string]string{} // address → best-status-seen
		for _, s := range p.BackendHTTPSettingsCollection {
			if s == nil {
				continue
			}
			for _, srv := range s.Servers {
				if srv == nil || srv.Address == nil {
					continue
				}
				addr := *srv.Address
				h := ""
				if srv.Health != nil {
					h = string(*srv.Health)
				}
				// Prefer the strongest health observation across HTTP
				// settings: Up > Partial > Draining > Down > Unknown.
				if rankHealth(h) > rankHealth(seen[addr]) {
					seen[addr] = h
				}
			}
		}
		ph := poolHealth{Total: len(seen)}
		for _, h := range seen {
			if isHealthy(h) {
				ph.Healthy++
			}
		}
		out[name] = ph
	}
	return out
}

// backendPoolName extracts the pool name from a BackendAddressPool, which
// may carry only an ID (e.g. .../backendAddressPools/<name>) when the
// LRO response is constructed by reference rather than by name.
func backendPoolName(p *armnetwork.ApplicationGatewayBackendAddressPool) string {
	if p == nil {
		return ""
	}
	if p.Name != nil && *p.Name != "" {
		return *p.Name
	}
	if p.ID == nil {
		return ""
	}
	id := *p.ID
	// Last "/<segment>" of the resource ID.
	for i := len(id) - 1; i >= 0; i-- {
		if id[i] == '/' {
			return id[i+1:]
		}
	}
	return id
}

// isHealthy returns whether a Health value should count toward HealthyCount.
// "Up" is unambiguously healthy; "Partial" means some probes succeeded and
// the instance is still in rotation, so it counts. Everything else
// (Down/Draining/Unknown/empty) does not.
func isHealthy(h string) bool {
	return h == string(armnetwork.ApplicationGatewayBackendHealthServerHealthUp) ||
		h == string(armnetwork.ApplicationGatewayBackendHealthServerHealthPartial)
}

// rankHealth orders Health values for tie-breaking when the same backend
// address appears under multiple HTTP settings.
func rankHealth(h string) int {
	switch h {
	case string(armnetwork.ApplicationGatewayBackendHealthServerHealthUp):
		return 5
	case string(armnetwork.ApplicationGatewayBackendHealthServerHealthPartial):
		return 4
	case string(armnetwork.ApplicationGatewayBackendHealthServerHealthDraining):
		return 3
	case string(armnetwork.ApplicationGatewayBackendHealthServerHealthDown):
		return 2
	case string(armnetwork.ApplicationGatewayBackendHealthServerHealthUnknown):
		return 1
	default:
		return 0
	}
}
