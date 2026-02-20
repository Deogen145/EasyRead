package handlers

import (
	"app/easyread/usecases"
	"fmt"
	"strconv"

	"github.com/gofiber/fiber/v2"
)

type ImageHandler struct {
	uc usecases.ImageUsecase
}

func NewImageHandler(uc usecases.ImageUsecase) *ImageHandler {
	return &ImageHandler{uc: uc}
}

func (h *ImageHandler) GetAll(c *fiber.Ctx) error {
	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 10)

	images, err := h.uc.GetAll(c.Context(), page, limit)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(images)
}

func (h *ImageHandler) GetByID(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid id"})
	}

	img, err := h.uc.GetByID(c.Context(), int64(id))
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(img)
}

func (h *ImageHandler) GetByName(c *fiber.Ctx) error {
	name := c.Params("name")

	img, err := h.uc.GetByName(c.Context(), name)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(img)
}

func (h *ImageHandler) Uploaded(c *fiber.Ctx) error {
	img, err := h.uc.UploadImage(c)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"data": img,
	})
}

func (h *ImageHandler) UploadCSV(c *fiber.Ctx) error {
	count, err := h.uc.UploadCSV(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"inserted": count,
	})
}


func (h *ImageHandler) Delete(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, _ := strconv.ParseInt(idStr, 10, 64)

	if err := h.uc.DeleteByID(c); err != nil {
		return c.Status(404).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": fmt.Sprintf("Deleted image %d", id),
	})
}
