// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

package azure

import (
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v6"
)

// appGatewayFrontendHostname feeds the Srenix Enterprise "(lb: ...)" join key —
// the AppGW's public hostname out of the already-fetched listener
// config (no extra API call).

func strPtr(s string) *string { return &s }

func TestAppGWFrontendHostname_PrefersListenerHostName(t *testing.T) {
	gw := &armnetwork.ApplicationGateway{
		Properties: &armnetwork.ApplicationGatewayPropertiesFormat{
			HTTPListeners: []*armnetwork.ApplicationGatewayHTTPListener{
				{Properties: &armnetwork.ApplicationGatewayHTTPListenerPropertiesFormat{HostName: strPtr("www.example.com")}},
			},
		},
	}
	if got := appGatewayFrontendHostname(gw); got != "www.example.com" {
		t.Errorf("got %q want www.example.com", got)
	}
}

func TestAppGWFrontendHostname_FallsBackToHostNamesList(t *testing.T) {
	gw := &armnetwork.ApplicationGateway{
		Properties: &armnetwork.ApplicationGatewayPropertiesFormat{
			HTTPListeners: []*armnetwork.ApplicationGatewayHTTPListener{
				{Properties: &armnetwork.ApplicationGatewayHTTPListenerPropertiesFormat{
					HostNames: []*string{nil, strPtr(""), strPtr("api.example.com")},
				}},
			},
		},
	}
	if got := appGatewayFrontendHostname(gw); got != "api.example.com" {
		t.Errorf("got %q want api.example.com", got)
	}
}

func TestAppGWFrontendHostname_EmptyWhenNoListenerHostnames(t *testing.T) {
	for _, gw := range []*armnetwork.ApplicationGateway{
		nil,
		{},
		{Properties: &armnetwork.ApplicationGatewayPropertiesFormat{}},
		{Properties: &armnetwork.ApplicationGatewayPropertiesFormat{
			HTTPListeners: []*armnetwork.ApplicationGatewayHTTPListener{nil, {Properties: &armnetwork.ApplicationGatewayHTTPListenerPropertiesFormat{}}},
		}},
	} {
		if got := appGatewayFrontendHostname(gw); got != "" {
			t.Errorf("got %q want \"\" for %+v", got, gw)
		}
	}
}

// derefNonEmpty feeds the cert "(domains: ...)" join key from the App
// Service certificate's HostNames (SAN list).

func TestDerefNonEmpty(t *testing.T) {
	got := derefNonEmpty([]*string{strPtr("a.com"), nil, strPtr(""), strPtr("b.com")})
	if len(got) != 2 || got[0] != "a.com" || got[1] != "b.com" {
		t.Errorf("got %v want [a.com b.com]", got)
	}
	if got := derefNonEmpty(nil); got != nil {
		t.Errorf("got %v want nil", got)
	}
}
