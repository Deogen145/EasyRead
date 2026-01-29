package repositories

import (
	"context"
	"fmt"

	"app/easyread/entities"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pgvector/pgvector-go"
)

type ImageRepository interface {
	GetAll(ctx context.Context) ([]entities.Images_vit_b32norm, error)
	Insert(ctx context.Context, img entities.Images_vit_b32norm) (entities.Images_vit_b32norm, error)
	Delete(ctx context.Context, id int64) error
	SearchByVector(ctx context.Context, queryVec []float32, limit int) ([]entities.Images_vit_b32norm, error)
}

type pgImageRepository struct {
	db *pgxpool.Pool
}

func NewImageRepository(db *pgxpool.Pool) ImageRepository {
	return &pgImageRepository{db: db}
}

func (r *pgImageRepository) GetAll(ctx context.Context) ([]entities.Images_vit_b32norm, error) {
	rows, err := r.db.Query(ctx, `SELECT id, name, path, img_embedding FROM Images_vit_b32norm;`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var images []entities.Images_vit_b32norm

	for rows.Next() {
		var img entities.Images_vit_b32norm

		if err := rows.Scan(
			&img.ID,
			&img.Name,
			&img.Path,
			&img.Img_Embedding,
		); err != nil {
			return nil, err
		}

		images = append(images, img)
	}

	return images, nil
}

func (r *pgImageRepository) Insert(ctx context.Context, img entities.Images_vit_b32norm) (entities.Images_vit_b32norm, error) {
	var id int64
	err := r.db.AcquireFunc(ctx, func(c *pgxpool.Conn) error {
		return c.QueryRow(ctx,
			`INSERT INTO Images_vit_b32norm (name, path, img_embedding) VALUES ($1, $2, $3) RETURNING id`,
			img.Name,
			img.Path,
			img.Img_Embedding,
		).Scan(&id)
	})
	if err != nil {
		return entities.Images_vit_b32norm{}, err
	}

	img.ID = id
	return img, nil
}

func (r *pgImageRepository) Delete(ctx context.Context, id int64) error {
	cmd, err := r.db.Exec(
		ctx,
		`DELETE FROM Images_vit_b32norm WHERE id = $1`,
		id,
	)
	if err != nil {
		return err
	}

	if cmd.RowsAffected() == 0 {
		return fmt.Errorf("image %d not found", id)
	}

	return nil
}

func (r *pgImageRepository) SearchByVector(
	ctx context.Context,
	queryVec []float32, // รับเวกเตอร์ที่เราต้องการเปรียบเทียบ
	limit int,
) ([]entities.Images_vit_b32norm, error) {
	// สร้าง pgvector จาก queryVec
	vec := pgvector.NewVector(queryVec)

	// สร้าง SQL Query ที่ใช้คำนวณ cosine distance และคล้ายกัน
	rows, err := r.db.Query(ctx, `
		SELECT
			id,
			name,
			img_embedding,
			img_embedding <=> $1::vector AS cosine_dist,
			ROUND(((1 - (img_embedding <=> $1::vector)) * 100)::numeric, 2) AS cosine_percent
		FROM Images_vit_b32norm
		ORDER BY cosine_dist
		LIMIT $2
	`, vec, limit) // ส่งเวกเตอร์และ limit ไปที่ฐานข้อมูล
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var images []entities.Images_vit_b32norm
	for rows.Next() {
		var img entities.Images_vit_b32norm
		// ดึงข้อมูลจากผลลัพธ์
		if err := rows.Scan(
			&img.ID,
			&img.Name,
			&img.Img_Embedding,
			&img.CosineDist,
			&img.CosinePercent,
		); err != nil {
			return nil, err
		}
		images = append(images, img)
	}

	return images, nil
}
