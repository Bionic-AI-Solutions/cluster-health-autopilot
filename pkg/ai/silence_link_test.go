// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package ai

import (
	"net/url"
	"testing"
	"time"
)

func TestMintSilenceLinks_WellFormed(t *testing.T) {
	pub, priv, _ := GenerateSigningKey()
	now := time.Now()
	req := SilenceLinkRequest{
		Source:   "StaleErrorPods",
		Subject:  "Pod/default/legacy",
		ShortDur: 24 * time.Hour,
		LongDur:  2160 * time.Hour, // 90d
	}
	links, err := MintSilenceLinks(priv, "kid-1", "https://cha-approve.example.com", req, now)
	if err != nil {
		t.Fatal(err)
	}

	// Both URLs point at /silence with a token query param.
	for name, raw := range map[string]string{"subject": links.SubjectURL, "class": links.ClassURL} {
		u, perr := url.Parse(raw)
		if perr != nil {
			t.Fatalf("%s URL unparseable: %v", name, perr)
		}
		if u.Path != "/silence" {
			t.Errorf("%s URL path = %q; want /silence", name, u.Path)
		}
		if u.Query().Get("token") == "" {
			t.Errorf("%s URL missing token", name)
		}
	}

	// Subject token: scope=subject, carries Subject, until = now+24h.
	su, _ := url.Parse(links.SubjectURL)
	subjClaims, err := VerifySilenceToken(pub, su.Query().Get("token"))
	if err != nil {
		t.Fatal(err)
	}
	if subjClaims.Scope != SilenceScopeSubject {
		t.Errorf("subject scope = %q", subjClaims.Scope)
	}
	if subjClaims.Subject != req.Subject {
		t.Errorf("subject = %q", subjClaims.Subject)
	}
	if subjClaims.Source != req.Source {
		t.Errorf("subject source = %q", subjClaims.Source)
	}
	if got, want := subjClaims.UntilUnix, now.Add(req.ShortDur).Unix(); got != want {
		t.Errorf("subject until = %d want %d", got, want)
	}

	// Class token: scope=class, NO subject, until = now+90d.
	cu, _ := url.Parse(links.ClassURL)
	classClaims, err := VerifySilenceToken(pub, cu.Query().Get("token"))
	if err != nil {
		t.Fatal(err)
	}
	if classClaims.Scope != SilenceScopeClass {
		t.Errorf("class scope = %q", classClaims.Scope)
	}
	if classClaims.Subject != "" {
		t.Errorf("class subject should be empty, got %q", classClaims.Subject)
	}
	if classClaims.Source != req.Source {
		t.Errorf("class source = %q", classClaims.Source)
	}
	if got, want := classClaims.UntilUnix, now.Add(req.LongDur).Unix(); got != want {
		t.Errorf("class until = %d want %d", got, want)
	}

	// JTIs must differ (one link == one mint).
	if subjClaims.JTI == classClaims.JTI {
		t.Errorf("subject and class JTIs must differ; both = %q", subjClaims.JTI)
	}
}

func TestMintSilenceLinks_Errors(t *testing.T) {
	_, priv, _ := GenerateSigningKey()
	now := time.Now()
	if _, err := MintSilenceLinks(priv, "k", "https://x", SilenceLinkRequest{Source: ""}, now); err == nil {
		t.Error("expected error for empty Source")
	}
	if _, err := MintSilenceLinks(priv, "k", "", SilenceLinkRequest{Source: "S"}, now); err == nil {
		t.Error("expected error for empty baseURL")
	}
}

func TestMintSilenceLinks_MessagePatternPropagates(t *testing.T) {
	pub, priv, _ := GenerateSigningKey()
	now := time.Now()
	req := SilenceLinkRequest{
		Source:         "SecurityDrift",
		Subject:        "Deployment/ns/app",
		MessagePattern: "without digest pin",
		ShortDur:       time.Hour,
		LongDur:        2 * time.Hour,
	}
	links, err := MintSilenceLinks(priv, "kid-1", "https://x.example", req, now)
	if err != nil {
		t.Fatal(err)
	}
	su, _ := url.Parse(links.SubjectURL)
	c, _ := VerifySilenceToken(pub, su.Query().Get("token"))
	if c.MessagePattern != "without digest pin" {
		t.Errorf("messagePattern not propagated: got %q", c.MessagePattern)
	}
}
