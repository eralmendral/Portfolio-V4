package intro

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

func (h *Handler) uploadProfilePicture(w http.ResponseWriter, r *http.Request) {
	current, err := h.store.Get(r.Context())
	if err != nil {
		writeHandledError(w, err)
		return
	}

	image, err := h.readSingleUpload(w, r, "image")
	if err != nil {
		writeUploadError(w, err)
		return
	}

	previous := current.ProfilePicture
	current.ProfilePicture = &image
	updated, err := h.store.Save(r.Context(), current)
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

func (h *Handler) deleteProfilePicture(w http.ResponseWriter, r *http.Request) {
	current, err := h.store.Get(r.Context())
	if err != nil {
		writeHandledError(w, err)
		return
	}
	if current.ProfilePicture == nil {
		writeHandledError(w, ErrNotFound)
		return
	}

	removed := *current.ProfilePicture
	current.ProfilePicture = nil
	updated, err := h.store.Save(r.Context(), current)
	if err != nil {
		writeHandledError(w, err)
		return
	}

	h.removeImageFile(removed)
	writeJSON(w, http.StatusOK, updated)
}

func (h *Handler) readSingleUpload(w http.ResponseWriter, r *http.Request, field string) (IntroImage, error) {
	r.Body = http.MaxBytesReader(w, r.Body, h.uploads.MaxBytes+1024)
	if err := r.ParseMultipartForm(h.uploads.MaxBytes); err != nil {
		return IntroImage{}, uploadError{status: http.StatusBadRequest, message: "invalid multipart upload"}
	}

	file, header, err := r.FormFile(field)
	if err != nil {
		return IntroImage{}, uploadError{status: http.StatusBadRequest, message: "image file is required"}
	}
	defer file.Close()

	return h.saveUploadedImage(r.Context(), file, header, r.FormValue("alt_text"), r.FormValue("caption"))
}

func (h *Handler) saveUploadedImage(ctx context.Context, file multipart.File, header *multipart.FileHeader, altText string, caption string) (IntroImage, error) {
	data, err := io.ReadAll(io.LimitReader(file, h.uploads.MaxBytes+1))
	if err != nil {
		return IntroImage{}, err
	}
	if int64(len(data)) > h.uploads.MaxBytes {
		return IntroImage{}, uploadError{status: http.StatusRequestEntityTooLarge, message: fmt.Sprintf("image exceeds the %d MB limit", h.uploads.MaxBytes/(1<<20))}
	}

	contentType := http.DetectContentType(data)
	extension, err := validateImageFile(header.Filename, contentType)
	if err != nil {
		return IntroImage{}, err
	}

	imageID := newID()
	key := filepath.ToSlash(filepath.Join("intro", "profile-picture", imageID+extension))
	stored, err := h.uploads.Store.Put(ctx, key, data, contentType)
	if err != nil {
		return IntroImage{}, err
	}

	width, height := imageDimensions(data)

	return IntroImage{
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

func (h *Handler) removeIntroFiles(intro Intro) {
	if intro.ProfilePicture != nil {
		h.removeImageFile(*intro.ProfilePicture)
	}
}

func (h *Handler) removeImageFile(image IntroImage) {
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
