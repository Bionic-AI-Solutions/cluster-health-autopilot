// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

// Package investigator implements the read-only investigator tier (Layer 2)
// of the CHA pipeline. It exposes a concrete Environment backed by net/http,
// net.Resolver, crypto/tls, and the watcher's snapshot.Source, plus a
// deterministic rule-based Investigator that pattern-matches the failure
// mode and runs the appropriate tools.
//
// The LLM-backed Investigator implementation lives in the paid CHA-com
// binary; both implementations share the same pkg/ai.Investigator interface
// and Environment surface.
package investigator

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/internal/snapshot"
	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/pkg/ai"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
)

// LiveEnvironment is the production Environment implementation. It wires
// net/http and crypto/tls for the network tools and a snapshot.Source for
// the kubectl-style reads. The same instance is reused across investigations.
type LiveEnvironment struct {
	src      snapshot.Source
	resolver *net.Resolver
	// logs is the typed clientset used by Logs() to stream pod logs. nil in
	// snapshot mode (and when no clientset was threaded through), in which
	// case Logs() returns a soft error instead of streaming.
	logs kubernetes.Interface
}

// NewLiveEnvironment constructs an Environment using the given Source for
// kubectl-style reads. The Source is typically the live SnapshotSource of
// the watcher; snapshot-mode callers may pass a File source and the
// network tools will still work (against the real network — there is no
// offline simulation of HTTP/DNS).
//
// Pod-logs access is disabled (Logs() returns a soft error). Use
// NewLiveEnvironmentWithLogs to enable log-based investigation.
func NewLiveEnvironment(src snapshot.Source) *LiveEnvironment {
	return &LiveEnvironment{
		src:      src,
		resolver: net.DefaultResolver,
	}
}

// NewLiveEnvironmentWithLogs is NewLiveEnvironment plus a typed clientset so
// Logs() can stream container logs (the capability that lets the investigator
// find the actual crash cause). A nil logsClient degrades to the no-logs
// behaviour of NewLiveEnvironment.
func NewLiveEnvironmentWithLogs(src snapshot.Source, logsClient kubernetes.Interface) *LiveEnvironment {
	e := NewLiveEnvironment(src)
	e.logs = logsClient
	return e
}

var _ ai.Environment = (*LiveEnvironment)(nil)

// DNSLookup resolves host using the cluster resolver and returns timing.
func (e *LiveEnvironment) DNSLookup(ctx context.Context, host string) (ai.DNSResult, error) {
	host = strings.TrimSpace(host)
	if host == "" {
		return ai.DNSResult{}, fmt.Errorf("empty host")
	}
	start := time.Now()
	addrs, err := e.resolver.LookupHost(ctx, host)
	r := ai.DNSResult{Host: host, Elapsed: time.Since(start)}
	if err != nil {
		r.Error = err.Error()
		return r, nil // soft-fail — surface error to caller, not Go error
	}
	r.Addresses = addrs
	return r, nil
}

// HTTPProbe performs one HTTP request and captures detailed timing.
func (e *LiveEnvironment) HTTPProbe(ctx context.Context, target string, opts ai.HTTPProbeOpts) (ai.HTTPProbeResult, error) {
	method := opts.Method
	if method == "" {
		method = http.MethodGet
	}
	timeout := opts.Timeout
	if timeout == 0 {
		timeout = 5 * time.Second
	}
	followRedirects := true
	if opts.FollowRedirects != nil {
		followRedirects = *opts.FollowRedirects
	}

	r := ai.HTTPProbeResult{URL: target, Method: method}
	parsed, err := url.Parse(target)
	if err != nil {
		r.Error = "invalid URL: " + err.Error()
		return r, nil
	}
	_ = parsed // sanity-check only; net/http handles parsing too

	var dnsStart, connStart, tlsStart time.Time
	trace := &httptrace.ClientTrace{
		DNSStart:          func(httptrace.DNSStartInfo) { dnsStart = time.Now() },
		DNSDone:           func(httptrace.DNSDoneInfo) { r.DNSTime = time.Since(dnsStart) },
		ConnectStart:      func(network, addr string) { connStart = time.Now() },
		ConnectDone:       func(network, addr string, err error) { r.ConnectTime = time.Since(connStart) },
		TLSHandshakeStart: func() { tlsStart = time.Now() },
		TLSHandshakeDone:  func(tls.ConnectionState, error) { r.TLSTime = time.Since(tlsStart) },
	}
	ctx = httptrace.WithClientTrace(ctx, trace)
	reqCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, method, target, nil)
	if err != nil {
		r.Error = "request build: " + err.Error()
		return r, nil
	}
	req.Header.Set("User-Agent", "cha-investigator/1.0")

	client := &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: opts.InsecureSkipVerify},
		},
	}
	if !followRedirects {
		client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	}

	start := time.Now()
	resp, err := client.Do(req)
	r.ResponseTime = time.Since(start)
	if err != nil {
		r.Error = err.Error()
		return r, nil
	}
	defer func() { _ = resp.Body.Close() }()

	r.StatusCode = resp.StatusCode
	r.FinalURL = resp.Request.URL.String()
	r.TLSVerified = resp.TLS != nil && len(resp.TLS.VerifiedChains) > 0
	// Capture a small, useful header subset; never the full Set-Cookie.
	r.Headers = map[string]string{}
	for _, key := range []string{"Server", "Content-Type", "Location", "X-Powered-By", "Strict-Transport-Security"} {
		if v := resp.Header.Get(key); v != "" {
			r.Headers[key] = v
		}
	}
	return r, nil
}

// TLSInspect dials host:port and inspects the served certificate chain.
// Skips TLS verification at dial time so the function can describe the
// cert even when it would normally be rejected (e.g. expired, SAN mismatch).
func (e *LiveEnvironment) TLSInspect(ctx context.Context, host string, port int) (ai.TLSResult, error) {
	host = strings.TrimSpace(host)
	if port == 0 {
		port = 443
	}
	r := ai.TLSResult{Host: host, Port: port}
	if host == "" {
		r.HandshakeErr = "empty host"
		return r, nil
	}

	dialer := &net.Dialer{Timeout: 5 * time.Second}
	start := time.Now()
	addr := net.JoinHostPort(host, strconv.Itoa(port))
	rawConn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		r.HandshakeErr = "tcp dial: " + err.Error()
		r.Elapsed = time.Since(start)
		return r, nil
	}
	tlsConn := tls.Client(rawConn, &tls.Config{
		ServerName:         host,
		InsecureSkipVerify: true, // we WANT to see broken certs
	})
	defer func() { _ = tlsConn.Close() }()
	if err := tlsConn.HandshakeContext(ctx); err != nil {
		r.HandshakeErr = err.Error()
		r.Elapsed = time.Since(start)
		return r, nil
	}
	state := tlsConn.ConnectionState()
	r.Elapsed = time.Since(start)
	if len(state.PeerCertificates) == 0 {
		r.HandshakeErr = "peer returned no certificates"
		return r, nil
	}
	leaf := state.PeerCertificates[0]
	r.Subject = leaf.Subject.String()
	r.Issuer = leaf.Issuer.CommonName
	r.SANs = leaf.DNSNames
	r.NotBefore = leaf.NotBefore
	r.NotAfter = leaf.NotAfter
	r.Expired = time.Now().After(leaf.NotAfter)
	// Host-match check honors SAN; matches the rules net/http uses.
	r.HostMatches = leaf.VerifyHostname(host) == nil
	for _, c := range state.PeerCertificates {
		r.ChainSummary = append(r.ChainSummary,
			fmt.Sprintf("%s (issued by %s, expires %s)",
				c.Subject.CommonName, c.Issuer.CommonName,
				c.NotAfter.Format("2006-01-02")))
	}
	return r, nil
}

// Describe returns a compact human-readable description for one resource.
// Uses snapshot.Source — works in both live and snapshot mode.
func (e *LiveEnvironment) Describe(ctx context.Context, kind, namespace, name string) (ai.DescribeResult, error) {
	r := ai.DescribeResult{Kind: kind, Namespace: namespace, Name: name}
	gvr, ok := kindToGVR(kind)
	if !ok {
		r.Error = "unknown kind: " + kind
		return r, nil
	}
	obj, err := e.src.Get(ctx, gvr, namespace, name)
	if err != nil || obj == nil {
		r.Error = fmt.Sprintf("not found: %s/%s/%s", kind, namespace, name)
		return r, nil
	}
	r.Labels = obj.GetLabels()
	r.Annotations = obj.GetAnnotations()
	r.Status, r.Reason, r.Message = readCommonStatus(obj)
	r.Notes = readSpecHighlights(obj, kind)
	return r, nil
}

// GetEvents returns recent events involving the given object, newest-first.
func (e *LiveEnvironment) GetEvents(ctx context.Context, namespace, kind, name string, since time.Duration) ([]ai.EventInfo, error) {
	events, err := e.src.List(ctx, snapshot.GVREvent, namespace)
	if err != nil || events == nil {
		return nil, nil
	}
	cutoff := time.Time{}
	if since > 0 {
		cutoff = time.Now().Add(-since)
	}
	out := []ai.EventInfo{}
	for i := range events.Items {
		ev := events.Items[i]
		invKind, _, _ := unstructured.NestedString(ev.Object, "involvedObject", "kind")
		invName, _, _ := unstructured.NestedString(ev.Object, "involvedObject", "name")
		if !strings.EqualFold(invKind, kind) || invName != name {
			continue
		}
		typ, _, _ := unstructured.NestedString(ev.Object, "type")
		reason, _, _ := unstructured.NestedString(ev.Object, "reason")
		msg, _, _ := unstructured.NestedString(ev.Object, "message")
		count, _, _ := unstructured.NestedInt64(ev.Object, "count")
		lastSeenStr, _, _ := unstructured.NestedString(ev.Object, "lastTimestamp")
		firstSeenStr, _, _ := unstructured.NestedString(ev.Object, "firstTimestamp")
		lastSeen, _ := time.Parse(time.RFC3339, lastSeenStr)
		firstSeen, _ := time.Parse(time.RFC3339, firstSeenStr)
		if !cutoff.IsZero() && !lastSeen.IsZero() && lastSeen.Before(cutoff) {
			continue
		}
		src, _, _ := unstructured.NestedString(ev.Object, "source", "component")
		out = append(out, ai.EventInfo{
			Type: typ, Reason: reason, Message: msg, Count: int32(count),
			FirstSeen: firstSeen, LastSeen: lastSeen, Source: src,
		})
	}
	// newest first
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	// Scrub secrets / IPs / hostnames from event messages BEFORE they
	// can reach an LLM-backed investigator. This is one layer of
	// defense; the LLM prompt itself also wraps observed_data in
	// untrusted markers.
	return ai.RedactEvents(out), nil
}

// kindToGVR maps the human-friendly kind names the investigator uses to the
// schema.GroupVersionResource records the snapshot.Source indexes by. Keep
// in sync with internal/snapshot/source.go's GVR constants.
func kindToGVR(kind string) (schema.GroupVersionResource, bool) {
	switch strings.ToLower(kind) {
	case "pod":
		return snapshot.GVRPod, true
	case "deployment":
		return snapshot.GVRDeployment, true
	case "replicaset":
		return snapshot.GVRReplicaSet, true
	case "statefulset":
		return snapshot.GVRStatefulSet, true
	case "job":
		return snapshot.GVRJob, true
	case "cronjob":
		return snapshot.GVRCronJob, true
	case "service":
		// no explicit GVR const; fall through
		return schema.GroupVersionResource{Group: "", Version: "v1", Resource: "services"}, true
	case "ingress":
		return snapshot.GVRIngress, true
	case "secret":
		return snapshot.GVRSecret, true
	case "externalsecret":
		return snapshot.GVRExtSecret, true
	case "certificate":
		return snapshot.GVRCertificate, true
	case "certificaterequest":
		return snapshot.GVRCertificateRequest, true
	case "node":
		return snapshot.GVRNode, true
	}
	return schema.GroupVersionResource{}, false
}

// readCommonStatus extracts a status/reason/message triple from common
// Kubernetes status patterns. Returns empty strings when not present.
func readCommonStatus(obj *unstructured.Unstructured) (status, reason, msg string) {
	// status.phase (pods, PVCs)
	if v, _, _ := unstructured.NestedString(obj.Object, "status", "phase"); v != "" {
		status = v
	}
	// status.conditions[Ready] (most controller-managed resources)
	conds, _, _ := unstructured.NestedSlice(obj.Object, "status", "conditions")
	for _, raw := range conds {
		cm, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		if cm["type"] == "Ready" {
			if s, ok := cm["status"].(string); ok && s != "" {
				status = "Ready=" + s
			}
			if r, ok := cm["reason"].(string); ok && r != "" {
				reason = r
			}
			if m, ok := cm["message"].(string); ok && m != "" {
				msg = m
			}
			break
		}
	}
	// Top-level status.reason / status.message — where rejected / Failed-phase
	// pods record the cause (admission + scheduling failures, e.g. "Pod was
	// rejected: Allocate failed ... nvidia.com/gpu unavailable"). These do NOT
	// appear in a Ready condition, so the investigator would otherwise miss them.
	if reason == "" {
		reason, _, _ = unstructured.NestedString(obj.Object, "status", "reason")
	}
	if msg == "" {
		msg, _, _ = unstructured.NestedString(obj.Object, "status", "message")
	}
	return
}

// readSpecHighlights produces a short list of kind-specific spec callouts.
// Designed to fit in a Slack code block — keeps the Describe output skimmable.
func readSpecHighlights(obj *unstructured.Unstructured, kind string) []string {
	notes := []string{}
	switch strings.ToLower(kind) {
	case "pod":
		// container images + restart counts
		containers, _, _ := unstructured.NestedSlice(obj.Object, "spec", "containers")
		for _, raw := range containers {
			cm, _ := raw.(map[string]any)
			n, _ := cm["name"].(string)
			img, _ := cm["image"].(string)
			notes = append(notes, fmt.Sprintf("container %s: %s", n, img))
		}
		cs, _, _ := unstructured.NestedSlice(obj.Object, "status", "containerStatuses")
		for _, raw := range cs {
			cm, _ := raw.(map[string]any)
			n, _ := cm["name"].(string)
			restartCount, _ := cm["restartCount"].(int64)
			if restartCount > 0 {
				notes = append(notes, fmt.Sprintf("container %s restarts: %d", n, restartCount))
			}
			if waiting, ok := cm["state"].(map[string]any)["waiting"].(map[string]any); ok && waiting != nil {
				reason, _ := waiting["reason"].(string)
				wmsg, _ := waiting["message"].(string)
				if reason != "" {
					note := fmt.Sprintf("container %s waiting: %s", n, reason)
					if wmsg != "" {
						// The detailed pull / config error ("pull access denied,
						// repository does not exist …") persists here even after
						// the Failed event ages out.
						note += " — " + wmsg
					}
					notes = append(notes, note)
				}
			}
		}
	case "ingress":
		rules, _, _ := unstructured.NestedSlice(obj.Object, "spec", "rules")
		for _, raw := range rules {
			rm, _ := raw.(map[string]any)
			h, _ := rm["host"].(string)
			if h != "" {
				notes = append(notes, "host: "+h)
			}
		}
		tls, _, _ := unstructured.NestedSlice(obj.Object, "spec", "tls")
		for _, raw := range tls {
			tm, _ := raw.(map[string]any)
			s, _ := tm["secretName"].(string)
			if s != "" {
				notes = append(notes, "tls secret: "+s)
			}
		}
	case "certificate":
		s, _, _ := unstructured.NestedString(obj.Object, "spec", "secretName")
		if s != "" {
			notes = append(notes, "target secret: "+s)
		}
		dns, _, _ := unstructured.NestedStringSlice(obj.Object, "spec", "dnsNames")
		if len(dns) > 0 {
			notes = append(notes, "dnsNames: "+strings.Join(dns, ", "))
		}
	}
	return notes
}

// Logs streams the tail of a pod container's logs (kubectl logs [--previous]).
// Returns a soft LogsResult.Error (never a hard error for the common cases) so
// the investigation pass degrades gracefully when logs aren't available:
//   - no clientset wired (snapshot mode) -> "pod logs unavailable ..."
//   - container never started / no previous instance -> API error surfaced
//     in LogsResult.Error
//
// Lines are redacted via ai.RedactEventMessage before return so secrets in
// log output never reach a prompt or a Slack post.
func (e *LiveEnvironment) Logs(ctx context.Context, namespace, pod string, opts ai.LogsOptions) (ai.LogsResult, error) {
	return ai.FetchPodLogs(ctx, e.logs, namespace, pod, opts), nil
}

// LatestByPrefix lists objects of `kind` in the namespace and returns the name
// of the most-recently-created one whose name starts with prefix. For pods it
// prefers a not-Running/Succeeded (failed) instance; for other kinds it takes
// the newest.
func (e *LiveEnvironment) LatestByPrefix(ctx context.Context, kind, namespace, prefix string) (string, error) {
	gvr, ok := kindToGVR(kind)
	if !ok {
		return "", nil
	}
	list, err := e.src.List(ctx, gvr, namespace)
	if err != nil || list == nil {
		return "", nil // soft-fail: investigation degrades to events
	}
	isPod := strings.EqualFold(kind, "pod")
	var bestName, bestTS string
	var bestFailed bool
	for i := range list.Items {
		o := &list.Items[i]
		name := o.GetName()
		if !strings.HasPrefix(name, prefix) {
			continue
		}
		failed := true
		if isPod {
			phase, _, _ := unstructured.NestedString(o.Object, "status", "phase")
			failed = phase != "Running" && phase != "Succeeded"
		}
		ts := o.GetCreationTimestamp().UTC().Format(time.RFC3339Nano)
		if bestName == "" ||
			(failed && !bestFailed) ||
			(failed == bestFailed && ts > bestTS) {
			bestName, bestTS, bestFailed = name, ts, failed
		}
	}
	return bestName, nil
}
