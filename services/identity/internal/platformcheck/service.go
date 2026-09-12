package platformcheck

import "context"

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Check(ctx context.Context) (SchemaState, error) {
	state, err := s.repo.SchemaState(ctx)
	if err != nil {
		return SchemaState{}, err
	}
	if state.Dirty {
		return SchemaState{}, ErrDirtySchema
	}
	if state.Version == 0 {
		return SchemaState{}, ErrNoMigrations
	}
	return state, nil
}
