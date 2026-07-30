package model

import (
	"time"
)

type CollectorStatus string

const (
	StatusSuccess  CollectorStatus = "success"
	StatusFailed   CollectorStatus = "failed"
	StatusTimedOut CollectorStatus = "timed_out"
	StatusSkipped  CollectorStatus = "skipped"
)

type CollectorDefinition struct {
	ID          string
	Category    string
	Description string
	Provider    string
	Timeout     time.Duration
	Privilege   string
	Mode        string
}

type Artifact struct {
	Path      string `json:"path"`
	Size      int64  `json:"size"`
	SHA256    string `json:"sha256"`
	MediaType string `json:"media_type"`
}

type CollectorResult struct {
	ID         string          `json:"id"`
	Category   string          `json:"category"`
	Provider   string          `json:"provider,omitempty"`
	Status     CollectorStatus `json:"status"`
	StartedAt  time.Time       `json:"started_at"`
	FinishedAt time.Time       `json:"finished_at"`
	DurationMS int64           `json:"duration_ms"`
	ErrorCode  string          `json:"error_code,omitempty"`
	Error      string          `json:"error,omitempty"`
	Artifacts  []Artifact      `json:"artifacts"`
}
