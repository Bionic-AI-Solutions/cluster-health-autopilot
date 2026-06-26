// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

package diagnose

import (
	"context"
	"strings"
	"testing"
)

const esoFailingWithEvent = `{
  "apiVersion": "external-secrets.io/v1",
  "kind": "ExternalSecret",
  "metadata": {"name": "mail-service-api-key", "namespace": "mail"},
  "spec": {"target": {"name": "mail-service-api-key"}},
  "status": {
    "conditions": [{
      "type": "Ready",
      "status": "False",
      "message": "could not get secret data from provider"
    }]
  }
}`

const eventUpdateFailed = `{
  "apiVersion": "v1",
  "kind": "Event",
  "metadata": {"name": "mail-service-api-key.evt", "namespace": "mail"},
  "involvedObject": {"name": "mail-service-api-key", "kind": "ExternalSecret"},
  "reason": "UpdateFailed",
  "message": "error processing spec.data[0] (key: t6-apps/mail/config), err: cannot find secret data for key: \"mail_service_api_key\"",
  "lastTimestamp": "2026-04-29T18:30:00Z"
}`

const eventOlderUpdateFailed = `{
  "apiVersion": "v1",
  "kind": "Event",
  "metadata": {"name": "mail-service-api-key.evt-old", "namespace": "mail"},
  "involvedObject": {"name": "mail-service-api-key", "kind": "ExternalSecret"},
  "reason": "UpdateFailed",
  "message": "stale older error from yesterday",
  "lastTimestamp": "2026-04-28T18:30:00Z"
}`

const esoHealthy = `{
  "apiVersion": "external-secrets.io/v1",
  "kind": "ExternalSecret",
  "metadata": {"name": "mail-service-config", "namespace": "mail"},
  "status": {
    "conditions": [{"type": "Ready", "status": "True", "message": "secret synced"}]
  }
}`

func TestFailingExternalSecrets_PrefersEventOverConditionMessage(t *testing.T) {
	src := loadSrc(t, map[string]string{
		"eso.json":       esoFailingWithEvent,
		"event.json":     eventUpdateFailed,
		"event-old.json": eventOlderUpdateFailed,
	})
	got := FailingExternalSecrets{}.Run(context.Background(), src)
	if len(got) != 1 {
		t.Fatalf("want 1 diagnostic, got %d", len(got))
	}
	d := got[0]
	if !strings.Contains(d.Message, `cannot find secret data for key: "mail_service_api_key"`) {
		t.Errorf("expected event message preferred, got: %s", d.Message)
	}
	if strings.Contains(d.Message, "stale older error") {
		t.Errorf("older event should not have been chosen: %s", d.Message)
	}
	if d.Subject != "ExternalSecret/mail/mail-service-api-key" {
		t.Errorf("subject = %q", d.Subject)
	}
}

func TestFailingExternalSecrets_FallsBackToConditionMessage(t *testing.T) {
	// No events in the snapshot — should use the condition's own message.
	src := loadSrc(t, map[string]string{"eso.json": esoFailingWithEvent})
	got := FailingExternalSecrets{}.Run(context.Background(), src)
	if len(got) != 1 {
		t.Fatalf("want 1 diagnostic, got %d", len(got))
	}
	if !strings.Contains(got[0].Message, "could not get secret data from provider") {
		t.Errorf("expected condition-message fallback, got: %s", got[0].Message)
	}
}

func TestFailingExternalSecrets_SkipsHealthy(t *testing.T) {
	src := loadSrc(t, map[string]string{"eso.json": esoHealthy})
	got := FailingExternalSecrets{}.Run(context.Background(), src)
	if len(got) != 0 {
		t.Errorf("want 0 diagnostics for Ready=True ESO, got %d", len(got))
	}
}

func TestFailingExternalSecrets_NoCRDInstalled(t *testing.T) {
	// Empty snapshot — no ExternalSecret kind at all. Should be quiet, not error.
	src := loadSrc(t, map[string]string{})
	got := FailingExternalSecrets{}.Run(context.Background(), src)
	if len(got) != 0 {
		t.Errorf("want 0 diagnostics on empty snapshot, got %d", len(got))
	}
}

func TestFailingExternalSecrets_NonStandardVaultPath(t *testing.T) {
	// ESO references a non-standard flat path — diagnostic should fire with
	// the path in the message so the operator knows where to look.
	esoNonStd := `{
  "apiVersion": "external-secrets.io/v1",
  "kind": "ExternalSecret",
  "metadata": {"name": "counsellor-secrets", "namespace": "livekit-agents"},
  "spec": {
    "target": {"name": "counsellor-secrets"},
    "data": [{"remoteRef": {"key": "counsellor/config", "property": "livekit_url"}}]
  },
  "status": {
    "conditions": [{"type": "Ready", "status": "False", "message": "vault path not found"}]
  }
}`
	src := loadSrc(t, map[string]string{"eso.json": esoNonStd})
	got := FailingExternalSecrets{}.Run(context.Background(), src)
	if len(got) != 1 {
		t.Fatalf("want 1 diagnostic, got %d", len(got))
	}
	d := got[0]
	if !strings.Contains(d.Message, "not Ready") {
		t.Errorf("expected 'not Ready' in message: %s", d.Message)
	}
}

func TestFailingExternalSecrets_TruncatesLongMessages(t *testing.T) {
	long := strings.Repeat("x", 500)
	esoLong := `{
  "apiVersion": "external-secrets.io/v1",
  "kind": "ExternalSecret",
  "metadata": {"name": "longerr", "namespace": "demo"},
  "status": {
    "conditions": [{"type": "Ready", "status": "False", "message": "` + long + `"}]
  }
}`
	src := loadSrc(t, map[string]string{"eso.json": esoLong})
	got := FailingExternalSecrets{}.Run(context.Background(), src)
	if len(got) != 1 {
		t.Fatalf("want 1 diagnostic, got %d", len(got))
	}
	// Full message includes the prefix "ExternalSecret `demo/longerr` not Ready: " (≈42 bytes)
	// plus the truncated body (capped at 200) plus the trailing ". Check ..." (~37 bytes).
	if len(got[0].Message) > 300 {
		t.Errorf("diagnostic message length %d exceeds reasonable cap", len(got[0].Message))
	}
}
