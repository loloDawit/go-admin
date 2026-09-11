package seed_test

import (
	"testing"

	"github.com/loloDawit/go-admin/internal/auth"
	"github.com/loloDawit/go-admin/internal/seed"
	"github.com/loloDawit/go-admin/internal/testutil"
	"github.com/loloDawit/go-admin/models"
)

func TestRunCreatesPermissionsRolesAndOwner(t *testing.T) {
	db := testutil.NewDB(t)

	if err := seed.Run(db, auth.NewTestHasher(), "owner@example.com", "s3cret-password"); err != nil {
		t.Fatalf("seed: %v", err)
	}

	var permCount int64
	db.Model(&models.Permission{}).Count(&permCount)
	if permCount != 8 {
		t.Errorf("want 8 permissions, got %d", permCount)
	}

	var owner models.Role
	if err := db.Preload("Permissions").Where("name = ?", "owner").First(&owner).Error; err != nil {
		t.Fatalf("owner role missing: %v", err)
	}
	if len(owner.Permissions) != 8 {
		t.Errorf("owner must hold all 8 permissions, got %d", len(owner.Permissions))
	}

	var user models.User
	if err := db.Where("email = ?", "owner@example.com").First(&user).Error; err != nil {
		t.Fatalf("owner user missing: %v", err)
	}
	if user.RoleId != owner.Id {
		t.Errorf("owner user must hold the owner role")
	}
	if err := auth.NewTestHasher().Check(user.Password, "s3cret-password"); err != nil {
		t.Errorf("owner password must verify: %v", err)
	}
}

func TestRunIsIdempotent(t *testing.T) {
	db := testutil.NewDB(t)

	if err := seed.Run(db, auth.NewTestHasher(), "owner@example.com", "s3cret-password"); err != nil {
		t.Fatalf("seed run 0: %v", err)
	}

	// Re-run with a different password and 2 more times: row counts alone
	// can't tell a Replace from an Append, and can't tell whether the owner's
	// password got overwritten, so both are checked explicitly below.
	for i := 1; i < 3; i++ {
		if err := seed.Run(db, auth.NewTestHasher(), "owner@example.com", "a-different-password"); err != nil {
			t.Fatalf("seed run %d: %v", i, err)
		}
	}

	var users, perms, roles int64
	db.Model(&models.User{}).Count(&users)
	db.Model(&models.Permission{}).Count(&perms)
	db.Model(&models.Role{}).Count(&roles)

	if users != 1 || perms != 8 || roles != 3 {
		t.Fatalf("re-running seed must not duplicate: users=%d perms=%d roles=%d", users, perms, roles)
	}

	var owner models.Role
	if err := db.Preload("Permissions").Where("name = ?", "owner").First(&owner).Error; err != nil {
		t.Fatalf("owner role missing: %v", err)
	}
	if len(owner.Permissions) != 8 {
		t.Errorf("re-running must not accumulate grants: owner has %d permissions, want 8", len(owner.Permissions))
	}

	var user models.User
	if err := db.Where("email = ?", "owner@example.com").First(&user).Error; err != nil {
		t.Fatalf("owner user missing: %v", err)
	}
	if err := auth.NewTestHasher().Check(user.Password, "s3cret-password"); err != nil {
		t.Errorf("re-running must not reset the owner's password: original password no longer verifies: %v", err)
	}
	if err := auth.NewTestHasher().Check(user.Password, "a-different-password"); err == nil {
		t.Error("re-running must not reset the owner's password: the later run's password verifies instead")
	}
}

func TestRunRejectsEmptyOwnerCredentials(t *testing.T) {
	db := testutil.NewDB(t)

	if err := seed.Run(db, auth.NewTestHasher(), "", "s3cret-password"); err == nil {
		t.Error("an empty owner email must be rejected")
	}
	if err := seed.Run(db, auth.NewTestHasher(), "owner@example.com", ""); err == nil {
		t.Error("an empty owner password must be rejected")
	}
}
