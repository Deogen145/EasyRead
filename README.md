# EasyRead

EasyRead is an image similarity detection system using CLIP embeddings and pgvector.
The system prevents duplicate or highly similar images from being uploaded by
comparing vector embeddings with cosine similarity.

---

## Overview

This project allows users to upload images and checks whether the image is too
similar to existing images in the database before saving it.

Key features:
- Image embedding using CLIP (ViT-B/32)
- Vector similarity search using pgvector
- Duplicate / near-duplicate image prevention

---

## Architecture
Client
|
v
API (Go / Fiber)
|
|-- Upload image(s)
|-- Call CLIP server
|-- Similarity check (pgvector)
|-- Save image + embedding
|
PostgreSQL (pgvector)
|
CLIP Server (FastAPI + PyTorch)


---

## Tech Stack

- Backend API: Go (Fiber)
- Embedding Service: Python (FastAPI, PyTorch, CLIP)
- Database: PostgreSQL 16 + pgvector
- Containerization: Docker & Docker Compose
- ML Model: OpenAI CLIP (ViT-B/32)

---

## Services

| Service       | Description                         | Port |
|--------------|-------------------------------------|------|
| api          | Main API server (Go)                | 3000 |
| clip_server  | Image embedding service (CLIP)      | 8001 |
| pgvector     | PostgreSQL + vector extension       | 5433 |

---

## API Flow

1. Client uploads one or multiple images
2. API sends image bytes to CLIP server
3. CLIP server returns image embedding (vector)
4. API finds nearest vectors using pgvector
5. Cosine similarity is calculated
6. If similarity >= threshold → reject upload
7. If similarity < threshold → save image + embedding

---

## Database

### Enable pgvector extension

```sql
CREATE EXTENSION IF NOT EXISTS vector;

CREATE TABLE images_vit_b32norm (
  id BIGSERIAL PRIMARY KEY,
  name TEXT,
  path TEXT,
  img_embedding VECTOR(512)
);

```

## How to Run
Prerequisites
- Docker
- Docker Compose

## Start all services
docker compose up -d --build

## Stop all services
docker compose down

## Upload Multiple Images (Go / Fiber)
func (uc *imageUsecaseImpl) UploadImages(c *fiber.Ctx) ([]entities.Images_vit_b32norm, error) {
	ctx := context.Background()

	form, err := c.MultipartForm()
	if err != nil {
		return nil, err
	}

	files := form.File["files"]
	if len(files) == 0 {
		return nil, fmt.Errorf("no files uploaded")
	}

	var savedImages []entities.Images_vit_b32norm

	for _, file := range files {
		opened, err := file.Open()
		if err != nil {
			return nil, err
		}

		fileBytes, err := io.ReadAll(opened)
		opened.Close()
		if err != nil {
			return nil, err
		}

		// Generate embedding
		vector, err := utils.CLIPEmbedding(fileBytes)
		if err != nil {
			return nil, fmt.Errorf("embedding error (%s): %v", file.Filename, err)
		}

		// Similarity check
		results, err := uc.repo.SearchByVector(ctx, vector, 1)
		if err != nil {
			return nil, err
		}

		const threshold = 90.0
		if len(results) > 0 && results[0].CosinePercent >= threshold {
			return nil, fmt.Errorf(
				"image %s too similar (%.2f%%)",
				file.Filename,
				results[0].CosinePercent,
			)
		}

		// Save file
		saveDir := "storage/images"
		os.MkdirAll(saveDir, os.ModePerm)
		savePath := filepath.Join(saveDir, filepath.Base(file.Filename))
		if err := os.WriteFile(savePath, fileBytes, 0644); err != nil {
			return nil, err
		}

		// Save DB
		img := entities.Images_vit_b32norm{
			Name:          file.Filename,
			Path:          "/" + savePath,
			Img_Embedding: pgvector.NewVector(vector),
		}

		saved, err := uc.repo.Insert(ctx, img)
		if err != nil {
			return nil, err
		}

		savedImages = append(savedImages, saved)
	}

	return savedImages, nil
}
---