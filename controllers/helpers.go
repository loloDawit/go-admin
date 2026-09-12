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

// MySQL's two "row is referenced" error numbers: 1451 on the parent side
// (deleting a row another table still references), 1452 on the child side.
// Deletes in this API only trigger 1451.
const (
	mysqlRowIsReferenced        = 1451
	mysqlNoReferencedRowFailure = 1452
)

func isForeignKeyError(err error) bool {
	var mysqlErr *mysql.MySQLError
	if !errors.As(err, &mysqlErr) {
		return false
	}
	return mysqlErr.Number == mysqlRowIsReferenced || mysqlErr.Number == mysqlNoReferencedRowFailure
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
	return httpx.Fail(ctx, notFoundOrDBErrorPlain(err, resource))
}

// callerPermissions returns the calling user's granted permission names, by
// name, keyed for O(1) lookup.
func callerPermissions(ctx *fiber.Ctx) (map[string]bool, error) {
	callerId, err := currentUserId(ctx)
	if err != nil {
		return nil, errs.Unauthenticated
	}

	var caller models.User
	if err := database.DB.First(&caller, callerId).Error; err != nil {
		return nil, errs.SessionInvalid.Wrap(err)
	}

	var callerRole models.Role
	if err := database.DB.Preload("Permissions").First(&callerRole, caller.RoleId).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errs.NoRoleAssigned
		}
		return nil, errs.Database.Wrap(err)
	}

	granted := make(map[string]bool, len(callerRole.Permissions))
	for _, p := range callerRole.Permissions {
		granted[p.Name] = true
	}
	return granted, nil
}

// permissionsSubsetOf reports whether every permission in target is granted.
func permissionsSubsetOf(granted map[string]bool, target []models.Permission) bool {
	for _, p := range target {
		if !granted[p.Name] {
			return false
		}
	}
	return true
}

// ensureCanAssignRole rejects a target role whose permission set is not a
// subset of the caller's own; otherwise edit_users lets any holder mint an
// owner.
func ensureCanAssignRole(ctx *fiber.Ctx, target models.Role) error {
	granted, err := callerPermissions(ctx)
	if err != nil {
		return err
	}
	if !permissionsSubsetOf(granted, target.Permissions) {
		return errs.Forbidden
	}
	return nil
}

// ensureCanModifyUser rejects modifying or deleting a target user whose role
// holds permissions the caller lacks — otherwise the roleId subset check in
// ensureCanAssignRole only guards which role gets assigned, never who it
// gets taken away from, letting a caller demote or delete a more-privileged
// user outright. A caller acting on their own row is always allowed.
func ensureCanModifyUser(ctx *fiber.Ctx, targetUserId int) error {
	callerId, err := currentUserId(ctx)
	if err != nil {
		return errs.Unauthenticated
	}
	if callerId == targetUserId {
		return nil
	}

	var target models.User
	if err := database.DB.Preload("Role.Permissions").First(&target, targetUserId).Error; err != nil {
		return notFoundOrDBErrorPlain(err, "user")
	}

	granted, err := callerPermissions(ctx)
	if err != nil {
		return err
	}
	if !permissionsSubsetOf(granted, target.Role.Permissions) {
		return errs.Forbidden
	}
	return nil
}

// notFoundOrDBErrorPlain is notFoundOrDBError's error-only counterpart, for
// callers that return an error rather than writing the response themselves.
func notFoundOrDBErrorPlain(err error, resource string) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return errs.NotFound.WithMessage("%s not found", resource)
	}
	return errs.Database.Wrap(err)
}
