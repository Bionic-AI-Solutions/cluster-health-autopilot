// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

// Package v1alpha1 contains the API Schema definitions for the
// srenix.ai/v1alpha1 API group.
//
// The CRDs in this group make Srenix an in-cluster operator: the operator
// reconciles a AgenticSRE resource into the existing
// watcher Deployment + diagnose CronJob + RBAC. The DriftReport CRD
// remains the read surface for active drift findings (its types live
// in pkg/snapshot for now; future PR may move them here).
//
// Phase 1 (this package) ships ONLY the type definitions — no
// controller-runtime dependency, no reconcile loop. The pure-function
// builders in internal/operator turn a AgenticSRESpec
// into concrete K8s manifests. Phase 2 will add the manager binary
// and the controller wiring.
//
// +kubebuilder:object:generate=true
// +groupName=srenix.ai
package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// GroupVersion is the API group and version for the Srenix operator types.
var GroupVersion = schema.GroupVersion{
	Group:   "srenix.ai",
	Version: "v1alpha1",
}

// SchemeBuilder collects the v1alpha1 types and exposes AddToScheme
// for the manager to wire in. Using runtime.SchemeBuilder directly
// (rather than controller-runtime's deprecated scheme.Builder) keeps
// the api package free of controller-runtime imports — recommended
// in the upstream guidance for v0.24+.
var SchemeBuilder = runtime.NewSchemeBuilder(addKnownTypes)

// AddToScheme adds the v1alpha1 types to the given Scheme. Called
// from cmd/srenix-operator/main.go so the manager can encode/decode
// AgenticSRE resources.
var AddToScheme = SchemeBuilder.AddToScheme

func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(GroupVersion,
		&AgenticSRE{},
		&AgenticSREList{},
		&Silence{},
		&SilenceList{},
	)
	// MUST register metav1 types (ListOptions, GetOptions, …) against
	// this GroupVersion. Without it, client-go can't serialize a List
	// request body for the CR's group: the manager cache start-up
	// crashes with `v1.ListOptions is not suitable for converting to
	// "<group>/<version>"`. Bundle-smoke caught this; unit tests with
	// the fake client don't, because the fake client bypasses the
	// wire conversion path entirely.
	metav1.AddToGroupVersion(scheme, GroupVersion)
	return nil
}

// Resource takes an unqualified resource name and returns the
// fully-qualified Group/Version/Resource. Used by the controller to
// build informer requests in Phase 2.
func Resource(resource string) schema.GroupResource {
	return GroupVersion.WithResource(resource).GroupResource()
}
