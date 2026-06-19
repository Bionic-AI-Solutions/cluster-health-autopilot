// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package probe

import (
	"context"
	"testing"
)

// podsServiceTerminating has 1 terminating pod and 1 ready pod; the
// terminating one must not count as "matched but not ready".
const podsServiceTerminating = `{
  "apiVersion": "v1", "kind": "PodList",
  "items": [
    {"apiVersion": "v1", "kind": "Pod",
     "metadata": {"name": "myapp-old", "namespace": "demo",
                  "labels": {"app": "myapp"},
                  "deletionTimestamp": "2026-06-19T10:00:00Z"},
     "status": {"containerStatuses": [{"ready": false}]}},
    {"apiVersion": "v1", "kind": "Pod",
     "metadata": {"name": "myapp-new", "namespace": "demo",
                  "labels": {"app": "myapp"}},
     "status": {"containerStatuses": [{"ready": true}]}}
  ]
}`

func TestServices_TerminatingPodNotCountedAsNotReady(t *testing.T) {
	src := loadProbeSrc(t, map[string]string{"pods.json": podsServiceTerminating})
	s := Services{Targets: []ServiceTarget{
		{Namespace: "demo", Selector: "app=myapp", Display: "myapp"},
	}}
	r := s.Run(context.Background(), src)
	if r.Component.Status != "HEALTHY" {
		t.Errorf("Terminating pod must not count as not-ready; got Status=%q Detail=%q",
			r.Component.Status, r.Component.Detail)
	}
}

func TestServices_NonTerminatingNotReadyPodStillFlagged(t *testing.T) {
	// A not-ready pod without deletionTimestamp must still trigger the finding.
	const podsServiceNotReady = `{
  "apiVersion": "v1", "kind": "PodList",
  "items": [
    {"apiVersion": "v1", "kind": "Pod",
     "metadata": {"name": "myapp-abc", "namespace": "demo",
                  "labels": {"app": "myapp"}},
     "status": {"containerStatuses": [{"ready": false}]}}
  ]
}`
	src := loadProbeSrc(t, map[string]string{"pods.json": podsServiceNotReady})
	s := Services{Targets: []ServiceTarget{
		{Namespace: "demo", Selector: "app=myapp", Display: "myapp"},
	}}
	r := s.Run(context.Background(), src)
	if r.Component.Status == "HEALTHY" {
		t.Errorf("non-terminating not-ready pod must still be flagged; got HEALTHY")
	}
}
