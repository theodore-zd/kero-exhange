package crypto

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
)

const (
	passphraseEntropyBytes = 32
)

func GenerateSecurePassphrase() (string, error) {
	bytes := make([]byte, passphraseEntropyBytes)
	if _, err := rand.Read(bytes); err != nil {
		return "", errors.New("failed to generate random bytes")
	}

	return base64.URLEncoding.EncodeToString(bytes), nil
}

func HashAccessToken(walletUUID, passphrase string) string {
	combined := walletUUID + passphrase
	hash := sha256.Sum256([]byte(combined))
	return hex.EncodeToString(hash[:])
}

func HashPassphrase(passphrase string) string {
	hash := sha256.Sum256([]byte(passphrase))
	return hex.EncodeToString(hash[:])
}
