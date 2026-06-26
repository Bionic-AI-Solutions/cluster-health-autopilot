// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

package probe

// asInt64 coerces a value from unstructured K8s data to int64. The
// apimachinery unstructured unmarshaler preserves int64 for K8s API
// integer fields (per Kubernetes JSON convention), but defensive code
// also handles float64 in case a probe ever consumes externally-supplied
// JSON. Returns 0 for nil or non-numeric input.
func asInt64(v any) int64 {
	switch x := v.(type) {
	case int64:
		return x
	case int:
		return int64(x)
	case float64:
		return int64(x)
	case float32:
		return int64(x)
	}
	return 0
}
