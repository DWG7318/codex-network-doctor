package networkcore

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
)

const maxContractBytes = 1 << 20

func decodeStrict[T any](reader io.Reader, kind string, validate func(T) []ValidationIssue) (T, error) {
	var value T
	data, err := io.ReadAll(io.LimitReader(reader, maxContractBytes+1))
	if err != nil {
		return value, fmt.Errorf("%s_READ_FAILED", strings.ToUpper(kind))
	}
	if len(data) > maxContractBytes {
		return value, fmt.Errorf("%s_INPUT_TOO_LARGE", strings.ToUpper(kind))
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&value); err != nil {
		return value, fmt.Errorf("%s_DECODE_FAILED: %w", strings.ToUpper(kind), sanitizeJSONError(err))
	}

	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return value, fmt.Errorf("%s_TRAILING_DATA", strings.ToUpper(kind))
	}
	if issues := validate(value); len(issues) != 0 {
		return value, ValidationError{Kind: kind, Issues: issues}
	}
	return value, nil
}

func DecodeNode(reader io.Reader) (NodeDescriptor, error) {
	return decodeStrict(reader, "node", NodeDescriptor.Validate)
}

func DecodeTargetSet(reader io.Reader) (TargetSet, error) {
	return decodeStrict(reader, "target_set", TargetSet.Validate)
}

func DecodeRouteProfile(reader io.Reader) (RouteProfile, error) {
	return decodeStrict(reader, "route_profile", RouteProfile.Validate)
}

type ValidationError struct {
	Kind   string
	Issues []ValidationIssue
}

func (err ValidationError) Error() string {
	codes := make([]string, 0, len(err.Issues))
	for _, issue := range err.Issues {
		codes = append(codes, issue.Code)
	}
	return strings.ToUpper(err.Kind) + "_INVALID: " + strings.Join(codes, ",")
}

func sanitizeJSONError(err error) error {
	var syntax *json.SyntaxError
	if errors.As(err, &syntax) {
		return fmt.Errorf("invalid JSON near byte %d", syntax.Offset)
	}
	var typeError *json.UnmarshalTypeError
	if errors.As(err, &typeError) {
		return fmt.Errorf("invalid type for field %s", typeError.Field)
	}
	if strings.HasPrefix(err.Error(), "json: unknown field ") {
		return errors.New("UNKNOWN_FIELD")
	}
	return errors.New("INVALID_JSON")
}
