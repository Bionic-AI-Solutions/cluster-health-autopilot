// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

package diagnose

import (
	"context"
	"strings"
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func makeEvent(invKind, invNS, invName, message string) unstructured.Unstructured {
	u := unstructured.Unstructured{Object: map[string]any{}}
	u.SetAPIVersion("v1")
	u.SetKind("Event")
	u.SetNamespace(invNS)
	u.SetName(invName + "-evt")
	_ = unstructured.SetNestedField(u.Object, message, "message")
	_ = unstructured.SetNestedField(u.Object, invKind, "involvedObject", "kind")
	_ = unstructured.SetNestedField(u.Object, invName, "involvedObject", "name")
	_ = unstructured.SetNestedField(u.Object, invNS, "involvedObject", "namespace")
	return u
}

func TestLogPatternMatcher_Name(t *testing.T) {
	if (LogPatternMatcher{}).Name() != "LogPatternMatcher" {
		t.Error("Name mismatch")
	}
}

func TestLogPatternMatcher_ImagePullBackOff_Critical(t *testing.T) {
	src := &memSourceDD{byResource: map[string][]unstructured.Unstructured{
		"events": {
			makeEvent("Pod", "ns", "app-1", "Back-off pulling image: ErrImagePull: manifest unknown"),
		},
	}}
	got := LogPatternMatcher{}.Run(context.Background(), src)
	if len(got) != 1 {
		t.Fatalf("expected 1 diagnostic; got %d", len(got))
	}
	if got[0].Source != "LogPatternMatcher.ImagePullBackOff" {
		t.Errorf("source: %q", got[0].Source)
	}
	// A container kubelet cannot pull is hard-down → critical.
	if got[0].Severity != "critical" {
		t.Errorf("severity: %q", got[0].Severity)
	}
}

func TestLogPatternMatcher_OOMKilled(t *testing.T) {
	src := &memSourceDD{byResource: map[string][]unstructured.Unstructured{
		"events": {
			makeEvent("Pod", "ns", "memhog", "Container memhog was OOMKilled"),
		},
	}}
	got := LogPatternMatcher{}.Run(context.Background(), src)
	if len(got) != 1 || !strings.Contains(got[0].Source, "OOMKilled") {
		t.Errorf("expected OOMKilled finding; got %+v", got)
	}
}

func TestLogPatternMatcher_VolumeAttachFailed_Critical(t *testing.T) {
	src := &memSourceDD{byResource: map[string][]unstructured.Unstructured{
		"events": {
			makeEvent("Pod", "ns", "storage-app", "AttachVolume.Attach failed for volume \"pvc-abc\": rpc error"),
		},
	}}
	got := LogPatternMatcher{}.Run(context.Background(), src)
	if len(got) != 1 {
		t.Fatalf("expected 1 diagnostic; got %d", len(got))
	}
	if got[0].Severity != "critical" {
		t.Errorf("volume attach failed should be critical; got %q", got[0].Severity)
	}
}

func TestLogPatternMatcher_DedupsBySubjectAndLabel(t *testing.T) {
	// Same Pod with 30 OOMKilled events should produce 1 diagnostic.
	events := make([]unstructured.Unstructured, 0, 30)
	for i := 0; i < 30; i++ {
		events = append(events, makeEvent("Pod", "ns", "memhog", "Container memhog was OOMKilled (attempt N)"))
	}
	src := &memSourceDD{byResource: map[string][]unstructured.Unstructured{"events": events}}
	got := LogPatternMatcher{}.Run(context.Background(), src)
	if len(got) != 1 {
		t.Errorf("dedup failed; got %d, expected 1", len(got))
	}
}

func TestLogPatternMatcher_NoMatch_NoFindings(t *testing.T) {
	src := &memSourceDD{byResource: map[string][]unstructured.Unstructured{
		"events": {
			makeEvent("Pod", "ns", "happy", "Pulled image gracefully"),
		},
	}}
	got := LogPatternMatcher{}.Run(context.Background(), src)
	if len(got) != 0 {
		t.Errorf("unrelated message must not fire; got %+v", got)
	}
}

func TestLogPatternMatcher_EmptyEventListNoOp(t *testing.T) {
	src := &memSourceDD{byResource: map[string][]unstructured.Unstructured{}}
	got := LogPatternMatcher{}.Run(context.Background(), src)
	if len(got) != 0 {
		t.Errorf("no events must yield nothing; got %+v", got)
	}
}

func TestLogPatternMatcher_TruncatesLongMessages(t *testing.T) {
	long := strings.Repeat("x", 500) + " OOMKilled"
	src := &memSourceDD{byResource: map[string][]unstructured.Unstructured{
		"events": {
			makeEvent("Pod", "ns", "big", long),
		},
	}}
	got := LogPatternMatcher{}.Run(context.Background(), src)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding")
	}
	if !strings.HasSuffix(got[0].Message, "…") {
		t.Errorf("long message should be truncated with …; got %q", got[0].Message)
	}
}
