package auth

import "testing"

func TestValidateUsername(t *testing.T) {
	valid := []string{"admin", "viewer-01", "ops.user", "read_only"}
	for _, value := range valid {
		if err := ValidateUsername(value); err != nil {
			t.Fatalf("expected %q to be valid: %v", value, err)
		}
	}
	invalid := []string{"ab", "contains space", "用户"}
	if NormalizeUsername("UPPER") != "upper" {
		t.Fatal("expected username normalization to lowercase input")
	}
	for _, value := range invalid {
		if err := ValidateUsername(value); err == nil {
			t.Fatalf("expected %q to be invalid", value)
		}
	}
}

func TestValidatePasswordAndRole(t *testing.T) {
	if err := ValidatePassword("12345678"); err != nil {
		t.Fatalf("expected password to be valid: %v", err)
	}
	if err := ValidatePassword("short"); err == nil {
		t.Fatal("expected short password to fail")
	}
	if err := ValidateRole("admin"); err != nil {
		t.Fatal(err)
	}
	if err := ValidateRole("viewer"); err != nil {
		t.Fatal(err)
	}
	if err := ValidateRole("editor"); err == nil {
		t.Fatal("expected unsupported role to fail")
	}
}

func TestHashPassword(t *testing.T) {
	hash, err := HashPassword("correct-horse-battery-staple")
	if err != nil {
		t.Fatal(err)
	}
	if hash == "correct-horse-battery-staple" || len(hash) < 50 {
		t.Fatalf("unexpected bcrypt hash %q", hash)
	}
}
