package utils

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

func CheckPassword(password, hash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

// GenerateCode generates a random alphanumeric code of given length
// Uses rejection sampling to avoid modulo bias in the random byte distribution.
func GenerateCode(length int) string {
	chars := "abcdefghijklmnopqrstuvwxyz0123456789"
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

// ExpireKey returns a unique key for expiry tracking
func ExpireKey(code string) string {
	h := sha256.Sum256([]byte("file_expire:" + code))
	return hex.EncodeToString(h[:8])
}

// ParseSizeString parses a size string like "1GB200MB500kb" into bytes
func ParseSizeString(s string) (int64, error) {
	if s == "" {
		return 0, nil
	}
	var total int64
	var cur int64
	for i := 0; i < len(s); i++ {
		if s[i] >= '0' && s[i] <= '9' {
			cur = cur*10 + int64(s[i]-'0')
		} else if s[i] == 'G' || s[i] == 'g' {
			if i+1 < len(s) && (s[i+1] == 'B' || s[i+1] == 'b') {
				total += cur * 1024 * 1024 * 1024
				cur = 0
				i++
			} else {
				return 0, fmt.Errorf("无效的大小格式: %s", s)
			}
		} else if s[i] == 'M' || s[i] == 'm' {
			if i+1 < len(s) && (s[i+1] == 'B' || s[i+1] == 'b') {
				total += cur * 1024 * 1024
				cur = 0
				i++
			} else {
				return 0, fmt.Errorf("无效的大小格式: %s", s)
			}
		} else if s[i] == 'K' || s[i] == 'k' {
			if i+1 < len(s) && (s[i+1] == 'B' || s[i+1] == 'b') {
				total += cur * 1024
				cur = 0
				i++
			} else {
				return 0, fmt.Errorf("无效的大小格式: %s", s)
			}
		} else {
			return 0, fmt.Errorf("无效的大小格式: %s", s)
		}
	}
	if cur > 0 {
		total += cur
	}
	return total, nil
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
