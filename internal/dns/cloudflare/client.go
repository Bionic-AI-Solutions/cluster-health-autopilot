// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

// Package cloudflare provides a thin HTTP client for the Cloudflare v4 API.
// Only the zone-list and DNS-record-list endpoints are implemented; they are
// sufficient for the DNSChainDrift analyzer.
package cloudflare

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const (
	defaultBaseURL = "https://api.cloudflare.com/client/v4"
	defaultTimeout = 5 * time.Second
	perPage        = 100
)

// Zone is a Cloudflare DNS zone.
type Zone struct {
	ID   string
	Name string
}

// DNSRecord is a single DNS record within a Cloudflare zone.
type DNSRecord struct {
	Name    string
	Type    string
	Content string
	Proxied bool
}

// Client is the Cloudflare API surface used by this package.
type Client interface {
	ListZones(ctx context.Context) ([]Zone, error)
	ListDNSRecords(ctx context.Context, zoneID string) ([]DNSRecord, error)
}

type client struct {
	token   string
	baseURL string
	http    *http.Client
}

// New returns a Client authenticated with token. When baseURL is empty the
// production Cloudflare API endpoint is used.
func New(token, baseURL string) Client {
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	return &client{
		token:   token,
		baseURL: baseURL,
		http:    &http.Client{Timeout: defaultTimeout},
	}
}

type cfResponse[T any] struct {
	Result     []T        `json:"result"`
	ResultInfo resultInfo `json:"result_info"`
	Success    bool       `json:"success"`
	Errors     []cfError  `json:"errors"`
}

type resultInfo struct {
	Page       int `json:"page"`
	PerPage    int `json:"per_page"`
	TotalPages int `json:"total_pages"`
	Count      int `json:"count"`
	Total      int `json:"total_count"`
}

type cfError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type cfZone struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type cfDNSRecord struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	Content string `json:"content"`
	Proxied bool   `json:"proxied"`
}

func (c *client) do(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")
	return c.http.Do(req)
}

func (c *client) ListZones(ctx context.Context) ([]Zone, error) {
	var out []Zone
	for page := 1; ; page++ {
		url := fmt.Sprintf("%s/zones?page=%d&per_page=%d", c.baseURL, page, perPage)
		resp, err := c.do(ctx, url)
		if err != nil {
			return nil, err
		}
		var parsed cfResponse[cfZone]
		if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
			_ = resp.Body.Close()
			return nil, err
		}
		_ = resp.Body.Close()
		if !parsed.Success {
			if len(parsed.Errors) > 0 {
				return nil, fmt.Errorf("cloudflare API error %d: %s", parsed.Errors[0].Code, parsed.Errors[0].Message)
			}
			return nil, fmt.Errorf("cloudflare ListZones: unknown API error")
		}
		for _, z := range parsed.Result {
			out = append(out, Zone(z))
		}
		if page >= parsed.ResultInfo.TotalPages || len(parsed.Result) == 0 {
			break
		}
	}
	return out, nil
}

func (c *client) ListDNSRecords(ctx context.Context, zoneID string) ([]DNSRecord, error) {
	var out []DNSRecord
	for page := 1; ; page++ {
		url := fmt.Sprintf("%s/zones/%s/dns_records?page=%d&per_page=%d", c.baseURL, zoneID, page, perPage)
		resp, err := c.do(ctx, url)
		if err != nil {
			return nil, err
		}
		var parsed cfResponse[cfDNSRecord]
		if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
			_ = resp.Body.Close()
			return nil, err
		}
		_ = resp.Body.Close()
		if !parsed.Success {
			if len(parsed.Errors) > 0 {
				return nil, fmt.Errorf("cloudflare API error %d: %s", parsed.Errors[0].Code, parsed.Errors[0].Message)
			}
			return nil, fmt.Errorf("cloudflare ListDNSRecords: unknown API error")
		}
		for _, r := range parsed.Result {
			out = append(out, DNSRecord(r))
		}
		if page >= parsed.ResultInfo.TotalPages || len(parsed.Result) == 0 {
			break
		}
	}
	return out, nil
}
