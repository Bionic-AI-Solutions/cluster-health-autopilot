// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

package watcher

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// leaseObjectMeta is a tiny constructor kept in its own file so the lease
// metadata stays trivial to test (and reusable from a future operator
// port). Returns ObjectMeta with the given namespace and name.
func leaseObjectMeta(namespace, name string) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Namespace: namespace,
		Name:      name,
	}
}
