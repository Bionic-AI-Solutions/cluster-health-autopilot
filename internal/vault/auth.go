// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package vault

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

// KubernetesAuthConfig configures `vault login -method=kubernetes`.
type KubernetesAuthConfig struct {
	// Role is the Vault role bound to the cha ServiceAccount.
	Role string
	// MountPath is the kubernetes auth method's mount path. Default "kubernetes".
	MountPath string
	// JWTPath is the in-cluster ServiceAccount projected-token path.
	// Default "/var/run/secrets/kubernetes.io/serviceaccount/token".
	JWTPath string
}

// loginKubernetes performs the kubernetes auth login and returns a Vault
// token. Called once at probe initialization; the analyzer holds the token
// for the rest of the run. If the token expires mid-tick the next tick will
// re-login (the cron CronJob constructs a fresh client each invocation).
func loginKubernetes(httpClient *http.Client, addr string, cfg KubernetesAuthConfig) (string, error) {
	if cfg.Role == "" {
		return "", fmt.Errorf("KubernetesAuthConfig.Role is required")
	}
	mount := cfg.MountPath
	if mount == "" {
		mount = "kubernetes"
	}
	jwtPath := cfg.JWTPath
	if jwtPath == "" {
		jwtPath = "/var/run/secrets/kubernetes.io/serviceaccount/token"
	}
	jwtBytes, err := os.ReadFile(jwtPath)
	if err != nil {
		return "", fmt.Errorf("read SA JWT at %s: %w", jwtPath, err)
	}
	body, _ := json.Marshal(map[string]string{
		"role": cfg.Role,
		"jwt":  strings.TrimSpace(string(jwtBytes)),
	})
	url := strings.TrimRight(addr, "/") + "/v1/auth/" + mount + "/login"
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		preview, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return "", fmt.Errorf("login HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(preview)))
	}
	var out struct {
		Auth struct {
			ClientToken string `json:"client_token"`
		} `json:"auth"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", fmt.Errorf("decode login response: %w", err)
	}
	if out.Auth.ClientToken == "" {
		return "", fmt.Errorf("login response had empty client_token")
	}
	return out.Auth.ClientToken, nil
}
