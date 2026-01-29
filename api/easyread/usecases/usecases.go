package usecases

import (
	"context"
	"fmt"
	"io"

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
	GetAllImages(ctx context.Context) ([]entities.Images_vit_b32norm, error)
	DeleteImage(ctx context.Context, id int64) error
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

	// ---------- input ----------
	file, err := c.FormFile("file")
	if err != nil {
		return nil, fmt.Errorf("no file uploaded: %v", err)
	}

	// ---------- read ----------
	opened, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer opened.Close()

	fileBytes, err := io.ReadAll(opened)
	if err != nil {
		return nil, err
	}

	// ---------- embedding ----------
	vector, err := utils.CLIPEmbedding(fileBytes)
	if err != nil {
		return nil, fmt.Errorf("embedding error (%s): %v", file.Filename, err)
	}

	// ---------- similarity check ----------
	results, err := uc.repo.SearchByVector(ctx, vector, 1)
	if err != nil {
		return nil, err
	}
/////////////////////////////////////////////////////////////////////////////////
	const threshold = 90.0
/////////////////////////////////////////////////////////////////////////////////
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

	// ---------- save file ----------
	saveDir := "storage/images"
	if err := os.MkdirAll(saveDir, os.ModePerm); err != nil {
		return nil, err
	}

	fileName := filepath.Base(file.Filename)
	savePath := filepath.Join(saveDir, fileName)

	if err := os.WriteFile(savePath, fileBytes, 0644); err != nil {
		return nil, err
	}

	// ---------- save DB ----------
	img := entities.Images_vit_b32norm{
		Name:          fileName,
		Path:          "/" + savePath,
		Img_Embedding: pgvector.NewVector(vector),
	}

	savedImg, err := uc.repo.Insert(ctx, img)
	if err != nil {
		return nil, err
	}

	return []entities.Images_vit_b32norm{savedImg}, nil
}

// Get all images
func (uc *imageUsecaseImpl) GetAllImages(
	ctx context.Context,
) ([]entities.Images_vit_b32norm, error) {
	return uc.repo.GetAll(ctx)
}

// Delete image
func (uc *imageUsecaseImpl) DeleteImage(
	ctx context.Context,
	id int64,
) error {
	return uc.repo.Delete(ctx, id)
}
