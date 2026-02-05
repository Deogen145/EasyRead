package usecases

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"

	"os"
	"path/filepath"

	"app/easyread/entities"
	"app/easyread/repositories"
	"app/easyread/utils"

	"github.com/gofiber/fiber/v2"
	"github.com/pgvector/pgvector-go"
)

type ImageUsecase interface {
	UploadImage(c *fiber.Ctx) ([]entities.Images_vit_b32norm, error)
	GetAll(ctx context.Context, page, limit int) ([]entities.ImageGET, error)
	GetByID(ctx context.Context, id int64) (entities.ImageGET, error)
	GetByName(ctx context.Context, name string) (entities.ImageGET, error)
	DeleteByID(c *fiber.Ctx) error
}

type imageUsecaseImpl struct {
	repo repositories.ImageRepository
}

func NewImageUsecase(repo repositories.ImageRepository) ImageUsecase {
	return &imageUsecaseImpl{repo: repo}
}

// Upload multiple images + generate embedding + check similarity + save
func (uc *imageUsecaseImpl) UploadImage(c *fiber.Ctx) ([]entities.Images_vit_b32norm, error) {
	ctx := context.Background()

	form, err := c.MultipartForm()
	if err != nil {
		return nil, fmt.Errorf("cannot read multipart form: %v", err)
	}

	// input
	files := form.File["files"]
	if len(files) == 0 {
		return nil, fmt.Errorf("no files uploaded")
	}
	var savedImages []entities.Images_vit_b32norm

	// read
	for _, file := range files {
		if !isAllowed(file.Filename) {
			return nil, fmt.Errorf("only .jpg, .jpeg, .png are allowed")
		}
		opened, err := file.Open()
		if err != nil {
			return nil, err
		}
		defer opened.Close()

		fileBytes, err := io.ReadAll(opened)
		if err != nil {
			return nil, err
		}

		// embedding
		vector, err := utils.CLIPEmbedding(fileBytes)
		if err != nil {
			return nil, fmt.Errorf("embedding error (%s): %v", file.Filename, err)
		}

		// similarity check
		results, err := uc.repo.SearchByVector(ctx, vector, 1)// search top 1
		if err != nil {
			return nil, err
		}

		const threshold = 90.0

		if len(results) > 0 {
			sim := results[0].CosinePercent
			fmt.Printf(
				"[CHECK] %s → %.2f%% (id=%d name=%s)\n",
				file.Filename,
				sim,
				results[0].ID,
				results[0].Name,
			)

			if sim >= threshold {
				return nil, fmt.Errorf(
					"image too similar %.2f%% (id=%d name=%s)",
					sim,
					results[0].ID,
					results[0].Name,
				)
			}
		}

		// save file
		saveDir := "storage/images"
		if err := os.MkdirAll(saveDir, os.ModePerm); err != nil {
			return nil, err
		}

		fileName := filepath.Base(file.Filename)
		savePath := filepath.Join(saveDir, fileName)

		if err := os.WriteFile(savePath, fileBytes, 0644); err != nil {
			return nil, err
		}

		// save DB
		img := entities.Images_vit_b32norm{
			Name:          fileName,
			Path:          "/" + savePath,
			Img_Embedding: pgvector.NewVector(vector),
		}

		savedImg, err := uc.repo.Insert(ctx, img)
		if err != nil {
			return nil, err
		}
		savedImages = append(savedImages, savedImg)
	}
	return savedImages, nil
}

// Get
func (uc *imageUsecaseImpl) GetAll(ctx context.Context, page int, limit int) ([]entities.ImageGET, error) {

	if page < 1 {
		page = 1
	}
	if limit <= 0 || limit > 100 {
		limit = 10
	}

	offset := (page - 1) * limit

	images, err := uc.repo.GetAll(ctx, limit, offset)
	if err != nil {
		return nil, err
	}

	return images, nil
}

func (uc *imageUsecaseImpl) GetByID(ctx context.Context, id int64) (entities.ImageGET, error) {
	img, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return entities.ImageGET{}, err
	}

	return img, nil
}

func (uc *imageUsecaseImpl) GetByName(ctx context.Context, name string) (entities.ImageGET, error) {
	img, err := uc.repo.GetByName(ctx, name)
	if err != nil {
		return entities.ImageGET{}, err
	}

	return img, nil
}

// Delete
func (uc *imageUsecaseImpl) DeleteByID(c *fiber.Ctx) error {
	idParam := c.Params("id")

	id, err := strconv.Atoi(idParam)
	if err != nil {
		return fiber.NewError(
			fiber.StatusBadRequest,
			"id must be a number",
		)
	}

	ctx := c.Context()

	img, err := uc.repo.GetByID(ctx, int64(id))
	if err != nil {
		return fiber.NewError(
			fiber.StatusNotFound,
			"image not found",
		)
	}

	if img.Path != "" {
		filePath := "." + img.Path
		if err := os.Remove(filePath); err != nil {
			if !os.IsNotExist(err) {
				return err
			}
		}
	}

	if err := uc.repo.Delete(ctx, int(id)); err != nil {
		return err
	}

	return nil
}

func isAllowed(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	return ext == ".jpg" || ext == ".jpeg" || ext == ".png"
}
