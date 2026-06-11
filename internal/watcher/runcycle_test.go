// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package watcher

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"

	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/internal/diagnose"
	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/internal/fix"
	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/internal/report"
	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/internal/snapshot"
	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/pkg/registry"
)

// ---- runCycle end-to-end tests -------------------------------------------
//
// runCycle is the trickiest state machine in the repo: it diffs pre-fix vs
// post-fix cluster state, dedups via the seen map, and routes posts to the
// Alerts ("CHA acted") vs Critical ("human needed") channels. These tests
// drive the REAL runCycle through scripted analyzers/fixers and observe its
// externally visible effects: Slack webhook posts (captured by httptest
// servers) and the seen-map state. No refactor of runCycle was needed —
// every dependency is injectable via existing Watcher fields:
//
//   - lv is nil: safe because scripted analyzers/fixers never touch the
//     source, CloudSource is nil, no investigator/enricher is registered,
//     AlertmanagerURL is empty, and WriteDriftReports is false — all the
//     paths that would dereference lv are config-gated off.
//   - reg carries the scripted analyzer (+ optional scripted fixer).
//   - mut is a no-op Mutator, only set for remediation cases.

// scriptedAnalyzer returns script[i] on its i-th Run call and nil once the
// script is exhausted. Mirrors how internal/diagnose tests script
// snapshot.Source fixtures, but at the Analyzer layer so runCycle's own
// pre-fix/post-fix double-diagnose is exercised for real.
type scriptedAnalyzer struct {
	mu     sync.Mutex
	script [][]diagnose.Diagnostic
	call   int
}

func (a *scriptedAnalyzer) Name() string { return "scripted" }

func (a *scriptedAnalyzer) Run(context.Context, snapshot.Source) []diagnose.Diagnostic {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.call >= len(a.script) {
		return nil
	}
	out := a.script[a.call]
	a.call++
	return out
}

// scriptedFixer returns fix.Result{Actions: script[i]} on its i-th Run call
// and an empty Result once exhausted.
type scriptedFixer struct {
	mu     sync.Mutex
	script [][]fix.Action
	call   int
}

func (f *scriptedFixer) Name() string { return "scripted-fixer" }

func (f *scriptedFixer) Run(context.Context, snapshot.Source, snapshot.Mutator) fix.Result {
	f.mu.Lock()
	defer f.mu.Unlock()
	r := fix.Result{Fixer: "scripted-fixer"}
	if f.call < len(f.script) {
		r.Actions = f.script[f.call]
	}
	f.call++
	return r
}

// nopMutator satisfies snapshot.Mutator with no-ops; runCycle only checks
// it for nil-ness to decide whether remediation runs.
type nopMutator struct{}

func (nopMutator) Delete(context.Context, schema.GroupVersionResource, string, string) error {
	return nil
}

func (nopMutator) Patch(context.Context, schema.GroupVersionResource, string, string, types.PatchType, []byte) error {
	return nil
}

func (nopMutator) PatchStatus(context.Context, schema.GroupVersionResource, string, string, types.PatchType, []byte) error {
	return nil
}

func (nopMutator) Create(context.Context, schema.GroupVersionResource, string, *unstructured.Unstructured) error {
	return nil
}

// slackRec captures Slack webhook POST bodies from runCycle.
type slackRec struct {
	mu     sync.Mutex
	bodies []string
}

func (r *slackRec) count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.bodies)
}

func (r *slackRec) all() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return strings.Join(r.bodies, "\n---\n")
}

func newSlackServer(t *testing.T) (*httptest.Server, *slackRec) {
	t.Helper()
	rec := &slackRec{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		body, _ := io.ReadAll(req.Body)
		rec.mu.Lock()
		rec.bodies = append(rec.bodies, string(body))
		rec.mu.Unlock()
		_, _ = w.Write([]byte("ok"))
	}))
	t.Cleanup(srv.Close)
	return srv, rec
}

func TestRunCycle(t *testing.T) {
	const subject = "Pod/demo/web-0"
	finding := diagnose.Diagnostic{
		Subject:  subject,
		Severity: "critical",
		Message:  "container OOMKilled",
	}
	fixAction := fix.Action{
		Description: "Deleted stale Failed pod",
		Object:      "Pod/demo/web-0",
	}

	cases := []struct {
		name string
		// analyzer outputs per Run call. With remediation ON the
		// analyzer runs TWICE per cycle (pre-fix + post-fix re-diagnose).
		analyzer [][]diagnose.Diagnostic
		// fixer actions per Run call; nil = no fixer registered.
		fixer          [][]fix.Action
		remediation    bool
		postOnResolved bool
		cycles         int

		wantCriticalPosts int      // posts to the Critical ("human needed") channel
		wantAlertsPosts   int      // posts to the Alerts ("CHA acted") channel
		wantSeenSubjects  []string // exact seen-map keys after the last cycle
		// substring that must appear in the captured channel bodies
		wantCriticalContains string
		wantAlertsContains   string
	}{
		{
			// A brand-new finding must be posted exactly once and
			// recorded in the seen map.
			name:                 "new finding posted and recorded once",
			analyzer:             [][]diagnose.Diagnostic{{finding}},
			cycles:               1,
			wantCriticalPosts:    1,
			wantSeenSubjects:     []string{subject},
			wantCriticalContains: subject,
		},
		{
			// The same finding (same fingerprint) on the next cycle must
			// be deduped: no second Slack post, entry stays in seen.
			name:              "persisting finding deduped on second cycle",
			analyzer:          [][]diagnose.Diagnostic{{finding}, {finding}},
			cycles:            2,
			wantCriticalPosts: 1,
			wantSeenSubjects:  []string{subject},
		},
		{
			// Finding fixed mid-cycle: present pre-fix, absent in the
			// post-fix re-diagnose. Per the code's documented intent
			// (see the preFix comment in runCycle): the Slack diff uses
			// the PRE-fix state so the just-fixed issue still appears in
			// the alert — routed to the Alerts channel because it's
			// absent from postFixSubjects ("CHA acted"); NOT to the
			// Critical channel. The seen map is updated from the
			// POST-fix state so the subject does not linger, and the
			// follow-up cycle emits no spurious "resolved" post.
			name: "finding fixed mid-cycle alerts once and leaves seen clean",
			analyzer: [][]diagnose.Diagnostic{
				{finding}, {}, // cycle 1: pre-fix sees it, post-fix doesn't
				{}, {}, // cycle 2: clean
			},
			fixer:              [][]fix.Action{{fixAction}, {}},
			remediation:        true,
			postOnResolved:     true,
			cycles:             2,
			wantCriticalPosts:  0,
			wantAlertsPosts:    1,
			wantSeenSubjects:   nil,
			wantAlertsContains: subject,
		},
		{
			// Finding clears on its own (no fixer): with PostOnResolved
			// the resolved transition posts to Critical, and the seen
			// map drops the subject.
			name:              "cleared finding posts resolved and prunes seen",
			analyzer:          [][]diagnose.Diagnostic{{finding}, {}},
			postOnResolved:    true,
			cycles:            2,
			wantCriticalPosts: 2, // initial alert + resolved notice
			wantSeenSubjects:  nil,
		},
		{
			// Same clear, but PostOnResolved=false: seen map is still
			// pruned, no resolved post fires.
			name:              "cleared finding prunes seen silently without PostOnResolved",
			analyzer:          [][]diagnose.Diagnostic{{finding}, {}},
			postOnResolved:    false,
			cycles:            2,
			wantCriticalPosts: 1,
			wantSeenSubjects:  nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			alertsSrv, alertsRec := newSlackServer(t)
			criticalSrv, criticalRec := newSlackServer(t)

			reg := registry.New()
			reg.RegisterAnalyzer(&scriptedAnalyzer{script: tc.analyzer})
			var mut snapshot.Mutator
			if tc.fixer != nil {
				reg.RegisterFixer(&scriptedFixer{script: tc.fixer})
			}
			if tc.remediation {
				mut = nopMutator{}
			}

			w := New(nil, reg, mut, Config{
				SlackChannels: report.SlackChannels{
					Alerts:   alertsSrv.URL,
					Critical: criticalSrv.URL,
				},
				PostOnResolved: tc.postOnResolved,
				RunRemediation: tc.remediation,
			})

			for i := 0; i < tc.cycles; i++ {
				w.runCycle(context.Background())
			}

			if got := criticalRec.count(); got != tc.wantCriticalPosts {
				t.Errorf("critical-channel posts = %d, want %d; bodies:\n%s",
					got, tc.wantCriticalPosts, criticalRec.all())
			}
			if got := alertsRec.count(); got != tc.wantAlertsPosts {
				t.Errorf("alerts-channel posts = %d, want %d; bodies:\n%s",
					got, tc.wantAlertsPosts, alertsRec.all())
			}
			if tc.wantCriticalContains != "" && !strings.Contains(criticalRec.all(), tc.wantCriticalContains) {
				t.Errorf("critical-channel bodies missing %q:\n%s", tc.wantCriticalContains, criticalRec.all())
			}
			if tc.wantAlertsContains != "" && !strings.Contains(alertsRec.all(), tc.wantAlertsContains) {
				t.Errorf("alerts-channel bodies missing %q:\n%s", tc.wantAlertsContains, alertsRec.all())
			}

			w.mu.Lock()
			defer w.mu.Unlock()
			if len(w.seen) != len(tc.wantSeenSubjects) {
				t.Fatalf("seen map has %d entries, want %d: %+v", len(w.seen), len(tc.wantSeenSubjects), w.seen)
			}
			for _, s := range tc.wantSeenSubjects {
				if _, ok := w.seen[s]; !ok {
					t.Errorf("seen map missing subject %q: %+v", s, w.seen)
				}
			}
		})
	}
}

// TestRunCycle_DedupPreservesLastPosted pins the repeat-interval anchor:
// a deduped (not re-posted) finding must keep its original lastPosted
// timestamp, otherwise RepeatInterval would never elapse and stable
// findings would never re-post.
func TestRunCycle_DedupPreservesLastPosted(t *testing.T) {
	criticalSrv, _ := newSlackServer(t)
	const subject = "Deploy/demo/api"
	finding := diagnose.Diagnostic{Subject: subject, Severity: "warning", Message: "drift"}

	reg := registry.New()
	reg.RegisterAnalyzer(&scriptedAnalyzer{script: [][]diagnose.Diagnostic{{finding}, {finding}}})
	w := New(nil, reg, nil, Config{
		SlackChannels: report.SlackChannels{Critical: criticalSrv.URL},
	})

	w.runCycle(context.Background())
	w.mu.Lock()
	first := w.seen[subject]
	if first == nil || first.lastPosted.IsZero() {
		w.mu.Unlock()
		t.Fatalf("cycle 1 must record a posted entry with lastPosted set; got %+v", first)
	}
	firstPosted := first.lastPosted
	w.mu.Unlock()

	time.Sleep(5 * time.Millisecond) // ensure a re-post would move the clock
	w.runCycle(context.Background())

	w.mu.Lock()
	defer w.mu.Unlock()
	second := w.seen[subject]
	if second == nil {
		t.Fatal("subject must survive cycle 2 in the seen map")
	}
	if !second.lastPosted.Equal(firstPosted) {
		t.Errorf("deduped finding must preserve lastPosted: cycle1=%v cycle2=%v",
			firstPosted, second.lastPosted)
	}
}
