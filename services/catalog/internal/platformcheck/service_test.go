package platformcheck_test

import (
	"context"
	"errors"
	"testing"

	"github.com/loloDawit/go-admin/services/catalog/internal/platformcheck"
)

type stubRepo struct {
	state platformcheck.SchemaState
	err   error
}

func (s stubRepo) SchemaState(context.Context) (platformcheck.SchemaState, error) {
	return s.state, s.err
}

func TestCheckAcceptsACleanAppliedSchema(t *testing.T) {
	svc := platformcheck.NewService(stubRepo{state: platformcheck.SchemaState{Version: 3, Dirty: false}})

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
	svc := platformcheck.NewService(stubRepo{state: platformcheck.SchemaState{Version: 3, Dirty: true}})

	if _, err := svc.Check(context.Background()); !errors.Is(err, platformcheck.ErrDirtySchema) {
		t.Fatalf("want ErrDirtySchema, got %v", err)
	}
}

func TestCheckRejectsAnUnmigratedSchema(t *testing.T) {
	svc := platformcheck.NewService(stubRepo{state: platformcheck.SchemaState{Version: 0}})

	if _, err := svc.Check(context.Background()); !errors.Is(err, platformcheck.ErrNoMigrations) {
		t.Fatalf("want ErrNoMigrations, got %v", err)
	}
}

// A repository failure is always a connection or query failure surfacing
// from the driver, never a schema-shape problem: Check must classify it as
// ErrDatabaseUnavailable while keeping the original cause reachable via %w,
// so the log (not the client) can still show what the driver said.
func TestCheckPropagatesRepositoryFailure(t *testing.T) {
	want := errors.New("connection refused")
	svc := platformcheck.NewService(stubRepo{err: want})

	_, err := svc.Check(context.Background())
	if !errors.Is(err, platformcheck.ErrDatabaseUnavailable) {
		t.Fatalf("want ErrDatabaseUnavailable, got %v", err)
	}
	if !errors.Is(err, want) {
		t.Fatalf("want the repository error reachable via %%w, got %v", err)
	}
}
