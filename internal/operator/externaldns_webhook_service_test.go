// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

package operator

import (
	"context"
	"testing"

	chav1alpha1 "github.com/srenix-ai/agentic-sre/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
)

// P1.5 — spec.externalDNS.cloudflare.* and
// spec.watcher.triggers.webhook.{serviceEnabled,servicePort} were
// accepted by the CRD schema but consumed nowhere in the operator:
// catalog.go reads SRENIX_CLOUDFLARE_TOKEN at registration time but
// nothing supplied it on operator-managed installs, and the webhook
// receiver was reachable only by pod IP because no Service and no
// containerPort were ever built. These tests pin the wiring.

// --- spec.externalDNS → SRENIX_CLOUDFLARE_TOKEN env ---

// externalDNSCR returns a watcher-enabled CR with Cloudflare DNS
// drift enabled and a token Secret reference.
func externalDNSCR() *chav1alpha1.AgenticSRE {
	cr := sampleCR()
	cr.Spec.Watcher = &chav1alpha1.WatcherSpec{Enabled: true}
	cr.Spec.ExternalDNS = &chav1alpha1.ExternalDNSSpec{
		Cloudflare: &chav1alpha1.CloudflareSpec{
			Enabled: true,
			APITokenSecretRef: &chav1alpha1.CloudflareSecretRef{
				Name: "cloudflare-token-secret",
			},
		},
	}
	return cr
}

func findEnv(env []corev1.EnvVar, name string) *corev1.EnvVar {
	for i := range env {
		if env[i].Name == name {
			return &env[i]
		}
	}
	return nil
}

func TestBuildWatcherDeployment_ExternalDNSCloudflareEnv_DefaultKey(t *testing.T) {
	cr := externalDNSCR()
	d := BuildWatcherDeployment(cr)
	e := findEnv(d.Spec.Template.Spec.Containers[0].Env, "SRENIX_CLOUDFLARE_TOKEN")
	if e == nil {
		t.Fatalf("externalDNS.cloudflare.enabled must inject SRENIX_CLOUDFLARE_TOKEN; env=%+v",
			d.Spec.Template.Spec.Containers[0].Env)
	}
	if e.Value != "" {
		t.Errorf("SRENIX_CLOUDFLARE_TOKEN must NEVER carry a literal value; got %q", e.Value)
	}
	if e.ValueFrom == nil || e.ValueFrom.SecretKeyRef == nil {
		t.Fatalf("SRENIX_CLOUDFLARE_TOKEN must use secretKeyRef; got %+v", e)
	}
	if got := e.ValueFrom.SecretKeyRef.Name; got != "cloudflare-token-secret" {
		t.Errorf("secretKeyRef.Name=%q want cloudflare-token-secret", got)
	}
	// CloudflareSecretRef.Key documents `Defaults to "token"`.
	if got := e.ValueFrom.SecretKeyRef.Key; got != "token" {
		t.Errorf("secretKeyRef.Key=%q want default token", got)
	}
}

func TestBuildWatcherDeployment_ExternalDNSCloudflareEnv_ExplicitKey(t *testing.T) {
	cr := externalDNSCR()
	cr.Spec.ExternalDNS.Cloudflare.APITokenSecretRef.Key = "cf-api-token"
	d := BuildWatcherDeployment(cr)
	e := findEnv(d.Spec.Template.Spec.Containers[0].Env, "SRENIX_CLOUDFLARE_TOKEN")
	if e == nil {
		t.Fatal("expected SRENIX_CLOUDFLARE_TOKEN env var")
	}
	if got := e.ValueFrom.SecretKeyRef.Key; got != "cf-api-token" {
		t.Errorf("secretKeyRef.Key=%q want cf-api-token", got)
	}
}

func TestBuildWatcherDeployment_ExternalDNSDisabled_NoEnv(t *testing.T) {
	// Enabled=false with a ref set — the CRD documents Enabled as the
	// master switch, so no env must leak.
	cr := externalDNSCR()
	cr.Spec.ExternalDNS.Cloudflare.Enabled = false
	d := BuildWatcherDeployment(cr)
	if e := findEnv(d.Spec.Template.Spec.Containers[0].Env, "SRENIX_CLOUDFLARE_TOKEN"); e != nil {
		t.Errorf("cloudflare.enabled=false must not inject SRENIX_CLOUDFLARE_TOKEN; got %+v", e)
	}

	// Nil stanza — legacy CRs stay byte-identical.
	cr2 := sampleCR()
	cr2.Spec.Watcher = &chav1alpha1.WatcherSpec{Enabled: true}
	d2 := BuildWatcherDeployment(cr2)
	if e := findEnv(d2.Spec.Template.Spec.Containers[0].Env, "SRENIX_CLOUDFLARE_TOKEN"); e != nil {
		t.Errorf("nil externalDNS must not inject SRENIX_CLOUDFLARE_TOKEN; got %+v", e)
	}
}

func TestBuildWatcherDeployment_ExternalDNSEnabledWithoutRef_NoEnv(t *testing.T) {
	// Enabled but no APITokenSecretRef — there is nothing to
	// reference; an empty-name secretKeyRef would fail pod admission.
	cr := externalDNSCR()
	cr.Spec.ExternalDNS.Cloudflare.APITokenSecretRef = nil
	d := BuildWatcherDeployment(cr)
	if e := findEnv(d.Spec.Template.Spec.Containers[0].Env, "SRENIX_CLOUDFLARE_TOKEN"); e != nil {
		t.Errorf("enabled without apiTokenSecretRef must not inject env; got %+v", e)
	}
}

// --- webhook receiver containerPort + ClusterIP Service ---

// webhookCR returns a watcher-enabled CR with the class-E webhook
// receiver listening and its Service rendered.
func webhookCR() *chav1alpha1.AgenticSRE {
	cr := sampleCR()
	cr.Spec.Watcher = &chav1alpha1.WatcherSpec{
		Enabled: true,
		Triggers: &chav1alpha1.WatcherTriggersSpec{
			Webhook: &chav1alpha1.WatcherWebhookTriggerSpec{
				Listen:         ":8090",
				ServiceEnabled: true,
			},
		},
	}
	return cr
}

// webhookPort returns the named "webhook" containerPort, or nil. The
// always-on "health" port (P1.9) is present on every watcher container,
// so these tests assert the webhook port specifically rather than count.
func webhookPort(ports []corev1.ContainerPort) *corev1.ContainerPort {
	for i := range ports {
		if ports[i].Name == "webhook" {
			return &ports[i]
		}
	}
	return nil
}

func TestBuildWatcherDeployment_WebhookListen_ContainerPort(t *testing.T) {
	// The chart declares the named `webhook` containerPort whenever
	// listen is set (watcher-deployment.yaml), so the Service's
	// targetPort:webhook resolves. Mirror that.
	cr := webhookCR()
	d := BuildWatcherDeployment(cr)
	wp := webhookPort(d.Spec.Template.Spec.Containers[0].Ports)
	if wp == nil {
		t.Fatalf("webhook.listen set: want a named webhook containerPort; got %+v", d.Spec.Template.Spec.Containers[0].Ports)
	}
	if wp.ContainerPort != 8090 {
		t.Errorf("containerPort=%d want default 8090", wp.ContainerPort)
	}
	if wp.Protocol != corev1.ProtocolTCP {
		t.Errorf("protocol=%q want TCP", wp.Protocol)
	}
}

func TestBuildWatcherDeployment_WebhookExplicitServicePort_ContainerPort(t *testing.T) {
	cr := webhookCR()
	cr.Spec.Watcher.Triggers.Webhook.ServicePort = 9099
	d := BuildWatcherDeployment(cr)
	wp := webhookPort(d.Spec.Template.Spec.Containers[0].Ports)
	if wp == nil || wp.ContainerPort != 9099 {
		t.Errorf("servicePort=9099 must drive the webhook containerPort (chart parity); got %+v", d.Spec.Template.Spec.Containers[0].Ports)
	}
}

func TestBuildWatcherDeployment_NoWebhookListen_NoWebhookPort(t *testing.T) {
	cr := sampleCR()
	cr.Spec.Watcher = &chav1alpha1.WatcherSpec{Enabled: true}
	d := BuildWatcherDeployment(cr)
	// Health port is always present (P1.9); only the webhook port must be absent.
	if wp := webhookPort(d.Spec.Template.Spec.Containers[0].Ports); wp != nil {
		t.Errorf("no webhook receiver: want no webhook containerPort; got %+v", wp)
	}
}

func TestBuildWatcherWebhookService_BasicShape(t *testing.T) {
	cr := webhookCR()
	svc := BuildWatcherWebhookService(cr)
	if svc == nil {
		t.Fatal("serviceEnabled=true must produce a Service")
	}
	if svc.Name != "bionic-webhook" {
		t.Errorf("name=%q want bionic-webhook (chart: <fullname>-webhook)", svc.Name)
	}
	if svc.Namespace != "srenix-system" {
		t.Errorf("namespace=%q want srenix-system", svc.Namespace)
	}
	if svc.Spec.Type != corev1.ServiceTypeClusterIP {
		t.Errorf("type=%q want ClusterIP", svc.Spec.Type)
	}
	// Must select the watcher pods (the chart Service selects on the
	// watcher Deployment's selector labels).
	watcherLabels := CommonLabels(cr, "watcher")
	for k, v := range watcherLabels {
		if svc.Spec.Selector[k] != v {
			t.Errorf("selector[%s]=%q want %q (must match watcher pod labels)",
				k, svc.Spec.Selector[k], v)
		}
	}
	if svc.Labels["srenix.ai/role"] != "watcher-webhook" {
		t.Errorf("role label=%q want watcher-webhook", svc.Labels["srenix.ai/role"])
	}
	if len(svc.Spec.Ports) != 1 {
		t.Fatalf("want exactly 1 port; got %+v", svc.Spec.Ports)
	}
	p := svc.Spec.Ports[0]
	if p.Name != "webhook" {
		t.Errorf("port name=%q want webhook", p.Name)
	}
	if p.Port != 8090 {
		t.Errorf("port=%d want default 8090", p.Port)
	}
	if p.TargetPort.String() != "webhook" {
		t.Errorf("targetPort=%q want named port webhook (chart parity)", p.TargetPort.String())
	}
	if p.Protocol != corev1.ProtocolTCP {
		t.Errorf("protocol=%q want TCP", p.Protocol)
	}
}

func TestBuildWatcherWebhookService_ExplicitPort(t *testing.T) {
	cr := webhookCR()
	cr.Spec.Watcher.Triggers.Webhook.ServicePort = 9099
	svc := BuildWatcherWebhookService(cr)
	if svc == nil {
		t.Fatal("expected a Service")
	}
	if got := svc.Spec.Ports[0].Port; got != 9099 {
		t.Errorf("port=%d want 9099", got)
	}
}

func TestBuildWatcherWebhookService_DisabledReturnsNil(t *testing.T) {
	// serviceEnabled=false (default) — chart renders nothing.
	cr := webhookCR()
	cr.Spec.Watcher.Triggers.Webhook.ServiceEnabled = false
	if svc := BuildWatcherWebhookService(cr); svc != nil {
		t.Errorf("serviceEnabled=false must produce nil; got %+v", svc)
	}

	// serviceEnabled=true but no listen — the chart's template gates
	// on BOTH (`and ...webhook.listen ...service.enabled`): a Service
	// with no receiver behind it would blackhole.
	cr2 := webhookCR()
	cr2.Spec.Watcher.Triggers.Webhook.Listen = ""
	if svc := BuildWatcherWebhookService(cr2); svc != nil {
		t.Errorf("listen unset must produce nil even with serviceEnabled=true; got %+v", svc)
	}

	// Watcher disabled — no pods to select.
	cr3 := webhookCR()
	cr3.Spec.Watcher.Enabled = false
	if svc := BuildWatcherWebhookService(cr3); svc != nil {
		t.Errorf("watcher disabled must produce nil; got %+v", svc)
	}

	// No triggers stanza at all.
	cr4 := sampleCR()
	cr4.Spec.Watcher = &chav1alpha1.WatcherSpec{Enabled: true}
	if svc := BuildWatcherWebhookService(cr4); svc != nil {
		t.Errorf("nil triggers must produce nil; got %+v", svc)
	}
}

// --- Reconcile-level wiring ---

func TestReconcile_WebhookServiceEnabled_CreatesService(t *testing.T) {
	cr := webhookCR()
	r, c := newReconciler(t, cr)
	reconcileOnce(t, r, cr)

	var svc corev1.Service
	if err := c.Get(context.Background(),
		types.NamespacedName{Namespace: "srenix-system", Name: "bionic-webhook"},
		&svc); err != nil {
		t.Fatalf("webhook Service not created: %v", err)
	}
	if len(svc.Spec.Ports) != 1 || svc.Spec.Ports[0].Port != 8090 {
		t.Errorf("Service ports=%+v want single port 8090", svc.Spec.Ports)
	}
	// Owner-ref'd to the CR so deletion cascades.
	found := false
	for _, ref := range svc.OwnerReferences {
		if ref.Kind == "AgenticSRE" && ref.Name == "bionic" {
			found = true
		}
	}
	if !found {
		t.Errorf("webhook Service missing CR ownerRef; got %+v", svc.OwnerReferences)
	}
}

func TestReconcile_WebhookServiceDisabled_NoService(t *testing.T) {
	cr := webhookCR()
	cr.Spec.Watcher.Triggers.Webhook.ServiceEnabled = false
	r, c := newReconciler(t, cr)
	reconcileOnce(t, r, cr)

	var svc corev1.Service
	err := c.Get(context.Background(),
		types.NamespacedName{Namespace: "srenix-system", Name: "bionic-webhook"},
		&svc)
	if !apierrors.IsNotFound(err) {
		t.Errorf("expected NotFound for serviceEnabled=false; got err=%v", err)
	}
}

func TestReconcile_WebhookServiceFlipOff_DeletesService(t *testing.T) {
	cr := webhookCR()
	r, c := newReconciler(t, cr)
	reconcileOnce(t, r, cr)
	reconcileOnce(t, r, cr)

	// Flip serviceEnabled off.
	var stored chav1alpha1.AgenticSRE
	if err := c.Get(context.Background(),
		types.NamespacedName{Namespace: "srenix-system", Name: "bionic"},
		&stored); err != nil {
		t.Fatalf("get cr: %v", err)
	}
	stored.Spec.Watcher.Triggers.Webhook.ServiceEnabled = false
	stored.Generation = 2
	if err := c.Update(context.Background(), &stored); err != nil {
		t.Fatalf("update cr: %v", err)
	}
	reconcileOnce(t, r, &stored)

	var svc corev1.Service
	err := c.Get(context.Background(),
		types.NamespacedName{Namespace: "srenix-system", Name: "bionic-webhook"},
		&svc)
	if !apierrors.IsNotFound(err) {
		t.Errorf("webhook Service not deleted after serviceEnabled flip-off; got err=%v", err)
	}
}

func TestReconcile_ExternalDNSEnabled_WatcherDeploymentHasEnv(t *testing.T) {
	cr := externalDNSCR()
	r, c := newReconciler(t, cr)
	reconcileOnce(t, r, cr)

	var dep appsv1.Deployment
	if err := c.Get(context.Background(),
		types.NamespacedName{Namespace: "srenix-system", Name: "bionic-watcher"},
		&dep); err != nil {
		t.Fatalf("get watcher deployment: %v", err)
	}
	e := findEnv(dep.Spec.Template.Spec.Containers[0].Env, "SRENIX_CLOUDFLARE_TOKEN")
	if e == nil {
		t.Fatal("reconciled watcher Deployment missing SRENIX_CLOUDFLARE_TOKEN env")
	}
	if e.ValueFrom == nil || e.ValueFrom.SecretKeyRef == nil ||
		e.ValueFrom.SecretKeyRef.Name != "cloudflare-token-secret" {
		t.Errorf("SRENIX_CLOUDFLARE_TOKEN must be a secretKeyRef to cloudflare-token-secret; got %+v", e)
	}
}
