// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package azure

import (
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v6"
)

// subnetUsedIPCount returns the number of primary-range IPs in s that are
// already consumed, summed across every resource type that pulls an IP
// from the subnet:
//
//   - NIC IP configurations (VMs, AKS nodes, internal LBs)
//   - Application Gateway IP configurations
//   - IP-configuration profiles (Container Instances etc.)
//   - Private Endpoints (each consumes one IP)
//
// These fields are READ-ONLY on the Subnet resource and are populated by
// the apiserver automatically — no $expand needed.
//
// Available = (chart's usableIPsFromCIDR) - used. The chart already
// accounts for Azure's 5-IP reservation; subtracting `used` here yields
// the primary-range free count the IP-exhaustion probe expects.
func subnetUsedIPCount(s *armnetwork.Subnet) int {
	if s == nil || s.Properties == nil {
		return 0
	}
	p := s.Properties
	return len(p.IPConfigurations) +
		len(p.ApplicationGatewayIPConfigurations) +
		len(p.IPConfigurationProfiles) +
		len(p.PrivateEndpoints)
}
