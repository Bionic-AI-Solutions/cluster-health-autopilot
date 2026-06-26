// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

package probe

import (
	"context"
	"sort"
	"strings"

	"github.com/srenix-ai/agentic-sre/internal/snapshot"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// ProbeCriticalAnnotation marks a workload as critical for the Services probe.
// Set on a Deployment / StatefulSet to opt-in to readiness probing without
// editing Srenix's compiled-in defaults.
//
// Example:
//
//	metadata:
//	  annotations:
//	    srenix.ai/probe-critical: "true"
//	    srenix.ai/probe-display: "Billing API"
const (
	ProbeCriticalAnnotation = "srenix.ai/probe-critical"
	ProbeDisplayAnnotation  = "srenix.ai/probe-display"
)

// TargetsFromEnv parses a semicolon-separated list of "namespace/selector"
// or "namespace/selector|Display Name" pairs. Empty or whitespace-only
// input returns nil — callers can then fall back to defaults or
// auto-discovery.
//
// Example:
//
//	SRENIX_CRITICAL_SERVICES="prod/app=billing|Billing API;prod/app=auth"
//
// Selectors must currently be single-equals labels (Sprint 2 scope);
// multi-label LabelSelector parsing is a follow-up.
func TargetsFromEnv(s string) []ServiceTarget {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	var out []ServiceTarget
	for _, raw := range strings.Split(s, ";") {
		entry := strings.TrimSpace(raw)
		if entry == "" {
			continue
		}
		display := ""
		if i := strings.Index(entry, "|"); i >= 0 {
			display = strings.TrimSpace(entry[i+1:])
			entry = strings.TrimSpace(entry[:i])
		}
		slash := strings.IndexByte(entry, '/')
		if slash <= 0 {
			continue
		}
		ns := strings.TrimSpace(entry[:slash])
		selector := strings.TrimSpace(entry[slash+1:])
		if ns == "" || selector == "" || !strings.Contains(selector, "=") {
			continue
		}
		if display == "" {
			display = ns + "/" + selector
		}
		out = append(out, ServiceTarget{
			Namespace: ns,
			Selector:  selector,
			Display:   display,
		})
	}
	return out
}

// TargetsFromAnnotation walks Deployments and StatefulSets in the snapshot
// and returns a ServiceTarget for any workload that carries the
// ProbeCriticalAnnotation set to "true". Display name comes from
// ProbeDisplayAnnotation if present, otherwise "<namespace>/<workload-name>".
//
// Selector priority: "app" > "app.kubernetes.io/name" > first label
// alphabetically. Workloads with no usable selector are silently skipped.
func TargetsFromAnnotation(ctx context.Context, src snapshot.Source) []ServiceTarget {
	var out []ServiceTarget
	for _, gvr := range []struct {
		gvr  interface{}
		kind string
	}{
		{gvr: snapshot.GVRDeployment, kind: "Deployment"},
		{gvr: snapshot.GVRStatefulSet, kind: "StatefulSet"},
	} {
		_ = gvr.kind
	}
	// Inline fetch — keeps the code straightforward.
	collect := func(list *unstructured.UnstructuredList) {
		if list == nil {
			return
		}
		for i := range list.Items {
			u := &list.Items[i]
			if !strings.EqualFold(u.GetAnnotations()[ProbeCriticalAnnotation], "true") {
				continue
			}
			t := deriveTarget(u)
			if t.Selector == "" {
				continue
			}
			out = append(out, t)
		}
	}
	deps, _ := src.List(ctx, snapshot.GVRDeployment, "")
	collect(deps)
	stss, _ := src.List(ctx, snapshot.GVRStatefulSet, "")
	collect(stss)

	sort.Slice(out, func(i, j int) bool {
		if out[i].Namespace != out[j].Namespace {
			return out[i].Namespace < out[j].Namespace
		}
		return out[i].Display < out[j].Display
	})
	return out
}

// deriveTarget builds a ServiceTarget from a Deployment/StatefulSet using
// the priority rule documented on TargetsFromAnnotation.
func deriveTarget(u *unstructured.Unstructured) ServiceTarget {
	display := u.GetAnnotations()[ProbeDisplayAnnotation]
	if display == "" {
		display = u.GetNamespace() + "/" + u.GetName()
	}
	ml := matchLabels(u)
	if v, ok := ml["app"]; ok && v != "" {
		return ServiceTarget{Namespace: u.GetNamespace(), Selector: "app=" + v, Display: display}
	}
	if v, ok := ml["app.kubernetes.io/name"]; ok && v != "" {
		return ServiceTarget{Namespace: u.GetNamespace(), Selector: "app.kubernetes.io/name=" + v, Display: display}
	}
	keys := make([]string, 0, len(ml))
	for k := range ml {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	if len(keys) > 0 {
		return ServiceTarget{Namespace: u.GetNamespace(), Selector: keys[0] + "=" + ml[keys[0]], Display: display}
	}
	return ServiceTarget{Namespace: u.GetNamespace(), Display: display}
}

func matchLabels(u *unstructured.Unstructured) map[string]string {
	sel, _, _ := unstructured.NestedMap(u.Object, "spec", "selector", "matchLabels")
	out := make(map[string]string, len(sel))
	for k, v := range sel {
		if s, ok := v.(string); ok {
			out[k] = s
		}
	}
	return out
}
