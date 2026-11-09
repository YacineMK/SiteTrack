package hash

import (
	"strings"
	"testing"
)

func TestHashAndVerify(t *testing.T) {
	const password = "CorrectHorseBatteryStaple!"

	hashed, err := Hash(password)
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}
	if hashed == password {
		t.Fatal("hash must not equal the plaintext password")
	}
	if !strings.HasPrefix(hashed, "$argon2id$") {
		t.Fatalf("expected argon2id hash, got %q", hashed)
	}
	if !strings.Contains(hashed, "m=32768,t=3,p=1") {
		t.Fatalf("unexpected argon2 parameters in %q", hashed)
	}

	match, err := Verify(password, hashed)
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if !match {
		t.Fatal("Verify() = false for the correct password")
	}

	match, err = Verify("wrong-password", hashed)
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if match {
		t.Fatal("Verify() = true for a wrong password")
	}
}

func TestHashUsesUniqueSalt(t *testing.T) {
	first, err := Hash("same-password")
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}
	second, err := Hash("same-password")
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}
	if first == second {
		t.Fatal("two hashes of the same password must differ due to unique salts")
	}
}
