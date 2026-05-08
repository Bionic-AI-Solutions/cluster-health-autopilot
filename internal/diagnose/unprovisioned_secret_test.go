// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package diagnose

import (
	"context"
	"strings"
	"testing"
)

// ── fixture helpers ──────────────────────────────────────────────────────────

const unprovDeployEnvFrom = `{
  "apiVersion": "apps/v1",
  "kind": "Deployment",
  "metadata": {"name": "playground-agent", "namespace": "playground"},
  "spec": {
    "template": {
      "spec": {
        "containers": [{
          "name": "agent",
          "envFrom": [{"secretRef": {"name": "playground-agent-secrets"}}]
        }]
      }
    }
  }
}`

const unprovDeployVolume = `{
  "apiVersion": "apps/v1",
  "kind": "Deployment",
  "metadata": {"name": "config-reader", "namespace": "infra"},
  "spec": {
    "template": {
      "spec": {
        "containers": [{"name": "reader"}],
        "volumes": [{"name": "cfg", "secret": {"secretName": "infra-config-secret"}}]
      }
    }
  }
}`

const unprovStatefulSet = `{
  "apiVersion": "apps/v1",
  "kind": "StatefulSet",
  "metadata": {"name": "db", "namespace": "data"},
  "spec": {
    "template": {
      "spec": {
        "containers": [{
          "name": "db",
          "envFrom": [{"secretRef": {"name": "db-credentials"}}]
        }]
      }
    }
  }
}`

const unprovCronJob = `{
  "apiVersion": "batch/v1",
  "kind": "CronJob",
  "metadata": {"name": "nightly-sync", "namespace": "ops"},
  "spec": {
    "jobTemplate": {
      "spec": {
        "template": {
          "spec": {
            "containers": [{
              "name": "sync",
              "envFrom": [{"secretRef": {"name": "sync-secrets"}}]
            }]
          }
        }
      }
    }
  }
}`

const unprovDeployOptional = `{
  "apiVersion": "apps/v1",
  "kind": "Deployment",
  "metadata": {"name": "tolerant", "namespace": "demo"},
  "spec": {
    "template": {
      "spec": {
        "containers": [{
          "name": "x",
          "envFrom": [{"secretRef": {"name": "optional-secret", "optional": true}}]
        }]
      }
    }
  }
}`

// esoProvisionedSecret has spec.target.name matching the referenced Secret.
const esoProvisionedSecret = `{
  "apiVersion": "external-secrets.io/v1",
  "kind": "ExternalSecret",
  "metadata": {"name": "playground-agent-secrets", "namespace": "playground"},
  "spec": {
    "target": {"name": "playground-agent-secrets"},
    "dataFrom": [{"extract": {"key": "t6-apps/playground/config"}}]
  },
  "status": {
    "conditions": [{"type": "Ready", "status": "True"}]
  }
}`

// secretExists is a Secret that physically exists in the cluster.
const unprovExistingSecret = `{
  "apiVersion": "v1",
  "kind": "Secret",
  "metadata": {"name": "playground-agent-secrets", "namespace": "playground"},
  "data": {"some_key": "dmFsdWU="}
}`

// ── tests ────────────────────────────────────────────────────────────────────

func TestUnprovisionedSecret_EmitsWhenNoESO(t *testing.T) {
	src := loadSrc(t, map[string]string{
		"deploy.json": unprovDeployEnvFrom,
	})
	got := UnprovisionedSecret{}.Run(context.Background(), src)
	if len(got) != 1 {
		t.Fatalf("want 1 diagnostic, got %d: %+v", len(got), got)
	}
	d := got[0]
	for _, want := range []string{
		"Secret `playground/playground-agent-secrets`",
		"Deployment/playground-agent",
		"no ExternalSecret provisioning it",
		"secret/t6-apps/playground/config",
		"spec.target.name=playground-agent-secrets",
	} {
		if !strings.Contains(d.Message, want) {
			t.Errorf("missing %q in message: %s", want, d.Message)
		}
	}
	if d.Subject != "unprovisioned/playground/playground-agent-secrets" {
		t.Errorf("unexpected subject: %s", d.Subject)
	}
}

func TestUnprovisionedSecret_SilentWhenESOProvisions(t *testing.T) {
	// ESO targets the same Secret name — CHA should trust ESO is handling it.
	src := loadSrc(t, map[string]string{
		"deploy.json": unprovDeployEnvFrom,
		"eso.json":    esoProvisionedSecret,
	})
	got := UnprovisionedSecret{}.Run(context.Background(), src)
	if len(got) != 0 {
		t.Errorf("want 0 diagnostics when ESO provisions the Secret, got %d: %+v", len(got), got)
	}
}

func TestUnprovisionedSecret_SilentWhenSecretExists(t *testing.T) {
	// Secret exists in cluster (e.g. manually created or created by Helm).
	src := loadSrc(t, map[string]string{
		"deploy.json":  unprovDeployEnvFrom,
		"secret.json":  unprovExistingSecret,
	})
	got := UnprovisionedSecret{}.Run(context.Background(), src)
	if len(got) != 0 {
		t.Errorf("want 0 diagnostics when Secret exists, got %d: %+v", len(got), got)
	}
}

func TestUnprovisionedSecret_OptionalSkipped(t *testing.T) {
	src := loadSrc(t, map[string]string{
		"deploy.json": unprovDeployOptional,
	})
	got := UnprovisionedSecret{}.Run(context.Background(), src)
	if len(got) != 0 {
		t.Errorf("optional envFrom should not emit, got %d", len(got))
	}
}

func TestUnprovisionedSecret_VolumeMountedSecret(t *testing.T) {
	src := loadSrc(t, map[string]string{
		"deploy.json": unprovDeployVolume,
	})
	got := UnprovisionedSecret{}.Run(context.Background(), src)
	if len(got) != 1 {
		t.Fatalf("want 1 diagnostic for volume-mounted secret, got %d", len(got))
	}
	if !strings.Contains(got[0].Message, "infra/infra-config-secret") {
		t.Errorf("expected secret name in message: %s", got[0].Message)
	}
	if !strings.Contains(got[0].Message, "secret/t6-apps/infra/config") {
		t.Errorf("expected t6 vault path hint in message: %s", got[0].Message)
	}
}

func TestUnprovisionedSecret_StatefulSet(t *testing.T) {
	src := loadSrc(t, map[string]string{
		"ss.json": unprovStatefulSet,
	})
	got := UnprovisionedSecret{}.Run(context.Background(), src)
	if len(got) != 1 {
		t.Fatalf("want 1 diagnostic for StatefulSet, got %d", len(got))
	}
	if !strings.Contains(got[0].Message, "StatefulSet/db") {
		t.Errorf("expected StatefulSet kind in message: %s", got[0].Message)
	}
	if !strings.Contains(got[0].Message, "secret/t6-apps/data/config") {
		t.Errorf("expected t6 path hint: %s", got[0].Message)
	}
}

func TestUnprovisionedSecret_CronJob(t *testing.T) {
	src := loadSrc(t, map[string]string{
		"cj.json": unprovCronJob,
	})
	got := UnprovisionedSecret{}.Run(context.Background(), src)
	if len(got) != 1 {
		t.Fatalf("want 1 diagnostic for CronJob, got %d", len(got))
	}
	if !strings.Contains(got[0].Message, "CronJob/nightly-sync") {
		t.Errorf("expected CronJob kind in message: %s", got[0].Message)
	}
}

func TestUnprovisionedSecret_DeduplicatesAcrossWorkloads(t *testing.T) {
	// Two Deployments in the same namespace reference the same missing Secret.
	second := strings.Replace(unprovDeployEnvFrom, `"name": "playground-agent"`, `"name": "second-agent"`, 1)
	src := loadSrc(t, map[string]string{
		"deploy1.json": unprovDeployEnvFrom,
		"deploy2.json": second,
	})
	got := UnprovisionedSecret{}.Run(context.Background(), src)
	if len(got) != 1 {
		t.Errorf("want 1 deduplicated diagnostic, got %d", len(got))
	}
}
