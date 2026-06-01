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
//
// The canonical interface types live in pkg/snapshot; the aliases below keep
// all internal packages compiling without import changes.
package snapshot

import (
	pkgsnapshot "github.com/Bionic-AI-Solutions/cluster-health-autopilot/pkg/snapshot"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// Source is re-exported from pkg/snapshot; see that package for the canonical definition.
type Source = pkgsnapshot.Source

// Mutator is re-exported from pkg/snapshot; see that package for the canonical definition.
type Mutator = pkgsnapshot.Mutator

// Mode is re-exported from pkg/snapshot; see that package for the canonical definition.
type Mode = pkgsnapshot.Mode

// Mode constants re-exported from pkg/snapshot.
const (
	ModeLive     = pkgsnapshot.ModeLive
	ModeSnapshot = pkgsnapshot.ModeSnapshot
)

// Common GVRs used across probes and analyzers.
var (
	GVRPod         = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}
	GVRNode        = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "nodes"}
	GVRPVC         = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "persistentvolumeclaims"}
	GVREvent       = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "events"}
	GVRDeployment  = schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}
	GVRReplicaSet  = schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "replicasets"}
	GVRStatefulSet = schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "statefulsets"}
	GVRDaemonSet   = schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "daemonsets"}
	GVRJob         = schema.GroupVersionResource{Group: "batch", Version: "v1", Resource: "jobs"}
	GVRCronJob     = schema.GroupVersionResource{Group: "batch", Version: "v1", Resource: "cronjobs"}
	GVRExtSecret   = schema.GroupVersionResource{Group: "external-secrets.io", Version: "v1", Resource: "externalsecrets"}
	GVRCNPGCluster = schema.GroupVersionResource{Group: "postgresql.cnpg.io", Version: "v1", Resource: "clusters"}
	GVRCephCluster = schema.GroupVersionResource{Group: "ceph.rook.io", Version: "v1", Resource: "cephclusters"}
	GVRSecret      = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "secrets"}
	GVRDriftReport = schema.GroupVersionResource{Group: "cha.bionicaisolutions.com", Version: "v1alpha1", Resource: "driftreports"}
	// GVRResolutionRecord is the append-only outcome log: one record per
	// applied+verified (or rejected/reverted) remediation. Source of truth
	// for the RAG memory layer (P1/P2 of the AI-remediation plan).
	GVRResolutionRecord = schema.GroupVersionResource{Group: "cha.bionicaisolutions.com", Version: "v1alpha1", Resource: "resolutionrecords"}

	GVRSecretStore        = schema.GroupVersionResource{Group: "external-secrets.io", Version: "v1", Resource: "secretstores"}
	GVRClusterSecretStore = schema.GroupVersionResource{Group: "external-secrets.io", Version: "v1", Resource: "clustersecretstores"}

	GVRCertificate          = schema.GroupVersionResource{Group: "cert-manager.io", Version: "v1", Resource: "certificates"}
	GVRCertificateRequest   = schema.GroupVersionResource{Group: "cert-manager.io", Version: "v1", Resource: "certificaterequests"}
	GVRCertManagerOrder     = schema.GroupVersionResource{Group: "acme.cert-manager.io", Version: "v1", Resource: "orders"}
	GVRCertManagerChallenge = schema.GroupVersionResource{Group: "acme.cert-manager.io", Version: "v1", Resource: "challenges"}

	GVRIngress = schema.GroupVersionResource{Group: "networking.k8s.io", Version: "v1", Resource: "ingresses"}

	// GVRConfigMap is used by the K3sDatastore probe to check etcd snapshot
	// ConfigMaps written by k3s in kube-system.
	GVRConfigMap = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "configmaps"}

	// GVRService is the GVR for core/v1 Service objects.
	// Used by the TraefikRoutes probe to validate backend service references.
	GVRService = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "services"}

	// GVRServiceAccount is the GVR for core/v1 ServiceAccount objects.
	// Used by RBACDrift to read automountServiceAccountToken so SAs that
	// explicitly opt out of token mounting are NOT flagged as "unbound".
	GVRServiceAccount = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "serviceaccounts"}

	// GVRStorageClass is the GVR for storage.k8s.io/v1 StorageClass objects.
	// Used by K3sLocalPathStorage to detect the cluster default StorageClass.
	GVRStorageClass = schema.GroupVersionResource{Group: "storage.k8s.io", Version: "v1", Resource: "storageclasses"}

	// GVREndpoints is the GVR for core/v1 Endpoints objects.
	// Used by K3sDatastore to check apiserver reachability from within the cluster.
	GVREndpoints = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "endpoints"}

	// Traefik IngressRoute CRDs — present on any k3s cluster with the default
	// Traefik ingress controller. Probes auto-skip when the list call returns a
	// "no such resource" error (same pattern as Kong). The canonical group for
	// Traefik v3 (k3s 1.30+) is traefik.io; v2 used traefik.containo.us.
	// These GVRs reference the v3 group; the TraefikRoutes probe and
	// DiscoverTraefikRouteTargets fall back to traefik.containo.us on list error.
	GVRTraefikIngressRoute = schema.GroupVersionResource{
		Group: "traefik.io", Version: "v1alpha1", Resource: "ingressroutes",
	}
	GVRTraefikIngressRouteTCP = schema.GroupVersionResource{
		Group: "traefik.io", Version: "v1alpha1", Resource: "ingressroutetcps",
	}
	GVRTraefikMiddleware = schema.GroupVersionResource{
		Group: "traefik.io", Version: "v1alpha1", Resource: "middlewares",
	}
	GVRTraefikTLSStore = schema.GroupVersionResource{
		Group: "traefik.io", Version: "v1alpha1", Resource: "tlsstores",
	}
)
