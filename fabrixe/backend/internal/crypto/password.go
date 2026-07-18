// Package crypto provides password hashing using PBKDF2-HMAC-SHA256.
// Uses only Go standard library — no external dependencies.
package crypto

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

const (
	pbkdf2Iterations = 120_000
	saltLen          = 16
	keyLen           = 32
	hashVersion      = "v1"
)

// pbkdf2Key derives a key using PBKDF2 with HMAC-SHA256.
// This is RFC 2898 / PKCS#5 v2.0.
func pbkdf2Key(password, salt []byte, iter, keyLen int) []byte {
	prf := hmac.New(sha256.New, password)
	hashLen := prf.Size()
	numBlocks := (keyLen + hashLen - 1) / hashLen

	var buf [4]byte
	dk := make([]byte, 0, numBlocks*hashLen)
	U := make([]byte, hashLen)
	for block := 1; block <= numBlocks; block++ {
		prf.Reset()
		_, _ = prf.Write(salt)
		buf[0] = byte(block >> 24)
		buf[1] = byte(block >> 16)
		buf[2] = byte(block >> 8)
		buf[3] = byte(block)
		_, _ = prf.Write(buf[:4])
		dk = prf.Sum(dk)
		T := dk[len(dk)-hashLen:]
		copy(U, T)
		for n := 2; n <= iter; n++ {
			prf.Reset()
			_, _ = prf.Write(U)
			U = U[:0]
			U = prf.Sum(U)
			for x := range U {
				T[x] ^= U[x]
			}
		}
	}
	return dk[:keyLen]
}

// HashPassword derives a hash from a plaintext password.
// Returns a string in the form: pbkdf2$v1$<iterations>$<salt_hex>$<hash_hex>
func HashPassword(password string) (string, error) {
	salt := make([]byte, saltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("generating salt: %w", err)
	}
	dk := pbkdf2Key([]byte(password), salt, pbkdf2Iterations, keyLen)
	return fmt.Sprintf("pbkdf2$%s$%d$%s$%s",
		hashVersion,
		pbkdf2Iterations,
		hex.EncodeToString(salt),
		hex.EncodeToString(dk),
	), nil
}

// CompareHashAndPassword compares a hash string against a plaintext password.
// Returns nil on match, an error on mismatch.
func CompareHashAndPassword(hash, password string) error {
	parts := strings.Split(hash, "$")
	if len(parts) != 5 || parts[0] != "pbkdf2" {
		// Support legacy bcrypt hashes (starts with $2a$ or $2b$)
		// During a migration window we treat them as invalid so users must reset.
		return errors.New("unsupported hash format — please reset password")
	}
	iter, err := strconv.Atoi(parts[2])
	if err != nil {
		return errors.New("invalid iteration count")
	}
	salt, err := hex.DecodeString(parts[3])
	if err != nil {
		return errors.New("invalid salt encoding")
	}
	stored, err := hex.DecodeString(parts[4])
	if err != nil {
		return errors.New("invalid hash encoding")
	}
	computed := pbkdf2Key([]byte(password), salt, iter, keyLen)
	if subtle.ConstantTimeCompare(computed, stored) != 1 {
		return errors.New("password mismatch")
	}
	return nil
}
