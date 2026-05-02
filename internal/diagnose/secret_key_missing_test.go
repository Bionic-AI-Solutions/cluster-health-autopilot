// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package diagnose

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/internal/snapshot"
)

// loadSrc writes a fixture set into a tempdir snapshot and returns the loaded Source.
func loadSrc(t *testing.T, files map[string]string) snapshot.Source {
	t.Helper()
	dir := t.TempDir()
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	src, err := snapshot.LoadFile(dir)
	if err != nil {
		t.Fatalf("LoadFile: %v", err)
	}
	return src
}

const podCCEViaContainerStatus = `{
  "apiVersion": "v1",
  "kind": "PodList",
  "items": [
    {
      "apiVersion": "v1",
      "kind": "Pod",
      "metadata": {
        "name": "mcp-openproject-server-69df8b57bd-fx82n",
        "namespace": "mcp",
        "ownerReferences": [{"kind": "ReplicaSet", "name": "mcp-openproject-server-69df8b57bd"}]
      },
      "status": {
        "containerStatuses": [{
          "name": "mcp-openproject",
          "state": {
            "waiting": {
              "reason": "CreateContainerConfigError",
              "message": "couldn't find key openproject-url in Secret mcp/mcp-openproject-secrets"
            }
          }
        }]
      }
    }
  ]
}`

const replicaSetForOpenProject = `{
  "apiVersion": "apps/v1",
  "kind": "ReplicaSet",
  "metadata": {
    "name": "mcp-openproject-server-69df8b57bd",
    "namespace": "mcp",
    "ownerReferences": [{"kind": "Deployment", "name": "mcp-openproject-server"}]
  }
}`

const externalSecretOpenProject = `{
  "apiVersion": "external-secrets.io/v1",
  "kind": "ExternalSecret",
  "metadata": {"name": "mcp-openproject-secrets", "namespace": "mcp"},
  "spec": {"target": {"name": "mcp-openproject-secrets"}}
}`

func TestSecretKeyMissing_FullChain(t *testing.T) {
	src := loadSrc(t, map[string]string{
		"pods.json": podCCEViaContainerStatus,
		"rs.json":   replicaSetForOpenProject,
		"eso.json":  externalSecretOpenProject,
	})
	got := SecretKeyMissing{}.Run(context.Background(), src)
	if len(got) != 1 {
		t.Fatalf("want 1 diagnostic, got %d (%+v)", len(got), got)
	}
	d := got[0]
	wantSubstrs := []string{
		"`mcp/mcp-openproject-secrets`",
		"missing key `openproject-url`",
		"Deployment/mcp-openproject-server",
		"Owning ExternalSecret: `mcp/mcp-openproject-secrets`",
	}
	for _, sub := range wantSubstrs {
		if !strings.Contains(d.Message, sub) {
			t.Errorf("message missing %q\nfull message: %s", sub, d.Message)
		}
	}
	if d.Subject != "Secret/mcp/mcp-openproject-secrets/openproject-url" {
		t.Errorf("subject = %q", d.Subject)
	}
}

func TestSecretKeyMissing_NoESO(t *testing.T) {
	// Same pod + RS but no ExternalSecret in the snapshot — analyzer should
	// fall back to the "Secret is hand-managed" wording.
	src := loadSrc(t, map[string]string{
		"pods.json": podCCEViaContainerStatus,
		"rs.json":   replicaSetForOpenProject,
	})
	got := SecretKeyMissing{}.Run(context.Background(), src)
	if len(got) != 1 {
		t.Fatalf("want 1 diagnostic, got %d", len(got))
	}
	if !strings.Contains(got[0].Message, "No owning ExternalSecret") {
		t.Errorf("expected fallback wording, got: %s", got[0].Message)
	}
}

func TestSecretKeyMissing_DedupesAcrossPods(t *testing.T) {
	// Two pods of the same Deployment, both stuck on the same missing key —
	// should produce exactly one diagnostic.
	multiPod := strings.Replace(podCCEViaContainerStatus,
		`"items": [`,
		`"items": [
    {
      "apiVersion": "v1",
      "kind": "Pod",
      "metadata": {
        "name": "mcp-openproject-server-69df8b57bd-other",
        "namespace": "mcp",
        "ownerReferences": [{"kind": "ReplicaSet", "name": "mcp-openproject-server-69df8b57bd"}]
      },
      "status": {
        "containerStatuses": [{
          "name": "mcp-openproject",
          "state": {
            "waiting": {
              "reason": "CreateContainerConfigError",
              "message": "couldn't find key openproject-url in Secret mcp/mcp-openproject-secrets"
            }
          }
        }]
      }
    },`, 1)

	src := loadSrc(t, map[string]string{
		"pods.json": multiPod,
		"rs.json":   replicaSetForOpenProject,
		"eso.json":  externalSecretOpenProject,
	})
	got := SecretKeyMissing{}.Run(context.Background(), src)
	if len(got) != 1 {
		t.Fatalf("want 1 (deduped) diagnostic, got %d", len(got))
	}
}

func TestSecretKeyMissing_NoCCEPods(t *testing.T) {
	healthy := `{
  "apiVersion": "v1",
  "kind": "PodList",
  "items": [{
    "apiVersion": "v1", "kind": "Pod",
    "metadata": {"name": "ok", "namespace": "demo"},
    "status": {"containerStatuses": [{"name": "c", "ready": true, "state": {"running": {}}}]}
  }]
}`
	src := loadSrc(t, map[string]string{"pods.json": healthy})
	got := SecretKeyMissing{}.Run(context.Background(), src)
	if len(got) != 0 {
		t.Errorf("want 0 diagnostics, got %d", len(got))
	}
}

func TestSecretKeyMissing_ApostropheVariants(t *testing.T) {
	// Some Kubernetes versions emit "couldnt" without the apostrophe — both
	// must match.
	withoutApos := strings.Replace(podCCEViaContainerStatus, "couldn't", "couldnt", 1)
	src := loadSrc(t, map[string]string{"pods.json": withoutApos})
	got := SecretKeyMissing{}.Run(context.Background(), src)
	if len(got) != 1 {
		t.Fatalf("want 1 diagnostic for apostrophe-less variant, got %d", len(got))
	}
}
