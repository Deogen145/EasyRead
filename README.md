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

## API server
- GET ALL url = http://localhost:3000/api/images?page=1
- GET BY ID url = http://localhost:3000/api/images/:id
- GET BY NAME url = http://localhost:3000/api/images/name/:name

- POST UPLOAD url = http://localhost:3000/api/images/upload
  - form-data key = files
- POST UPLOAD CSV url = http://localhost:3000/api/images/uploadCSV
  - form-data key = file (single), files (multiple)

- DELETE url = http://localhost:3000/api/images/:id

## Start all services
docker compose up -d --build

## Stop all services
docker compose down



## CSV Upload Format

The system supports bulk image uploads using a CSV file.

File Requirements

- The file must be in CSV format (.csv)

- The file must contain exactly 2 columns

- The file must not contain empty rows

## Column Structure

1. First column: image filename (e.g., "image1.jpg")
2. Second column: image URL (e.g., "https://example.com/image1.jpg")

## Notes

- The image URL must be accessible (no authentication required).

- Supported image formats: .jpg, .jpeg, .png

- If the image is duplicate (based on embedding similarity threshold), it will not be stored.