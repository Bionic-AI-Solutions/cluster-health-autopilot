// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

package watcher

import (
	"context"
	"fmt"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"

	"github.com/srenix-ai/agentic-sre/internal/snapshot"
)

// TestWatchGVR_EndpointSliceEventTriggersSignal runs the real watchGVR
// goroutine against client-go's fake dynamic client: an EndpointSlice ADDED
// event (an endpoint-membership change behind a Service) must push the
// trigger signal that the debounce loop turns into a diagnose cycle —
// preserving the trigger semantics the endpoint layer had before the
// migration off the deprecated core/v1 Endpoints API.
func TestWatchGVR_EndpointSliceEventTriggersSignal(t *testing.T) {
	gvr := snapshot.GVREndpointSlice
	dyn := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(
		runtime.NewScheme(),
		map[schema.GroupVersionResource]string{gvr: "EndpointSliceList"},
	)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	w := &Watcher{lv: snapshot.NewLiveFromDynamic(dyn)}
	trigCh := make(chan struct{}, 1)
	go w.watchGVR(ctx, gvr, trigCh)

	newSlice := func(name string) *unstructured.Unstructured {
		return &unstructured.Unstructured{Object: map[string]any{
			"apiVersion": "discovery.k8s.io/v1",
			"kind":       "EndpointSlice",
			"metadata": map[string]any{
				"name":      name,
				"namespace": "default",
				"labels":    map[string]any{"kubernetes.io/service-name": "api-svc"},
			},
			"addressType": "IPv4",
			"ports": []any{
				map[string]any{"name": "http", "protocol": "TCP", "port": int64(8080)},
			},
		}}
	}

	// The fake's watch channel is only wired once watchGVR's Watch() call
	// lands; events created before that are dropped. Retry bounded creates
	// until the signal arrives (or fail at the deadline).
	deadline := time.After(3 * time.Second)
	for i := 0; ; i++ {
		obj := newSlice(fmt.Sprintf("api-svc-%05d", i))
		if _, err := dyn.Resource(gvr).Namespace("default").Create(ctx, obj, metav1.CreateOptions{}); err != nil {
			t.Fatalf("create fake EndpointSlice: %v", err)
		}
		select {
		case <-trigCh:
			return // EndpointSlice event reached the trigger channel
		case <-deadline:
			t.Fatal("EndpointSlice ADDED event never reached the watcher trigger channel")
		case <-time.After(50 * time.Millisecond):
		}
	}
}
