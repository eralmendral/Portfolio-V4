package products

import (
	"crypto/rand"
	"encoding/hex"
	"regexp"
	"strings"
)

var nonSlugCharPattern = regexp.MustCompile(`[^a-z0-9]+`)

func newID() string {
	var bytes [16]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		panic(err)
	}
	return hex.EncodeToString(bytes[:])
}

func slugify(value string) string {
	slug := strings.ToLower(strings.TrimSpace(value))
	slug = nonSlugCharPattern.ReplaceAllString(slug, "-")
	slug = strings.Trim(slug, "-")
	if slug == "" {
		return newID()
	}
	return slug
}
