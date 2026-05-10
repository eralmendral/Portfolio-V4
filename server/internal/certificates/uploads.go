package certificates

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/eralme/server/internal/projects"
)

type UploadConfig struct {
	Store    projects.AssetStore
	Dir      string
	BaseURL  string
	MaxBytes int64
}

func (c UploadConfig) withDefaults() UploadConfig {
	if c.Store == nil {
		c.Store = projects.LocalAssetStore{
			Dir:     firstNonEmpty(c.Dir, filepath.Join("public", "uploads", "projects")),
			BaseURL: firstNonEmpty(c.BaseURL, "/uploads/projects"),
		}
	}
	if c.MaxBytes <= 0 {
		c.MaxBytes = 300 << 20
	}
	return c
}

func (h *Handler) uploadImage(w http.ResponseWriter, r *http.Request, idOrSlug string) {
	certificate, err := h.store.Get(r.Context(), idOrSlug)
	if err != nil {
		writeHandledError(w, err)
		return
	}

	image, err := h.readSingleUpload(w, r, certificate.ID, "image")
	if err != nil {
		writeUploadError(w, err)
		return
	}

	var previous *CertificateImage
	updated, err := h.store.Update(r.Context(), idOrSlug, func(certificate *Certificate) error {
		if certificate.Image != nil {
			imageCopy := *certificate.Image
			previous = &imageCopy
		}
		certificate.Image = &image
		return nil
	})
	if err != nil {
		h.removeImageFile(image)
		writeHandledError(w, err)
		return
	}

	if previous != nil {
		h.removeImageFile(*previous)
	}
	writeJSON(w, http.StatusOK, updated)
}

func (h *Handler) deleteImage(w http.ResponseWriter, r *http.Request, idOrSlug string) {
	var removed *CertificateImage
	updated, err := h.store.Update(r.Context(), idOrSlug, func(certificate *Certificate) error {
		if certificate.Image == nil {
			return ErrNotFound
		}

		imageCopy := *certificate.Image
		removed = &imageCopy
		certificate.Image = nil
		return nil
	})
	if err != nil {
		writeHandledError(w, err)
		return
	}

	if removed != nil {
		h.removeImageFile(*removed)
	}
	writeJSON(w, http.StatusOK, updated)
}

func (h *Handler) readSingleUpload(w http.ResponseWriter, r *http.Request, certificateID string, field string) (CertificateImage, error) {
	r.Body = http.MaxBytesReader(w, r.Body, h.uploads.MaxBytes+1024)
	if err := r.ParseMultipartForm(h.uploads.MaxBytes); err != nil {
		return CertificateImage{}, uploadError{status: http.StatusBadRequest, message: "invalid multipart upload"}
	}

	file, header, err := r.FormFile(field)
	if err != nil {
		return CertificateImage{}, uploadError{status: http.StatusBadRequest, message: "image file is required"}
	}
	defer file.Close()

	return h.saveUploadedImage(r.Context(), certificateID, file, header, r.FormValue("alt_text"), r.FormValue("caption"))
}

func (h *Handler) saveUploadedImage(ctx context.Context, certificateID string, file multipart.File, header *multipart.FileHeader, altText string, caption string) (CertificateImage, error) {
	data, err := io.ReadAll(io.LimitReader(file, h.uploads.MaxBytes+1))
	if err != nil {
		return CertificateImage{}, err
	}
	if int64(len(data)) > h.uploads.MaxBytes {
		return CertificateImage{}, uploadError{status: http.StatusRequestEntityTooLarge, message: fmt.Sprintf("image exceeds the %d MB limit", h.uploads.MaxBytes/(1<<20))}
	}

	contentType := http.DetectContentType(data)
	extension, err := validateImageFile(header.Filename, contentType)
	if err != nil {
		return CertificateImage{}, err
	}

	imageID := newID()
	key := filepath.ToSlash(filepath.Join("certificates", certificateID, imageID+extension))
	stored, err := h.uploads.Store.Put(ctx, key, data, contentType)
	if err != nil {
		return CertificateImage{}, err
	}

	width, height := imageDimensions(data)

	return CertificateImage{
		ID:          imageID,
		URL:         stored.URL,
		Path:        stored.Path,
		AltText:     strings.TrimSpace(altText),
		Caption:     strings.TrimSpace(caption),
		ContentType: contentType,
		SizeBytes:   int64(len(data)),
		Width:       width,
		Height:      height,
		UploadedAt:  time.Now().UTC(),
	}, nil
}

func validateImageFile(filename string, contentType string) (string, error) {
	extension := strings.ToLower(filepath.Ext(filename))
	switch extension {
	case ".png":
		if contentType == "image/png" {
			return ".png", nil
		}
	case ".jpg", ".jpeg":
		if contentType == "image/jpeg" {
			return ".jpg", nil
		}
	}

	return "", uploadError{
		status:  http.StatusBadRequest,
		message: "only .png, .jpg, and .jpeg image files are allowed",
	}
}

func imageDimensions(data []byte) (int, int) {
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return 0, 0
	}

	return cfg.Width, cfg.Height
}

type uploadError struct {
	status  int
	message string
}

func (e uploadError) Error() string {
	return e.message
}

func writeUploadError(w http.ResponseWriter, err error) {
	var upload uploadError
	if errors.As(err, &upload) {
		writeError(w, upload.status, upload.message, nil)
		return
	}

	writeError(w, http.StatusInternalServerError, "could not save upload", nil)
}

func (h *Handler) removeCertificateFiles(certificate Certificate) {
	if certificate.Image != nil {
		h.removeImageFile(*certificate.Image)
	}
}

func (h *Handler) removeImageFile(image CertificateImage) {
	if image.Path == "" {
		return
	}
	_ = h.uploads.Store.Delete(context.Background(), image.Path)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
