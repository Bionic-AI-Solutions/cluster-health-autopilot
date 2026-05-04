// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

// Package snapshot provides the data-layer abstraction that decouples the
// probes/analyzers from where Kubernetes object data comes from.
//
// Two implementations are provided:
//   - Live (live.go): backed by a Kubernetes API server via client-go.
//   - File (file.go): backed by a directory of `kubectl get -o json` outputs,
//     enabling zero-trust offline diagnose with no install / no RBAC.
//
// Probes and analyzers consume the Source interface — they never touch
// client-go or the filesystem directly. This is the contract that makes the
// "run on a captured snapshot from your laptop" headline possible.
package snapshot

import (
	"context"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// Source returns Kubernetes objects to a probe.
//
// Implementations must:
//   - Return cluster-scoped resources when ns == "" and namespaced resources
//     when ns is provided (or "" to mean "all namespaces").
//   - Return an empty list (not error) when a resource type does not exist
//     in this cluster (e.g. CNPG CRD not installed); callers distinguish
//     "not installed" from "permission denied" via the err return.
type Source interface {
	// List returns all instances of the given GVR, optionally filtered by namespace.
	// If ns == "", returns objects across all namespaces (for namespaced resources)
	// or the single cluster-scoped collection (for cluster resources).
	List(ctx context.Context, gvr schema.GroupVersionResource, ns string) (*unstructured.UnstructuredList, error)

	// Get returns a single instance by namespace + name.
	Get(ctx context.Context, gvr schema.GroupVersionResource, ns, name string) (*unstructured.Unstructured, error)

	// Mode reports whether this source is live (allows fixers to run) or
	// snapshot-backed (read-only — fixers refuse to operate).
	Mode() Mode
}

// Mode reports whether a Source is backed by a live cluster or a snapshot.
type Mode int

// Mode values.
const (
	ModeLive Mode = iota
	ModeSnapshot
)

func (m Mode) String() string {
	switch m {
	case ModeLive:
		return "live"
	case ModeSnapshot:
		return "snapshot"
	default:
		return "unknown"
	}
}

// Common GVRs used across probes.
var (
	GVRPod         = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}
	GVRNode        = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "nodes"}
	GVRPVC         = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "persistentvolumeclaims"}
	GVREvent       = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "events"}
	GVRDeployment  = schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}
	GVRReplicaSet  = schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "replicasets"}
	GVRStatefulSet = schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "statefulsets"}
	GVRJob         = schema.GroupVersionResource{Group: "batch", Version: "v1", Resource: "jobs"}
	GVRCronJob     = schema.GroupVersionResource{Group: "batch", Version: "v1", Resource: "cronjobs"}
	GVRExtSecret   = schema.GroupVersionResource{Group: "external-secrets.io", Version: "v1", Resource: "externalsecrets"}
	GVRCNPGCluster = schema.GroupVersionResource{Group: "postgresql.cnpg.io", Version: "v1", Resource: "clusters"}
	GVRCephCluster = schema.GroupVersionResource{Group: "ceph.rook.io", Version: "v1", Resource: "cephclusters"}
	GVRSecret      = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "secrets"}
	GVRDriftReport = schema.GroupVersionResource{Group: "cha.bionicaisolutions.com", Version: "v1alpha1", Resource: "driftreports"}

	// External-Secrets Operator store kinds. Used by VaultPathMissing to
	// filter ExternalSecrets to only those whose backing store is Vault —
	// avoids false-positive diagnostics for ESOs backed by AWS Secrets
	// Manager, GCP Secret Manager, etc.
	GVRSecretStore        = schema.GroupVersionResource{Group: "external-secrets.io", Version: "v1", Resource: "secretstores"}
	GVRClusterSecretStore = schema.GroupVersionResource{Group: "external-secrets.io", Version: "v1", Resource: "clustersecretstores"}
)
