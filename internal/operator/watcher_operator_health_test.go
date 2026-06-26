// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

package operator

import (
	"testing"

	chav1alpha1 "github.com/srenix-ai/agentic-sre/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Operator-built watcher Deployment must carry the same always-on
// health port + liveness/readiness probes the Helm chart ships, so the
// two deployment paths stay at parity (P1.9 follow-up).
func TestBuildWatcherDeployment_HealthPortAndProbes(t *testing.T) {
	cr := &chav1alpha1.AgenticSRE{
		ObjectMeta: metav1.ObjectMeta{Name: "srenix", Namespace: "ops"},
		Spec: chav1alpha1.AgenticSRESpec{
			Watcher: &chav1alpha1.WatcherSpec{Enabled: true},
		},
	}
	dep := BuildWatcherDeployment(cr)
	if dep == nil {
		t.Fatal("expected a Deployment")
	}
	c := dep.Spec.Template.Spec.Containers[0]

	var hasHealthPort bool
	for _, p := range c.Ports {
		if p.Name == "health" && p.ContainerPort == watcherHealthPort {
			hasHealthPort = true
		}
	}
	if !hasHealthPort {
		t.Errorf("watcher container missing always-on health port %d; got %+v", watcherHealthPort, c.Ports)
	}
	if c.LivenessProbe == nil || c.LivenessProbe.HTTPGet == nil ||
		c.LivenessProbe.HTTPGet.Path != "/healthz" || c.LivenessProbe.HTTPGet.Port.StrVal != "health" {
		t.Errorf("watcher container liveness probe not wired to /healthz on health port; got %+v", c.LivenessProbe)
	}
	if c.ReadinessProbe == nil || c.ReadinessProbe.HTTPGet == nil ||
		c.ReadinessProbe.HTTPGet.Path != "/healthz" {
		t.Errorf("watcher container readiness probe not wired to /healthz; got %+v", c.ReadinessProbe)
	}
}
