// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package ai

import (
	"bufio"
	"context"
	"io"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

// FetchPodLogs streams the tail of a pod container's logs and returns them as a
// redacted LogsResult. It is the shared implementation behind every
// Environment.Logs() — the OSS LiveEnvironment and the CHA-com aiwatch env both
// delegate here so the streaming + redaction behaviour can't drift between the
// free and paid investigators.
//
// All common failure modes are SOFT: a nil client, a missing previous
// instance, or a still-starting container set LogsResult.Error and return a nil
// Go error, so the investigation pass continues without logs.
func FetchPodLogs(ctx context.Context, client kubernetes.Interface, namespace, pod string, opts LogsOptions) LogsResult {
	res := LogsResult{Namespace: namespace, Pod: pod, Container: opts.Container, Previous: opts.Previous}
	if client == nil {
		res.Error = "pod logs unavailable (no typed client; snapshot mode or logs not enabled)"
		return res
	}
	tail := opts.TailLines
	if tail <= 0 {
		tail = DefaultLogTailLines
	}
	req := client.CoreV1().Pods(namespace).GetLogs(pod, &corev1.PodLogOptions{
		Container:  opts.Container,
		Previous:   opts.Previous,
		TailLines:  &tail,
		Timestamps: false,
	})
	stream, err := req.Stream(ctx)
	if err != nil {
		res.Error = "log stream: " + err.Error()
		return res
	}
	defer stream.Close()

	const maxBytes = 64 * 1024 // cap memory; tail is already line-bounded
	sc := bufio.NewScanner(io.LimitReader(stream, maxBytes))
	sc.Buffer(make([]byte, 0, 8*1024), 64*1024)
	var lines []string
	for sc.Scan() {
		line := strings.TrimRight(sc.Text(), "\r")
		if line == "" {
			continue
		}
		lines = append(lines, RedactEventMessage(line))
		if int64(len(lines)) > tail {
			lines = lines[len(lines)-int(tail):]
			res.Truncated = true
		}
	}
	res.Lines = lines
	return res
}
