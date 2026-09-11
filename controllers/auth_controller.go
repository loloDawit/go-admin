package controllers

import (
	"errors"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/loloDawit/go-admin/database"
	"github.com/loloDawit/go-admin/internal/errs"
	"github.com/loloDawit/go-admin/internal/httpx"
	"github.com/loloDawit/go-admin/models"
	"github.com/loloDawit/go-admin/utils"
	"gorm.io/gorm"
)

// sessionCookieName is the one place this name is written. It was previously
// repeated as the literal "jwt" in five handlers and both middlewares.
const sessionCookieName = "jwt"

// sessionTTL is how long a session lasts. M4 replaces this with a revocable
// server-side session; until then the JWT's own expiry and the cookie's must
// agree, so both derive from this constant.
const sessionTTL = 24 * time.Hour

// Register creates an account.
//
// NOTE: this endpoint is removed entirely in Task 6 per
// docs/decisions/0001-remove-public-registration.md — it is public and grants
// RoleId 1, which is the admin role. It is kept here only so the repository
// is never in a state with no way to create a user at all; the seed command
// arrives in the same task that deletes this.
func Register(ctx *fiber.Ctx) error {
	var req httpx.RegisterRequest
	if err := ctx.BodyParser(&req); err != nil {
		return httpx.Fail(ctx, errs.InvalidBody.Wrap(err))
	}

	if req.Password != req.PasswordConfirm {
		return httpx.Fail(ctx, errs.PasswordMismatch)
	}

	user := models.User{
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Email:     req.Email,
		RoleId:    1,
	}

	if err := user.Validate(); err != nil {
		return httpx.Fail(ctx, errs.ValidationFailed.WithMessage("%s", err))
	}
	if err := user.SetPassword(hasher, req.Password); err != nil {
		return httpx.Fail(ctx, err)
	}

	if err := database.DB.Create(&user).Error; err != nil {
		// Never return the driver error: it names tables and constraints.
		return httpx.Fail(ctx, errs.EmailTaken.Wrap(err))
	}

	return ctx.Status(fiber.StatusCreated).JSON(user)
}

func Login(ctx *fiber.Ctx) error {
	var req httpx.LoginRequest
	if err := ctx.BodyParser(&req); err != nil {
		return httpx.Fail(ctx, errs.InvalidBody.Wrap(err))
	}

	var user models.User
	err := database.DB.Where("email = ?", req.Email).First(&user).Error

	// An unknown email and a wrong password return the SAME error, so the
	// endpoint cannot be used to enumerate accounts (ASSESSMENT 4i). The
	// original returned 404 "user not found" versus 400 "password|email is
	// incorrect", which told an attacker exactly which emails were registered.
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return httpx.Fail(ctx, errs.Database.Wrap(err))
		}
		return httpx.Fail(ctx, errs.InvalidCredentials)
	}
	if err := user.CompareHashAndPassword(req.Password); err != nil {
		return httpx.Fail(ctx, errs.InvalidCredentials)
	}

	token, err := utils.GenerateJWT(strconv.Itoa(user.Id))
	if err != nil {
		return httpx.Fail(ctx, errs.TokenIssueFailed.Wrap(err))
	}

	setSessionCookie(ctx, token)

	// The token is deliberately absent from the body. The HTTPOnly cookie is
	// the only channel; returning it here would hand it to any XSS on the
	// page, which is the whole thing HTTPOnly exists to prevent.
	return ctx.JSON(fiber.Map{"message": "ok"})
}

func Logout(ctx *fiber.Ctx) error {
	clearSessionCookie(ctx)
	return ctx.JSON(fiber.Map{"message": "ok"})
}

func User(ctx *fiber.Ctx) error {
	userId, err := currentUserId(ctx)
	if err != nil {
		return httpx.Fail(ctx, errs.Unauthenticated)
	}

	var user models.User
	if err := database.DB.Preload("Role").First(&user, userId).Error; err != nil {
		return notFoundOrDBError(ctx, err, "user")
	}
	return ctx.JSON(user)
}

func UpdateUserInfo(ctx *fiber.Ctx) error {
	var req httpx.UpdateUserInfoRequest
	if err := ctx.BodyParser(&req); err != nil {
		return httpx.Fail(ctx, errs.InvalidBody.Wrap(err))
	}

	userId, err := currentUserId(ctx)
	if err != nil {
		return httpx.Fail(ctx, errs.Unauthenticated)
	}

	if req.FirstName == "" || req.LastName == "" {
		return httpx.Fail(ctx, errs.MissingField.WithMessage("first and last name are required"))
	}
	if req.Email == "" {
		return httpx.Fail(ctx, errs.MissingField.WithMessage("email is required"))
	}

	// Updates with a MAP, not a struct. GORM's struct form skips zero values,
	// so a struct update can never clear a field (ASSESSMENT 4p) — and that
	// is precisely what masked the key-casing bug: the empty strings produced
	// by reading data["firstname"] were skipped, so the request reported
	// success and silently changed nothing.
	result := database.DB.Model(&models.User{Id: userId}).Updates(map[string]any{
		"first_name": req.FirstName,
		"last_name":  req.LastName,
		"email":      req.Email,
	})
	if result.Error != nil {
		return httpx.Fail(ctx, errs.Database.Wrap(result.Error))
	}

	var user models.User
	if err := database.DB.First(&user, userId).Error; err != nil {
		return notFoundOrDBError(ctx, err, "user")
	}
	return ctx.JSON(user)
}

func UpdatePassword(ctx *fiber.Ctx) error {
	var req httpx.UpdatePasswordRequest
	if err := ctx.BodyParser(&req); err != nil {
		return httpx.Fail(ctx, errs.InvalidBody.Wrap(err))
	}
	if req.Password != req.PasswordConfirm {
		return httpx.Fail(ctx, errs.PasswordMismatch)
	}

	userId, err := currentUserId(ctx)
	if err != nil {
		return httpx.Fail(ctx, errs.Unauthenticated)
	}

	var user models.User
	if err := user.SetPassword(hasher, req.Password); err != nil {
		return httpx.Fail(ctx, err)
	}

	if err := database.DB.Model(&models.User{Id: userId}).
		Update("password", user.Password).Error; err != nil {
		return httpx.Fail(ctx, errs.Database.Wrap(err))
	}

	return ctx.JSON(fiber.Map{"message": "ok"})
}

// currentUserId reads the authenticated user's id from the session cookie.
// It replaces four copies of the same cookie-parse-and-discard-the-error
// block, each of which treated a malformed token as user id 0.
func currentUserId(ctx *fiber.Ctx) (int, error) {
	issuer, err := utils.ParseJWT(ctx.Cookies(sessionCookieName))
	if err != nil {
		return 0, err
	}
	id, err := strconv.Atoi(issuer)
	if err != nil || id <= 0 {
		return 0, errs.SessionInvalid
	}
	return id, nil
}

func setSessionCookie(ctx *fiber.Ctx, token string) {
	ctx.Cookie(&fiber.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		Expires:  time.Now().Add(sessionTTL),
		HTTPOnly: true,
		SameSite: "Lax",
		Secure:   cookieSecure,
		Path:     "/",
	})
}

func clearSessionCookie(ctx *fiber.Ctx) {
	ctx.Cookie(&fiber.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Expires:  time.Now().Add(-time.Hour),
		HTTPOnly: true,
		SameSite: "Lax",
		Secure:   cookieSecure,
		Path:     "/",
	})
}
