// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

// Package cloud holds the multi-provider cloud.Source assembler. It
// pulls together AWS / GCP / Azure sub-clients (configured or nil)
// into a single cloud.Source that probes consume.
package cloud

import (
	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/pkg/cloud"
	pkgaws "github.com/Bionic-AI-Solutions/cluster-health-autopilot/pkg/cloud/aws"
	pkgazure "github.com/Bionic-AI-Solutions/cluster-health-autopilot/pkg/cloud/azure"
	pkggcp "github.com/Bionic-AI-Solutions/cluster-health-autopilot/pkg/cloud/gcp"
)

// Source is the implementation of pkg/cloud.Source used by the
// watcher. Each provider sub-client is either nil (provider not
// configured → probes return SKIPPED) or a Live/Snapshot client.
type Source struct {
	aws   pkgaws.Client
	gcp   pkggcp.Client
	azure pkgazure.Client
	mode  cloud.Mode
}

// NewSource constructs the multi-provider source. Any of aws, gcp,
// azure may be nil — the matching probes will return SKIPPED.
func NewSource(aws pkgaws.Client, gcp pkggcp.Client, azure pkgazure.Client, mode cloud.Mode) *Source {
	return &Source{aws: aws, gcp: gcp, azure: azure, mode: mode}
}

// AWS satisfies cloud.Source.
func (s *Source) AWS() pkgaws.Client { return s.aws }

// GCP satisfies cloud.Source.
func (s *Source) GCP() pkggcp.Client { return s.gcp }

// Azure satisfies cloud.Source.
func (s *Source) Azure() pkgazure.Client { return s.azure }

// Mode satisfies cloud.Source.
func (s *Source) Mode() cloud.Mode { return s.mode }

// AnyConfigured reports whether at least one provider sub-client is
// non-nil. Callers use this to skip cloud-probe iteration entirely
// when nothing is configured.
func (s *Source) AnyConfigured() bool {
	return s.aws != nil || s.gcp != nil || s.azure != nil
}
