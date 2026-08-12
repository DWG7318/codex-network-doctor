package networkcore

import "io"

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
	if !attachment.Rollback.DetachOnly || attachment.Rollback.RestoreRouteID != "direct" {
		issues = addIssue(issues, "ROLLBACK_SCOPE_INVALID", "rollback", "rollback must detach one attachment and restore direct")
	}
	if attachment.ProductionAuthorized {
		issues = addIssue(issues, "PRODUCTION_AUTHORITY_FORBIDDEN", "production_authorized", "repository attachments cannot authorize production")
	}
	return issues
}
