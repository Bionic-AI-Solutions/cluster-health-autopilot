// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package snapshot

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"k8s.io/apimachinery/pkg/runtime/schema"
)

// CaptureGVRs is the canonical resource set the offline diagnose mode
// needs. Anything not in this list is invisible to probes/analyzers.
//
// Order is stable so file layouts in tarballs are reproducible.
// Note on GVRSecret: we list Secrets so the proactive L5 analyzer can see
// which keys exist on each Secret, but `cha snapshot capture` deliberately
// does NOT include them — capturing Secret values to disk would be a
// privacy regression (any auditor reading the snapshot tarball would see
// every secret in the cluster). Live mode reads Secrets directly and
// inspects only the key NAMES, never the byte values.
var CaptureGVRs = []schema.GroupVersionResource{
	GVRPod,
	GVRNode,
	GVRPVC,
	GVREvent,
	GVRDeployment,
	GVRReplicaSet,
	GVRJob,
	GVRCronJob,
	GVRExtSecret,
	GVRCNPGCluster,
	GVRCephCluster,
	// GVRSecret intentionally excluded — see comment above.
}

// CaptureSummary records what a Capture call wrote.
type CaptureSummary struct {
	OutDir string         `json:"outDir"`
	Items  []CapturedFile `json:"items"`
}

// CapturedFile is one JSON file written during capture.
type CapturedFile struct {
	GVR     string `json:"gvr"`
	Path    string `json:"path"`
	Items   int    `json:"items"`
	SkipErr string `json:"skipErr,omitempty"`
}

// Capture lists every resource in CaptureGVRs from src and writes each as a
// kubectl-style JSON List into outDir. The directory is created if missing.
//
// Resources whose CRD is not installed (List returns 0 items) are still
// written as empty Lists — this lets the offline loader handle the absence
// gracefully and lets users diff snapshots over time even when CRDs come
// and go.
//
// Errors on individual resources do not abort the whole capture; they are
// recorded per-file in the returned CaptureSummary so the caller can decide
// how to surface them.
func Capture(ctx context.Context, src Source, outDir string) (*CaptureSummary, error) {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return nil, fmt.Errorf("create %q: %w", outDir, err)
	}
	out := &CaptureSummary{OutDir: outDir}
	for _, gvr := range CaptureGVRs {
		fname := filenameFor(gvr)
		path := filepath.Join(outDir, fname)
		entry := CapturedFile{GVR: gvrString(gvr), Path: path}

		list, err := src.List(ctx, gvr, "")
		if err != nil {
			entry.SkipErr = err.Error()
			out.Items = append(out.Items, entry)
			continue
		}
		entry.Items = len(list.Items)

		// Marshal manually so the file matches `kubectl get -o json` shape
		// (apiVersion + kind + items[]) — the same shape our offline loader
		// already consumes.
		fh, err := os.Create(path)
		if err != nil {
			entry.SkipErr = err.Error()
			out.Items = append(out.Items, entry)
			continue
		}
		enc := json.NewEncoder(fh)
		enc.SetIndent("", "  ")
		if err := enc.Encode(list); err != nil {
			_ = fh.Close()
			entry.SkipErr = err.Error()
			out.Items = append(out.Items, entry)
			continue
		}
		if err := fh.Close(); err != nil {
			entry.SkipErr = err.Error()
		}
		out.Items = append(out.Items, entry)
	}
	return out, nil
}

// CaptureTarGZ is Capture + tar.gz: writes the files into a temp dir, then
// gzips the directory into outPath. The temp dir is removed on success.
func CaptureTarGZ(ctx context.Context, src Source, outPath string) (*CaptureSummary, error) {
	tmp, err := os.MkdirTemp("", "cha-capture-*")
	if err != nil {
		return nil, err
	}
	defer func() { _ = os.RemoveAll(tmp) }()

	summary, err := Capture(ctx, src, tmp)
	if err != nil {
		return nil, err
	}

	out, err := os.Create(outPath)
	if err != nil {
		return summary, err
	}
	gz := gzip.NewWriter(out)
	tw := tar.NewWriter(gz)

	entries, err := os.ReadDir(tmp)
	if err != nil {
		_ = tw.Close()
		_ = gz.Close()
		_ = out.Close()
		return summary, err
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if err := writeTarFile(tw, filepath.Join(tmp, e.Name()), e.Name()); err != nil {
			_ = tw.Close()
			_ = gz.Close()
			_ = out.Close()
			return summary, err
		}
	}
	if err := tw.Close(); err != nil {
		_ = gz.Close()
		_ = out.Close()
		return summary, err
	}
	if err := gz.Close(); err != nil {
		_ = out.Close()
		return summary, err
	}
	if err := out.Close(); err != nil {
		return summary, err
	}
	summary.OutDir = outPath
	return summary, nil
}

func writeTarFile(tw *tar.Writer, src, name string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	hdr := &tar.Header{
		Name:    name,
		Mode:    0o644,
		Size:    info.Size(),
		ModTime: info.ModTime(),
	}
	if err := tw.WriteHeader(hdr); err != nil {
		return err
	}
	fh, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = fh.Close() }()
	_, err = io.Copy(tw, fh)
	return err
}

// filenameFor returns the canonical on-disk name for a GVR's capture file.
// "<group-or-core>-<resource>.json" — group-prefixed so two CRDs sharing
// a resource name (rare but possible) don't collide.
func filenameFor(gvr schema.GroupVersionResource) string {
	if gvr.Group == "" {
		return "core-" + gvr.Resource + ".json"
	}
	return gvr.Group + "-" + gvr.Resource + ".json"
}

// gvrString renders a GVR as "group/version/resource" with empty group elided.
func gvrString(gvr schema.GroupVersionResource) string {
	if gvr.Group == "" {
		return gvr.Version + "/" + gvr.Resource
	}
	return gvr.Group + "/" + gvr.Version + "/" + gvr.Resource
}
