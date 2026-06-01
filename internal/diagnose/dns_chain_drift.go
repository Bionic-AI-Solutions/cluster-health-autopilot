// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package diagnose

import (
	"context"
	"fmt"
	"net/url"
	"sort"
	"strings"

	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/internal/snapshot"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// CloudflareClient is the interface satisfied by a Cloudflare API client.
// Only the zone-list and record-list operations are required; the concrete
// implementation lives in internal/dns/cloudflare (to be added). Tests inject
// a fake that returns pre-seeded data.
//
// Fail-open contract: any error returned by these methods is treated as
// "Cloudflare hop not verifiable for this zone" — the analyzer continues with
// the K8s-layer checks and emits the "external DNS hop not verified" info
// diagnostic rather than propagating errors.
type CloudflareClient interface {
	ListZones(ctx context.Context) ([]Zone, error)
	ListDNSRecords(ctx context.Context, zoneID string) ([]DNSRecord, error)
}

// Zone is a Cloudflare DNS zone (account-level container for DNS records).
type Zone struct {
	ID   string
	Name string // e.g. "bionicaisolutions.com"
}

// DNSRecord is a single DNS record within a Cloudflare zone.
type DNSRecord struct {
	Name    string // fully-qualified host, e.g. "livekit.bionicaisolutions.com"
	Type    string // "A", "AAAA", "CNAME", etc.
	Content string // IP address or CNAME target
	Proxied bool   // true when Cloudflare proxy (orange cloud) is enabled
}

// ingressBackend carries the K8s references resolved from an Ingress rule.
type ingressBackend struct {
	ns      string // namespace of the Ingress
	ingName string // Ingress name
	svcNs   string // namespace that contains the backend Service (same as ns for standard Ingress)
	svcName string // backend Service name
	svcPort string // backend Service port (name or number as string)
}

// DNSChainDrift walks the external DNS → Ingress → Service → Endpoints chain
// for every host the cluster claims to serve and emits a structured diagnostic
// naming the specific broken link when any layer fails.
//
// # Chain
//
//	Cloudflare DNS record for H  ──→ matches cluster ingress-controller LB IP
//	                                           │
//	                                           ▼
//	    Ingress with H in spec.rules[].host  ──→ backend Service N/S
//	                                           │
//	                                           ▼
//	                      Service N/S exists ──→ Endpoints has ready addresses
//
// Cloudflare integration is optional. When Client is nil the analyzer runs
// the K8s hops and emits a single info-level "external-dns-not-verified"
// diagnostic instead of the per-host Cloudflare checks.
//
// # Host discovery
//
// Every Ingress in every namespace contributes its spec.rules[].host values.
// Ingresses carrying the OptOutAnno annotation (value "true") are excluded.
// SeedTargets (typically probe.DefaultEndpointTargets()) are merged in and
// de-duped. Seed hosts with no matching Ingress are treated as
// "external-only" and only the Cloudflare layer is checked for them.
//
// # Fail-open contract
//
// Per pkg/diagnose.Analyzer contract, Run never returns an error. Any
// transient Cloudflare API failure causes the analyzer to emit the
// "external DNS hop not verified" info diagnostic (same as Client == nil)
// and continue with the K8s layers. In snapshot mode, Cloudflare HTTP calls
// are skipped entirely.
type DNSChainDrift struct {
	// Client is the Cloudflare API client. When nil the external DNS hop is
	// skipped and the analyzer emits an info-level summary diagnostic instead.
	Client CloudflareClient

	// SeedTargets is the static list of endpoint targets to merge with
	// Ingress-discovered hosts. Typically probe.DefaultEndpointTargets().
	// The analyzer extracts the hostname from each URL.
	SeedTargets []string

	// OptOutAnno is the Ingress annotation key whose value, when set to
	// "true" (case-insensitive), suppresses all hosts from that Ingress.
	// Defaults to "cha.bionicaisolutions.com/probe-disable".
	OptOutAnno string
}

// Name returns the analyzer's stable identifier.
func (DNSChainDrift) Name() string { return "DNSChainDrift" }

// Run executes the full DNS-chain analysis and returns diagnostics for every
// broken link found. Returns nil when there is nothing to report or when the
// cluster has no Ingress resources.
func (a DNSChainDrift) Run(ctx context.Context, src snapshot.Source) []Diagnostic {
	optOut := a.OptOutAnno
	if optOut == "" {
		optOut = "cha.bionicaisolutions.com/probe-disable"
	}

	// ── Step 1: collect all Ingress hosts and build the host→backend map ──────

	ingresses, err := src.List(ctx, snapshot.GVRIngress, "")
	if err != nil || ingresses == nil || len(ingresses.Items) == 0 {
		// CRD absent or RBAC denied — fail open, nothing to analyze.
		return nil
	}

	// hostBackend maps lowercased host → ingressBackend (first Ingress wins for
	// de-dup purposes; duplicates are flagged separately).
	hostBackend := make(map[string]*ingressBackend)
	// hostPathClaimants records WHICH (path, pathType) keys are claimed under
	// each host and by which Ingresses. Path-disjoint co-tenancy on a single
	// host is a legitimate pattern (oauth2-proxy on `/` + MCP on `/mcp`, or
	// HTTP webhooks + WebSocket split across two Ingresses for distinct Kong
	// plugins/timeouts) — flagging those as "duplicate hosts" is a false
	// positive. Real duplication = the SAME (path, pathType) pair appearing
	// on ≥2 Ingresses for the same host.
	type pathKey struct{ path, pathType string }
	hostPathClaimants := make(map[string]map[pathKey][]string) // host → key → list of "ns/name"
	// hostAllClaimants tracks every Ingress that lists this host on any rule,
	// independent of paths. Used for the duplicate message context.
	hostAllClaimants := make(map[string]map[string]struct{})

	for i := range ingresses.Items {
		ing := ingresses.Items[i]
		ns := ing.GetNamespace()
		ingName := ing.GetName()

		// Skip opted-out Ingresses.
		if v, ok := ing.GetAnnotations()[optOut]; ok && strings.EqualFold(v, "true") {
			continue
		}
		// Skip cert-manager ephemeral ACME challenge Ingresses.
		if strings.HasPrefix(ingName, "cm-acme-http-solver-") {
			continue
		}

		rules, _, _ := unstructured.NestedSlice(ing.Object, "spec", "rules")
		for _, rawRule := range rules {
			rm, ok := rawRule.(map[string]any)
			if !ok {
				continue
			}
			host, _ := rm["host"].(string)
			host = strings.ToLower(strings.TrimSpace(host))
			if host == "" {
				continue
			}

			claimant := ns + "/" + ingName
			if hostAllClaimants[host] == nil {
				hostAllClaimants[host] = map[string]struct{}{}
			}
			hostAllClaimants[host][claimant] = struct{}{}

			// Record every (path, pathType) this Ingress claims on this host
			// so the duplicate detector can find real collisions.
			httpMap, _ := rm["http"].(map[string]any)
			var pathsSlice []any
			if httpMap != nil {
				pathsSlice, _ = httpMap["paths"].([]any)
			}
			if hostPathClaimants[host] == nil {
				hostPathClaimants[host] = map[pathKey][]string{}
			}
			if len(pathsSlice) == 0 {
				// No paths under this rule — treat as "/" Prefix (default).
				k := pathKey{path: "/", pathType: "Prefix"}
				hostPathClaimants[host][k] = append(hostPathClaimants[host][k], claimant)
			}
			for _, rawPath := range pathsSlice {
				pm, ok := rawPath.(map[string]any)
				if !ok {
					continue
				}
				p, _ := pm["path"].(string)
				pt, _ := pm["pathType"].(string)
				if p == "" {
					p = "/"
				}
				if pt == "" {
					pt = "Prefix"
				}
				k := pathKey{path: p, pathType: pt}
				hostPathClaimants[host][k] = append(hostPathClaimants[host][k], claimant)
			}

			// Only record the first Ingress backend per host for chain analysis.
			if _, seen := hostBackend[host]; seen {
				continue
			}

			// Extract backend service from paths[0] (the first path is the
			// primary backend; multiple paths are handled by the Ingress itself).
			be := resolveIngressBackend(ing, ns, ingName, rm)
			hostBackend[host] = be
		}
	}

	// ── Step 2: merge seed targets (de-duped) ─────────────────────────────────

	// seedHostSet tracks which hosts came exclusively from SeedTargets and have
	// no matching Ingress (external-only hosts that CF checks still apply to).
	seedHostSet := make(map[string]struct{})
	for _, rawURL := range a.SeedTargets {
		h := hostFromURL(rawURL)
		if h == "" {
			continue
		}
		seedHostSet[h] = struct{}{}
		if _, inIngress := hostBackend[h]; !inIngress {
			// Seed host has no Ingress — external-only placeholder.
			hostBackend[h] = nil
		}
	}

	if len(hostBackend) == 0 {
		return nil
	}

	// ── Step 3: find ingress-controller LB IP ─────────────────────────────────

	lbIP := findIngressControllerLBIP(ctx, src)

	// ── Step 4: build Cloudflare record cache (when Client is set) ────────────

	// cfRecords maps lowercased hostname → slice of matching DNSRecords.
	// Built once per Run from a full zone dump — avoids per-host API calls and
	// stays within the CF rate limit (2 calls per zone, not N).
	var cfRecords map[string][]DNSRecord
	cfDisabled := a.Client == nil || src.Mode() == snapshot.ModeSnapshot
	if !cfDisabled {
		cfRecords, cfDisabled = buildCFRecordCache(ctx, a.Client)
	}

	// ── Step 5: analyze each host ─────────────────────────────────────────────

	var out []Diagnostic
	cfMissingHosts := 0

	// Iterate in a deterministic order: sort hosts for stable output.
	sortedHosts := sortedBackendKeys(hostBackend)

	for _, host := range sortedHosts {
		be := hostBackend[host]
		_, isExternalOnly := seedHostSet[host]
		// A seed host that also has an Ingress is NOT external-only.
		if be != nil {
			isExternalOnly = false
		}

		// Describe the K8s-side state for use in CF diagnostic messages.
		k8sSummary := buildK8sSummary(ctx, src, host, be)

		// ── 5a: Cloudflare layer ──────────────────────────────────────────────

		if cfDisabled {
			cfMissingHosts++
		} else {
			cfDiag, cfOK := checkCFLayer(host, cfRecords, lbIP, k8sSummary)
			if cfDiag != nil {
				out = append(out, *cfDiag)
				if !cfOK {
					// Missing record: the host isn't in CF at all — no point
					// checking K8s layers; the traffic can't arrive.
					continue
				}
			}
		}

		// External-only hosts: Cloudflare is the only layer we check.
		if isExternalOnly {
			continue
		}

		// ── 5b: Ingress layer ─────────────────────────────────────────────────

		if be == nil {
			// Host is on seed list but has no Ingress and is not external-only.
			out = append(out, Diagnostic{
				Subject:  subj("missing-ingress", host),
				Severity: "error",
				Source:   "DNSChainDrift",
				Message: fmt.Sprintf(
					"*%s* is in the static endpoint target list but has no Ingress in any namespace. "+
						"Create an Ingress with `spec.rules[].host=%s` pointing at the intended backend Service, "+
						"or mark this host external-only by removing it from the static target list.",
					host, host,
				),
				Remediation: fmt.Sprintf(
					"Create an Ingress for %s, e.g.:\n"+
						"  kubectl create ingress %s-ingress --rule=%s/*=<svc-name>:<port>",
					host, sanitizeForName(host), host,
				),
			})
			continue
		}

		// Duplicate Ingress hosts — warn ONLY when the SAME (path, pathType)
		// pair appears on ≥2 Ingresses. Co-tenancy where each Ingress claims
		// distinct paths under one host is a legitimate pattern (Kong path-
		// based routing, MCP gateway fan-out, oauth2-proxy on `/` + bypass
		// on `/<route>`), not a misconfig. Flagging those caused
		// false-positive noise on production multi-tenant hosts pre-1.10.4.
		if pathMap, ok := hostPathClaimants[host]; ok {
			var colliding []string // human-readable "/path (pathType) ↔ ns1/ing-a + ns2/ing-b"
			for k, ings := range pathMap {
				if len(ings) <= 1 {
					continue
				}
				// dedupe + stable order so the message is deterministic
				seen := map[string]struct{}{}
				uniq := make([]string, 0, len(ings))
				for _, n := range ings {
					if _, dup := seen[n]; dup {
						continue
					}
					seen[n] = struct{}{}
					uniq = append(uniq, n)
				}
				if len(uniq) <= 1 {
					continue
				}
				sort.Strings(uniq)
				colliding = append(colliding,
					fmt.Sprintf("`%s` (%s) on %s", k.path, k.pathType, strings.Join(uniq, " + ")))
			}
			sort.Strings(colliding)
			if len(colliding) > 0 {
				out = append(out, Diagnostic{
					Subject:  subj("duplicate-ingress-host", host),
					Severity: "warn",
					Source:   "DNSChainDrift",
					Message: fmt.Sprintf(
						"*%s* has %d colliding path(s) claimed by multiple Ingresses: %s. "+
							"Pick one Ingress per (path, pathType); the router otherwise picks non-deterministically.",
						host, len(colliding), strings.Join(colliding, "; "),
					),
				})
			}
		}

		// Ingress references a Service — verify the Service exists.
		if be.svcName != "" {
			svc, svcErr := src.Get(ctx, snapshot.GVRService, be.svcNs, be.svcName)
			if svcErr != nil || svc == nil {
				out = append(out, Diagnostic{
					Subject:  subj("ingress-orphan-service", be.ns+"/"+be.ingName+"/"+host),
					Severity: "error",
					Source:   "DNSChainDrift",
					Message: fmt.Sprintf(
						"Ingress `%s/%s` routes host *%s* to Service `%s/%s` (port %s) "+
							"but that Service does not exist in the cluster.",
						be.ns, be.ingName, host, be.svcNs, be.svcName, be.svcPort,
					),
					Remediation: fmt.Sprintf(
						"Either deploy Service `%s/%s`, or update Ingress `%s` to reference an existing Service.\n"+
							"To list Services in the namespace: kubectl -n %s get svc",
						be.svcNs, be.svcName, be.ingName, be.svcNs,
					),
				})
				continue
			}

			// ── 5c: Service type check ────────────────────────────────────────

			svcType, _, _ := unstructured.NestedString(svc.Object, "spec", "type")
			if strings.EqualFold(svcType, "ExternalName") {
				out = append(out, Diagnostic{
					Subject:  subj("service-external-name-mismatch", be.svcNs+"/"+be.svcName+"/"+host),
					Severity: "warn",
					Source:   "DNSChainDrift",
					Message: fmt.Sprintf(
						"Ingress `%s/%s` for host *%s* references Service `%s/%s` "+
							"which has `spec.type=ExternalName`. "+
							"ExternalName Services proxy to an external DNS name, not in-cluster pods; "+
							"the Ingress may not behave as expected.",
						be.ns, be.ingName, host, be.svcNs, be.svcName,
					),
				})
				// Don't check Endpoints for ExternalName services — they have none.
				continue
			}

			// ── 5d: Endpoints layer ───────────────────────────────────────────

			ep, epErr := src.Get(ctx, snapshot.GVREndpoints, be.svcNs, be.svcName)
			if epErr != nil || ep == nil || !hasReadyEndpoints(ep) {
				out = append(out, Diagnostic{
					Subject:  subj("service-no-endpoints", be.svcNs+"/"+be.svcName+"/"+host),
					Severity: "error",
					Source:   "DNSChainDrift",
					Message: fmt.Sprintf(
						"Service `%s/%s` (backing host *%s* via Ingress `%s/%s`) "+
							"has zero ready endpoint addresses. "+
							"Traffic to %s will be dropped at the Service layer.",
						be.svcNs, be.svcName, host, be.ns, be.ingName, host,
					),
					Remediation: fmt.Sprintf(
						"Check the Deployment/StatefulSet whose selector matches Service `%s/%s`:\n"+
							"  kubectl -n %s get pods -l $(kubectl -n %s get svc %s -o jsonpath='{.spec.selector}' 2>/dev/null)\n"+
							"Look for CrashLoopBackOff, Pending, or failed readiness probes. "+
							"The WorkloadStateDrift analyzer may have additional detail.",
						be.svcNs, be.svcName, be.svcNs, be.svcNs, be.svcName,
					),
				})
			}
		}
	}

	// ── Step 6: emit "CF disabled" summary when N > 0 hosts were skipped ──────

	if cfMissingHosts > 0 {
		out = append(out, Diagnostic{
			Subject:  "DNSChainDrift/external-dns-not-verified",
			Severity: "info",
			Source:   "DNSChainDrift",
			Message: fmt.Sprintf(
				"Cloudflare credentials not configured; external DNS hop not checked for %d host(s). "+
					"Set `CHA_CLOUDFLARE_API_TOKEN` (and optionally `CHA_CLOUDFLARE_ZONE_ID`) "+
					"to enable the full DNS-chain analysis including the Cloudflare layer.",
				cfMissingHosts,
			),
		})
	}

	return out
}

// ── helpers ───────────────────────────────────────────────────────────────────

// subj builds the canonical Subject field for a DNSChainDrift diagnostic.
// Format: "DNSChainDrift/<category>/<detail>"
func subj(category, detail string) string {
	return "DNSChainDrift/" + category + "/" + detail
}

// hostFromURL extracts the hostname component from a raw URL string.
// Returns "" on parse failure or when the hostname is empty.
func hostFromURL(rawURL string) string {
	if rawURL == "" {
		return ""
	}
	// Fast path: no scheme present — treat entire string as a hostname.
	if !strings.Contains(rawURL, "://") {
		return strings.ToLower(strings.TrimSpace(rawURL))
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	h := u.Hostname()
	return strings.ToLower(strings.TrimSpace(h))
}

// resolveIngressBackend pulls the first backend Service reference out of an
// Ingress rule. Returns an ingressBackend with svcName="" when no backend
// can be resolved (e.g. default-backend only Ingress, or non-standard rule shape).
func resolveIngressBackend(ing unstructured.Unstructured, ns, ingName string, rule map[string]any) *ingressBackend {
	be := &ingressBackend{
		ns:      ns,
		ingName: ingName,
		svcNs:   ns,
	}

	// networking.k8s.io/v1 Ingress: spec.rules[].http.paths[].backend.service
	http, ok := rule["http"].(map[string]any)
	if !ok {
		return be
	}
	paths, ok := http["paths"].([]any)
	if !ok || len(paths) == 0 {
		return be
	}
	path0, ok := paths[0].(map[string]any)
	if !ok {
		return be
	}
	backend, ok := path0["backend"].(map[string]any)
	if !ok {
		return be
	}
	// networking.k8s.io/v1 shape: backend.service.{name,port.{name|number}}
	svcMap, ok := backend["service"].(map[string]any)
	if ok {
		be.svcName, _ = svcMap["name"].(string)
		if portMap, ok := svcMap["port"].(map[string]any); ok {
			if portName, ok := portMap["name"].(string); ok && portName != "" {
				be.svcPort = portName
			} else if portNum, ok := portMap["number"].(int64); ok {
				be.svcPort = fmt.Sprintf("%d", portNum)
			}
		}
		return be
	}
	// extensions/v1beta1 shape: backend.{serviceName,servicePort} (legacy clusters).
	if svcName, ok := backend["serviceName"].(string); ok {
		be.svcName = svcName
	}
	if svcPort, ok := backend["servicePort"].(string); ok {
		be.svcPort = svcPort
	} else if portNum, ok := backend["servicePort"].(int64); ok {
		be.svcPort = fmt.Sprintf("%d", portNum)
	}
	return be
}

// findIngressControllerLBIP lists Services in all namespaces and returns the
// first LoadBalancer IP found on a Service that matches common ingress
// controller label selectors (Kong, NGINX, Traefik). Returns "" when not found.
func findIngressControllerLBIP(ctx context.Context, src snapshot.Source) string {
	svcs, err := src.List(ctx, snapshot.GVRService, "")
	if err != nil || svcs == nil {
		return ""
	}
	for i := range svcs.Items {
		svc := svcs.Items[i]
		labels := svc.GetLabels()
		if !isIngressControllerService(labels) {
			continue
		}
		ip := extractLBIP(svc)
		if ip != "" {
			return ip
		}
	}
	return ""
}

// isIngressControllerService returns true when the label map matches a
// known ingress controller (Kong, NGINX, Traefik).
func isIngressControllerService(labels map[string]string) bool {
	if v, ok := labels["app.kubernetes.io/name"]; ok {
		switch strings.ToLower(v) {
		case "ingress-kong", "kong", "ingress-nginx", "nginx-ingress", "traefik":
			return true
		}
	}
	if v, ok := labels["app"]; ok {
		switch strings.ToLower(v) {
		case "ingress-kong", "kong", "ingress-nginx", "nginx", "traefik":
			return true
		}
	}
	return false
}

// extractLBIP reads status.loadBalancer.ingress[0].ip from a Service.
func extractLBIP(svc unstructured.Unstructured) string {
	ingresses, _, _ := unstructured.NestedSlice(svc.Object, "status", "loadBalancer", "ingress")
	if len(ingresses) == 0 {
		return ""
	}
	first, ok := ingresses[0].(map[string]any)
	if !ok {
		return ""
	}
	ip, _ := first["ip"].(string)
	return strings.TrimSpace(ip)
}

// buildCFRecordCache fetches all zones and their DNS records from Cloudflare
// and returns a hostname→records map. On any API error it returns (nil, true)
// which signals cfDisabled=true (fail-open). The bool return value indicates
// whether the caller should treat CF as disabled.
func buildCFRecordCache(ctx context.Context, client CloudflareClient) (map[string][]DNSRecord, bool) {
	zones, err := client.ListZones(ctx)
	if err != nil {
		// Fail open: treat as CF disabled.
		return nil, true
	}

	cache := make(map[string][]DNSRecord)
	for _, z := range zones {
		records, err := client.ListDNSRecords(ctx, z.ID)
		if err != nil {
			// One zone failed — skip it, not the whole cache.
			continue
		}
		for _, r := range records {
			name := strings.ToLower(r.Name)
			cache[name] = append(cache[name], r)
		}
	}
	return cache, false
}

// checkCFLayer checks the Cloudflare DNS record for a single host.
// Returns:
//   - (nil, true) when the record exists and matches — no finding, chain continues.
//   - (*Diagnostic, false) when the record is completely absent — emit error, chain stops.
//   - (*Diagnostic, true) when the record points elsewhere — emit error, chain continues.
func checkCFLayer(host string, cfRecords map[string][]DNSRecord, lbIP, k8sSummary string) (*Diagnostic, bool) {
	records, found := cfRecords[host]
	if !found || len(records) == 0 {
		msg := fmt.Sprintf(
			"*%s*: no Cloudflare DNS record found. "+
				"The host is not reachable from the internet until a DNS record is created.",
			host,
		)
		if k8sSummary != "" {
			msg += " " + k8sSummary
		}
		rem := fmt.Sprintf(
			"Add `%s` to `deploy/lib/dns.sh` with target `%s` and re-run the deploy step.\n"+
				"Suggested record: A %s → %s (proxied=false).",
			host, lbIP, host, lbIP,
		)
		return &Diagnostic{
			Subject:     subj("missing-cloudflare-record", host),
			Severity:    "error",
			Source:      "DNSChainDrift",
			Message:     msg,
			Remediation: rem,
		}, false
	}

	// Check whether any record points at the expected target (lbIP).
	if lbIP == "" {
		// Can't auto-derive expected target — skip value comparison.
		return nil, true
	}

	for _, r := range records {
		if r.Content == lbIP {
			return nil, true // at least one record matches
		}
	}

	// Record(s) exist but none match the expected LB IP.
	observed := make([]string, 0, len(records))
	for _, r := range records {
		observed = append(observed, fmt.Sprintf("%s→%s", r.Type, r.Content))
	}
	return &Diagnostic{
		Subject:  subj("cloudflare-points-elsewhere", host),
		Severity: "error",
		Source:   "DNSChainDrift",
		Message: fmt.Sprintf(
			"*%s*: Cloudflare DNS record exists but points elsewhere. "+
				"Observed: [%s]. Expected target (ingress-controller LB): `%s`.",
			host, strings.Join(observed, ", "), lbIP,
		),
		Remediation: fmt.Sprintf(
			"Update the Cloudflare record for `%s` to point at `%s`, "+
				"or update `expectedTargets` in the CHA config if the current target is intentional.\n"+
				"File to update: `deploy/lib/dns.sh`.",
			host, lbIP,
		),
	}, true // chain continues — K8s layers may still reveal issues
}

// buildK8sSummary produces a short inline description of the K8s-side health
// for use in CF-layer diagnostic messages. Returns "" when the backend is nil
// (external-only) or when the Service doesn't exist.
func buildK8sSummary(ctx context.Context, src snapshot.Source, host string, be *ingressBackend) string {
	if be == nil || be.svcName == "" {
		return ""
	}
	svc, err := src.Get(ctx, snapshot.GVRService, be.svcNs, be.svcName)
	if err != nil || svc == nil {
		return fmt.Sprintf("Ingress `%s/%s` exists but references a missing Service `%s/%s`.",
			be.ns, be.ingName, be.svcNs, be.svcName)
	}
	ep, epErr := src.Get(ctx, snapshot.GVREndpoints, be.svcNs, be.svcName)
	if epErr != nil || ep == nil || !hasReadyEndpoints(ep) {
		return fmt.Sprintf("Ingress `%s/%s` exists and points at Service `%s/%s` (zero ready endpoints).",
			be.ns, be.ingName, be.svcNs, be.svcName)
	}
	count := readyAddressCount(ep)
	return fmt.Sprintf("Ingress `%s/%s` exists and points at Service `%s/%s` (%d ready endpoint(s)) — the cluster side is healthy; only the external DNS hop is missing.",
		be.ns, be.ingName, be.svcNs, be.svcName, count)
}

// hasReadyEndpoints returns true when the Endpoints object has at least one
// ready address across all subsets.
func hasReadyEndpoints(ep *unstructured.Unstructured) bool {
	return readyAddressCount(ep) > 0
}

// readyAddressCount counts the total ready addresses across all Endpoints subsets.
func readyAddressCount(ep *unstructured.Unstructured) int {
	subsets, _, _ := unstructured.NestedSlice(ep.Object, "subsets")
	total := 0
	for _, raw := range subsets {
		sm, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		addrs, _, _ := unstructured.NestedSlice(sm, "addresses")
		total += len(addrs)
	}
	return total
}

// sortedBackendKeys returns the keys of a host→ingressBackend map sorted for
// deterministic iteration. Named distinctly from the package-wide sortedKeys
// (which operates on map[string]struct{}) to avoid a redeclaration conflict.
func sortedBackendKeys(m map[string]*ingressBackend) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	// Simple insertion sort — the map is bounded by cluster Ingress count (~hundreds).
	for i := 1; i < len(keys); i++ {
		for j := i; j > 0 && keys[j] < keys[j-1]; j-- {
			keys[j], keys[j-1] = keys[j-1], keys[j]
		}
	}
	return keys
}

// sanitizeForName converts a hostname to a valid Kubernetes resource name
// fragment (replaces dots with dashes, strips invalid chars).
func sanitizeForName(host string) string {
	return strings.NewReplacer(".", "-", "_", "-").Replace(host)
}
