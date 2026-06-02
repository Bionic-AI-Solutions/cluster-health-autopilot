// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package operator

import (
	"testing"

	chav1alpha1 "github.com/Bionic-AI-Solutions/cluster-health-autopilot/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func sampleCR() *chav1alpha1.ClusterHealthAutopilot {
	return &chav1alpha1.ClusterHealthAutopilot{
		ObjectMeta: metav1.ObjectMeta{Name: "bionic", Namespace: "cha-system"},
		Spec: chav1alpha1.ClusterHealthAutopilotSpec{
			Image: chav1alpha1.ImageSpec{
				Repository: "docker4zerocool/cluster-health-autopilot",
				Tag:        "1.7.0",
			},
		},
	}
}

// --- ServiceAccount ---

func TestBuildServiceAccount_DefaultName(t *testing.T) {
	cr := sampleCR()
	sa := BuildServiceAccount(cr)
	if sa.Name != "bionic-sa" {
		t.Errorf("SA name=%q want bionic-sa", sa.Name)
	}
	if sa.Namespace != "cha-system" {
		t.Errorf("SA namespace=%q want cha-system", sa.Namespace)
	}
	if sa.Labels["app.kubernetes.io/managed-by"] != FieldManager {
		t.Errorf("managed-by label missing/wrong: %v", sa.Labels)
	}
}

func TestBuildServiceAccount_ExplicitNameWins(t *testing.T) {
	cr := sampleCR()
	cr.Spec.ServiceAccountName = "shared-sa"
	sa := BuildServiceAccount(cr)
	if sa.Name != "shared-sa" {
		t.Errorf("SA name=%q want shared-sa (operator must honor explicit override)", sa.Name)
	}
}

// --- Watcher Deployment ---

func TestBuildWatcherDeployment_DisabledReturnsNil(t *testing.T) {
	cr := sampleCR()
	cr.Spec.Watcher = &chav1alpha1.WatcherSpec{Enabled: false}
	if d := BuildWatcherDeployment(cr); d != nil {
		t.Errorf("disabled watcher should produce nil; got %v", d)
	}
}

func TestBuildWatcherDeployment_DefaultReplicasOne(t *testing.T) {
	cr := sampleCR()
	cr.Spec.Watcher = &chav1alpha1.WatcherSpec{Enabled: true}
	d := BuildWatcherDeployment(cr)
	if d == nil {
		t.Fatal("enabled watcher must produce a Deployment")
	}
	if d.Spec.Replicas == nil || *d.Spec.Replicas != 1 {
		t.Errorf("default replicas=%v want 1", d.Spec.Replicas)
	}
}

func TestBuildWatcherDeployment_HonorsExplicitReplicas(t *testing.T) {
	cr := sampleCR()
	cr.Spec.Watcher = &chav1alpha1.WatcherSpec{Enabled: true, Replicas: 3}
	d := BuildWatcherDeployment(cr)
	if d.Spec.Replicas == nil || *d.Spec.Replicas != 3 {
		t.Errorf("explicit replicas=3 not honored; got %v", d.Spec.Replicas)
	}
}

func TestBuildWatcherDeployment_DownwardAPIEnvVars(t *testing.T) {
	cr := sampleCR()
	cr.Spec.Watcher = &chav1alpha1.WatcherSpec{Enabled: true}
	d := BuildWatcherDeployment(cr)
	c := d.Spec.Template.Spec.Containers[0]
	want := map[string]string{
		"MY_POD_NAMESPACE": "metadata.namespace",
		"MY_POD_NAME":      "metadata.name",
	}
	for _, e := range c.Env {
		if path, ok := want[e.Name]; ok {
			if e.ValueFrom == nil || e.ValueFrom.FieldRef == nil ||
				e.ValueFrom.FieldRef.FieldPath != path {
				t.Errorf("env %s missing downward-API binding to %s", e.Name, path)
			}
			delete(want, e.Name)
		}
	}
	if len(want) != 0 {
		t.Errorf("missing env vars: %v", want)
	}
}

func TestBuildWatcherDeployment_DebounceAndResyncDefaults(t *testing.T) {
	cr := sampleCR()
	cr.Spec.Watcher = &chav1alpha1.WatcherSpec{Enabled: true}
	d := BuildWatcherDeployment(cr)
	args := d.Spec.Template.Spec.Containers[0].Args
	mustContain(t, args, "--debounce=10s")
	mustContain(t, args, "--resync-period=10m")
}

func TestBuildWatcherDeployment_DebounceAndResyncOverrides(t *testing.T) {
	cr := sampleCR()
	cr.Spec.Watcher = &chav1alpha1.WatcherSpec{
		Enabled:      true,
		Debounce:     "5s",
		ResyncPeriod: "30s",
	}
	d := BuildWatcherDeployment(cr)
	args := d.Spec.Template.Spec.Containers[0].Args
	mustContain(t, args, "--debounce=5s")
	mustContain(t, args, "--resync-period=30s")
}

func TestBuildWatcherDeployment_AlertingArgs(t *testing.T) {
	cr := sampleCR()
	cr.Spec.Watcher = &chav1alpha1.WatcherSpec{Enabled: true}
	cr.Spec.Alerting = &chav1alpha1.AlertingSpec{
		Alertmanager: &chav1alpha1.AlertmanagerSpec{
			URL:         "http://alertmanager.pg.svc:9093",
			ClusterName: "bionic-cluster",
		},
		Slack: &chav1alpha1.SlackSpec{
			Critical: &chav1alpha1.SlackChannelSpec{SecretName: "cha-critical"},
		},
	}
	d := BuildWatcherDeployment(cr)
	args := d.Spec.Template.Spec.Containers[0].Args
	mustContain(t, args, "--alertmanager-url=http://alertmanager.pg.svc:9093")
	mustContain(t, args, "--cluster-name=bionic-cluster")
	mustContain(t, args, "--slack-critical=$(SLACK_CRITICAL_URL)")
}

// v1.16.0 — when the CR sets both ai.approvalServerUrl and
// approval.signingKey.secretName, the watcher gets the new flags +
// signing-key mount so it can mint approve/deny URLs directly (rather
// than the pre-v1.16.0 architecture where URLs only existed in the
// separate aiwatch pod's stdout).
func TestBuildWatcherDeployment_ApprovalURLMintingWiring(t *testing.T) {
	cr := sampleCR()
	cr.Spec.Watcher = &chav1alpha1.WatcherSpec{Enabled: true}
	cr.Spec.AI = &chav1alpha1.AISpec{
		ApprovalServerURL: "https://cha-approve.example.com",
	}
	cr.Spec.Approval = &chav1alpha1.ApprovalSpec{
		SigningKey: &chav1alpha1.ApprovalSigningKeySpec{
			SecretName: "cha-approval-signing-key",
		},
	}
	d := BuildWatcherDeployment(cr)
	if d == nil {
		t.Fatal("watcher deploy must build")
	}
	args := d.Spec.Template.Spec.Containers[0].Args
	mustContain(t, args, "--approval-server-url=https://cha-approve.example.com")
	mustContain(t, args, "--signing-key-path=/etc/cha/keys/signing.key")
	// Volume + mount wired
	vols := d.Spec.Template.Spec.Volumes
	if len(vols) != 1 || vols[0].Name != "signing-key" || vols[0].Secret == nil ||
		vols[0].Secret.SecretName != "cha-approval-signing-key" {
		t.Errorf("signing-key volume not wired correctly: %+v", vols)
	}
	mounts := d.Spec.Template.Spec.Containers[0].VolumeMounts
	if len(mounts) != 1 || mounts[0].Name != "signing-key" || mounts[0].MountPath != "/etc/cha/keys" {
		t.Errorf("signing-key mount not wired correctly: %+v", mounts)
	}
}

// When only ai.approvalServerUrl is set but no signing key is
// configured, the watcher does NOT get the new flags (because URL
// minting requires the key). This guards against half-configured
// installs producing broken pods.
func TestBuildWatcherDeployment_NoApprovalSignerNoFlags(t *testing.T) {
	cr := sampleCR()
	cr.Spec.Watcher = &chav1alpha1.WatcherSpec{Enabled: true}
	cr.Spec.AI = &chav1alpha1.AISpec{
		ApprovalServerURL: "https://cha-approve.example.com",
	}
	// no cr.Spec.Approval
	d := BuildWatcherDeployment(cr)
	args := d.Spec.Template.Spec.Containers[0].Args
	for _, a := range args {
		if a == "--approval-server-url=https://cha-approve.example.com" {
			t.Errorf("watcher should NOT get --approval-server-url when no signing key is configured; args=%v", args)
		}
	}
	if len(d.Spec.Template.Spec.Volumes) != 0 {
		t.Errorf("watcher should have no volumes when signing key not configured; got %+v", d.Spec.Template.Spec.Volumes)
	}
}

// --- Diagnose CronJob ---

func TestBuildDiagnoseCronJob_DisabledReturnsNil(t *testing.T) {
	cr := sampleCR()
	cr.Spec.Diagnose = &chav1alpha1.DiagnoseSpec{Enabled: false}
	if c := BuildDiagnoseCronJob(cr); c != nil {
		t.Errorf("disabled diagnose should produce nil; got %v", c)
	}
}

func TestBuildDiagnoseCronJob_DefaultScheduleAndCaps(t *testing.T) {
	cr := sampleCR()
	cr.Spec.Diagnose = &chav1alpha1.DiagnoseSpec{Enabled: true}
	c := BuildDiagnoseCronJob(cr)
	if c == nil {
		t.Fatal("enabled diagnose must produce a CronJob")
	}
	if c.Spec.Schedule != "0 9 * * *" {
		t.Errorf("schedule=%q want '0 9 * * *'", c.Spec.Schedule)
	}
	if *c.Spec.JobTemplate.Spec.BackoffLimit != 1 {
		t.Errorf("backoffLimit=%d want 1", *c.Spec.JobTemplate.Spec.BackoffLimit)
	}
	if *c.Spec.JobTemplate.Spec.ActiveDeadlineSeconds != 120 {
		t.Errorf("activeDeadline=%d want 120", *c.Spec.JobTemplate.Spec.ActiveDeadlineSeconds)
	}
	if c.Spec.ConcurrencyPolicy != "Forbid" {
		t.Errorf("concurrencyPolicy=%q want Forbid", c.Spec.ConcurrencyPolicy)
	}
	args := c.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Args
	mustContain(t, args, "diagnose")
}

// --- Remediate CronJob ---

func TestBuildRemediateCronJob_DefaultDisabled(t *testing.T) {
	cr := sampleCR()
	if c := BuildRemediateCronJob(cr); c != nil {
		t.Errorf("nil remediate spec should produce nil; got %v", c)
	}
}

func TestBuildRemediateCronJob_DryRunFlag(t *testing.T) {
	cr := sampleCR()
	cr.Spec.Remediate = &chav1alpha1.RemediateSpec{Enabled: true, DryRun: true}
	c := BuildRemediateCronJob(cr)
	if c == nil {
		t.Fatal("enabled remediate must produce a CronJob")
	}
	args := c.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Args
	mustContain(t, args, "remediate")
	mustContain(t, args, "--dry-run=true")
}

func TestBuildRemediateCronJob_DefaultSchedule(t *testing.T) {
	cr := sampleCR()
	cr.Spec.Remediate = &chav1alpha1.RemediateSpec{Enabled: true}
	c := BuildRemediateCronJob(cr)
	if c.Spec.Schedule != "*/30 * * * *" {
		t.Errorf("schedule=%q want '*/30 * * * *'", c.Spec.Schedule)
	}
}

// --- Image policy + pull secrets ---

func TestPullPolicy_MutableTagDefaultsAlways(t *testing.T) {
	cr := sampleCR()
	cr.Spec.Image.Tag = "latest"
	cr.Spec.Watcher = &chav1alpha1.WatcherSpec{Enabled: true}
	d := BuildWatcherDeployment(cr)
	if got := d.Spec.Template.Spec.Containers[0].ImagePullPolicy; got != corev1.PullAlways {
		t.Errorf("latest tag pullPolicy=%v want Always", got)
	}
}

func TestPullPolicy_SemverTagDefaultsIfNotPresent(t *testing.T) {
	cr := sampleCR()
	cr.Spec.Watcher = &chav1alpha1.WatcherSpec{Enabled: true}
	d := BuildWatcherDeployment(cr)
	if got := d.Spec.Template.Spec.Containers[0].ImagePullPolicy; got != corev1.PullIfNotPresent {
		t.Errorf("semver tag pullPolicy=%v want IfNotPresent", got)
	}
}

func TestPullPolicy_ExplicitOverrideWins(t *testing.T) {
	cr := sampleCR()
	cr.Spec.Image.PullPolicy = "Never"
	cr.Spec.Watcher = &chav1alpha1.WatcherSpec{Enabled: true}
	d := BuildWatcherDeployment(cr)
	if got := d.Spec.Template.Spec.Containers[0].ImagePullPolicy; got != "Never" {
		t.Errorf("explicit pullPolicy=Never not honored; got %v", got)
	}
}

func TestPullSecrets_RoundTrip(t *testing.T) {
	cr := sampleCR()
	cr.Spec.Image.PullSecrets = []string{"ghcr-secret", "dockerhub-secret"}
	cr.Spec.Watcher = &chav1alpha1.WatcherSpec{Enabled: true}
	d := BuildWatcherDeployment(cr)
	refs := d.Spec.Template.Spec.ImagePullSecrets
	if len(refs) != 2 {
		t.Fatalf("expected 2 pull-secret refs; got %d", len(refs))
	}
	if refs[0].Name != "ghcr-secret" || refs[1].Name != "dockerhub-secret" {
		t.Errorf("pull-secret names not preserved: %v", refs)
	}
}

// --- Labels + names ---

func TestCommonLabels_AlwaysCarriesInstanceAndManagedBy(t *testing.T) {
	cr := sampleCR()
	labels := CommonLabels(cr, "watcher")
	if labels["app.kubernetes.io/instance"] != "bionic" {
		t.Errorf("instance label=%q want bionic", labels["app.kubernetes.io/instance"])
	}
	if labels["app.kubernetes.io/managed-by"] != "cha-operator" {
		t.Errorf("managed-by label=%q want cha-operator", labels["app.kubernetes.io/managed-by"])
	}
	if labels["cha.bionicaisolutions.com/role"] != "watcher" {
		t.Errorf("role label=%q want watcher", labels["cha.bionicaisolutions.com/role"])
	}
}

func TestNamesFor_DerivedFromCRName(t *testing.T) {
	cr := sampleCR()
	names := NamesFor(cr)
	wants := map[string]string{
		"sa":        "bionic-sa",
		"watcher":   "bionic-watcher",
		"diagnose":  "bionic-diagnose",
		"remediate": "bionic-remediate",
	}
	got := map[string]string{
		"sa":        names.ServiceAccount,
		"watcher":   names.Watcher,
		"diagnose":  names.Diagnose,
		"remediate": names.Remediate,
	}
	for k, w := range wants {
		if got[k] != w {
			t.Errorf("name[%s]=%q want %q", k, got[k], w)
		}
	}
}

func mustContain(t *testing.T, args []string, want string) {
	t.Helper()
	for _, a := range args {
		if a == want {
			return
		}
	}
	t.Errorf("args missing %q; have: %v", want, args)
}
