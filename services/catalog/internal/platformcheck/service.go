package platformcheck

import (
	"context"
	"fmt"
)

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Check(ctx context.Context) (SchemaState, error) {
	state, err := s.repo.SchemaState(ctx)
	// Repository has exactly one implementation path that returns a non-nil
	// error: a connection or query failure surfacing from the driver. Wrap it
	// as ErrDatabaseUnavailable here, the one seam every Repository
	// implementation (real or test double) passes through, rather than in
	// the Postgres implementation, which a fake repository bypasses entirely.
	if err != nil {
		return SchemaState{}, fmt.Errorf("%w: %w", ErrDatabaseUnavailable, err)
	}
	if state.Dirty {
		return SchemaState{}, ErrDirtySchema
	}
	if state.Version == 0 {
		return SchemaState{}, ErrNoMigrations
	}
	return state, nil
}

// Probe adapts Check to platform/readiness.Probe: a readiness check only
// cares whether the service can serve, not the schema version it saw.
func (s *Service) Probe(ctx context.Context) error {
	_, err := s.Check(ctx)
	return err
}
