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

	req, err := http.NewRequest("POST", "http://clip_server:8001/clip/encode", &buf)
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
	for _, v := range vec {
		sum += float64(v * v)
	}
	norm := float32(math.Sqrt(sum))
	if norm == 0 {
		return vec
	}
	for i := range vec {
		vec[i] /= norm
	}
	return vec
}
