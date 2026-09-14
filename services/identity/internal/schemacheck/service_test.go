package schemacheck_test

import (
	"context"
	"errors"
	"testing"

	"github.com/loloDawit/go-admin/services/identity/internal/schemacheck"
)

type stubRepo struct {
	state schemacheck.SchemaState
	err   error
}

func (s stubRepo) SchemaState(context.Context) (schemacheck.SchemaState, error) {
	return s.state, s.err
}

func TestCheckAcceptsACleanAppliedSchema(t *testing.T) {
	svc := schemacheck.NewService(stubRepo{state: schemacheck.SchemaState{Version: 3, Dirty: false}})

	state, err := svc.Check(context.Background())
	if err != nil {
		t.Fatalf("want success, got %v", err)
	}
	if state.Version != 3 {
		t.Errorf("version: want 3, got %d", state.Version)
	}
}

// A dirty row means a migration failed partway and the schema is in an unknown
// state. A non-zero version alone is not success.
func TestCheckRejectsADirtySchema(t *testing.T) {
	svc := schemacheck.NewService(stubRepo{state: schemacheck.SchemaState{Version: 3, Dirty: true}})

	if _, err := svc.Check(context.Background()); !errors.Is(err, schemacheck.ErrDirtySchema) {
		t.Fatalf("want ErrDirtySchema, got %v", err)
	}
}

func TestCheckRejectsAnUnmigratedSchema(t *testing.T) {
	svc := schemacheck.NewService(stubRepo{state: schemacheck.SchemaState{Version: 0}})

	if _, err := svc.Check(context.Background()); !errors.Is(err, schemacheck.ErrNoMigrations) {
		t.Fatalf("want ErrNoMigrations, got %v", err)
	}
}

// Check must classify a repository failure as ErrDatabaseUnavailable while keeping the cause reachable via %w for the log.
func TestCheckPropagatesRepositoryFailure(t *testing.T) {
	want := errors.New("connection refused")
	svc := schemacheck.NewService(stubRepo{err: want})

	_, err := svc.Check(context.Background())
	if !errors.Is(err, schemacheck.ErrDatabaseUnavailable) {
		t.Fatalf("want ErrDatabaseUnavailable, got %v", err)
	}
	if !errors.Is(err, want) {
		t.Fatalf("want the repository error reachable via %%w, got %v", err)
	}
}
