// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

package diagnose

import (
	"context"
	"fmt"

	"github.com/srenix-ai/agentic-sre/internal/snapshot"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// UnprovisionedSecret walks every Deployment, StatefulSet, and CronJob and
// checks whether each Secret referenced via envFrom.secretRef or
// spec.template.spec.volumes[].secret is either present in the cluster OR
// has an ExternalSecret configured to provision it.
//
// When a Secret is missing AND no ExternalSecret targets it, the diagnostic
// includes a suggested Vault path derived from the namespace name following
// the t6 canonical hierarchy (secret/t6-apps/<namespace>/config).
//
// This is the root-cause complement to ProactiveSecretKeyCheck: PSKC says
// "this reference will crash", UnprovisionedSecret says "here's how to wire
// the ESO so the Secret gets created in the first place".
//
// In snapshot mode without Secrets captured, the analyzer acts purely on ESO
// coverage: any envFrom or volume-secret reference with no backing
// ExternalSecret is reported regardless of whether the Secret exists on-cluster.
type UnprovisionedSecret struct{}

// Name returns the analyzer's identifier.
func (UnprovisionedSecret) Name() string { return "UnprovisionedSecret" }

// Run produces diagnostics for workload → Secret references that have no
// ExternalSecret provisioning them.
func (UnprovisionedSecret) Run(ctx context.Context, src snapshot.Source) []Diagnostic {
	// Build existing-secret set (may be empty in snapshot mode if secrets were excluded).
	existing := make(map[string]struct{})
	if secrets, err := src.List(ctx, snapshot.GVRSecret, ""); err == nil && secrets != nil {
		for _, s := range secrets.Items {
			existing[s.GetNamespace()+"/"+s.GetName()] = struct{}{}
		}
	}

	// Build provisioned map: (ns/secret-name) → ESO name.
	// If ESO CRD not installed, provisioned is empty — detection still works.
	provisioned := make(map[string]string)
	if esos, err := src.List(ctx, snapshot.GVRExtSecret, ""); err == nil && esos != nil {
		for _, eso := range esos.Items {
			targetName, _, _ := unstructured.NestedString(eso.Object, "spec", "target", "name")
			if targetName == "" {
				targetName = eso.GetName()
			}
			provisioned[eso.GetNamespace()+"/"+targetName] = eso.GetName()
		}
	}

	seen := make(map[string]struct{})
	var out []Diagnostic

	emit := func(workloadKind, ns, workloadName, secretName string) {
		key := ns + "/" + secretName
		if _, ok := existing[key]; ok {
			return // Secret is present — not our concern.
		}
		if _, ok := provisioned[key]; ok {
			return // ESO will provision it; FailingExternalSecrets covers sync failures.
		}
		dedupe := "unprovisioned/" + key
		if _, dup := seen[dedupe]; dup {
			return
		}
		seen[dedupe] = struct{}{}
		out = append(out, Diagnostic{
			Subject: dedupe,
			Message: fmt.Sprintf(
				"Secret `%s` referenced by %s/%s has no ExternalSecret provisioning it. "+
					"Create an ExternalSecret with spec.target.name=%s pointing to "+
					"Vault path `secret/t6-apps/%s/config`.",
				key, workloadKind, workloadName, secretName, ns,
			),
		})
	}

	scanEnvFrom := func(workloadKind, ns, name string, containers []any) {
		for _, c := range containers {
			cm, ok := c.(map[string]any)
			if !ok {
				continue
			}
			envFroms, _, _ := unstructured.NestedSlice(cm, "envFrom")
			for _, ef := range envFroms {
				efm, ok := ef.(map[string]any)
				if !ok {
					continue
				}
				sref, _ := efm["secretRef"].(map[string]any)
				if sref == nil {
					continue
				}
				secretName, _ := sref["name"].(string)
				optional, _ := sref["optional"].(bool)
				if secretName != "" && !optional {
					emit(workloadKind, ns, name, secretName)
				}
			}
		}
	}

	scanVolumes := func(workloadKind, ns, name string, volumes []any) {
		for _, v := range volumes {
			vm, ok := v.(map[string]any)
			if !ok {
				continue
			}
			sv, _ := vm["secret"].(map[string]any)
			if sv == nil {
				continue
			}
			secretName, _ := sv["secretName"].(string)
			if secretName != "" {
				emit(workloadKind, ns, name, secretName)
			}
		}
	}

	scanWorkload := func(kind, ns, name string, obj unstructured.Unstructured, basePath []string) {
		containers, _, _ := unstructured.NestedSlice(obj.Object, append(basePath, "containers")...)
		initContainers, _, _ := unstructured.NestedSlice(obj.Object, append(basePath, "initContainers")...)
		scanEnvFrom(kind, ns, name, append(containers, initContainers...))

		volumes, _, _ := unstructured.NestedSlice(obj.Object, append(basePath, "volumes")...)
		scanVolumes(kind, ns, name, volumes)
	}

	podSpecPath := []string{"spec", "template", "spec"}
	cronJobPath := []string{"spec", "jobTemplate", "spec", "template", "spec"}

	if deploys, _ := src.List(ctx, snapshot.GVRDeployment, ""); deploys != nil {
		for i := range deploys.Items {
			obj := deploys.Items[i]
			scanWorkload("Deployment", obj.GetNamespace(), obj.GetName(), obj, podSpecPath)
		}
	}
	if statefulsets, _ := src.List(ctx, snapshot.GVRStatefulSet, ""); statefulsets != nil {
		for i := range statefulsets.Items {
			obj := statefulsets.Items[i]
			scanWorkload("StatefulSet", obj.GetNamespace(), obj.GetName(), obj, podSpecPath)
		}
	}
	if cronjobs, _ := src.List(ctx, snapshot.GVRCronJob, ""); cronjobs != nil {
		for i := range cronjobs.Items {
			obj := cronjobs.Items[i]
			scanWorkload("CronJob", obj.GetNamespace(), obj.GetName(), obj, cronJobPath)
		}
	}
	return out
}
