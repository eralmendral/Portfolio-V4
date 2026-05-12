package contact

import (
	"crypto/rand"
	"encoding/hex"
)

func newID() string {
	var bytes [16]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return ""
	}
	return "contact_" + hex.EncodeToString(bytes[:])
}
