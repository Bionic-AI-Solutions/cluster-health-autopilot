// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package ai

import "testing"

func TestIsGitOpsManaged(t *testing.T) {
	cases := []struct {
		name        string
		labels      map[string]string
		annotations map[string]string
		want        bool
	}{
		{
			name:        "argo tracking-id annotation",
			annotations: map[string]string{"argocd.argoproj.io/tracking-id": "app:apps/Deployment:ns/x"},
			want:        true,
		},
		{
			name:   "argo instance label",
			labels: map[string]string{"argocd.argoproj.io/instance": "my-app"},
			want:   true,
		},
		{
			name:        "flux kustomize annotation",
			annotations: map[string]string{"kustomize.toolkit.fluxcd.io/name": "apps"},
			want:        true,
		},
		{
			name:        "empty annotation value is not managed",
			annotations: map[string]string{"argocd.argoproj.io/tracking-id": "  "},
			want:        false,
		},
		{
			name:   "plain helm managed-by is NOT gitops",
			labels: map[string]string{"app.kubernetes.io/managed-by": "Helm"},
			want:   false,
		},
		{
			name:   "no markers",
			labels: map[string]string{"app": "x"},
			want:   false,
		},
		{
			name: "nil maps",
			want: false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsGitOpsManaged(tc.labels, tc.annotations); got != tc.want {
				t.Errorf("IsGitOpsManaged() = %v, want %v", got, tc.want)
			}
		})
	}
}
