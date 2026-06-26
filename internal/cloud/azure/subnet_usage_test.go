// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

package azure

import (
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v6"
)

func TestSubnetUsedIPCount(t *testing.T) {
	cases := []struct {
		name string
		in   *armnetwork.Subnet
		want int
	}{
		{"nil subnet", nil, 0},
		{"nil properties", &armnetwork.Subnet{}, 0},
		{
			"all four IP-consuming fields summed",
			&armnetwork.Subnet{Properties: &armnetwork.SubnetPropertiesFormat{
				IPConfigurations: []*armnetwork.IPConfiguration{
					{}, {}, {}, // 3 NIC configs
				},
				ApplicationGatewayIPConfigurations: []*armnetwork.ApplicationGatewayIPConfiguration{
					{}, // 1 AppGW config
				},
				IPConfigurationProfiles: []*armnetwork.IPConfigurationProfile{
					{}, {}, // 2 container groups
				},
				PrivateEndpoints: []*armnetwork.PrivateEndpoint{
					{}, // 1 private endpoint
				},
			}},
			7,
		},
		{
			"only NICs",
			&armnetwork.Subnet{Properties: &armnetwork.SubnetPropertiesFormat{
				IPConfigurations: []*armnetwork.IPConfiguration{{}, {}, {}, {}, {}},
			}},
			5,
		},
		{
			"empty subnet (no consumers)",
			&armnetwork.Subnet{Properties: &armnetwork.SubnetPropertiesFormat{}},
			0,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := subnetUsedIPCount(c.in); got != c.want {
				t.Errorf("subnetUsedIPCount = %d; want %d", got, c.want)
			}
		})
	}
}
