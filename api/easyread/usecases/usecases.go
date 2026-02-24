package usecases

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"os"
	"path/filepath"

	"app/easyread/entities"
	"app/easyread/repositories"
	"app/easyread/utils"

	"github.com/gofiber/fiber/v2"
	"github.com/pgvector/pgvector-go"
	"github.com/xuri/excelize/v2"
)

type ImageUsecase interface {
	UploadImage(c *fiber.Ctx) (*entities.Images_vit_b32norm, error)
	UploadCSV(c *fiber.Ctx) (int, error)
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
func (uc *imageUsecaseImpl) UploadImage(c *fiber.Ctx) (*entities.Images_vit_b32norm, error) {
	ctx := context.Background()

	// รับไฟล์เดียวจาก key "file"
	file, err := c.FormFile("files")
	if err != nil {
		return nil, fmt.Errorf("no file uploaded")
	}

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
	results, err := uc.repo.SearchByVector(ctx, vector, 1)
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

	return &savedImg, nil
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


func (uc *imageUsecaseImpl) UploadCSV(c *fiber.Ctx) (int, error) {
    ctx := context.Background()

    file, err := c.FormFile("file")
    if err != nil {
        return 0, fiber.NewError(400, "csv required")
    }

    f, err := file.Open()
    if err != nil {
        return 0, err
    }
    defer f.Close()

    reader := csv.NewReader(f)
    reader.Read() // skip header

    // ===== excel
    xlsx := excelize.NewFile()
    sheet := "report"
    xlsx.NewSheet(sheet)

    xlsx.SetCellValue(sheet, "A1", "รูปที่เข้ามา")
    xlsx.SetCellValue(sheet, "B1", "คล้ายกับกับ")
    xlsx.SetCellValue(sheet, "C1", "cosine")

    var mu sync.Mutex
    rowIndex := 2
    inserted := 0

    // ===== worker pool
    workerCount := 5
    jobs := make(chan []string, 100)
    wg := sync.WaitGroup{}

    worker := func() {
        defer wg.Done()

        for record := range jobs {
            filename := record[0]
            url := record[1]

            // download
            imgBytes, err := downloadImage(url)
            if err != nil {
                fmt.Println("download fail:", url)
                continue
            }

            // embedding
            vector, err := utils.CLIPEmbedding(imgBytes)
            if err != nil {
                fmt.Println("embed fail:", filename)
                continue
            }

            // search
            results, _ := uc.repo.SearchByVector(ctx, vector, 1)

            var simName string
            var simPercent float32

            if len(results) > 0 {
                simName = results[0].Name
                simPercent = float32(results[0].CosinePercent)
            }

            // excel log
            mu.Lock()
            xlsx.SetCellValue(sheet, fmt.Sprintf("A%d", rowIndex), filename)
            xlsx.SetCellValue(sheet, fmt.Sprintf("B%d", rowIndex), simName)
            xlsx.SetCellValue(sheet, fmt.Sprintf("C%d", rowIndex), simPercent)
            rowIndex++
            mu.Unlock()

            // skip similar
            if simPercent >= 0 {
                fmt.Println("skip similar:", filename)
                continue
            } 

            // save file
            savePath := "storage/images/" + filename
            os.WriteFile(savePath, imgBytes, 0644)

            img := entities.Images_vit_b32norm{
                Name:          filename,
                Path:          "/" + savePath,
                Img_Embedding: pgvector.NewVector(vector),
            }

            _, err = uc.repo.Insert(ctx, img)
            if err == nil {
                mu.Lock()
                inserted++
                mu.Unlock()
            }
        }
    }

    // start workers
    for i := 0; i < workerCount; i++ {
        wg.Add(1)
        go worker()
    }

    // push jobs
    for {
        record, err := reader.Read()
        if err == io.EOF {
            break
        }
        if err != nil {
            continue
        }
        jobs <- record
    }

    close(jobs)
    wg.Wait()

    // save excel
    reportPath := "storage/report_similarity.xlsx"
    xlsx.SaveAs(reportPath)

    return inserted, nil
}



func downloadImage(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return io.ReadAll(resp.Body)
}
