package main

import "testing"

func TestVerifyPasswordWithBcrypt(t *testing.T) {
	const password = "s3cret-pass"

	strong, err := hashPasswordStrong(password)
	if err != nil {
		t.Fatalf("hashPasswordStrong failed: %v", err)
	}
	if !verifyPassword(password, strong) {
		t.Fatalf("bcrypt password should verify")
	}
	if verifyPassword("wrong", strong) {
		t.Fatalf("bcrypt password should reject wrong value")
	}
}
