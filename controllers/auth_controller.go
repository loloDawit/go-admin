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

const (
	sessionCookieName = "jwt"

	// Sets only the cookie's expiry. utils.GenerateJWT hardcodes its own
	// 24-hour token expiry separately; nothing keeps the two in sync — see
	// "Known limitations in M0" in the README.
	sessionTTL = 24 * time.Hour
)

func Login(ctx *fiber.Ctx) error {
	var req httpx.LoginRequest
	if err := ctx.BodyParser(&req); err != nil {
		return httpx.Fail(ctx, errs.InvalidBody.Wrap(err))
	}

	var user models.User
	err := database.DB.Where("email = ?", req.Email).First(&user).Error

	// Both branches must return the same error; see errs.InvalidCredentials.
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

	// The cookie is the only channel; putting the token in the body would
	// expose it to any XSS on the page.
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
	if err := database.DB.Preload("Role.Permissions").First(&user, userId).Error; err != nil {
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

	if err := models.ValidateContact(req.FirstName, req.LastName, req.Email); err != nil {
		return httpx.Fail(ctx, err)
	}

	// Must be a map: GORM's struct form skips zero values, so a struct update
	// silently cannot clear a field.
	result := database.DB.Model(&models.User{Id: userId}).Updates(map[string]any{
		"first_name": req.FirstName,
		"last_name":  req.LastName,
		"email":      req.Email,
	})
	if result.Error != nil {
		if isDuplicateKeyError(result.Error) {
			return httpx.Fail(ctx, errs.EmailTaken.Wrap(result.Error))
		}
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
