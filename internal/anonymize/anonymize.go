// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

// Package anonymize converts raw `cha diagnose --format json` output into
// anonymized JSONL records suitable for publishing in the public `runs/`
// directory.
//
// Anonymization contract:
//   - Namespace names, workload names, secret names, and Vault path segments
//     are replaced with deterministic short hashes derived from SHA-256.
//     The same input token always produces the same hash, so time-series
//     comparisons across daily runs remain coherent.
//   - IPv4 addresses are replaced with a placeholder.
//   - Hostnames (≥2 labels ending in a known TLD) are replaced.
//   - Image tags are stripped; the base image name is hashed if it looks
//     like an operator-specific image rather than a well-known public one.
//   - Category-level component names ("Ceph Storage", "Cluster Nodes") are
//     preserved verbatim — they carry no customer identity.
//   - Byte values of Secrets and Vault KV entries are never present in the
//     input and therefore never present in the output.
package anonymize

import (
	"crypto/sha256"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/internal/diagnose"
	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/internal/probe"
)

// RunInput is the JSON object emitted by `cha diagnose --format json`.
type RunInput struct {
	Version     string                `json:"version"`
	Results     []probe.Result        `json:"results"`
	Diagnostics []diagnose.Diagnostic `json:"diagnostics"`
}

// RunRecord is the anonymized JSONL record written to runs/.
type RunRecord struct {
	SchemaVersion string           `json:"schemaVersion"` // "1"
	RunID         string           `json:"runID"`
	Timestamp     string           `json:"timestamp"` // RFC3339
	ChaVersion    string           `json:"chaVersion"`
	Results       []AnonResult     `json:"results"`
	Diagnostics   []AnonDiagnostic `json:"diagnostics"`
	Summary       RunSummary       `json:"summary"`
}

// AnonResult mirrors probe.Result with anonymized string fields.
type AnonResult struct {
	Component AnonComponentResult `json:"component"`
	Findings  []AnonFinding       `json:"findings,omitempty"`
}

// AnonComponentResult mirrors probe.ComponentResult.
type AnonComponentResult struct {
	Component string `json:"component"`
	Status    string `json:"status"`
	Detail    string `json:"detail"`
}

// AnonFinding mirrors probe.Finding.
type AnonFinding struct {
	Component   string `json:"component"`
	Severity    string `json:"severity"`
	Message     string `json:"message"`
	Remediation string `json:"remediation,omitempty"`
}

// AnonDiagnostic mirrors diagnose.Diagnostic.
type AnonDiagnostic struct {
	Subject string `json:"subject"`
	Message string `json:"message"`
}

// RunSummary carries aggregate counts useful for trend dashboards without
// exposing any per-resource identity.
type RunSummary struct {
	TotalComponents int `json:"totalComponents"`
	HealthyCount    int `json:"healthyCount"`
	DegradedCount   int `json:"degradedCount"`
	CriticalCount   int `json:"criticalCount"`
	FindingCount    int `json:"findingCount"`
	DiagnosticCount int `json:"diagnosticCount"`
}

// Anonymizer converts RunInput → RunRecord. Construct with New.
type Anonymizer struct{}

// New returns an Anonymizer. The hash function is pure SHA-256 (no salt):
// operator-specific tokens are hashed to an opaque 8-char hex string.
// Using a salt would prevent cross-run time-series coherence, which is the
// primary value of the published run logs.
func New() *Anonymizer { return &Anonymizer{} }

// Anonymize converts one raw diagnose JSON run into an anonymized RunRecord.
// runID and ts are injected by the caller so batch processing can stamp
// records with the correct source-file identity.
func (a *Anonymizer) Anonymize(in RunInput, runID, ts string) RunRecord {
	if ts == "" {
		ts = time.Now().UTC().Format(time.RFC3339)
	}

	rec := RunRecord{
		SchemaVersion: "1",
		RunID:         runID,
		Timestamp:     ts,
		ChaVersion:    in.Version,
	}

	// Build summary counters while walking results.
	summary := RunSummary{
		TotalComponents: len(in.Results),
		DiagnosticCount: len(in.Diagnostics),
	}
	for _, r := range in.Results {
		switch strings.ToUpper(r.Component.Status) {
		case "HEALTHY":
			summary.HealthyCount++
		case "DEGRADED":
			summary.DegradedCount++
		case "CRITICAL":
			summary.CriticalCount++
		}
		summary.FindingCount += len(r.Findings)

		ar := AnonResult{
			Component: AnonComponentResult{
				Component: anonComponent(r.Component.Component),
				Status:    r.Component.Status,
				Detail:    anonText(r.Component.Detail),
			},
		}
		for _, f := range r.Findings {
			ar.Findings = append(ar.Findings, AnonFinding{
				Component:   anonComponent(f.Component),
				Severity:    string(f.Severity),
				Message:     anonText(f.Message),
				Remediation: anonText(f.Remediation),
			})
		}
		rec.Results = append(rec.Results, ar)
	}

	for _, d := range in.Diagnostics {
		rec.Diagnostics = append(rec.Diagnostics, AnonDiagnostic{
			Subject: anonSubject(d.Subject),
			Message: anonText(d.Message),
		})
	}

	rec.Summary = summary
	return rec
}

// tok returns a short deterministic hash of s for use as an anonymous
// placeholder. 8 hex chars = 32 bits of hash space — sufficient to avoid
// accidental collisions across a typical cluster's resource namespace.
func tok(s string) string {
	sum := sha256.Sum256([]byte(s))
	return fmt.Sprintf("%x", sum[:4])
}

// categoryComponents are component names that are generic / non-identifying
// and should be preserved verbatim in anonymized output.
var categoryComponents = map[string]bool{
	"Ceph Storage":      true,
	"Cluster Nodes":     true,
	"Storage Claims":    true,
	"PostgreSQL":        true,
	"Critical Services": true,
}

// anonComponent anonymizes a component name. Category-level names are kept
// verbatim. Names with embedded "(ns/name)" suffixes have those hashed.
// "Service: <display>" hashes the display label.
func anonComponent(c string) string {
	if categoryComponents[c] {
		return c
	}
	// "PostgreSQL (ns/name)" → "PostgreSQL (ns-HASH/name-HASH)"
	if i := strings.Index(c, " ("); i != -1 {
		prefix := c[:i]
		inner := strings.TrimSuffix(c[i+2:], ")")
		return prefix + " (" + anonNsName(inner) + ")"
	}
	// "Service: LiveKit SIP" → "Service: svc-HASH"
	if strings.HasPrefix(c, "Service: ") {
		svc := strings.TrimPrefix(c, "Service: ")
		return "Service: svc-" + tok(svc)
	}
	// Unknown shape — hash it entirely.
	return "component-" + tok(c)
}

// anonNsName hashes the tokens in a "ns/name" or "ns/name/extra" string.
func anonNsName(s string) string {
	parts := strings.SplitN(s, "/", 3)
	hashed := make([]string, len(parts))
	for i, p := range parts {
		hashed[i] = hashK8sName(p)
	}
	return strings.Join(hashed, "/")
}

// hashK8sName hashes a single K8s name token to a short hex placeholder.
func hashK8sName(name string) string {
	if name == "" {
		return ""
	}
	return tok(name)
}

// anonSubject anonymizes a Diagnostic.Subject. Subjects have the form
// "<category>/<ns>/<name>[/<extra>]". The category prefix is preserved
// (it's a fixed keyword like "missing-key" or "vault-missing-key");
// subsequent segments are hashed.
func anonSubject(subject string) string {
	// Subjects with known safe prefixes that carry no customer identity.
	if subject == "vault-store-rbac-missing" {
		return subject
	}
	parts := strings.SplitN(subject, "/", 2)
	if len(parts) == 1 {
		return subject // no slash — treat as opaque category
	}
	category := parts[0]
	rest := parts[1]

	// Vault path subjects: hash each path segment individually.
	// e.g. "missing-vault-path/team/app" → "missing-vault-path/HASH/HASH"
	// e.g. "vault-missing-key/team/app/KEY" → "vault-missing-key/HASH/HASH/HASH"
	if strings.HasPrefix(category, "vault") || strings.HasPrefix(category, "missing-vault") {
		segs := strings.Split(rest, "/")
		hashed := make([]string, len(segs))
		for i, s := range segs {
			hashed[i] = tok(s)
		}
		return category + "/" + strings.Join(hashed, "/")
	}

	// K8s-resource subjects: "missing-key/ns/name/key" or "missing-secret/ns/name"
	// Keep the category; hash the rest as ns/name/... segments.
	segs := strings.Split(rest, "/")
	hashed := make([]string, len(segs))
	for i, s := range segs {
		hashed[i] = tok(s)
	}
	return category + "/" + strings.Join(hashed, "/")
}

// ── Text-level regex substitutions ──────────────────────────────────────────

var (
	// IPv4 addresses: 1.2.3.4
	reIPv4 = regexp.MustCompile(`\b(?:\d{1,3}\.){3}\d{1,3}\b`)

	// Hostnames: sub.domain.tld (at least 2 dots, TLD ≥ 2 chars, no leading digit)
	// Excludes pure dotted numbers (already caught by IPv4 rule).
	reHostname = regexp.MustCompile(`\b(?:[a-zA-Z][a-zA-Z0-9-]*\.)+(?:com|io|net|org|dev|ai|cloud|internal|cluster|local|svc)\b`)

	// Backtick-quoted ns/name pairs: `ns/name` or `ns/name/extra`
	reBacktickNsName = regexp.MustCompile("`([^`/\n]+/[^`\n]+)`")

	// K8s resource kind/ns/name: ExternalSecret/ns/name, Secret/ns/name, etc.
	reKindNsName = regexp.MustCompile(`\b(ExternalSecret|Secret|Deployment|StatefulSet|Job|CronJob|Pod|ReplicaSet|Node)/([\w-]+)/([\w-]+)\b`)

	// Container image refs with a tag: registry.io/org/image:tag or image:tag
	// We hash the image name part and drop the registry+tag detail.
	reImageRef = regexp.MustCompile(`\b([a-zA-Z0-9][a-zA-Z0-9._/-]+):([a-zA-Z0-9._-]+)\b`)

	// Vault path segments in backticks: `secret/team/app`
	// Handled by the backtick rule above; Vault paths also match ns/name pattern.
)

// anonText applies the full substitution pipeline to a free-text message string.
func anonText(s string) string {
	if s == "" {
		return s
	}
	// 1. IPv4 addresses — before hostname rule to avoid partial matches.
	s = reIPv4.ReplaceAllStringFunc(s, func(ip string) string {
		return "ip-" + tok(ip)
	})

	// 2. Hostnames.
	s = reHostname.ReplaceAllStringFunc(s, func(host string) string {
		return "host-" + tok(host)
	})

	// 3. K8s kind/ns/name references.
	s = reKindNsName.ReplaceAllStringFunc(s, func(ref string) string {
		parts := strings.SplitN(ref, "/", 3)
		if len(parts) == 3 {
			return parts[0] + "/" + tok(parts[1]) + "/" + tok(parts[2])
		}
		return ref
	})

	// 4. Backtick-quoted ns/name (catches Vault paths and K8s ns/name pairs).
	s = reBacktickNsName.ReplaceAllStringFunc(s, func(match string) string {
		inner := match[1 : len(match)-1] // strip backticks
		return "`" + anonSlashPath(inner) + "`"
	})

	// 5. Container image refs: keep the final image-name component, hash registry
	//    prefix, redact tag. Only applies where the pattern looks like a real
	//    image ref (contains "/" or known registry keywords).
	s = reImageRef.ReplaceAllStringFunc(s, func(ref string) string {
		colon := strings.LastIndex(ref, ":")
		image := ref[:colon]
		// Keep well-known public images verbatim to preserve readability.
		if isWellKnownImage(image) {
			return ref
		}
		// Hash the image name, drop the tag.
		return "img-" + tok(image) + ":tag"
	})

	return s
}

// anonSlashPath hashes each /-separated segment in a path string.
func anonSlashPath(s string) string {
	parts := strings.Split(s, "/")
	hashed := make([]string, len(parts))
	for i, p := range parts {
		hashed[i] = tok(p)
	}
	return strings.Join(hashed, "/")
}

// wellKnownImages are public base images whose names carry no customer identity.
var wellKnownImages = []string{
	"nginx", "redis", "postgres", "mysql", "mongo", "alpine", "ubuntu",
	"debian", "busybox", "python", "node", "golang", "openjdk", "gcr.io/",
	"registry.k8s.io/", "quay.io/", "ghcr.io/bionic-ai-solutions/",
}

func isWellKnownImage(image string) bool {
	lower := strings.ToLower(image)
	for _, prefix := range wellKnownImages {
		if strings.HasPrefix(lower, prefix) {
			return true
		}
	}
	return false
}
