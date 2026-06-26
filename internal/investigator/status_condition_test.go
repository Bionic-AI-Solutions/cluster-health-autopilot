// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

package investigator

import (
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// A failed Job records its cause in a terminal `type: Failed` condition with
// no Ready condition and no top-level status.reason. readCommonStatus must
// surface that reason (e.g. DeadlineExceeded) so investigateCronJob reports
// the real cause instead of the "can't be read" fallback.
func TestReadCommonStatus_JobFailedCondition(t *testing.T) {
	job := &unstructured.Unstructured{Object: map[string]any{
		"status": map[string]any{
			"conditions": []any{
				map[string]any{
					"type": "Failed", "status": "True",
					"reason":  "DeadlineExceeded",
					"message": "Job was active longer than specified deadline",
				},
			},
		},
	}}
	status, reason, msg := readCommonStatus(job)
	if reason != "DeadlineExceeded" {
		t.Errorf("reason: got %q want DeadlineExceeded", reason)
	}
	if msg == "" {
		t.Errorf("message should be surfaced; got empty")
	}
	if status != "Failed=True" {
		t.Errorf("status: got %q want Failed=True", status)
	}
}

// A Ready condition still wins (terminal-condition handling is a fallback).
func TestReadCommonStatus_ReadyConditionWins(t *testing.T) {
	obj := &unstructured.Unstructured{Object: map[string]any{
		"status": map[string]any{
			"conditions": []any{
				map[string]any{"type": "Ready", "status": "False", "reason": "ContainersNotReady"},
				map[string]any{"type": "Failed", "status": "True", "reason": "DeadlineExceeded"},
			},
		},
	}}
	_, reason, _ := readCommonStatus(obj)
	if reason != "ContainersNotReady" {
		t.Errorf("Ready condition should win; got reason %q", reason)
	}
}

// A successful Job (Complete=True) surfaces as Complete with no failure noise.
func TestReadCommonStatus_JobCompleteCondition(t *testing.T) {
	job := &unstructured.Unstructured{Object: map[string]any{
		"status": map[string]any{
			"conditions": []any{
				map[string]any{"type": "Complete", "status": "True"},
			},
		},
	}}
	status, _, _ := readCommonStatus(job)
	if status != "Complete=True" {
		t.Errorf("status: got %q want Complete=True", status)
	}
}
