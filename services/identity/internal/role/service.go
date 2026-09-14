package role

import (
	"context"
	"errors"
	"slices"

	"github.com/loloDawit/go-admin/services/identity/internal/errs"
	"github.com/loloDawit/go-admin/services/identity/internal/permission"
)

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Create(ctx context.Context, in CreateRole) (Role, error) {
	if err := validatePermissions(in.Permissions); err != nil {
		return Role{}, err
	}
	created, err := s.repo.Create(ctx, in)
	if err != nil {
		if errors.Is(err, ErrNameTaken) {
			return Role{}, ErrNameTaken
		}
		return Role{}, errs.Wrap(errs.OpCreateRole, err)
	}
	return created, nil
}

// Update applies a partial change, guarded against emptying edit_staff when Permissions is set.
func (s *Service) Update(ctx context.Context, id int64, in UpdateRole) (Role, error) {
	if in.Permissions != nil {
		if err := validatePermissions(*in.Permissions); err != nil {
			return Role{}, err
		}
		if err := s.guardLastAdmin(ctx, id, *in.Permissions); err != nil {
			return Role{}, err
		}
	}

	updated, err := s.repo.Update(ctx, id, in)
	if err != nil {
		if errors.Is(err, ErrNotFound) || errors.Is(err, ErrNameTaken) {
			return Role{}, err
		}
		return Role{}, errs.Wrap(errs.OpUpdateRole, err)
	}
	return updated, nil
}

func (s *Service) Delete(ctx context.Context, id int64) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		if errors.Is(err, ErrNotFound) || errors.Is(err, ErrInUse) {
			return err
		}
		return errs.Wrap(errs.OpDeleteRole, err)
	}
	return nil
}

func (s *Service) Get(ctx context.Context, id int64) (Role, error) {
	got, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return Role{}, ErrNotFound
		}
		return Role{}, errs.Wrap(errs.OpLookupRoleByID, err)
	}
	return got, nil
}

func (s *Service) List(ctx context.Context) ([]Role, error) {
	list, err := s.repo.List(ctx)
	if err != nil {
		return nil, errs.Wrap(errs.OpListRoles, err)
	}
	return list, nil
}

// guardLastAdmin refuses removing edit_staff from id when no active staff
// would hold it through any other role afterward.
func (s *Service) guardLastAdmin(ctx context.Context, id int64, newPermissions []string) error {
	hasEditStaffNow, err := s.repo.HasEditStaffPermission(ctx, id)
	if err != nil {
		return errs.Wrap(errs.OpCheckRoleEditStaff, err)
	}
	if !hasEditStaffNow || slices.Contains(newPermissions, string(permission.EditStaff)) {
		return nil
	}

	others, err := s.repo.CountActiveStaffWithEditStaffOutsideRole(ctx, id)
	if err != nil {
		return errs.Wrap(errs.OpCountActiveAdminsOutside, err)
	}
	if others == 0 {
		return ErrLastAdmin
	}
	return nil
}

func validatePermissions(names []string) error {
	for _, name := range names {
		if !slices.Contains(permission.All(), name) {
			return ErrUnknownPermission
		}
	}
	return nil
}
