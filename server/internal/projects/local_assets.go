package projects

import (
	"context"
	"os"
	pathpkg "path"
	"path/filepath"
	"strings"
)

type LocalAssetStore struct {
	Dir     string
	BaseURL string
}

func (s LocalAssetStore) Put(_ context.Context, key string, data []byte, _ string) (StoredAsset, error) {
	destination := filepath.Join(s.Dir, filepath.FromSlash(key))
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return StoredAsset{}, err
	}
	if err := os.WriteFile(destination, data, 0o644); err != nil {
		return StoredAsset{}, err
	}

	baseURL := strings.TrimRight(firstNonEmpty(s.BaseURL, "/uploads/projects"), "/")
	urlPath := pathpkg.Join(baseURL, key)
	if strings.HasPrefix(baseURL, "/") && !strings.HasPrefix(urlPath, "/") {
		urlPath = "/" + urlPath
	}

	return StoredAsset{
		URL:  urlPath,
		Path: key,
	}, nil
}

func (s LocalAssetStore) Delete(_ context.Context, key string) error {
	if key == "" {
		return nil
	}
	return os.Remove(filepath.Join(s.Dir, filepath.FromSlash(key)))
}
