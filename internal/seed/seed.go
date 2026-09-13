// Package seed creates the permission vocabulary, the built-in roles, and the
// first owner account.
//
// It is the only way an account comes into existence: there is no public
// registration endpoint, because self-registration granted admin rights.
package seed

import (
	"errors"
	"fmt"

	"github.com/loloDawit/go-admin/internal/auth"
	"github.com/loloDawit/go-admin/models"
	"gorm.io/gorm"
)

// The authorization middleware derives its checks from these names.
var AllPermissions = []string{
	"view_users", "edit_users",
	"view_products", "edit_products",
	"view_orders", "edit_orders",
	"view_roles", "edit_roles",
}

var roleGrants = map[string][]string{
	"owner": AllPermissions,
	"admin": {
		"view_users", "edit_users",
		"view_products", "edit_products",
		"view_orders", "edit_orders",
		"view_roles",
	},
	"staff": {
		"view_users",
		"view_products", "edit_products",
		"view_orders", "edit_orders",
	},
}

// Idempotent: never resets an existing owner's password. But every run
// replaces owner/admin/staff's permission grants with the hardcoded
// defaults above, reverting any customization made through UpdateRole.
func Run(db *gorm.DB, h auth.Hasher, ownerEmail, ownerPassword string) error {
	if ownerEmail == "" {
		return errors.New("owner email is required (set OWNER_EMAIL)")
	}
	if ownerPassword == "" {
		return errors.New("owner password is required (set OWNER_PASSWORD)")
	}

	return db.Transaction(func(tx *gorm.DB) error {
		permissions := make(map[string]models.Permission, len(AllPermissions))
		for _, name := range AllPermissions {
			var p models.Permission
			if err := tx.Where(models.Permission{Name: name}).
				FirstOrCreate(&p, models.Permission{Name: name}).Error; err != nil {
				return fmt.Errorf("permission %q: %w", name, err)
			}
			permissions[name] = p
		}

		roles := make(map[string]models.Role, len(roleGrants))
		for roleName, granted := range roleGrants {
			var role models.Role
			if err := tx.Where(models.Role{Name: roleName}).
				FirstOrCreate(&role, models.Role{Name: roleName}).Error; err != nil {
				return fmt.Errorf("role %q: %w", roleName, err)
			}

			attach := make([]models.Permission, 0, len(granted))
			for _, name := range granted {
				attach = append(attach, permissions[name])
			}
			// Replace, not Append: re-running must converge, not accumulate.
			if err := tx.Model(&role).Association("Permissions").Replace(attach); err != nil {
				return fmt.Errorf("grant permissions to %q: %w", roleName, err)
			}
			roles[roleName] = role
		}

		var existing models.User
		err := tx.Where("email = ?", ownerEmail).First(&existing).Error
		if err == nil {
			// Never reset an existing owner's password: that would make every
			// deploy an account takeover.
			return nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("look up owner: %w", err)
		}

		hash, err := h.Hash(ownerPassword)
		if err != nil {
			return fmt.Errorf("hash owner password: %w", err)
		}

		owner := models.User{
			FirstName: "Owner",
			LastName:  "Account",
			Email:     ownerEmail,
			Password:  hash,
			RoleId:    roles["owner"].Id,
		}
		if err := tx.Create(&owner).Error; err != nil {
			return fmt.Errorf("create owner: %w", err)
		}
		return nil
	})
}
