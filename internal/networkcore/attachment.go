package networkcore

import (
	"io"
	"regexp"
	"strings"
)

var selectorValuePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:/-]*$`)

type ApplicationSelector struct {
	Provider     string `json:"provider,omitempty"`
	Model        string `json:"model,omitempty"`
	Channel      string `json:"channel,omitempty"`
	TargetSetRef string `json:"target_set_ref,omitempty"`
}

type AttachmentRollback struct {
	DetachOnly     bool   `json:"detach_only"`
	RestoreRouteID string `json:"restore_route_id"`
}

type ApplicationAttachment struct {
	SchemaVersion        string                `json:"schema_version"`
	AttachmentID         string                `json:"attachment_id"`
	ConsumerID           string                `json:"consumer_id"`
	ConsumerIdentityRef  string                `json:"consumer_identity_ref"`
	Selectors            []ApplicationSelector `json:"selectors"`
	Route                RouteIdentity         `json:"route"`
	EndpointRef          string                `json:"endpoint_ref"`
	CredentialRef        string                `json:"credential_ref"`
	Timeouts             ServiceTimeouts       `json:"timeouts"`
	FailurePolicy        FailurePolicy         `json:"failure_policy"`
	BusinessProbeRef     string                `json:"business_probe_ref"`
	Rollback             AttachmentRollback    `json:"rollback"`
	ProductionAuthorized bool                  `json:"production_authorized"`
}

func DecodeApplicationAttachment(reader io.Reader) (ApplicationAttachment, error) {
	return decodeStrict(reader, "application_attachment", ApplicationAttachment.Validate)
}

func (attachment ApplicationAttachment) Validate() []ValidationIssue {
	var issues []ValidationIssue
	issues = validateSchemaVersion(issues, attachment.SchemaVersion)
	issues = validateID(issues, "attachment_id", attachment.AttachmentID)
	issues = validateID(issues, "consumer_id", attachment.ConsumerID)
	if !validReference(attachment.ConsumerIdentityRef, "lcnd-identity://consumers/") {
		issues = addIssue(issues, "CONSUMER_IDENTITY_REF_INVALID", "consumer_identity_ref", "consumer identity reference is invalid")
	} else if attachment.ConsumerIdentityRef != "lcnd-identity://consumers/"+attachment.ConsumerID {
		issues = addIssue(issues, "CONSUMER_IDENTITY_MISMATCH", "consumer_identity_ref", "consumer identity does not match consumer_id")
	}
	if len(attachment.Selectors) == 0 || len(attachment.Selectors) > 64 {
		issues = addIssue(issues, "SELECTOR_REQUIRED", "selectors", "a bounded explicit selector list is required")
	}
	for _, selector := range attachment.Selectors {
		if selector.Provider == "" && selector.Model == "" && selector.Channel == "" && selector.TargetSetRef == "" {
			issues = addIssue(issues, "SELECTOR_EMPTY", "selectors", "selector must name a provider, model, channel, or target set")
		}
		if selector.TargetSetRef != "" && !validReference(selector.TargetSetRef, "lcnd-targets://") {
			issues = addIssue(issues, "TARGET_SET_REF_INVALID", "selectors.target_set_ref", "target set reference is invalid")
		}
		if selector.Provider != "" && !validSelectorValue(selector.Provider, 64) {
			issues = addIssue(issues, "SELECTOR_PROVIDER_INVALID", "selectors.provider", "provider selector is invalid")
		}
		if selector.Model != "" && !validSelectorValue(selector.Model, 128) {
			issues = addIssue(issues, "SELECTOR_MODEL_INVALID", "selectors.model", "model selector is invalid")
		}
		if selector.Channel != "" && !validSelectorValue(selector.Channel, 64) {
			issues = addIssue(issues, "SELECTOR_CHANNEL_INVALID", "selectors.channel", "channel selector is invalid")
		}
	}
	issues = append(issues, attachment.Route.validate("route")...)
	if attachment.Route.Class == RouteDirect {
		issues = addIssue(issues, "EGRESS_ROUTE_REQUIRED", "route.class", "attachment must name an application egress route")
	}
	if !validReference(attachment.EndpointRef, "lcnd-endpoint://") {
		issues = addIssue(issues, "ENDPOINT_REF_INVALID", "endpoint_ref", "endpoint reference is invalid")
	}
	if !secretRefPattern.MatchString(attachment.CredentialRef) {
		issues = addIssue(issues, "CREDENTIAL_REF_INVALID", "credential_ref", "credential reference is invalid")
	}
	issues = append(issues, attachment.Timeouts.validate()...)
	issues = append(issues, attachment.FailurePolicy.validate()...)
	if !validReference(attachment.BusinessProbeRef, "lcnd-probe://") {
		issues = addIssue(issues, "BUSINESS_PROBE_REF_INVALID", "business_probe_ref", "business probe reference is invalid")
	}
	if domain, ok := egressDomainForRoute(attachment.Route); ok {
		if !referenceHasSegment(attachment.EndpointRef, domain) {
			issues = addIssue(issues, "ENDPOINT_DOMAIN_MISMATCH", "endpoint_ref", "endpoint belongs to another egress domain")
		}
		if !referenceHasSegment(attachment.CredentialRef, domain) {
			issues = addIssue(issues, "CREDENTIAL_DOMAIN_MISMATCH", "credential_ref", "credential belongs to another egress domain")
		}
		if !referenceHasSegment(attachment.BusinessProbeRef, domain) {
			issues = addIssue(issues, "BUSINESS_PROBE_DOMAIN_MISMATCH", "business_probe_ref", "business probe belongs to another egress domain")
		}
	}
	if !attachment.Rollback.DetachOnly || attachment.Rollback.RestoreRouteID != "direct" {
		issues = addIssue(issues, "ROLLBACK_SCOPE_INVALID", "rollback", "rollback must detach one attachment and restore direct")
	}
	if attachment.ProductionAuthorized {
		issues = addIssue(issues, "PRODUCTION_AUTHORITY_FORBIDDEN", "production_authorized", "repository attachments cannot authorize production")
	}
	return issues
}

func validSelectorValue(value string, maxLength int) bool {
	if len(value) == 0 || len(value) > maxLength || value != strings.TrimSpace(value) || !selectorValuePattern.MatchString(value) {
		return false
	}
	switch strings.ToLower(value) {
	case "all", "any", "default":
		return false
	default:
		return true
	}
}
