package staff

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"

	"github.com/loloDawit/go-admin/services/identity/internal/errs"
	"github.com/loloDawit/go-admin/services/identity/internal/session"
)

// generatedPasswordBytes is the entropy of a newly created staff member's
// one-time password, before base64url encoding.
const generatedPasswordBytes = 16

type Service struct {
	repo   Repository
	hasher *session.Hasher
}

func NewService(repo Repository, hasher *session.Hasher) *Service {
	return &Service{repo: repo, hasher: hasher}
}

// Create generates a one-time password and returns it once; only its hash is stored.
func (s *Service) Create(ctx context.Context, in CreateStaff) (Staff, string, error) {
	plain, err := generatePassword()
	if err != nil {
		return Staff{}, "", errs.Wrap(errs.OpGenerateStaffPassword, err)
	}
	hash, err := s.hasher.Hash(plain)
	if err != nil {
		return Staff{}, "", errs.Wrap(errs.OpHashStaffPassword, err)
	}

	created, err := s.repo.Create(ctx, in, hash)
	if err != nil {
		if errors.Is(err, ErrEmailTaken) {
			return Staff{}, "", ErrEmailTaken
		}
		return Staff{}, "", errs.Wrap(errs.OpCreateStaff, err)
	}
	return created, plain, nil
}

// Update applies a partial change, guarded against emptying edit_staff when it touches RoleID or IsActive.
func (s *Service) Update(ctx context.Context, id int64, in UpdateStaff) (Staff, error) {
	if in.RoleID != nil || in.IsActive != nil {
		if err := s.guardLastAdmin(ctx, id, in.RoleID, in.IsActive); err != nil {
			return Staff{}, err
		}
	}

	updated, err := s.repo.Update(ctx, id, in)
	if err != nil {
		if errors.Is(err, ErrNotFound) || errors.Is(err, ErrEmailTaken) {
			return Staff{}, err
		}
		return Staff{}, errs.Wrap(errs.OpUpdateStaff, err)
	}
	return updated, nil
}

// Deactivate sets is_active false, guarded the same way Update guards a
// change to IsActive.
func (s *Service) Deactivate(ctx context.Context, id int64) (Staff, error) {
	inactive := false
	return s.Update(ctx, id, UpdateStaff{IsActive: &inactive})
}

func (s *Service) List(ctx context.Context) ([]Staff, error) {
	list, err := s.repo.List(ctx)
	if err != nil {
		return nil, errs.Wrap(errs.OpListStaff, err)
	}
	return list, nil
}

func (s *Service) Get(ctx context.Context, id int64) (Staff, error) {
	got, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return Staff{}, ErrNotFound
		}
		return Staff{}, errs.Wrap(errs.OpLookupStaffByID, err)
	}
	return got, nil
}

// ChangePassword verifies currentPassword against the stored hash before replacing it.
func (s *Service) ChangePassword(ctx context.Context, id int64, currentPassword, newPassword string) error {
	hash, err := s.repo.PasswordHash(ctx, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return ErrNotFound
		}
		return errs.Wrap(errs.OpLookupStaffByID, err)
	}
	if err := s.hasher.Compare(hash, currentPassword); err != nil {
		return errs.ErrCurrentPasswordIncorrect
	}

	newHash, err := s.hasher.Hash(newPassword)
	if err != nil {
		return errs.Wrap(errs.OpHashStaffPassword, err)
	}
	if err := s.repo.SetPasswordHash(ctx, id, newHash); err != nil {
		return errs.Wrap(errs.OpChangeStaffPassword, err)
	}
	return nil
}

// guardLastAdmin refuses a change that would leave no active staff member holding edit_staff, checked by permission, not role name.
func (s *Service) guardLastAdmin(ctx context.Context, id int64, newRoleID *int64, newActive *bool) error {
	current, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return ErrNotFound
		}
		return errs.Wrap(errs.OpLookupStaffByID, err)
	}
	if !current.IsActive {
		return nil
	}

	currentlyHasEditStaff, err := s.repo.HasEditStaffPermission(ctx, current.RoleID)
	if err != nil {
		return errs.Wrap(errs.OpCheckAdminRole, err)
	}
	if !currentlyHasEditStaff {
		return nil
	}

	losingEditStaff := newActive != nil && !*newActive
	if newRoleID != nil {
		stillHasEditStaff, err := s.repo.HasEditStaffPermission(ctx, *newRoleID)
		if err != nil {
			return errs.Wrap(errs.OpCheckAdminRole, err)
		}
		losingEditStaff = losingEditStaff || !stillHasEditStaff
	}
	if !losingEditStaff {
		return nil
	}

	others, err := s.repo.CountOtherActiveStaffWithEditStaff(ctx, id)
	if err != nil {
		return errs.Wrap(errs.OpCountActiveAdmins, err)
	}
	if others == 0 {
		return ErrLastAdmin
	}
	return nil
}

// generatePassword returns generatedPasswordBytes of crypto/rand entropy,
// base64url-encoded so it is safe to render directly in a JSON response.
func generatePassword() (string, error) {
	raw := make([]byte, generatedPasswordBytes)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}
