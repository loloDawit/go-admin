package controllers

import (
	"errors"
	"strconv"

	"github.com/go-sql-driver/mysql"
	"github.com/gofiber/fiber/v2"
	"github.com/loloDawit/go-admin/database"
	"github.com/loloDawit/go-admin/internal/errs"
	"github.com/loloDawit/go-admin/internal/httpx"
	"github.com/loloDawit/go-admin/models"
	"gorm.io/gorm"
)

// MySQL's "Duplicate entry" error number. gorm's default config does not
// translate driver errors, so this is the only way to distinguish a unique
// constraint violation from any other write failure.
const mysqlDuplicateEntry = 1062

func isDuplicateKeyError(err error) bool {
	var mysqlErr *mysql.MySQLError
	return errors.As(err, &mysqlErr) && mysqlErr.Number == mysqlDuplicateEntry
}

func pathId(ctx *fiber.Ctx) (int, error) {
	id, err := strconv.Atoi(ctx.Params("id"))
	if err != nil || id <= 0 {
		return 0, errs.InvalidID
	}
	return id, nil
}

// Callers must use First, not Find: Find returns a zero-valued struct and no
// error for a missing row.
func notFoundOrDBError(ctx *fiber.Ctx, err error, resource string) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return httpx.Fail(ctx, errs.NotFound.WithMessage("%s not found", resource))
	}
	return httpx.Fail(ctx, errs.Database.Wrap(err))
}

// ensureCanAssignRole rejects a target role whose permission set is not a
// subset of the caller's own; otherwise edit_users lets any holder mint an
// owner.
func ensureCanAssignRole(ctx *fiber.Ctx, target models.Role) error {
	callerId, err := currentUserId(ctx)
	if err != nil {
		return errs.Unauthenticated
	}

	var caller models.User
	if err := database.DB.First(&caller, callerId).Error; err != nil {
		return errs.Database.Wrap(err)
	}

	var callerRole models.Role
	if err := database.DB.Preload("Permissions").First(&callerRole, caller.RoleId).Error; err != nil {
		return errs.Database.Wrap(err)
	}

	granted := make(map[string]bool, len(callerRole.Permissions))
	for _, p := range callerRole.Permissions {
		granted[p.Name] = true
	}

	for _, p := range target.Permissions {
		if !granted[p.Name] {
			return errs.Forbidden
		}
	}
	return nil
}
