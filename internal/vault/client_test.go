// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package vault

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"
)

func TestListKeys_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/secret/data/team/app" {
			t.Errorf("unexpected URL: %s", r.URL.Path)
		}
		if r.Header.Get("X-Vault-Token") != "tok-1" {
			t.Errorf("missing/wrong token")
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"data": map[string]any{
					"DB_USER":     "redacted",
					"DB_PASSWORD": "redacted",
					"API_KEY":     "redacted",
				},
			},
		})
	}))
	defer srv.Close()

	c, err := New(Config{Address: srv.URL, Token: "tok-1", HTTPClient: srv.Client()})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	keys, err := c.ListKeys(context.Background(), "team/app")
	if err != nil {
		t.Fatalf("ListKeys: %v", err)
	}
	sort.Strings(keys)
	want := []string{"API_KEY", "DB_PASSWORD", "DB_USER"}
	if strings.Join(keys, ",") != strings.Join(want, ",") {
		t.Errorf("got %v, want %v", keys, want)
	}
}

func TestListKeys_PathNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer srv.Close()

	c, _ := New(Config{Address: srv.URL, Token: "x", HTTPClient: srv.Client()})
	_, err := c.ListKeys(context.Background(), "missing/path")
	if !errors.Is(err, ErrPathNotFound) {
		t.Errorf("want ErrPathNotFound, got %v", err)
	}
}

func TestListKeys_AuthError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "permission denied", http.StatusForbidden)
	}))
	defer srv.Close()

	c, _ := New(Config{Address: srv.URL, Token: "x", HTTPClient: srv.Client()})
	_, err := c.ListKeys(context.Background(), "any")
	if err == nil || errors.Is(err, ErrPathNotFound) {
		t.Errorf("want non-nil non-NotFound error, got %v", err)
	}
}

func TestNew_RejectsBothAuth(t *testing.T) {
	_, err := New(Config{Address: "http://x", Token: "t", KubernetesAuth: &KubernetesAuthConfig{Role: "r"}})
	if err == nil {
		t.Errorf("want error when both Token and KubernetesAuth set")
	}
}

func TestNew_RejectsNoAuth(t *testing.T) {
	_, err := New(Config{Address: "http://x"})
	if err == nil {
		t.Errorf("want error when no auth set")
	}
}

func TestNew_RejectsEmptyAddress(t *testing.T) {
	_, err := New(Config{Token: "t"})
	if err == nil {
		t.Errorf("want error when Address empty")
	}
}

func TestListKeys_CustomMount(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/kv/data/foo" {
			t.Errorf("unexpected URL: %s, want /v1/kv/data/foo", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{"data": map[string]any{"k": "v"}},
		})
	}))
	defer srv.Close()
	c, _ := New(Config{Address: srv.URL, Token: "t", Mount: "kv", HTTPClient: srv.Client()})
	if _, err := c.ListKeys(context.Background(), "foo"); err != nil {
		t.Errorf("ListKeys: %v", err)
	}
}
