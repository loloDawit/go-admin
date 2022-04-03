package controllers

import (
	"github.com/gofiber/fiber/v2"
)

func Upload(ctx *fiber.Ctx) error {
	form, err := ctx.MultipartForm()

	if err != nil {
		return err
	}

	files := form.File["image"]
	filename := ""

	if len(files) == 0 {
		ctx.Status(400)
		return ctx.JSON(fiber.Map{
			"msg": "wrong format",
		})
	}

	for _, file := range files {
		filename = file.Filename

		if err := ctx.SaveFile(file, "./uploads/"+filename); err != nil {
			return err
		}
	}

	return ctx.JSON(fiber.Map{
		"url": "http://localhost:3000/api/v1/uploads/" + filename,
	})
}
