package runner

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
)

type Runner interface {
	Run(ctx context.Context) error
}

// Multi runs several Runners concurrently and waits for all of them to finish.
// If any runner returns a non-nil error the error is logged; the other runners
// keep going until the shared context is cancelled.
type Multi struct {
	runners []Runner
}

func NewMulti(runners ...Runner) *Multi {
	return &Multi{runners: runners}
}

func (m *Multi) Run(ctx context.Context) error {
	var wg sync.WaitGroup
	errs := make(chan error, len(m.runners))

	for _, r := range m.runners {
		wg.Add(1)
		go func(r Runner) {
			defer wg.Done()
			if err := r.Run(ctx); err != nil {
				errs <- fmt.Errorf("%T: %w", r, err)
			}
		}(r)
	}

	wg.Wait()
	close(errs)

	var combined error
	for err := range errs {
		slog.Error("runner exited with error", "err", err)
		combined = err
	}
	return combined
}
