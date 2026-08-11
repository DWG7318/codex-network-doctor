package singbox

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"regexp"

	"github.com/dwg7318/codex-network-doctor/internal/networkcore"
)

const maxMaterialBytes = 1 << 20

var secretRefPattern = regexp.MustCompile(`^lcnd-secret://[a-z0-9][a-z0-9/_-]{1,127}$`)

type NodeMaterial struct {
	SchemaVersion string          `json:"schema_version"`
	SecretRef     string          `json:"secret_ref"`
	Outbound      json.RawMessage `json:"outbound"`
}

type RenderSummary struct {
	SchemaVersion   string `json:"schema_version"`
	PlanID          string `json:"plan_id"`
	CandidateSHA256 string `json:"candidate_sha256"`
	CandidateBytes  int    `json:"candidate_bytes"`
}

func DecodeMaterial(reader io.Reader) (NodeMaterial, error) {
	var material NodeMaterial
	data, err := io.ReadAll(io.LimitReader(reader, maxMaterialBytes+1))
	if err != nil {
		return material, errors.New("MATERIAL_READ_FAILED")
	}
	if len(data) > maxMaterialBytes {
		return material, errors.New("MATERIAL_INPUT_TOO_LARGE")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&material); err != nil {
		return material, errors.New("MATERIAL_DECODE_FAILED")
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return material, errors.New("MATERIAL_TRAILING_DATA")
	}
	if material.SchemaVersion != networkcore.SchemaVersion || !secretRefPattern.MatchString(material.SecretRef) || len(material.Outbound) == 0 {
		return material, errors.New("MATERIAL_INVALID")
	}
	return material, nil
}
