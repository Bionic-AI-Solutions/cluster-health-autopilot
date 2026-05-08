// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package diagnose

import (
	"context"
	"strings"
	"testing"
)

const deployRefsValidKey = `{
  "apiVersion": "apps/v1",
  "kind": "Deployment",
  "metadata": {"name": "frontend", "namespace": "demo"},
  "spec": {
    "template": {
      "spec": {
        "containers": [{
          "name": "web",
          "env": [{
            "name": "API_KEY",
            "valueFrom": {"secretKeyRef": {"name": "frontend-secrets", "key": "api_key"}}
          }]
        }]
      }
    }
  }
}`

const deployRefsMissingKey = `{
  "apiVersion": "apps/v1",
  "kind": "Deployment",
  "metadata": {"name": "billing", "namespace": "billing"},
  "spec": {
    "template": {
      "spec": {
        "containers": [{
          "name": "api",
          "env": [{
            "name": "STRIPE_API_KEY",
            "valueFrom": {"secretKeyRef": {"name": "billing-secrets", "key": "stripe_api_key"}}
          }]
        }]
      }
    }
  }
}`

const deployRefsMissingSecret = `{
  "apiVersion": "apps/v1",
  "kind": "Deployment",
  "metadata": {"name": "loner", "namespace": "demo"},
  "spec": {
    "template": {
      "spec": {
        "containers": [{
          "name": "x",
          "env": [{
            "name": "PASS",
            "valueFrom": {"secretKeyRef": {"name": "ghost-secret", "key": "any"}}
          }]
        }]
      }
    }
  }
}`

const deployRefsOptional = `{
  "apiVersion": "apps/v1",
  "kind": "Deployment",
  "metadata": {"name": "tolerant", "namespace": "demo"},
  "spec": {
    "template": {
      "spec": {
        "containers": [{
          "name": "x",
          "env": [{
            "name": "MAYBE",
            "valueFrom": {"secretKeyRef": {"name": "ghost-secret", "key": "any", "optional": true}}
          }]
        }]
      }
    }
  }
}`

const ssRefsMissingKey = `{
  "apiVersion": "apps/v1",
  "kind": "StatefulSet",
  "metadata": {"name": "stateful-thing", "namespace": "data"},
  "spec": {
    "template": {
      "spec": {
        "initContainers": [{
          "name": "migrate",
          "env": [{
            "name": "DB_PASS",
            "valueFrom": {"secretKeyRef": {"name": "db-secrets", "key": "postgres_password_v2"}}
          }]
        }]
      }
    }
  }
}`

const secretFrontend = `{
  "apiVersion": "v1",
  "kind": "Secret",
  "metadata": {"name": "frontend-secrets", "namespace": "demo"},
  "data": {"api_key": "...", "csrf_token": "..."}
}`

const secretBillingMissingStripe = `{
  "apiVersion": "v1",
  "kind": "Secret",
  "metadata": {"name": "billing-secrets", "namespace": "billing"},
  "data": {"OPENPROJECT_API_KEY": "..."}
}`

const secretDBOldKey = `{
  "apiVersion": "v1",
  "kind": "Secret",
  "metadata": {"name": "db-secrets", "namespace": "data"},
  "data": {"postgres_password": "..."}
}`

func TestProactiveSecretKeyCheck_ValidReference(t *testing.T) {
	src := loadSrc(t, map[string]string{
		"deploy.json": deployRefsValidKey,
		"secret.json": secretFrontend,
	})
	got := ProactiveSecretKeyCheck{}.Run(context.Background(), src)
	if len(got) != 0 {
		t.Errorf("want 0 diagnostics for valid reference, got %d (%+v)", len(got), got)
	}
}

func TestProactiveSecretKeyCheck_MissingKey(t *testing.T) {
	src := loadSrc(t, map[string]string{
		"deploy.json": deployRefsMissingKey,
		"secret.json": secretBillingMissingStripe,
	})
	got := ProactiveSecretKeyCheck{}.Run(context.Background(), src)
	if len(got) != 1 {
		t.Fatalf("want 1 diagnostic, got %d", len(got))
	}
	for _, want := range []string{
		"Secret `billing/billing-secrets` exists but is missing key `stripe_api_key`",
		"Deployment/billing",
		"Existing keys: [OPENPROJECT_API_KEY]",
	} {
		if !strings.Contains(got[0].Message, want) {
			t.Errorf("missing %q in %q", want, got[0].Message)
		}
	}
	if got[0].Subject != "missing-key/billing/billing-secrets/stripe_api_key" {
		t.Errorf("subject = %q", got[0].Subject)
	}
}

func TestProactiveSecretKeyCheck_MissingSecret(t *testing.T) {
	src := loadSrc(t, map[string]string{
		"deploy.json": deployRefsMissingSecret,
	})
	got := ProactiveSecretKeyCheck{}.Run(context.Background(), src)
	if len(got) != 1 {
		t.Fatalf("want 1 diagnostic, got %d", len(got))
	}
	if !strings.Contains(got[0].Message, "Secret `demo/ghost-secret` does NOT exist") {
		t.Errorf("expected missing-secret diagnostic, got: %s", got[0].Message)
	}
}

func TestProactiveSecretKeyCheck_OptionalReferenceSkipped(t *testing.T) {
	src := loadSrc(t, map[string]string{
		"deploy.json": deployRefsOptional,
	})
	got := ProactiveSecretKeyCheck{}.Run(context.Background(), src)
	if len(got) != 0 {
		t.Errorf("optional ref should not produce diagnostic, got %d", len(got))
	}
}

func TestProactiveSecretKeyCheck_StatefulSetInitContainer(t *testing.T) {
	src := loadSrc(t, map[string]string{
		"ss.json":     ssRefsMissingKey,
		"secret.json": secretDBOldKey,
	})
	got := ProactiveSecretKeyCheck{}.Run(context.Background(), src)
	if len(got) != 1 {
		t.Fatalf("want 1 diagnostic for ss init container, got %d", len(got))
	}
	if !strings.Contains(got[0].Message, "StatefulSet/stateful-thing") {
		t.Errorf("expected StatefulSet kind in message: %s", got[0].Message)
	}
	if !strings.Contains(got[0].Message, "postgres_password_v2") {
		t.Errorf("expected missing-key name in message: %s", got[0].Message)
	}
}

func TestProactiveSecretKeyCheck_DedupesAcrossWorkloads(t *testing.T) {
	dup := strings.Replace(deployRefsMissingKey, `"name": "billing"`, `"name": "billing-2"`, 1)
	src := loadSrc(t, map[string]string{
		"deploy.json":  deployRefsMissingKey,
		"deploy2.json": dup,
		"secret.json":  secretBillingMissingStripe,
	})
	got := ProactiveSecretKeyCheck{}.Run(context.Background(), src)
	if len(got) != 1 {
		t.Errorf("want 1 deduped diagnostic, got %d", len(got))
	}
}

const deployEnvFromMissingSecret = `{
  "apiVersion": "apps/v1",
  "kind": "Deployment",
  "metadata": {"name": "bulk", "namespace": "demo"},
  "spec": {
    "template": {
      "spec": {
        "containers": [{
          "name": "x",
          "envFrom": [{"secretRef": {"name": "missing-bulk-secret"}}]
        }]
      }
    }
  }
}`

const deployEnvFromExistingSecret = `{
  "apiVersion": "apps/v1",
  "kind": "Deployment",
  "metadata": {"name": "bulk-ok", "namespace": "demo"},
  "spec": {
    "template": {
      "spec": {
        "containers": [{
          "name": "x",
          "envFrom": [{"secretRef": {"name": "frontend-secrets"}}]
        }]
      }
    }
  }
}`

const deployEnvFromOptional = `{
  "apiVersion": "apps/v1",
  "kind": "Deployment",
  "metadata": {"name": "bulk-opt", "namespace": "demo"},
  "spec": {
    "template": {
      "spec": {
        "containers": [{
          "name": "x",
          "envFrom": [{"secretRef": {"name": "missing-but-optional", "optional": true}}]
        }]
      }
    }
  }
}`

func TestProactiveSecretKeyCheck_EnvFromMissingSecret(t *testing.T) {
	src := loadSrc(t, map[string]string{
		"deploy.json": deployEnvFromMissingSecret,
		"secret.json": secretFrontend, // unrelated secret in another name
	})
	got := ProactiveSecretKeyCheck{}.Run(context.Background(), src)
	if len(got) != 1 {
		t.Fatalf("want 1 diagnostic, got %d: %+v", len(got), got)
	}
	for _, want := range []string{
		"Secret `demo/missing-bulk-secret` does NOT exist",
		"envFrom whole-secret import",
		"Deployment/bulk",
	} {
		if !strings.Contains(got[0].Message, want) {
			t.Errorf("missing %q in %q", want, got[0].Message)
		}
	}
	if got[0].Subject != "missing-secret/demo/missing-bulk-secret" {
		t.Errorf("subject = %q", got[0].Subject)
	}
}

func TestProactiveSecretKeyCheck_EnvFromExistingSecret(t *testing.T) {
	src := loadSrc(t, map[string]string{
		"deploy.json": deployEnvFromExistingSecret,
		"secret.json": secretFrontend,
	})
	got := ProactiveSecretKeyCheck{}.Run(context.Background(), src)
	if len(got) != 0 {
		t.Errorf("envFrom on existing Secret should not emit; got: %+v", got)
	}
}

func TestProactiveSecretKeyCheck_EnvFromOptionalSkipped(t *testing.T) {
	src := loadSrc(t, map[string]string{
		"deploy.json": deployEnvFromOptional,
	})
	got := ProactiveSecretKeyCheck{}.Run(context.Background(), src)
	if len(got) != 0 {
		t.Errorf("optional envFrom should not emit; got: %+v", got)
	}
}

// deployRefsWrongCaseKey references "github-token" but the Secret has "GITHUB_TOKEN".
const deployRefsWrongCaseKey = `{
  "apiVersion": "apps/v1",
  "kind": "Deployment",
  "metadata": {"name": "repomind", "namespace": "tools"},
  "spec": {
    "template": {
      "spec": {
        "containers": [{
          "name": "app",
          "env": [{
            "name": "GH_TOKEN",
            "valueFrom": {"secretKeyRef": {"name": "repomind-secrets", "key": "github-token"}}
          }]
        }]
      }
    }
  }
}`

const secretRepomindUpperCase = `{
  "apiVersion": "v1",
  "kind": "Secret",
  "metadata": {"name": "repomind-secrets", "namespace": "tools"},
  "data": {"GITHUB_TOKEN": "Z2hwXzEyMzQ1Njc4OQ=="}
}`

func TestProactiveSecretKeyCheck_NearMissKeyHint(t *testing.T) {
	// "github-token" normalizes to "github_token"; "GITHUB_TOKEN" also normalizes
	// to "github_token" — the hint should fire.
	src := loadSrc(t, map[string]string{
		"deploy.json": deployRefsWrongCaseKey,
		"secret.json": secretRepomindUpperCase,
	})
	got := ProactiveSecretKeyCheck{}.Run(context.Background(), src)
	if len(got) != 1 {
		t.Fatalf("want 1 diagnostic, got %d: %+v", len(got), got)
	}
	d := got[0]
	if !strings.Contains(d.Message, "missing key `github-token`") {
		t.Errorf("expected missing-key name in message: %s", d.Message)
	}
	if !strings.Contains(d.Message, "GITHUB_TOKEN") {
		t.Errorf("expected near-miss key name in message: %s", d.Message)
	}
	if !strings.Contains(d.Message, "case/format variant") {
		t.Errorf("expected near-miss hint in message: %s", d.Message)
	}
}

func TestProactiveSecretKeyCheck_NoNearMissWhenTrulyMissing(t *testing.T) {
	// "stripe_api_key" has no case/format variant in the Secret — no hint.
	src := loadSrc(t, map[string]string{
		"deploy.json": deployRefsMissingKey,
		"secret.json": secretBillingMissingStripe,
	})
	got := ProactiveSecretKeyCheck{}.Run(context.Background(), src)
	if len(got) != 1 {
		t.Fatalf("want 1 diagnostic, got %d", len(got))
	}
	if strings.Contains(got[0].Message, "case/format variant") {
		t.Errorf("near-miss hint should NOT appear when no variant exists: %s", got[0].Message)
	}
}

func TestProactiveSecretKeyCheck_NoSecretsInSnapshot(t *testing.T) {
	// Offline snapshot mode: capture deliberately excludes Secrets, so the
	// analyzer should be a clean no-op (not a flood of "missing secret"
	// false-positives).
	src := loadSrc(t, map[string]string{
		"deploy.json": deployRefsValidKey,
	})
	got := ProactiveSecretKeyCheck{}.Run(context.Background(), src)
	// Without Secrets, every reference looks like missing-secret. This is
	// expected snapshot-mode behavior — operators should run live or capture
	// Secrets explicitly. We DO emit diagnostics; the operator sees them
	// labeled as such. That's the intended behavior; we don't want silent
	// false-cleans.
	if len(got) != 1 {
		t.Errorf("want 1 missing-secret diagnostic in snapshot-mode-without-secrets, got %d", len(got))
	}
}
