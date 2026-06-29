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

// makePodStatus builds a Pod with one container's ready + restartCount status.
func makePodStatus(ns, name string, ready bool, restarts int64) unstructured.Unstructured {
	u := unstructured.Unstructured{Object: map[string]any{}}
	u.SetAPIVersion("v1")
	u.SetKind("Pod")
	u.SetNamespace(ns)
	u.SetName(name)
	cs := map[string]any{"name": "c", "ready": ready, "restartCount": restarts}
	_ = unstructured.SetNestedSlice(u.Object, []any{cs}, "status", "containerStatuses")
	return u
}

// makeEventCount is makeEvent with the Event aggregation count set.
func makeEventCount(invKind, invNS, invName, message string, count int64) unstructured.Unstructured {
	u := makeEvent(invKind, invNS, invName, message)
	_ = unstructured.SetNestedField(u.Object, count, "count")
	return u
}

const livenessFailMsg = "Liveness probe failed: zookeeper healthcheck failed"

func TestLogPatternMatcher_Name(t *testing.T) {
	if (LogPatternMatcher{}).Name() != "LogPatternMatcher" {
		t.Error("Name mismatch")
	}
}

// A single liveness-probe blip on a pod that is now Ready (0 restarts) is a
// transient, self-healed event → stays advisory (warning). This is the
// langfuse-zookeeper-0 case the user asked about.
func TestLogPatternMatcher_ProbeFailed_TransientBlip_StaysWarning(t *testing.T) {
	src := &memSourceDD{byResource: map[string][]unstructured.Unstructured{
		"events": {makeEvent("Pod", "langfuse", "langfuse-zookeeper-0", livenessFailMsg)},
		"pods":   {makePodStatus("langfuse", "langfuse-zookeeper-0", true, 0)},
	}}
	got := LogPatternMatcher{}.Run(context.Background(), src)
	if len(got) != 1 || got[0].Source != "LogPatternMatcher.ProbeFailed" {
		t.Fatalf("expected one ProbeFailed finding; got %+v", got)
	}
	if got[0].Severity != "warning" {
		t.Errorf("a recovered blip must stay warning (advisory); got %q", got[0].Severity)
	}
	if strings.Contains(got[0].Message, "escalated") {
		t.Errorf("transient blip must not carry an escalation marker; got %q", got[0].Message)
	}
}

// A probe failure on a pod that is currently NotReady is genuinely down →
// escalate to critical (→ Human Action Required channel).
func TestLogPatternMatcher_ProbeFailed_PodNotReady_Escalates(t *testing.T) {
	src := &memSourceDD{byResource: map[string][]unstructured.Unstructured{
		"events": {makeEvent("Pod", "data", "zk-0", livenessFailMsg)},
		"pods":   {makePodStatus("data", "zk-0", false, 0)},
	}}
	got := LogPatternMatcher{}.Run(context.Background(), src)
	if len(got) != 1 || got[0].Severity != "critical" {
		t.Fatalf("NotReady pod must escalate to critical; got %+v", got)
	}
	if !strings.Contains(got[0].Message, "escalated: pod NotReady") {
		t.Errorf("escalation reason missing; got %q", got[0].Message)
	}
}

// A restart loop (restartCount past threshold) escalates even if the pod
// briefly reports ready between restarts.
func TestLogPatternMatcher_ProbeFailed_RestartLoop_Escalates(t *testing.T) {
	src := &memSourceDD{byResource: map[string][]unstructured.Unstructured{
		"events": {makeEvent("Pod", "data", "zk-1", livenessFailMsg)},
		"pods":   {makePodStatus("data", "zk-1", true, 4)},
	}}
	got := LogPatternMatcher{}.Run(context.Background(), src)
	if len(got) != 1 || got[0].Severity != "critical" {
		t.Fatalf("restart loop must escalate; got %+v", got)
	}
	if !strings.Contains(got[0].Message, "restart-looping") {
		t.Errorf("escalation reason missing; got %q", got[0].Message)
	}
}

// A persistently flapping probe (recurrence past threshold) escalates on the
// recurrence signal alone — even without a readable Pod.
func TestLogPatternMatcher_ProbeFailed_Recurring_Escalates(t *testing.T) {
	src := &memSourceDD{byResource: map[string][]unstructured.Unstructured{
		"events": {makeEventCount("Pod", "data", "zk-2", livenessFailMsg, 9)},
		// no pod fixture — recurrence alone must escalate
	}}
	got := LogPatternMatcher{}.Run(context.Background(), src)
	if len(got) != 1 || got[0].Severity != "critical" {
		t.Fatalf("recurring probe failure must escalate; got %+v", got)
	}
	if !strings.Contains(got[0].Message, "escalated: recurring") {
		t.Errorf("escalation reason missing; got %q", got[0].Message)
	}
}

// When the live Pod can't be read and recurrence is low, do NOT over-escalate.
func TestLogPatternMatcher_ProbeFailed_NoPod_LowRecurrence_StaysWarning(t *testing.T) {
	src := &memSourceDD{byResource: map[string][]unstructured.Unstructured{
		"events": {makeEvent("Pod", "data", "zk-3", livenessFailMsg)},
	}}
	got := LogPatternMatcher{}.Run(context.Background(), src)
	if len(got) != 1 || got[0].Severity != "warning" {
		t.Fatalf("unreadable pod + low recurrence must stay warning; got %+v", got)
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
