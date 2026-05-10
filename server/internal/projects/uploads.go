package projects

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
)

type UploadConfig struct {
	Store    AssetStore
	Dir      string
	BaseURL  string
	MaxBytes int64
}

func (c UploadConfig) withDefaults() UploadConfig {
	if c.Store == nil {
		c.Store = LocalAssetStore{
			Dir:     firstNonEmpty(c.Dir, filepath.Join("public", "uploads", "projects")),
			BaseURL: firstNonEmpty(c.BaseURL, "/uploads/projects"),
		}
	}
	if c.MaxBytes <= 0 {
		c.MaxBytes = 300 << 20
	}
	return c
}

type AssetStore interface {
	Put(context.Context, string, []byte, string) (StoredAsset, error)
	Delete(context.Context, string) error
}

type StoredAsset struct {
	URL  string
	Path string
}

func (h *Handler) uploadMainImage(w http.ResponseWriter, r *http.Request, idOrSlug string) {
	project, err := h.store.Get(r.Context(), idOrSlug)
	if err != nil {
		writeHandledError(w, err)
		return
	}

	image, err := h.readSingleUpload(w, r, project.ID, "image")
	if err != nil {
		writeUploadError(w, err)
		return
	}

	var previous *ProjectImage
	updated, err := h.store.Update(r.Context(), idOrSlug, func(project *Project) error {
		if project.MainImage != nil {
			imageCopy := *project.MainImage
			previous = &imageCopy
		}
		project.MainImage = &image
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

func (h *Handler) uploadGalleryImages(w http.ResponseWriter, r *http.Request, idOrSlug string) {
	project, err := h.store.Get(r.Context(), idOrSlug)
	if err != nil {
		writeHandledError(w, err)
		return
	}

	images, err := h.readGalleryUploads(w, r, project.ID)
	if err != nil {
		writeUploadError(w, err)
		return
	}

	updated, err := h.store.Update(r.Context(), idOrSlug, func(project *Project) error {
		nextSortOrder := len(project.Images)
		for i := range images {
			if images[i].SortOrder == 0 {
				images[i].SortOrder = nextSortOrder
				nextSortOrder++
			}
			project.Images = append(project.Images, images[i])
		}
		return nil
	})
	if err != nil {
		for _, image := range images {
			h.removeImageFile(image)
		}
		writeHandledError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, updated)
}

func (h *Handler) deleteProjectImage(w http.ResponseWriter, r *http.Request, idOrSlug string, imageID string) {
	var removed ProjectImage
	updated, err := h.store.Update(r.Context(), idOrSlug, func(project *Project) error {
		if project.MainImage != nil && project.MainImage.ID == imageID {
			removed = *project.MainImage
			project.MainImage = nil
			return nil
		}

		for i, image := range project.Images {
			if image.ID == imageID {
				removed = image
				project.Images = append(project.Images[:i], project.Images[i+1:]...)
				return nil
			}
		}

		return ErrNotFound
	})
	if err != nil {
		writeHandledError(w, err)
		return
	}

	h.removeImageFile(removed)
	writeJSON(w, http.StatusOK, updated)
}

func (h *Handler) readSingleUpload(w http.ResponseWriter, r *http.Request, projectID string, field string) (ProjectImage, error) {
	r.Body = http.MaxBytesReader(w, r.Body, h.uploads.MaxBytes+1024)
	if err := r.ParseMultipartForm(h.uploads.MaxBytes); err != nil {
		return ProjectImage{}, uploadError{status: http.StatusBadRequest, message: "invalid multipart upload"}
	}

	file, header, err := r.FormFile(field)
	if err != nil {
		return ProjectImage{}, uploadError{status: http.StatusBadRequest, message: "image file is required"}
	}
	defer file.Close()

	return h.saveUploadedImage(r.Context(), projectID, file, header, 0, r.FormValue("alt_text"), r.FormValue("caption"))
}

func (h *Handler) readGalleryUploads(w http.ResponseWriter, r *http.Request, projectID string) ([]ProjectImage, error) {
	r.Body = http.MaxBytesReader(w, r.Body, h.uploads.MaxBytes*5+1024)
	if err := r.ParseMultipartForm(h.uploads.MaxBytes * 5); err != nil {
		return nil, uploadError{status: http.StatusBadRequest, message: "invalid multipart upload"}
	}

	files := r.MultipartForm.File["images"]
	if len(files) == 0 {
		files = r.MultipartForm.File["image"]
	}
	if len(files) == 0 {
		return nil, uploadError{status: http.StatusBadRequest, message: "at least one image file is required"}
	}

	images := make([]ProjectImage, 0, len(files))
	for i, header := range files {
		file, err := header.Open()
		if err != nil {
			return nil, err
		}

		image, err := h.saveUploadedImage(r.Context(), projectID, file, header, i, indexedFormValue(r.MultipartForm.Value, "alt_text", i), indexedFormValue(r.MultipartForm.Value, "caption", i))
		_ = file.Close()
		if err != nil {
			return nil, err
		}
		images = append(images, image)
	}

	return images, nil
}

func (h *Handler) saveUploadedImage(ctx context.Context, projectID string, file multipart.File, header *multipart.FileHeader, sortOrder int, altText string, caption string) (ProjectImage, error) {
	data, err := io.ReadAll(io.LimitReader(file, h.uploads.MaxBytes+1))
	if err != nil {
		return ProjectImage{}, err
	}
	if int64(len(data)) > h.uploads.MaxBytes {
		return ProjectImage{}, uploadError{status: http.StatusRequestEntityTooLarge, message: fmt.Sprintf("image exceeds the %d MB limit", h.uploads.MaxBytes/(1<<20))}
	}

	contentType := http.DetectContentType(data)
	extension, err := validateImageFile(header.Filename, contentType)
	if err != nil {
		return ProjectImage{}, err
	}

	imageID := newID()
	key := filepath.ToSlash(filepath.Join(projectID, imageID+extension))
	stored, err := h.uploads.Store.Put(ctx, key, data, contentType)
	if err != nil {
		return ProjectImage{}, err
	}

	width, height := imageDimensions(data)

	return ProjectImage{
		ID:          imageID,
		URL:         stored.URL,
		Path:        stored.Path,
		AltText:     strings.TrimSpace(altText),
		Caption:     strings.TrimSpace(caption),
		ContentType: contentType,
		SizeBytes:   int64(len(data)),
		Width:       width,
		Height:      height,
		SortOrder:   sortOrder,
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

func indexedFormValue(values map[string][]string, key string, index int) string {
	if len(values[key]) > index {
		return values[key][index]
	}
	if len(values[key+"[]"]) > index {
		return values[key+"[]"][index]
	}
	return ""
}

func (h *Handler) removeProjectFiles(project Project) {
	if project.MainImage != nil {
		h.removeImageFile(*project.MainImage)
	}
	for _, image := range project.Images {
		h.removeImageFile(image)
	}
}

func (h *Handler) removeImageFile(image ProjectImage) {
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
