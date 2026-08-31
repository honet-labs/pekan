package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"unicode"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrPasswordTooShort = errors.New("password must be at least 8 characters long")
	ErrPasswordNoUpper  = errors.New("password must contain at least one uppercase letter")
	ErrPasswordNoLower  = errors.New("password must contain at least one lowercase letter")
	ErrPasswordNoNumber = errors.New("password must contain at least one number")
	ErrPasswordNoSymbol = errors.New("password must contain at least one special character")
)

func ValidatePasswordComplexity(pwd string) error {
	if len(pwd) < 8 {
		return ErrPasswordTooShort
	}
	var hasUpper, hasLower, hasNumber, hasSymbol bool
	for _, char := range pwd {
		switch {
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsLower(char):
			hasLower = true
		case unicode.IsNumber(char):
			hasNumber = true
		case unicode.IsPunct(char) || unicode.IsSymbol(char):
			hasSymbol = true
		}
	}
	if !hasUpper {
		return ErrPasswordNoUpper
	}
	if !hasLower {
		return ErrPasswordNoLower
	}
	if !hasNumber {
		return ErrPasswordNoNumber
	}
	if !hasSymbol {
		return ErrPasswordNoSymbol
	}
	return nil
}

// Argon2id parameters
const (
	argonMemory      = 64 * 1024
	argonIterations  = 3
	argonParallelism = 2
	argonSaltLength  = 16
	argonKeyLength   = 32
)

func generateSalt(length int) ([]byte, error) {
	salt := make([]byte, length)
	if _, err := rand.Read(salt); err != nil {
		return nil, err
	}
	return salt, nil
}

// HashPassword applies complexity validation and then hashes using Bcrypt.
func HashPassword(raw string) (string, error) {
	if err := ValidatePasswordComplexity(raw); err != nil {
		return "", err
	}
	bytes, err := bcrypt.GenerateFromPassword([]byte(raw), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// VerifyPassword dynamically verifies either an Argon2id or a legacy bcrypt hash.
func VerifyPassword(hash, raw string) error {
	hash = strings.TrimSpace(hash)
	raw = strings.TrimSpace(raw)
	if strings.HasPrefix(hash, "$argon2id$") {
		return verifyArgon2id(hash, raw)
	}
	
	// Fallback to bcrypt for existing and new users
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(raw))
}

func verifyArgon2id(encodedHash, raw string) error {
	encodedHash = strings.TrimSpace(encodedHash)
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 {
		return errors.New("invalid argon2id hash format")
	}

	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil {
		return err
	}
	
	var memory, time, parallelism uint32
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &time, &parallelism); err != nil {
		return err
	}

	decodeB64 := func(s string) ([]byte, error) {
		if b, err := base64.RawStdEncoding.DecodeString(s); err == nil {
			return b, nil
		}
		return base64.StdEncoding.DecodeString(s)
	}

	salt, err := decodeB64(parts[4])
	if err != nil {
		return fmt.Errorf("failed to decode salt: %w", err)
	}

	decodedHash, err := decodeB64(parts[5])
	if err != nil {
		return fmt.Errorf("failed to decode hash: %w", err)
	}
	keyLen := uint32(len(decodedHash))

	hashToVerify := argon2.IDKey([]byte(raw), salt, time, memory, uint8(parallelism), keyLen)
	
	if subtle.ConstantTimeCompare(decodedHash, hashToVerify) == 1 {
		return nil
	}
	return errors.New("crypto/argon2id: hashedPassword is not the hash of the given password")
}
