package utils

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// MaxPasswordBytes bounds the raw password length accepted before hashing.
// It prevents abusing extremely long inputs for cpu/memory amplification.
const MaxPasswordBytes = 512

// prehashPassword maps the raw password onto a fixed-length 32-byte digest
// before bcrypt. bcrypt only uses the first 72 bytes of its input, so without
// this a password longer than 72 bytes would either fail or silently hash an
// unrelated prefix; with it, any length up to MaxPasswordBytes is safe.
func prehashPassword(password string) []byte {
	sum := sha256.Sum256([]byte(password))
	return sum[:]
}

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword(prehashPassword(password), bcrypt.DefaultCost)
	return string(bytes), err
}

func CheckPassword(password, hash string) bool {
	hashBytes := []byte(hash)
	if bcrypt.CompareHashAndPassword(hashBytes, prehashPassword(password)) == nil {
		return true
	}
	// Backward compatibility: releases before password pre-hashing was added
	// stored bcrypt(rawPassword). Keep accepting those hashes so an upgrade does
	// not lock out an existing admin or invalidate protected file passwords.
	return bcrypt.CompareHashAndPassword(hashBytes, []byte(password)) == nil
}

// GenerateCode generates a random alphanumeric code of given length
// Uses rejection sampling to avoid modulo bias in the random byte distribution.
func GenerateCode(length int) string {
	return generateRandomCode(length, "abcdefghijklmnopqrstuvwxyz0123456789")
}

// GenerateReceiveCode generates an uppercase, human-friendly code. Characters
// that are commonly confused when typed (0/O and 1/I/L) are intentionally
// omitted.
func GenerateReceiveCode(length int) string {
	return generateRandomCode(length, "ABCDEFGHJKLMNPQRSTUVWXYZ23456789")
}

func generateRandomCode(length int, chars string) string {
	if length <= 0 || len(chars) == 0 {
		return ""
	}
	charsLen := len(chars)
	// Largest multiple of charsLen that fits in a byte (0-255)
	maxValid := 256 - (256 % charsLen)
	b := make([]byte, length)
	for i := 0; i < length; i++ {
		var bb byte
		for {
			if _, err := rand.Read(b[i : i+1]); err != nil {
				// crypto/rand should never fail; fall back to first char
				bb = chars[0]
				break
			}
			bb = b[i]
			if int(bb) < maxValid {
				break
			}
		}
		b[i] = chars[int(bb)%charsLen]
	}
	return string(b)
}

// GenerateToken generates a random hex token
func GenerateToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// FormatFileSize formats bytes to human-readable string
func FormatFileSize(size int64) string {
	if size < 1024 {
		return fmt.Sprintf("%d B", size)
	} else if size < 1024*1024 {
		return fmt.Sprintf("%.1f KB", float64(size)/1024)
	} else if size < 1024*1024*1024 {
		return fmt.Sprintf("%.1f MB", float64(size)/(1024*1024))
	}
	return fmt.Sprintf("%.1f GB", float64(size)/(1024*1024*1024))
}
