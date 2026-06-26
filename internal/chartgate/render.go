// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

package chartgate

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"

	"sigs.k8s.io/yaml"
)

// ChartDir is the path to the chart relative to a test file in this
// package. Exported so the cmd/srenix and cmd/srenix-operator parity gates
// (which live in their own main packages) can render through a shared
// path resolved against this package's source location.
func chartDir() string {
	_, thisFile, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(thisFile), "..", "..", "charts", "agentic-sre")
}

// maximalValues turns on every feature that adds a container flag so the
// rendered args surface is the union of all flags the chart can emit.
// Kept deliberately broad — the parity gate must see the maximal flag
// set, not the default-install subset.
const maximalValues = `
watcher:
  enabled: true
  remedy:
    enabled: true
    dryRun: true
  triggers:
    prom:
      url: "http://alertmanager.monitoring.svc:9093"
      interval: 30s
      alertNameFilter: ["DiskFillUp"]
    webhook:
      listen: ":8090"
      sources: ["vault=SRENIX_WEBHOOK_VAULT_SECRET"]
      secretName: srenix-webhook-secrets
  slack:
    criticalRepeatInterval: "4h"
diagnose:
  enabled: true
remediation:
  enabled: true
operator:
  enabled: true
slack:
  alerts:
    enabled: true
    secretName: srenix-slack
  critical:
    enabled: true
    secretName: srenix-slack
  healthinfo:
    enabled: true
    secretName: srenix-slack
alertmanager:
  enabled: true
  url: "http://alertmanager.monitoring.svc:9093"
  clusterName: "test"
ticketing:
  enabled: true
  provider: openproject
  mcpURL: "http://op-mcp.svc"
  project: "1"
  typeID: "7"
  closedStatusID: "12"
  webURLPrefix: "https://op.example.com"
  labels: ["srenix"]
  dryRun: true
cloud:
  enabled: true
  aws:
    enabled: true
    region: us-east-1
  gcp:
    enabled: true
    project: my-proj
  azure:
    enabled: true
    subscriptionId: sub-123
`

// RoleArgs maps a workload role label
// (srenix.ai/role) to the de-duplicated set of
// container args rendered for it across all pod specs that carry that
// role. Operator Deployments are keyed under "operator" by name match
// since they carry no role label.
type RoleArgs map[string][]string

var flagRe = regexp.MustCompile(`^--?([A-Za-z][A-Za-z0-9-]*)`)

// flagName extracts the flag name from a rendered arg, stripping a
// leading "--"/"-" and any "=value" suffix. Returns "" for positional
// args (e.g. "watch", "diagnose", "remediate").
func flagName(arg string) string {
	if !strings.HasPrefix(arg, "-") {
		return ""
	}
	arg = strings.SplitN(arg, "=", 2)[0]
	m := flagRe.FindStringSubmatch(arg)
	if m == nil {
		return ""
	}
	return m[1]
}

// RenderMaximalArgs runs `helm template` with the maximal values file
// and returns, per workload role, the flag names appearing in each
// container's args/command. Skips (t.Skip) when the helm binary is
// absent so local `go test ./...` without helm still passes; CI has
// helm on PATH (ci.yml) so the gate runs there.
func RenderMaximalArgs(t *testing.T) RoleArgs {
	t.Helper()

	if _, err := exec.LookPath("helm"); err != nil {
		t.Skipf("helm not on PATH — skipping chart-args↔binary-flags parity gate locally (it MUST run in CI, where helm is installed)")
	}

	tmp, err := os.CreateTemp(t.TempDir(), "maxvals-*.yaml")
	if err != nil {
		t.Fatalf("temp values: %v", err)
	}
	if _, err := tmp.WriteString(maximalValues); err != nil {
		t.Fatalf("write values: %v", err)
	}
	if err := tmp.Close(); err != nil {
		t.Fatalf("close values: %v", err)
	}

	cmd := exec.Command("helm", "template", "gate", chartDir(), "-f", tmp.Name())
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("helm template failed: %v\n%s", err, out)
	}
	return parseRoleArgs(t, string(out))
}

// minimalManifest is the subset of a rendered k8s object the gate needs:
// kind, role label, and every container's args+command.
type minimalManifest struct {
	Kind     string `json:"kind"`
	Metadata struct {
		Name   string            `json:"name"`
		Labels map[string]string `json:"labels"`
	} `json:"metadata"`
	Spec struct {
		// Deployment.spec.template.spec.containers
		Template podTemplate `json:"template"`
		// CronJob.spec.jobTemplate.spec.template.spec.containers
		JobTemplate struct {
			Spec struct {
				Template podTemplate `json:"template"`
			} `json:"spec"`
		} `json:"jobTemplate"`
	} `json:"spec"`
}

type podTemplate struct {
	Metadata struct {
		Labels map[string]string `json:"labels"`
	} `json:"metadata"`
	Spec struct {
		Containers []struct {
			Args    []string `json:"args"`
			Command []string `json:"command"`
		} `json:"containers"`
	} `json:"spec"`
}

const roleLabel = "srenix.ai/role"

func parseRoleArgs(t *testing.T, rendered string) RoleArgs {
	t.Helper()
	out := RoleArgs{}
	add := func(role string, args, command []string) {
		for _, a := range append(append([]string{}, command...), args...) {
			if name := flagName(a); name != "" {
				out[role] = appendUnique(out[role], name)
			}
		}
	}

	for _, doc := range strings.Split(rendered, "\n---") {
		if strings.TrimSpace(doc) == "" {
			continue
		}
		var m minimalManifest
		if err := yaml.Unmarshal([]byte(doc), &m); err != nil {
			// Non-object docs (comments / NOTES) — skip.
			continue
		}
		switch m.Kind {
		case "Deployment":
			role := m.Spec.Template.Metadata.Labels[roleLabel]
			if role == "" {
				// Operator Deployment carries no role label; key by name.
				if strings.Contains(m.Metadata.Name, "operator") {
					role = "operator"
				} else {
					continue
				}
			}
			for _, c := range m.Spec.Template.Spec.Containers {
				add(role, c.Args, c.Command)
			}
		case "CronJob":
			tmpl := m.Spec.JobTemplate.Spec.Template
			role := tmpl.Metadata.Labels[roleLabel]
			if role == "" {
				continue
			}
			for _, c := range tmpl.Spec.Containers {
				add(role, c.Args, c.Command)
			}
		}
	}
	if len(out) == 0 {
		t.Fatalf("parsed zero container args from helm render — template shape changed; update parseRoleArgs()")
	}
	return out
}

func appendUnique(s []string, v string) []string {
	for _, x := range s {
		if x == v {
			return s
		}
	}
	return append(s, v)
}
