// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package ai

import "context"

// NoOpAuditSink is a sink that discards all events. Used as a safe
// fallback when callers need a non-nil sink but the operator has not
// registered one.
type NoOpAuditSink struct{}

// Write discards e and returns nil.
func (NoOpAuditSink) Write(_ context.Context, _ AuditEvent) error { return nil }
