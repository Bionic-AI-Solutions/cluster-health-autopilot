// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

package ai

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestActionKindIsValid(t *testing.T) {
	cases := map[ActionKind]bool{
		ActionDeletePod:            true,
		ActionDeleteJob:            true,
		ActionPatchDeployment:      true,
		ActionDeleteCertRequest:    true,
		ActionDeleteACMEOrder:      true,
		ActionKind(""):             false,
		ActionKind("DeleteSecret"): false,
		ActionKind("ExecPod"):      false,
		ActionKind("CreateBucket"): false,
	}
	for k, want := range cases {
		if got := k.IsValid(); got != want {
			t.Errorf("ActionKind(%q).IsValid() = %v; want %v", k, got, want)
		}
	}
}

func TestTierAllowsProposals(t *testing.T) {
	cases := map[Tier]bool{
		TierOff: false,
		TierT0:  false,
		TierT1:  true,
		TierT2:  true,
		TierT3:  true,
	}
	for k, want := range cases {
		if got := k.AllowsProposals(); got != want {
			t.Errorf("Tier(%q).AllowsProposals() = %v; want %v", k, got, want)
		}
	}
}

func newValidProposal() AIProposedAction {
	return AIProposedAction{
		ActionID:          "act-test-1",
		Tier:              TierT1,
		ActionKind:        ActionDeletePod,
		Target:            ObjectRef{Kind: "Pod", Namespace: "default", Name: "demo-abc"},
		Rationale:         "Pod is in terminal Error state; deletion will not cascade.",
		Rollback:          RollbackInfo{Description: "Re-run the Job; it will spawn a new pod."},
		DiagnosticSubject: "Pod/default/demo-abc",
		CreatedAt:         time.Now(),
		ExpiresAt:         time.Now().Add(DefaultProposalTTL),
	}
}

func TestProposalValidate_HappyPath(t *testing.T) {
	p := newValidProposal()
	if err := p.Validate(); err != nil {
		t.Fatalf("valid proposal rejected: %v", err)
	}
}

func TestProposalValidate_BadActionKind(t *testing.T) {
	p := newValidProposal()
	p.ActionKind = ActionKind("ExecPod") // not in whitelist
	if err := p.Validate(); !errors.Is(err, ErrInvalidActionKind) {
		t.Errorf("got %v; want ErrInvalidActionKind", err)
	}
}

func TestProposalValidate_ProtectedNamespace(t *testing.T) {
	for ns := range ProtectedNamespaces {
		p := newValidProposal()
		p.Target.Namespace = ns
		if err := p.Validate(); !errors.Is(err, ErrProtectedNamespace) {
			t.Errorf("ns=%q: got %v; want ErrProtectedNamespace", ns, err)
		}
	}
}

func TestProposalValidate_EmptyTarget(t *testing.T) {
	for _, mutate := range []func(*ObjectRef){
		func(o *ObjectRef) { o.Kind = "" },
		func(o *ObjectRef) { o.Namespace = "" },
		func(o *ObjectRef) { o.Name = "" },
	} {
		p := newValidProposal()
		mutate(&p.Target)
		if err := p.Validate(); !errors.Is(err, ErrEmptyTarget) {
			t.Errorf("got %v; want ErrEmptyTarget", err)
		}
	}
}

func TestProposalValidate_MissingRollback(t *testing.T) {
	p := newValidProposal()
	p.Rollback.Description = ""
	if err := p.Validate(); !errors.Is(err, ErrMissingRollback) {
		t.Errorf("got %v; want ErrMissingRollback", err)
	}
}

func TestProposalValidate_PatchPayloadRequiresPatchAction(t *testing.T) {
	p := newValidProposal()
	p.ActionKind = ActionDeletePod // not a patch action
	p.PatchPayload = []byte(`{"spec":{"replicas":3}}`)
	if err := p.Validate(); !errors.Is(err, ErrInvalidActionKind) {
		t.Errorf("got %v; want ErrInvalidActionKind for non-patch action with patch payload", err)
	}
}

func TestProposalValidate_TierMustAllowProposals(t *testing.T) {
	for _, tier := range []Tier{TierOff, TierT0, Tier("nope")} {
		p := newValidProposal()
		p.Tier = tier
		if err := p.Validate(); !errors.Is(err, ErrInvalidTier) {
			t.Errorf("tier=%q: got %v; want ErrInvalidTier", tier, err)
		}
	}
}

func TestProposalValidate_ExpiryWindow(t *testing.T) {
	p := newValidProposal()
	p.ExpiresAt = p.CreatedAt.Add(-1 * time.Minute) // expires before created
	if err := p.Validate(); !errors.Is(err, ErrTokenExpired) {
		t.Errorf("got %v; want ErrTokenExpired", err)
	}
}

// newValidManifestProposal returns a fully-valid ApplyManifest proposal with a
// safe-apply manifest that passes ValidateManifest.
func newValidManifestProposal(t *testing.T) AIProposedAction {
	t.Helper()
	p := newValidProposal()
	p.ActionID = "act-manifest-1"
	p.ActionKind = ActionApplyManifest
	p.Target = ObjectRef{Kind: "NetworkPolicy", Namespace: "app", Name: "srenix-proposed-allow-intracluster"}
	p.ManifestYAML = []byte(validNetworkPolicy)
	// Sanity: the fixture itself must be valid under the strict creation check.
	if err := p.Validate(); err != nil {
		t.Fatalf("manifest fixture not valid under Validate(): %v", err)
	}
	return p
}

// TestValidateForExecution_EmptyRollback pins the exact failing case: an
// otherwise-valid proposal whose Rollback.Description is empty (as it always is
// once reconstructed from a signed approval token) must PASS ValidateForExecution
// while it FAILS Validate.
func TestValidateForExecution_EmptyRollback(t *testing.T) {
	p := newValidProposal()
	p.Rollback.Description = ""
	if err := p.Validate(); !errors.Is(err, ErrMissingRollback) {
		t.Fatalf("Validate(): got %v; want ErrMissingRollback (regression pin)", err)
	}
	if err := p.ValidateForExecution(); err != nil {
		t.Errorf("ValidateForExecution() rejected empty-rollback proposal: %v", err)
	}
}

// TestValidateForExecution_AllExecutorKinds asserts that every action kind that
// reaches the executor validates for execution with an empty rollback.
func TestValidateForExecution_AllExecutorKinds(t *testing.T) {
	deleteKinds := []ActionKind{
		ActionDeletePod,
		ActionDeleteJob,
		ActionDeleteCertRequest,
		ActionDeleteACMEOrder,
	}
	for _, k := range deleteKinds {
		p := newValidProposal()
		p.ActionKind = k
		p.Rollback.Description = "" // token omits it
		if err := p.ValidateForExecution(); err != nil {
			t.Errorf("kind=%q: ValidateForExecution() = %v; want nil", k, err)
		}
	}

	// PatchDeployment with a payload.
	patch := newValidProposal()
	patch.ActionKind = ActionPatchDeployment
	patch.Target = ObjectRef{Kind: "Deployment", Namespace: "default", Name: "demo"}
	patch.PatchPayload = []byte(`{"spec":{"replicas":3}}`)
	patch.Rollback.Description = ""
	if err := patch.ValidateForExecution(); err != nil {
		t.Errorf("PatchDeployment: ValidateForExecution() = %v; want nil", err)
	}

	// ApplyManifest with a valid manifest.
	mani := newValidManifestProposal(t)
	mani.Rollback.Description = ""
	if err := mani.ValidateForExecution(); err != nil {
		t.Errorf("ApplyManifest: ValidateForExecution() = %v; want nil", err)
	}
}

// TestValidateForExecution_StillCatchesUnsafe asserts ValidateForExecution
// drops ONLY the rollback check — every other invariant Validate enforces is
// still enforced.
func TestValidateForExecution_StillCatchesUnsafe(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*AIProposedAction)
		want   error
	}{
		{
			name:   "invalid action kind",
			mutate: func(p *AIProposedAction) { p.ActionKind = ActionKind("ExecPod") },
			want:   ErrInvalidActionKind,
		},
		{
			name:   "empty action kind",
			mutate: func(p *AIProposedAction) { p.ActionKind = ActionKind("") },
			want:   ErrInvalidActionKind,
		},
		{
			name:   "missing target name",
			mutate: func(p *AIProposedAction) { p.Target.Name = "" },
			want:   ErrEmptyTarget,
		},
		{
			name:   "protected namespace",
			mutate: func(p *AIProposedAction) { p.Target.Namespace = "kube-system" },
			want:   ErrProtectedNamespace,
		},
		{
			name: "patch payload on non-patch kind",
			mutate: func(p *AIProposedAction) {
				p.ActionKind = ActionDeletePod
				p.PatchPayload = []byte(`{"spec":{"replicas":3}}`)
			},
			want: ErrInvalidActionKind,
		},
		{
			name: "invalid manifest for ApplyManifest",
			mutate: func(p *AIProposedAction) {
				p.ActionKind = ActionApplyManifest
				p.ManifestYAML = []byte("this is not: [valid yaml")
			},
			want: nil, // ValidateManifest error class; checked via non-nil below
		},
		{
			name:   "expired window",
			mutate: func(p *AIProposedAction) { p.ExpiresAt = p.CreatedAt.Add(-time.Minute) },
			want:   ErrTokenExpired,
		},
		{
			name:   "tier disallows proposals",
			mutate: func(p *AIProposedAction) { p.Tier = TierT0 },
			want:   ErrInvalidTier,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := newValidProposal()
			p.Rollback.Description = "" // prove rollback is NOT what trips it
			tc.mutate(&p)
			err := p.ValidateForExecution()
			if err == nil {
				t.Fatalf("ValidateForExecution() = nil; want error")
			}
			if tc.want != nil && !errors.Is(err, tc.want) {
				t.Errorf("got %v; want %v", err, tc.want)
			}
		})
	}
}

// TestValidate_ExecutionParity asserts the two checks agree when rollback is
// present (both nil) and diverge only on the rollback requirement.
func TestValidate_ExecutionParity(t *testing.T) {
	// Non-empty rollback: both agree (nil).
	p := newValidProposal()
	if err := p.Validate(); err != nil {
		t.Fatalf("Validate() with rollback: %v", err)
	}
	if err := p.ValidateForExecution(); err != nil {
		t.Fatalf("ValidateForExecution() with rollback: %v", err)
	}

	// Empty rollback: they diverge.
	p.Rollback.Description = ""
	if err := p.Validate(); !errors.Is(err, ErrMissingRollback) {
		t.Errorf("Validate() empty rollback: got %v; want ErrMissingRollback", err)
	}
	if err := p.ValidateForExecution(); err != nil {
		t.Errorf("ValidateForExecution() empty rollback: got %v; want nil", err)
	}
}

func TestEnrichedDiagnosticValidate_LengthBound(t *testing.T) {
	e := EnrichedDiagnostic{Enrichment: strings.Repeat("x", MaxEnrichmentChars+1)}
	if err := e.Validate(); !errors.Is(err, ErrEnrichmentTooLong) {
		t.Errorf("got %v; want ErrEnrichmentTooLong", err)
	}
	e.Enrichment = strings.Repeat("x", MaxEnrichmentChars)
	if err := e.Validate(); err != nil {
		t.Errorf("max-length enrichment rejected: %v", err)
	}
}

func TestEnrichedDiagnosticValidate_RelatedSignalsTruncate(t *testing.T) {
	signals := []string{"a", "b", "c", "d", "e", "f", "g"}
	e := EnrichedDiagnostic{Enrichment: "ok", RelatedSignals: signals}
	if err := e.Validate(); err != nil {
		t.Fatal(err)
	}
	if len(e.RelatedSignals) != MaxRelatedSignals {
		t.Errorf("got %d signals; want %d", len(e.RelatedSignals), MaxRelatedSignals)
	}
}

func TestContainsSecretLike(t *testing.T) {
	// Construct test fixtures at runtime so they don't appear as literals
	// in the source — GitHub's secret scanner would otherwise (correctly)
	// flag literal patterns matching real-world token shapes.
	vaultTok := "hvs." + "A1b2C3d4E5f6G7h8I9j0K1l2M3n4O5p6Q7r8"
	jwtTok := "eyJ" + "hbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ" + "zdWIiOiIxMjMifQ.abc"
	ghPAT := "ghp" + "_aBcDeFgHiJkLmNoPqRsTuVwXyZ0123456789"
	awsKey := "AK" + "IAIOSFODNN7EXAMPLE"
	slackTok := "xox" + "b-1234567890-abcdefghijklmnop"

	positives := []string{
		vaultTok,                // Vault token
		jwtTok,                  // JWT-shape
		ghPAT,                   // GH PAT
		awsKey,                  // AWS key
		slackTok,                // Slack bot token
		strings.Repeat("a", 64), // 64-char hex looks like a hash
		"YWJjZGVmZ2hpamtsbW5vcHFyc3R1dnd4eXowMTIzNDU2Nzg5ZGVm", // long base64
	}
	for _, s := range positives {
		if !ContainsSecretLike(s) {
			t.Errorf("did NOT flag %q as secret-like", s)
		}
	}
	negatives := []string{
		"normal text",
		"secret/t6-apps/billing/config",      // Vault path — legit
		"STRIPE_API_KEY",                     // key name — legit
		"https://hooks.slack.com/services/X", // generic URL
		"openproject-cron-environment",       // a Kubernetes Secret name
	}
	for _, s := range negatives {
		if ContainsSecretLike(s) {
			t.Errorf("falsely flagged %q as secret-like", s)
		}
	}
}

func newValidRunbook() VaultRunbook {
	return VaultRunbook{
		RunbookID:       "rb-test-1",
		VaultPath:       "secret/t6-apps/billing/config",
		KeyNames:        []string{"STRIPE_API_KEY", "STRIPE_WEBHOOK_SECRET"},
		CommandTemplate: `vault kv patch secret/t6-apps/billing/config STRIPE_API_KEY=${VALUE_STRIPE_API_KEY} STRIPE_WEBHOOK_SECRET=${VALUE_STRIPE_WEBHOOK_SECRET}`,
		Rationale:       "Two keys missing from Vault; ESO ready=False; pods will fail on next restart.",
		CreatedAt:       time.Now(),
		ExpiresAt:       time.Now().Add(DefaultRunbookTTL),
	}
}

func TestRunbookValidate_HappyPath(t *testing.T) {
	r := newValidRunbook()
	if err := r.Validate(); err != nil {
		t.Fatalf("valid runbook rejected: %v", err)
	}
}

func TestRunbookValidate_RejectsLeakedSecret(t *testing.T) {
	r := newValidRunbook()
	r.CommandTemplate = `vault kv patch secret/t6-apps/billing/config STRIPE_API_KEY=` +
		"hvs." + "A1b2C3d4E5f6G7h8I9j0K1l2M3n4O5p6Q7r8"
	if err := r.Validate(); !errors.Is(err, ErrSecretValueInOutput) {
		t.Errorf("got %v; want ErrSecretValueInOutput", err)
	}
}

func TestRunbookValidate_RequiresPlaceholderToken(t *testing.T) {
	r := newValidRunbook()
	r.CommandTemplate = "vault kv patch secret/t6-apps/billing/config STRIPE_API_KEY=abc"
	if err := r.Validate(); !errors.Is(err, ErrSecretValueInOutput) {
		t.Errorf("got %v; want ErrSecretValueInOutput for runbook missing ${VALUE} placeholder", err)
	}
}

func TestRunbookValidate_NoSecretInRationale(t *testing.T) {
	r := newValidRunbook()
	r.Rationale = "Customer's current API key is " +
		"ghp" + "_aBcDeFgHiJkLmNoPqRsTuVwXyZ0123456789 — rotate it"
	if err := r.Validate(); !errors.Is(err, ErrSecretValueInOutput) {
		t.Errorf("got %v; want ErrSecretValueInOutput for runbook with embedded GH PAT in rationale", err)
	}
}

func TestValidateT3DualApproval(t *testing.T) {
	t1 := time.Now()
	tests := []struct {
		name string
		d    DualApproval
		want error
	}{
		{
			name: "happy path",
			d: DualApproval{
				First:  ApprovedAction{Approver: "alice", ApprovedAt: t1},
				Second: ApprovedAction{Approver: "bob", ApprovedAt: t1.Add(MinT3Delay + time.Second)},
			},
			want: nil,
		},
		{
			name: "same approver",
			d: DualApproval{
				First:  ApprovedAction{Approver: "alice", ApprovedAt: t1},
				Second: ApprovedAction{Approver: "alice", ApprovedAt: t1.Add(MinT3Delay + time.Second)},
			},
			want: ErrSameApprover,
		},
		{
			name: "delay too short",
			d: DualApproval{
				First:  ApprovedAction{Approver: "alice", ApprovedAt: t1},
				Second: ApprovedAction{Approver: "bob", ApprovedAt: t1.Add(time.Minute)},
			},
			want: ErrT3DelayNotElapsed,
		},
		{
			name: "missing first approver",
			d: DualApproval{
				First:  ApprovedAction{Approver: ""},
				Second: ApprovedAction{Approver: "bob", ApprovedAt: t1.Add(MinT3Delay + time.Second)},
			},
			want: ErrTokenInvalid,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateT3DualApproval(tc.d)
			if !errors.Is(err, tc.want) {
				t.Errorf("got %v; want %v", err, tc.want)
			}
		})
	}
}
