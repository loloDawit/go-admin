package middlewares

import (
	"fmt"
	"github.com/gofiber/fiber/v2"
	"github.com/loloDawit/go-admin/database"
	"github.com/loloDawit/go-admin/internal/errs"
	"github.com/loloDawit/go-admin/models"
	"github.com/loloDawit/go-admin/utils"
	"strconv"
)

func IsAuthorized(ctx *fiber.Ctx, page string) error {

	cookie := ctx.Cookies("jwt")

	Id, err := utils.ParseJWT(cookie)
	if err != nil {
		return err
	}

	userId, _ := strconv.Atoi(Id)

	user := models.User{
		Id: userId,
	}

	database.DB.Preload("Role").Find(&user)

	role := models.Role{
		Id: user.RoleId,
	}

	database.DB.Preload("Permissions").Find(&role)

	if ctx.Method() == "GET" {
		for _, permission := range role.Permissions {
			fmt.Println(permission.Name, page)
			if permission.Name == "view_"+page || permission.Name == "edit_"+page {
				return nil
			}
		}
	} else {
		for _, permission := range role.Permissions {
			if permission.Name == "edit_"+page {
				return nil
			}
		}
	}

	return errs.Forbidden
}
