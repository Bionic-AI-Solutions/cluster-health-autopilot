// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

package ai_test

import (
	"errors"
	"testing"
	"time"

	"github.com/srenix-ai/agentic-sre/pkg/ai"
)

// validPullRequestProposal returns a proposal that should pass Validate.
// Used as the baseline for negative tests.
func validPullRequestProposal() ai.AIProposedAction {
	now := time.Now().UTC()
	return ai.AIProposedAction{
		ActionID:          "Pod/production/srenix-enterprise:propose-digest-pin",
		ActionKind:        ai.ActionProposePullRequest,
		Target:            ai.ObjectRef{Kind: "Pod", Namespace: "production", Name: "srenix-enterprise-xyz"},
		PullRequestURL:    "https://github.com/srenix-ai/agentic-sre-enterprise/pull/42",
		Rationale:         "Pin docker4zerocool/srenix-enterprise:1.10.0 to its observed @sha256:...",
		Rollback:          ai.RollbackInfo{Description: "Close PR + delete branch"},
		DiagnosticSubject: "Pod/production/srenix-enterprise-xyz",
		Tier:              ai.TierT1,
		CreatedAt:         now,
		ExpiresAt:         now.Add(15 * time.Minute),
	}
}

func TestActionKind_ProposePullRequest_IsValid(t *testing.T) {
	if !ai.ActionProposePullRequest.IsValid() {
		t.Error("ActionProposePullRequest should be in the whitelist")
	}
}

func TestValidate_ProposePullRequest_HappyPath(t *testing.T) {
	p := validPullRequestProposal()
	if err := p.Validate(); err != nil {
		t.Fatalf("valid proposal rejected: %v", err)
	}
}

func TestValidate_ProposePullRequest_EmptyURLRejected(t *testing.T) {
	p := validPullRequestProposal()
	p.PullRequestURL = ""
	if err := p.Validate(); !errors.Is(err, ai.ErrPullRequestURLEmpty) {
		t.Errorf("want ErrPullRequestURLEmpty, got %v", err)
	}
}

func TestValidate_ProposePullRequest_WhitespaceURLRejected(t *testing.T) {
	p := validPullRequestProposal()
	p.PullRequestURL = "   \n\t  "
	if err := p.Validate(); !errors.Is(err, ai.ErrPullRequestURLEmpty) {
		t.Errorf("want ErrPullRequestURLEmpty on whitespace, got %v", err)
	}
}

func TestValidate_ProposePullRequest_HTTPScheme_Rejected(t *testing.T) {
	p := validPullRequestProposal()
	p.PullRequestURL = "http://github.com/org/repo/pull/1" // not https
	if err := p.Validate(); !errors.Is(err, ai.ErrPullRequestURLInvalid) {
		t.Errorf("want ErrPullRequestURLInvalid for http://, got %v", err)
	}
}

func TestValidate_ProposePullRequest_NoHost_Rejected(t *testing.T) {
	p := validPullRequestProposal()
	p.PullRequestURL = "https:///path" // no host
	if err := p.Validate(); !errors.Is(err, ai.ErrPullRequestURLInvalid) {
		t.Errorf("want ErrPullRequestURLInvalid for missing host, got %v", err)
	}
}

func TestValidate_ProposePullRequest_GarbledURL_Rejected(t *testing.T) {
	p := validPullRequestProposal()
	p.PullRequestURL = "://" // unparseable
	if err := p.Validate(); !errors.Is(err, ai.ErrPullRequestURLInvalid) {
		t.Errorf("want ErrPullRequestURLInvalid for garbled url, got %v", err)
	}
}

func TestValidate_ProposePullRequest_SelfHostedGitLab_Accepted(t *testing.T) {
	// Operators running self-hosted GitLab / Gitea / Forgejo have
	// arbitrary hostnames. No allowlist enforcement at the OSS layer.
	p := validPullRequestProposal()
	p.PullRequestURL = "https://gitlab.internal.example.com/team/repo/-/merge_requests/123"
	if err := p.Validate(); err != nil {
		t.Errorf("self-hosted forge URL should validate: %v", err)
	}
}

func TestValidate_ProposePullRequest_ProtectedNamespaceRejected(t *testing.T) {
	// Even a PR-proposal action targeting a protected NS workload is
	// refused — we don't want Srenix proposing PRs that modify
	// kube-system / vault / etc. infra.
	p := validPullRequestProposal()
	p.Target.Namespace = "kube-system"
	if err := p.Validate(); !errors.Is(err, ai.ErrProtectedNamespace) {
		t.Errorf("want ErrProtectedNamespace, got %v", err)
	}
}

func TestValidate_ProposePullRequest_MissingRollback_Rejected(t *testing.T) {
	p := validPullRequestProposal()
	p.Rollback.Description = ""
	if err := p.Validate(); !errors.Is(err, ai.ErrMissingRollback) {
		t.Errorf("want ErrMissingRollback, got %v", err)
	}
}

func TestValidate_PullRequestURLOnWrongKind_Rejected(t *testing.T) {
	// PullRequestURL on a non-ProposePullRequest action is a misconfig
	// — refuse so the executor never sees mismatched fields.
	p := validPullRequestProposal()
	p.ActionKind = ai.ActionDeletePod
	p.PullRequestURL = "https://github.com/x/y/pull/1"
	if err := p.Validate(); !errors.Is(err, ai.ErrInvalidActionKind) {
		t.Errorf("want ErrInvalidActionKind on URL+wrong kind, got %v", err)
	}
}

func TestValidate_ProposePullRequest_TierT0_Rejected(t *testing.T) {
	// T0 = narration-only; doesn't authorize action proposals.
	p := validPullRequestProposal()
	p.Tier = ai.TierT0
	if err := p.Validate(); !errors.Is(err, ai.ErrInvalidTier) {
		t.Errorf("want ErrInvalidTier on T0, got %v", err)
	}
}

func TestValidate_ProposePullRequest_DoesNotNeedManifestYAML(t *testing.T) {
	// ProposePullRequest carries NO cluster mutation, so ManifestYAML
	// MUST be empty. Setting it should be refused (would mean the
	// proposer is confused about what kind of action it's emitting).
	p := validPullRequestProposal()
	p.ManifestYAML = []byte("apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: x\n  namespace: y")
	if err := p.Validate(); !errors.Is(err, ai.ErrInvalidActionKind) {
		t.Errorf("want ErrInvalidActionKind when ManifestYAML set on ProposePullRequest, got %v", err)
	}
}
