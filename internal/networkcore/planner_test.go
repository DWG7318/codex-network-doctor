package networkcore

import (
	"bytes"
	"slices"
	"strings"
	"testing"
)

func TestBuildDraftRoutesDomesticTargetsThroughChinaEgress(t *testing.T) {
	draft, err := BuildDraft(chinaNode(), domesticTargets(), managementTargets(), overseasHostProfile(), "0.3.0")
	if err != nil {
		t.Fatal(err)
	}
	if draft.DefaultOutbound != OutboundDirect {
		t.Fatalf("default = %q", draft.DefaultOutbound)
	}
	if draft.TunnelOutbound != OutboundTunnel {
		t.Fatalf("tunnel = %q", draft.TunnelOutbound)
	}
	if draft.NodeDirection != DirectionChinaEgress {
		t.Fatalf("direction = %q", draft.NodeDirection)
	}
	assertAllDecisions(t, draft.Decisions, "domestic-services", OutboundTunnel)
	assertAllDecisions(t, draft.ProtectedDecisions, "management", OutboundDirect)
}

func TestBuildDraftRoutesGlobalTargetsThroughGlobalEgress(t *testing.T) {
	node := validNode()
	node.NodeID = "global-primary"
	node.Direction = DirectionGlobalEgress
	profile := RouteProfile{
		SchemaVersion:        SchemaVersion,
		ProfileID:            "domestic-host",
		Name:                 ProfileDomesticDirectGlobalTunnel,
		NodeID:               node.NodeID,
		TargetSetID:          "global-services",
		ProtectedTargetSetID: "management",
	}
	targets := TargetSet{
		SchemaVersion: SchemaVersion,
		TargetSetID:   "global-services",
		Revision:      1,
		Purpose:       "synthetic global APIs",
		Targets:       []TargetRule{{Kind: TargetDomain, Value: "api.example.com", Protocol: "tcp", Ports: []int{443}}},
	}

	draft, err := BuildDraft(node, targets, managementTargets(), profile, "0.3.0")
	if err != nil {
		t.Fatal(err)
	}
	if draft.NodeDirection != DirectionGlobalEgress {
		t.Fatalf("direction = %q", draft.NodeDirection)
	}
	assertAllDecisions(t, draft.Decisions, "global-services", OutboundTunnel)
	assertAllDecisions(t, draft.ProtectedDecisions, "management", OutboundDirect)
}

func TestBuildDraftIsDeterministicAcrossInputOrder(t *testing.T) {
	firstTargets := domesticTargets()
	secondTargets := domesticTargets()
	slices.Reverse(secondTargets.Targets)
	first, err := BuildDraft(chinaNode(), firstTargets, managementTargets(), overseasHostProfile(), "0.3.0")
	if err != nil {
		t.Fatal(err)
	}
	second, err := BuildDraft(chinaNode(), secondTargets, managementTargets(), overseasHostProfile(), "0.3.0")
	if err != nil {
		t.Fatal(err)
	}
	firstJSON, firstSHA, err := CanonicalJSON(first)
	if err != nil {
		t.Fatal(err)
	}
	secondJSON, secondSHA, err := CanonicalJSON(second)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(firstJSON, secondJSON) || firstSHA != secondSHA {
		t.Fatalf("drafts differ")
	}
}

func TestBuildDraftRejectsProtectedTargetOverlap(t *testing.T) {
	targets := domesticTargets()
	protected := managementTargets()
	protected.Targets = append(protected.Targets, targets.Targets[0])
	_, err := BuildDraft(chinaNode(), targets, protected, overseasHostProfile(), "0.3.0")
	if err == nil || !strings.Contains(err.Error(), "TARGET_OVERLAP") {
		t.Fatalf("error = %v, want TARGET_OVERLAP", err)
	}
}

func chinaNode() NodeDescriptor { return validNode() }

func domesticTargets() TargetSet {
	return TargetSet{
		SchemaVersion: SchemaVersion,
		TargetSetID:   "domestic-services",
		Revision:      1,
		Purpose:       "synthetic domestic APIs",
		Targets: []TargetRule{
			{Kind: TargetDomain, Value: "api.example.cn", Protocol: "tcp", Ports: []int{443}},
			{Kind: TargetDomainSuffix, Value: "example.cn", Protocol: "tcp", Ports: []int{443}},
		},
	}
}

func managementTargets() TargetSet {
	return TargetSet{
		SchemaVersion: SchemaVersion,
		TargetSetID:   "management",
		Revision:      1,
		Purpose:       "protected management",
		Targets: []TargetRule{
			{Kind: TargetIPCIDR, Value: "10.0.0.0/8"},
			{Kind: TargetIPCIDR, Value: "127.0.0.0/8"},
		},
	}
}

func overseasHostProfile() RouteProfile {
	return RouteProfile{
		SchemaVersion:        SchemaVersion,
		ProfileID:            "overseas-host",
		Name:                 ProfileGlobalDirectDomesticTunnel,
		NodeID:               "cn-primary",
		TargetSetID:          "domestic-services",
		ProtectedTargetSetID: "management",
	}
}

func assertAllDecisions(t *testing.T, got []RouteDecision, targetSetID string, outbound Outbound) {
	t.Helper()
	if len(got) == 0 {
		t.Fatal("no decisions")
	}
	for _, decision := range got {
		if decision.TargetSetID != targetSetID || decision.Outbound != outbound {
			t.Fatalf("decision = %#v", decision)
		}
	}
}
