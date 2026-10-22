package handler

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"be-pm/pkg/response"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type UploadHandler struct {
	uploadDir string
}

func NewUploadHandler(uploadDir string) *UploadHandler {
	if uploadDir == "" {
		uploadDir = "./uploads"
	}
	return &UploadHandler{uploadDir: uploadDir}
}

func (h *UploadHandler) UploadImage(c *fiber.Ctx) error {
	file, err := c.FormFile("file")
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Berkas gambar/dokumen wajib disertakan pada field 'file'", err.Error())
	}

	// 1. Check file size (max 2MB)
	if file.Size > 2*1024*1024 {
		return response.Error(c, fiber.StatusBadRequest, "Ukuran berkas melebihi batas maksimal 2 MB", nil)
	}

	// 2. Extension validation
	ext := strings.ToLower(filepath.Ext(file.Filename))
	allowedExts := map[string]bool{
		".jpg":  true,
		".jpeg": true,
		".png":  true,
		".webp": true,
		".pdf":  true,
	}
	if !allowedExts[ext] {
		return response.Error(c, fiber.StatusBadRequest, "Format berkas tidak diizinkan. Hanya JPG, PNG, WEBP, dan PDF yang diperbolehkan.", nil)
	}

	// 3. Magic bytes inspection
	src, err := file.Open()
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Gagal membaca berkas", err.Error())
	}
	defer src.Close()

	buffer := make([]byte, 512)
	n, _ := src.Read(buffer)
	mimeType := http.DetectContentType(buffer[:n])

	allowedMimes := map[string]bool{
		"image/jpeg":      true,
		"image/png":       true,
		"image/webp":      true,
		"application/pdf": true,
	}
	if !allowedMimes[mimeType] {
		return response.Error(c, fiber.StatusBadRequest, fmt.Sprintf("Konten berkas tidak valid (%s). Upload ditolak demi keamanan.", mimeType), nil)
	}

	// Ensure uploads directory exists
	if err := os.MkdirAll(h.uploadDir, os.ModePerm); err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Gagal menyiapkan direktori penyimpanan", err.Error())
	}

	// 4. Safe randomized UUID file name (anti path traversal)
	safeFilename := fmt.Sprintf("%s_%d%s", uuid.New().String(), time.Now().Unix(), ext)
	dst := filepath.Join(h.uploadDir, safeFilename)

	if err := c.SaveFile(file, dst); err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Gagal menyimpan berkas ke server", err.Error())
	}

	protocol := "http"
	if c.Protocol() == "https" {
		protocol = "https"
	}
	publicURL := fmt.Sprintf("%s://%s/uploads/%s", protocol, c.Hostname(), safeFilename)

	return response.Success(c, fiber.StatusOK, "Berkas berhasil diunggah", fiber.Map{
		"url":           publicURL,
		"filename":      safeFilename,
		"original_name": filepath.Base(file.Filename),
		"size":          file.Size,
		"mime_type":     mimeType,
	})
}
