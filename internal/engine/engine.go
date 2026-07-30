package engine

import (
	"context"
	"time"

	"github.com/dwg7318/codex-network-doctor/internal/evidence"
	"github.com/dwg7318/codex-network-doctor/internal/model"
)

type Status = model.CollectorStatus

const (
	StatusSuccess  = model.StatusSuccess
	StatusFailed   = model.StatusFailed
	StatusTimedOut = model.StatusTimedOut
	StatusSkipped  = model.StatusSkipped
)

type Definition = model.CollectorDefinition

type Result = model.CollectorResult

type Collector interface {
	Definition() Definition
	Collect(context.Context, *evidence.Store) ([]evidence.Artifact, error)
}

type CollectorFunc struct {
	Spec Definition
	Run  func(context.Context, *evidence.Store) ([]evidence.Artifact, error)
}

func (c CollectorFunc) Definition() Definition {
	return c.Spec
}

func (c CollectorFunc) Collect(ctx context.Context, store *evidence.Store) ([]evidence.Artifact, error) {
	return c.Run(ctx, store)
}

type Engine struct {
	collectors []Collector
}

func New(collectors []Collector) *Engine {
	return &Engine{collectors: append([]Collector(nil), collectors...)}
}

func (e *Engine) Run(ctx context.Context, store *evidence.Store) []Result {
	results := make([]Result, 0, len(e.collectors))
	for _, collector := range e.collectors {
		definition := collector.Definition()
		timeout := definition.Timeout
		if timeout <= 0 {
			timeout = 10 * time.Second
		}

		started := time.Now().UTC()
		collectorContext, cancel := context.WithTimeout(ctx, timeout)
		artifacts, err := collector.Collect(collectorContext, store)
		contextError := collectorContext.Err()
		cancel()
		finished := time.Now().UTC()

		result := Result{
			ID:         definition.ID,
			Category:   definition.Category,
			Provider:   definition.Provider,
			Status:     StatusSuccess,
			StartedAt:  started,
			FinishedAt: finished,
			DurationMS: finished.Sub(started).Milliseconds(),
			Artifacts:  artifacts,
		}
		if contextError == context.DeadlineExceeded {
			result.Status = StatusTimedOut
			result.ErrorCode = "COLLECTOR_TIMEOUT"
		} else if err != nil {
			result.Status = StatusFailed
			result.ErrorCode = "COLLECTOR_FAILED"
		}
		if err != nil {
			result.Error = err.Error()
		}
		if result.Artifacts == nil {
			result.Artifacts = []evidence.Artifact{}
		}
		results = append(results, result)
	}
	return results
}
