package utils

import (
	"bytes"
	"encoding/json"
	"math"
	"mime/multipart"
	"net/http"
)

type ClipResponse struct {
	Vector []float32 `json:"vector"`
}

func CLIPEmbedding(imageBytes []byte) ([]float32, error) {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	part, err := writer.CreateFormFile("file", "image.jpg")
	if err != nil {
		return nil, err
	}
	part.Write(imageBytes)
	writer.Close()

	req, err := http.NewRequest("POST", "http://clip_server:8001/clip/encode", &buf) // http://clip_server:8001/clip/encode
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var clipResp ClipResponse
	if err := json.NewDecoder(resp.Body).Decode(&clipResp); err != nil {
		return nil, err
	}

	vec := clipResp.Vector
	vec = NormalizeL2(vec)
	return vec, nil

	// return clipResp.Vector, nil
}

func NormalizeL2(vec []float32) []float32 {
	var sum float64
	for _, v := range vec { // คำนวณผลรวมของกำลังสอง
		sum += float64(v * v)
	}
	norm := float32(math.Sqrt(sum)) // คำนวณนอร์ม (L2 norm)
	if norm == 0 {
		return vec // ถ้านอร์มเป็นศูนย์ คืนค่าเวกเตอร์เดิม
	}
	for i := range vec { // ปรับแต่งเวกเตอร์ให้เป็นนอร์ม 1
		vec[i] /= norm // แบ่งแต่ละองค์ประกอบด้วยนอร์ม
	}
	return vec // คืนค่าเวกเตอร์ที่ปรับแต่งแล้ว
}
