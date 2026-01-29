package entities

import "github.com/pgvector/pgvector-go"

type Images_vit_b32norm struct {
	ID            int64           `json:"id"`
	Name          string          `json:"name"`
	Path          string          `json:"path"`
	Img_Embedding pgvector.Vector `json:"embedding"`
	CosineDist    float64         `json:"cosine_dist"`    // ค่าความแตกต่าง
	CosinePercent float64         `json:"cosine_percent"` // ค่าความคล้ายกัน
}
