package engine

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/dwg7318/codex-network-doctor/internal/evidence"
)

func TestEngineContinuesAfterCollectorFailure(t *testing.T) {
	store, err := evidence.NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("evidence.NewStore() error = %v", err)
	}

	failing := CollectorFunc{
		Spec: Definition{ID: "failing", Category: "system", Timeout: time.Second},
		Run: func(context.Context, *evidence.Store) ([]evidence.Artifact, error) {
			return nil, errors.New("access denied")
		},
	}
	succeeding := CollectorFunc{
		Spec: Definition{ID: "succeeding", Category: "adapter", Timeout: time.Second},
		Run: func(_ context.Context, store *evidence.Store) ([]evidence.Artifact, error) {
			artifact, err := store.Write("evidence/succeeding.json", []byte("{}\n"), "application/json")
			return []evidence.Artifact{artifact}, err
		},
	}

	results := New([]Collector{failing, succeeding}).Run(context.Background(), store)

	if len(results) != 2 {
		t.Fatalf("len(results) = %d, want 2", len(results))
	}
	if results[0].Status != StatusFailed {
		t.Fatalf("first status = %q, want %q", results[0].Status, StatusFailed)
	}
	if results[0].ErrorCode != "COLLECTOR_FAILED" {
		t.Fatalf("first error code = %q, want COLLECTOR_FAILED", results[0].ErrorCode)
	}
	if results[1].Status != StatusSuccess {
		t.Fatalf("second status = %q, want %q", results[1].Status, StatusSuccess)
	}
	if len(results[1].Artifacts) != 1 {
		t.Fatalf("second artifact count = %d, want 1", len(results[1].Artifacts))
	}
}

func TestEngineMarksCollectorTimeoutAndContinues(t *testing.T) {
	store, err := evidence.NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("evidence.NewStore() error = %v", err)
	}

	timedOut := CollectorFunc{
		Spec: Definition{ID: "slow", Category: "route", Timeout: 20 * time.Millisecond},
		Run: func(ctx context.Context, _ *evidence.Store) ([]evidence.Artifact, error) {
			<-ctx.Done()
			return nil, ctx.Err()
		},
	}
	after := CollectorFunc{
		Spec: Definition{ID: "after", Category: "dns", Timeout: time.Second},
		Run: func(context.Context, *evidence.Store) ([]evidence.Artifact, error) {
			return nil, nil
		},
	}

	results := New([]Collector{timedOut, after}).Run(context.Background(), store)

	if results[0].Status != StatusTimedOut {
		t.Fatalf("first status = %q, want %q", results[0].Status, StatusTimedOut)
	}
	if results[0].ErrorCode != "COLLECTOR_TIMEOUT" {
		t.Fatalf("first error code = %q, want COLLECTOR_TIMEOUT", results[0].ErrorCode)
	}
	if results[1].Status != StatusSuccess {
		t.Fatalf("second status = %q, want %q", results[1].Status, StatusSuccess)
	}
}
