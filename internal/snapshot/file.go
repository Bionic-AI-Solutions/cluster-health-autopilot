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
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// File is a Source backed by a directory or a single JSON file containing
// `kubectl get … -o json` output. It supports two layouts:
//
//  1. Single file containing a single List object (any kind).
//  2. Directory containing one or more *.json files; each file is parsed
//     as a List (or single object) and merged into an in-memory index
//     keyed by GVR.
//
// The on-disk format intentionally matches what `kubectl get -o json`
// produces, so users can capture a snapshot with one familiar command.
type File struct {
	// objects[gvrKey] -> list of objects of that GVR
	objects map[string][]unstructured.Unstructured
}

// LoadFile reads a snapshot from path, which may be:
//   - a directory containing *.json captures
//   - a single *.json file
//   - a *.tar.gz / *.tgz archive produced by `cha snapshot capture --tar`
//
// It is tolerant of unknown kinds (silently skipped) and of mixing List /
// single-object payloads in the same file or tree.
func LoadFile(path string) (*File, error) {
	if strings.HasSuffix(path, ".tar.gz") || strings.HasSuffix(path, ".tgz") {
		return loadTarGZ(path)
	}
	st, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("snapshot path %q: %w", path, err)
	}
	f := &File{objects: make(map[string][]unstructured.Unstructured)}
	if st.IsDir() {
		err = filepath.WalkDir(path, func(p string, d os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if d.IsDir() || !strings.HasSuffix(p, ".json") {
				return nil
			}
			return f.loadJSONFile(p)
		})
	} else {
		err = f.loadJSONFile(path)
	}
	if err != nil {
		return nil, err
	}
	return f, nil
}

// loadTarGZ reads a .tar.gz / .tgz snapshot archive directly into memory
// without writing to disk. Each *.json entry in the archive is parsed the
// same way as a file on disk.
func loadTarGZ(path string) (*File, error) {
	fh, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %q: %w", path, err)
	}
	defer func() { _ = fh.Close() }()

	gz, err := gzip.NewReader(fh)
	if err != nil {
		return nil, fmt.Errorf("decompress %q: %w", path, err)
	}
	defer func() { _ = gz.Close() }()

	f := &File{objects: make(map[string][]unstructured.Unstructured)}
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read tar %q: %w", path, err)
		}
		if hdr.Typeflag != tar.TypeReg || !strings.HasSuffix(hdr.Name, ".json") {
			continue
		}
		data, err := io.ReadAll(tr)
		if err != nil {
			return nil, fmt.Errorf("read entry %q in %q: %w", hdr.Name, path, err)
		}
		if err := f.loadJSONBytes(hdr.Name, data); err != nil {
			return nil, err
		}
	}
	return f, nil
}

func (f *File) loadJSONFile(p string) error {
	fh, err := os.Open(p)
	if err != nil {
		return fmt.Errorf("open %q: %w", p, err)
	}
	defer func() { _ = fh.Close() }()
	data, err := io.ReadAll(fh)
	if err != nil {
		return fmt.Errorf("read %q: %w", p, err)
	}
	return f.loadJSONBytes(p, data)
}

func (f *File) loadJSONBytes(name string, data []byte) error {
	// Determine if this is a List or a single object by peeking at "kind".
	var probe struct {
		Kind  string            `json:"kind"`
		Items []json.RawMessage `json:"items"`
	}
	if err := json.Unmarshal(data, &probe); err != nil {
		return fmt.Errorf("parse %q: %w", name, err)
	}
	if strings.HasSuffix(probe.Kind, "List") && len(probe.Items) > 0 {
		for i, raw := range probe.Items {
			obj := unstructured.Unstructured{}
			if err := obj.UnmarshalJSON(raw); err != nil {
				return fmt.Errorf("parse item %d in %q: %w", i, name, err)
			}
			f.add(obj)
		}
		return nil
	}
	// Treat as a single object.
	obj := unstructured.Unstructured{}
	if err := obj.UnmarshalJSON(data); err != nil {
		return fmt.Errorf("parse %q as single object: %w", name, err)
	}
	if obj.GetKind() == "" {
		return nil
	}
	f.add(obj)
	return nil
}

func (f *File) add(obj unstructured.Unstructured) {
	gvr := gvrFromObject(obj)
	if gvr == "" {
		return // unknown — silently skip
	}
	f.objects[gvr] = append(f.objects[gvr], obj)
}

// gvrFromObject converts an Unstructured's apiVersion/kind into the
// resource-name key used by our index. We map kind→resource via a small
// hard-coded table covering the resources our probes use; unknown kinds
// are dropped (returning ""). This keeps us free of a discovery client
// at parse time.
func gvrFromObject(obj unstructured.Unstructured) string {
	gv := obj.GetAPIVersion()
	kind := obj.GetKind()
	res, ok := kindToResource[kind]
	if !ok {
		return ""
	}
	return gv + "/" + res
}

var kindToResource = map[string]string{
	"Pod":                   "pods",
	"Node":                  "nodes",
	"PersistentVolumeClaim": "persistentvolumeclaims",
	"Event":                 "events",
	"Deployment":            "deployments",
	"ReplicaSet":            "replicasets",
	"StatefulSet":           "statefulsets",
	"Job":                   "jobs",
	"CronJob":               "cronjobs",
	"ExternalSecret":        "externalsecrets",
	"SecretStore":           "secretstores",
	"ClusterSecretStore":    "clustersecretstores",
	"Cluster":               "clusters",
	"CephCluster":           "cephclusters",
	"Certificate":           "certificates",
	"CertificateRequest":    "certificaterequests",
	"Order":                 "orders",
	"Secret":                "secrets",
	"Ingress":               "ingresses",
}

func indexKey(gvr schema.GroupVersionResource) string {
	gv := gvr.Version
	if gvr.Group != "" {
		gv = gvr.Group + "/" + gvr.Version
	}
	return gv + "/" + gvr.Resource
}

// List returns all objects of the given GVR. If ns is non-empty, results
// are filtered to that namespace (or returned as-is for cluster-scoped).
func (f *File) List(_ context.Context, gvr schema.GroupVersionResource, ns string) (*unstructured.UnstructuredList, error) {
	key := indexKey(gvr)
	out := &unstructured.UnstructuredList{}
	out.SetAPIVersion(gvr.GroupVersion().String())
	out.SetKind(titleCase(trimTrailingS(gvr.Resource)) + "List")
	for _, obj := range f.objects[key] {
		if ns != "" && obj.GetNamespace() != ns {
			continue
		}
		out.Items = append(out.Items, obj)
	}
	return out, nil
}

// Get returns a single object by namespace + name. Returns an error if not
// found in the snapshot — callers should treat NotFound the same as live mode.
func (f *File) Get(_ context.Context, gvr schema.GroupVersionResource, ns, name string) (*unstructured.Unstructured, error) {
	key := indexKey(gvr)
	for _, obj := range f.objects[key] {
		if obj.GetName() == name && (ns == "" || obj.GetNamespace() == ns) {
			cp := obj.DeepCopy()
			return cp, nil
		}
	}
	return nil, fmt.Errorf("%s %q not found in snapshot", gvr.Resource, ns+"/"+name)
}

// Mode reports snapshot mode — fixers refuse to run.
func (f *File) Mode() Mode { return ModeSnapshot }

// titleCase upper-cases the first ASCII letter of s. We only feed it
// resource names ("pods", "events", "externalsecrets" …) which are all
// lowercase ASCII, so this is sufficient and avoids the unicode
// edge-case bugs that got strings.Title deprecated.
func titleCase(s string) string {
	if s == "" {
		return s
	}
	if c := s[0]; c >= 'a' && c <= 'z' {
		return string(c-'a'+'A') + s[1:]
	}
	return s
}

// trimTrailingS drops a trailing 's' if present. Used to convert plural
// resource names ("pods" → "Pod") into Kind values.
func trimTrailingS(s string) string {
	if len(s) > 0 && s[len(s)-1] == 's' {
		return s[:len(s)-1]
	}
	return s
}
