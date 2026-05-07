// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package snapshot

import (
	"context"
	"fmt"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// Live is a Source backed by a Kubernetes API server.
//
// It uses dynamic client (untyped) so probes can ask for any GVR including
// CRDs (CNPG, External Secrets) without us having to vendor those CRDs into
// our build.
type Live struct {
	client dynamic.Interface
}

// LoadLive builds a Live source. If kubeconfigPath is empty, the in-cluster
// config is used (required when running as a pod); otherwise the file at
// the given path is loaded with $HOME/.kube/config as a final fallback.
func LoadLive(kubeconfigPath string) (*Live, error) {
	cfg, err := buildConfig(kubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("load kubeconfig: %w", err)
	}
	dyn, err := dynamic.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("dynamic client: %w", err)
	}
	return &Live{client: dyn}, nil
}

func buildConfig(kubeconfigPath string) (*rest.Config, error) {
	if kubeconfigPath != "" {
		return clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	}
	// Try in-cluster first.
	if cfg, err := rest.InClusterConfig(); err == nil {
		return cfg, nil
	}
	// Fall back to default loading rules ($KUBECONFIG, ~/.kube/config).
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	cc := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, &clientcmd.ConfigOverrides{})
	return cc.ClientConfig()
}

// List returns all objects of the given GVR.
//
// "Resource not found" errors (CRD not installed) are returned as an empty
// list with nil error — probes treat this as "feature not in this cluster"
// rather than as a probe failure.
func (l *Live) List(ctx context.Context, gvr schema.GroupVersionResource, ns string) (*unstructured.UnstructuredList, error) {
	var ri dynamic.ResourceInterface
	if ns == "" {
		ri = l.client.Resource(gvr)
	} else {
		ri = l.client.Resource(gvr).Namespace(ns)
	}
	list, err := ri.List(ctx, v1.ListOptions{})
	if err != nil {
		// Distinguish "no such resource type" (CRD missing) from real errors.
		// The dynamic client returns an error containing "the server could not find the requested resource"
		// which we translate to an empty list.
		if isResourceNotFound(err) {
			return &unstructured.UnstructuredList{}, nil
		}
		return nil, err
	}
	return list, nil
}

// Get returns a single object by namespace + name from the live cluster.
func (l *Live) Get(ctx context.Context, gvr schema.GroupVersionResource, ns, name string) (*unstructured.Unstructured, error) {
	var ri dynamic.ResourceInterface
	if ns == "" {
		ri = l.client.Resource(gvr)
	} else {
		ri = l.client.Resource(gvr).Namespace(ns)
	}
	return ri.Get(ctx, name, v1.GetOptions{})
}

// Mode reports live mode — fixers are permitted.
func (l *Live) Mode() Mode { return ModeLive }

// Watch returns a watch.Interface for the given GVR across all namespaces.
// The caller must call Stop() on the returned watcher when done.
// ResourceNotFound errors (CRD not installed) are returned as-is so callers
// can decide to skip the watch for that GVR.
func (l *Live) Watch(ctx context.Context, gvr schema.GroupVersionResource) (watch.Interface, error) {
	return l.client.Resource(gvr).Watch(ctx, v1.ListOptions{})
}

// isResourceNotFound returns true for the API-server "no resource type" error
// (typically when a CRD is not installed). This is intentionally string-based
// rather than typed because the dynamic client returns a generic StatusError.
func isResourceNotFound(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return contains(msg, "the server could not find the requested resource") ||
		contains(msg, "no matches for kind")
}

func contains(s, sub string) bool {
	if len(sub) == 0 {
		return true
	}
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
