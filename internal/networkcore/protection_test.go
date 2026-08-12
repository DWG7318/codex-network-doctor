package networkcore

import "testing"

func TestProtectedTargetSetRequiresPermanentPlatformTargets(t *testing.T) {
	set := SyntheticProtectedTargetSet()
	if issues := set.Validate(); len(issues) != 0 {
		t.Fatalf("Validate() issues = %#v", issues)
	}
	set.Targets = set.Targets[1:]
	if !hasIssue(set.Validate(), "PROTECTED_TARGET_REQUIRED") {
		t.Fatal("missing permanent target accepted")
	}
}

func TestProtectedTargetSetIncludesCaddyFingerprint(t *testing.T) {
	set := SyntheticProtectedTargetSet()
	for _, target := range set.Targets {
		if target.ID == "old-caddy-service-fingerprint" && target.Kind == "read_only_fingerprint" {
			return
		}
	}
	t.Fatal("Caddy fingerprint is not protected")
}

func TestGuardCheckRejectsLCSPAccessLoopbackCollision(t *testing.T) {
	plan := validContinuityPlan()
	plan.Endpoint.Port = 18082
	result := GuardCheck(plan, SyntheticProtectedTargetSet())
	if result.Status != GuardBlocked || !containsString(result.Codes, "PROTECTED_ENDPOINT_COLLISION") {
		t.Fatalf("result = %#v", result)
	}
}

func TestGuardCheckAllowsReadOnlySyntheticEvidence(t *testing.T) {
	result := GuardCheck(validContinuityPlan(), SyntheticProtectedTargetSet())
	if result.Status != GuardPass || result.EvidenceMode != EvidenceSynthetic || !result.ReadOnly {
		t.Fatalf("result = %#v", result)
	}
}
