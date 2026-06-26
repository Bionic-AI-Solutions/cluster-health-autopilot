// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

// Command playground-viewer is a tiny, read-only live view of the DriftReport
// CRs Srenix produces for the srenix-playground namespace. It exists so the hosted
// playground is self-contained OSS (no Srenix Enterprise dashboard image required).
//
// It does ONE thing: list driftreports.srenix.ai cluster-wide
// (the CRD is cluster-scoped) and render them as an auto-refreshing HTML table.
// All dynamic values go through html/template, so report messages — which can
// contain arbitrary cluster strings — are HTML-escaped (XSS-safe).
//
// It mutates nothing and needs only get/list on driftreports (see the
// srenix-playground-viewer ClusterRole in namespace.yaml).
package main

import (
	"context"
	"html/template"
	"log"
	"net/http"
	"os"
	"sort"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var driftGVR = schema.GroupVersionResource{
	Group:    "srenix.ai",
	Version:  "v1alpha1",
	Resource: "driftreports",
}

// row is the flattened, already-escaped-by-template view of one DriftReport.
type row struct {
	Subject  string
	Severity string
	Source   string
	Message  string
	Count    int64
	LastObs  string
}

type pageData struct {
	Rows      []row
	Total     int
	Refreshed string
	Err       string
}

const pageTmpl = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta http-equiv="refresh" content="5">
<title>Srenix Playground — live drift</title>
<style>
 body{font-family:system-ui,sans-serif;margin:2rem;background:#0d1117;color:#c9d1d9}
 h1{font-size:1.4rem}
 .meta{color:#8b949e;font-size:.85rem;margin-bottom:1rem}
 table{border-collapse:collapse;width:100%}
 th,td{border:1px solid #30363d;padding:.5rem .7rem;text-align:left;vertical-align:top;font-size:.9rem}
 th{background:#161b22}
 .critical{color:#f85149;font-weight:600}
 .warning{color:#d29922;font-weight:600}
 .info{color:#58a6ff}
 .empty{color:#8b949e;font-style:italic;margin-top:1rem}
 .err{color:#f85149}
 code{background:#161b22;padding:.1rem .3rem;border-radius:3px}
</style>
</head>
<body>
<h1>Agentic SRE — live drift in <code>srenix-playground</code></h1>
<div class="meta">
  {{.Total}} active DriftReport(s). Auto-refreshes every 5s. Last fetched {{.Refreshed}}.
  These findings are produced live by the Srenix watcher detecting synthetic drift the injector creates.
</div>
{{if .Err}}<p class="err">Error listing DriftReports: {{.Err}}</p>{{end}}
{{if .Rows}}
<table>
  <thead><tr><th>Severity</th><th>Source (analyzer/probe)</th><th>Subject</th><th>Message</th><th>Seen</th><th>Last observed</th></tr></thead>
  <tbody>
  {{range .Rows}}
   <tr>
     <td class="{{.Severity}}">{{.Severity}}</td>
     <td>{{.Source}}</td>
     <td><code>{{.Subject}}</code></td>
     <td>{{.Message}}</td>
     <td>{{.Count}}x</td>
     <td>{{.LastObs}}</td>
   </tr>
  {{end}}
  </tbody>
</table>
{{else if not .Err}}
<p class="empty">No DriftReports yet — the injector or watcher may still be warming up (give it a minute).</p>
{{end}}
</body>
</html>`

func newClient() (dynamic.Interface, error) {
	if cfg, err := rest.InClusterConfig(); err == nil {
		return dynamic.NewForConfig(cfg)
	}
	// Fall back to kubeconfig for local runs.
	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		kubeconfig = os.Getenv("HOME") + "/.kube/config"
	}
	cfg, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, err
	}
	return dynamic.NewForConfig(cfg)
}

func fetchRows(ctx context.Context, c dynamic.Interface) ([]row, error) {
	list, err := c.Resource(driftGVR).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	rows := make([]row, 0, len(list.Items))
	for i := range list.Items {
		rows = append(rows, toRow(&list.Items[i]))
	}
	// critical first, then warning, then info; stable by subject within a tier.
	rank := map[string]int{"critical": 0, "warning": 1, "info": 2}
	sort.SliceStable(rows, func(a, b int) bool {
		ra, rb := rank[rows[a].Severity], rank[rows[b].Severity]
		if ra != rb {
			return ra < rb
		}
		return rows[a].Subject < rows[b].Subject
	})
	return rows, nil
}

func toRow(u *unstructured.Unstructured) row {
	getStr := func(path ...string) string { s, _, _ := unstructured.NestedString(u.Object, path...); return s }
	cnt, _, _ := unstructured.NestedInt64(u.Object, "status", "observationCount")
	return row{
		Subject:  getStr("spec", "subject"),
		Severity: getStr("spec", "severity"),
		Source:   getStr("spec", "source"),
		Message:  getStr("spec", "message"),
		Count:    cnt,
		LastObs:  getStr("status", "lastObserved"),
	}
}

func main() {
	addr := os.Getenv("LISTEN")
	if addr == "" {
		addr = ":8080"
	}
	client, err := newClient()
	if err != nil {
		log.Fatalf("kube client: %v", err)
	}
	tmpl := template.Must(template.New("page").Parse(pageTmpl))

	http.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
		defer cancel()
		data := pageData{Refreshed: time.Now().UTC().Format(time.RFC3339)}
		rows, err := fetchRows(ctx, client)
		if err != nil {
			data.Err = err.Error()
		} else {
			data.Rows = rows
			data.Total = len(rows)
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := tmpl.Execute(w, data); err != nil {
			log.Printf("render: %v", err)
		}
	})

	log.Printf("playground-viewer listening on %s", addr)
	srv := &http.Server{Addr: addr, ReadHeaderTimeout: 5 * time.Second}
	log.Fatal(srv.ListenAndServe())
}
