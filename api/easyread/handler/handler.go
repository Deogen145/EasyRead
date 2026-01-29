package handlers

import (
	"fmt"
	"app/easyread/usecases"
	"log"
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
	images, err := h.uc.GetAllImages(c.Context())
	if err != nil {
		log.Println("GetAll error:", err)
		return c.Status(500).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(images)
}

func (h *ImageHandler) Uploaded(c *fiber.Ctx) error {
	img, err := h.uc.UploadImage(c)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"data":    img,
	})
}

func (h *ImageHandler) Delete(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, _ := strconv.ParseInt(idStr, 10, 64)

	if err := h.uc.DeleteImage(c.Context(), id); err != nil {
		return c.Status(404).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": fmt.Sprintf("Deleted image %d", id),
	})
}

