package utils

import (
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestCheckPasswordAcceptsCurrentHash(t *testing.T) {
	hash, err := HashPassword("CurrentPassword!1")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	if !CheckPassword("CurrentPassword!1", hash) {
		t.Fatal("current password hash was rejected")
	}
	if CheckPassword("wrong", hash) {
		t.Fatal("wrong password was accepted")
	}
}

func TestCheckPasswordAcceptsLegacyRawBcryptHash(t *testing.T) {
	const password = "LegacyPassword!1"
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("create legacy hash: %v", err)
	}
	if !CheckPassword(password, string(hash)) {
		t.Fatal("legacy raw bcrypt hash was rejected")
	}
	if CheckPassword("wrong", string(hash)) {
		t.Fatal("wrong password was accepted for legacy hash")
	}
}
