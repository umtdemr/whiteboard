package token

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base32"
)

const (
	ScopeActivation     = "activation"
	ScopeAuthentication = "authentication"
)

func GenerateToken() (string, []byte, error) {
	randomBytes := make([]byte, 16)

	_, err := rand.Read(randomBytes)
	if err != nil {
		return "", nil, err
	}

	plaintext := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(randomBytes)
	hash := sha256.Sum256([]byte(plaintext))
	return plaintext, hash[:], nil
}
