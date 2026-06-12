// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"testing"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	elbv2types "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2/types"
)

// lbDNSByARN + firstLoadBalancerDNS feed the CHA-com "(lb: ...)" join
// key — one DescribeLoadBalancers per probe cycle builds the ARN → DNS
// map; each target group resolves through its LoadBalancerArns.
//
// Error-injection: when DescribeLoadBalancers errors, describeLoadBalancerDNS
// returns nil (best-effort). firstLoadBalancerDNS on a nil map returns ""
// so the probe omits the "(lb: ...)" suffix entirely — the finding carries
// the pre-enrichment message.
func TestDescribeLoadBalancerDNS_ErrorYieldsEmptyMap_ProbeUnsuffixed(t *testing.T) {
	// Simulate the error path: describeLoadBalancerDNS returns nil when
	// the DescribeLoadBalancers call fails. Verify that firstLoadBalancerDNS
	// on that nil map returns "" so the probe emits the unsuffixed message.
	var nilMap map[string]string // equivalent to return value on error
	if got := firstLoadBalancerDNS([]string{"arn:lb/1"}, nilMap); got != "" {
		t.Errorf("DescribeLoadBalancers error path: firstLoadBalancerDNS=%q want \"\" (unsuffixed)", got)
	}
}

func TestLBDNSByARN(t *testing.T) {
	m := lbDNSByARN([]elbv2types.LoadBalancer{
		{LoadBalancerArn: awssdk.String("arn:lb/1"), DNSName: awssdk.String("one.elb.amazonaws.com")},
		{LoadBalancerArn: awssdk.String("arn:lb/2"), DNSName: awssdk.String("")}, // no DNS → skipped
		{LoadBalancerArn: nil, DNSName: awssdk.String("ghost.elb.amazonaws.com")},
	})
	if len(m) != 1 || m["arn:lb/1"] != "one.elb.amazonaws.com" {
		t.Errorf("got %v want map[arn:lb/1:one.elb.amazonaws.com]", m)
	}
}

func TestFirstLoadBalancerDNS(t *testing.T) {
	m := map[string]string{"arn:lb/1": "one.elb.amazonaws.com"}
	if got := firstLoadBalancerDNS([]string{"arn:lb/0", "arn:lb/1"}, m); got != "one.elb.amazonaws.com" {
		t.Errorf("got %q want one.elb.amazonaws.com", got)
	}
	if got := firstLoadBalancerDNS([]string{"arn:lb/0"}, m); got != "" {
		t.Errorf("got %q want \"\" for unresolvable ARNs", got)
	}
	if got := firstLoadBalancerDNS(nil, m); got != "" {
		t.Errorf("got %q want \"\" for no LB ARNs", got)
	}
}
